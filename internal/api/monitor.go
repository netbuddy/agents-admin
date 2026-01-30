// Package api 工作流监控 API
//
// 本文件提供工作流监控相关的 HTTP API，聚合 etcd 事件/状态和 PostgreSQL 任务数据，
// 实现一站式全流程监控功能。
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"agents-admin/internal/model"
	"agents-admin/internal/storage"
)

// WorkflowSummary 工作流摘要信息
type WorkflowSummary struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`        // auth, task, run
	Name       string                 `json:"name"`        // 显示名称
	State      string                 `json:"state"`       // 当前状态
	Progress   int                    `json:"progress"`    // 进度百分比 0-100
	EventCount int                    `json:"event_count"` // 事件数量
	StartTime  *time.Time             `json:"start_time"`
	UpdateTime *time.Time             `json:"update_time"`
	EndTime    *time.Time             `json:"end_time,omitempty"`
	Duration   *int64                 `json:"duration_ms,omitempty"` // 持续时间（毫秒）
	NodeID     string                 `json:"node_id,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// WorkflowDetail 工作流详情
type WorkflowDetail struct {
	WorkflowSummary
	Events     []WorkflowEventView    `json:"events"`
	StateData  map[string]interface{} `json:"state_data,omitempty"`
	RelatedIDs map[string]string      `json:"related_ids,omitempty"` // 关联ID（account_id, task_id等）
}

// WorkflowEventView 事件视图
type WorkflowEventView struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Seq        int64                  `json:"seq"`
	Data       map[string]interface{} `json:"data,omitempty"`
	ProducerID string                 `json:"producer_id"`
	Timestamp  time.Time              `json:"timestamp"`
	Level      string                 `json:"level"` // info, warning, error, success
}

// MonitorStats 监控统计
type MonitorStats struct {
	TotalWorkflows   int            `json:"total_workflows"`
	ActiveWorkflows  int            `json:"active_workflows"`
	CompletedToday   int            `json:"completed_today"`
	FailedToday      int            `json:"failed_today"`
	AvgDurationMs    int64          `json:"avg_duration_ms"`
	WorkflowsByType  map[string]int `json:"workflows_by_type"`
	WorkflowsByState map[string]int `json:"workflows_by_state"`
}

// ListWorkflows 列出工作流
//
// 路由: GET /api/v1/monitor/workflows
// 查询参数:
//   - type: 工作流类型过滤 (auth, task, run)
//   - state: 状态过滤 (pending, running, waiting, completed, failed)
//   - limit: 返回数量限制 (默认50)
//   - offset: 分页偏移
func (h *Handler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := r.URL.Query()

	workflowType := query.Get("type")
	state := query.Get("state")
	limit := 50
	offset := 0

	if l := query.Get("limit"); l != "" {
		if _, err := json.Number(l).Int64(); err == nil {
			limit = int(mustParseInt(l, 50))
		}
	}
	if o := query.Get("offset"); o != "" {
		offset = int(mustParseInt(o, 0))
	}

	var workflows []WorkflowSummary

	// 聚合认证任务
	if workflowType == "" || workflowType == "auth" {
		authWorkflows := h.getAuthWorkflows(ctx, state)
		workflows = append(workflows, authWorkflows...)
	}

	// 聚合运行任务
	if workflowType == "" || workflowType == "run" {
		runWorkflows := h.getRunWorkflows(ctx, state)
		workflows = append(workflows, runWorkflows...)
	}

	// 按更新时间排序
	sort.Slice(workflows, func(i, j int) bool {
		if workflows[i].UpdateTime == nil {
			return false
		}
		if workflows[j].UpdateTime == nil {
			return true
		}
		return workflows[i].UpdateTime.After(*workflows[j].UpdateTime)
	})

	// 分页
	total := len(workflows)
	if offset < len(workflows) {
		workflows = workflows[offset:]
	} else {
		workflows = []WorkflowSummary{}
	}
	if len(workflows) > limit {
		workflows = workflows[:limit]
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"workflows": workflows,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

// GetWorkflow 获取工作流详情
//
// 路由: GET /api/v1/monitor/workflows/{type}/{id}
func (h *Handler) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	workflowType := r.PathValue("type")
	workflowID := r.PathValue("id")

	if workflowType == "" || workflowID == "" {
		writeError(w, http.StatusBadRequest, "type and id are required")
		return
	}

	var detail *WorkflowDetail
	var err error

	switch workflowType {
	case "auth":
		detail, err = h.getAuthWorkflowDetail(ctx, workflowID)
	case "run":
		detail, err = h.getRunWorkflowDetail(ctx, workflowID)
	default:
		writeError(w, http.StatusBadRequest, "unsupported workflow type")
		return
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if detail == nil {
		writeError(w, http.StatusNotFound, "workflow not found")
		return
	}

	writeJSON(w, http.StatusOK, detail)
}

// GetMonitorStats 获取监控统计
//
// 路由: GET /api/v1/monitor/stats
func (h *Handler) GetMonitorStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	stats := h.calculateStats(ctx)
	writeJSON(w, http.StatusOK, stats)
}

// GetWorkflowEvents 获取工作流事件流
//
// 路由: GET /api/v1/monitor/workflows/{type}/{id}/events
func (h *Handler) GetWorkflowEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	workflowType := r.PathValue("type")
	workflowID := r.PathValue("id")

	if workflowType == "" || workflowID == "" {
		writeError(w, http.StatusBadRequest, "type and id are required")
		return
	}

	events := h.getWorkflowEvents(ctx, workflowType, workflowID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
		"total":  len(events),
	})
}

// ========== 内部方法 ==========

func (h *Handler) getAuthWorkflows(ctx context.Context, stateFilter string) []WorkflowSummary {
	var workflows []WorkflowSummary

	// 从 Redis 获取认证会话
	sessions, err := h.redisStore.ListAuthSessions(ctx)
	if err != nil {
		sessions = []*storage.AuthSession{}
	}

	for _, session := range sessions {
		state := mapAuthSessionStatus(session.Status)
		if stateFilter != "" && state != stateFilter {
			continue
		}

		summary := WorkflowSummary{
			ID:         session.TaskID,
			Type:       "auth",
			Name:       "OAuth 认证: " + session.AccountID,
			State:      state,
			Progress:   calculateAuthProgressFromStatus(session.Status),
			StartTime:  &session.CreatedAt,
			UpdateTime: &session.CreatedAt,
			NodeID:     session.NodeID,
			Metadata: map[string]interface{}{
				"account_id": session.AccountID,
				"method":     session.Method,
			},
		}

		if session.Message != "" {
			summary.Metadata["message"] = session.Message
		}

		// 计算持续时间
		if session.Status == "success" || session.Status == "failed" {
			duration := time.Since(session.CreatedAt).Milliseconds()
			summary.Duration = &duration
			now := time.Now()
			summary.EndTime = &now
		}

		// 从 Redis Streams 获取事件数量
		if h.redisStore != nil {
			eventCount, _ := h.redisStore.GetEventCount(ctx, "auth", session.TaskID)
			summary.EventCount = int(eventCount)
		}

		workflows = append(workflows, summary)
	}

	return workflows
}

func (h *Handler) getRunWorkflows(ctx context.Context, stateFilter string) []WorkflowSummary {
	var workflows []WorkflowSummary

	// 从 PostgreSQL 获取运行任务（获取所有任务的最近执行）
	tasks, err := h.store.ListTasks(ctx, "", 1000, 0)
	if err != nil {
		return workflows
	}

	var runs []*model.Run
	for _, task := range tasks {
		taskRuns, _ := h.store.ListRunsByTask(ctx, task.ID)
		runs = append(runs, taskRuns...)
	}

	for _, run := range runs {
		state := mapRunStatus(run.Status)
		if stateFilter != "" && state != stateFilter {
			continue
		}

		nodeID := ""
		if run.NodeID != nil {
			nodeID = *run.NodeID
		}

		summary := WorkflowSummary{
			ID:         run.ID,
			Type:       "run",
			Name:       "任务执行: " + run.TaskID,
			State:      state,
			Progress:   calculateRunProgress(run.Status),
			StartTime:  &run.CreatedAt,
			UpdateTime: &run.UpdatedAt,
			NodeID:     nodeID,
			Metadata: map[string]interface{}{
				"task_id": run.TaskID,
			},
		}

		if run.Status == model.RunStatusDone || run.Status == model.RunStatusFailed {
			duration := run.UpdatedAt.Sub(run.CreatedAt).Milliseconds()
			summary.Duration = &duration
			summary.EndTime = &run.UpdatedAt
		}

		if run.Error != nil {
			summary.Error = *run.Error
		}

		// 获取事件数量
		events, _ := h.store.GetEventsByRun(ctx, run.ID, 0, 1000)
		summary.EventCount = len(events)

		workflows = append(workflows, summary)
	}

	return workflows
}

func (h *Handler) getAuthWorkflowDetail(ctx context.Context, id string) (*WorkflowDetail, error) {
	// 从 Redis 获取认证会话
	session, err := h.redisStore.GetAuthSession(ctx, id)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil
	}

	detail := &WorkflowDetail{
		WorkflowSummary: WorkflowSummary{
			ID:         session.TaskID,
			Type:       "auth",
			Name:       "OAuth 认证: " + session.AccountID,
			State:      mapAuthSessionStatus(session.Status),
			Progress:   calculateAuthProgressFromStatus(session.Status),
			StartTime:  &session.CreatedAt,
			UpdateTime: &session.CreatedAt,
			NodeID:     session.NodeID,
			Metadata: map[string]interface{}{
				"account_id": session.AccountID,
				"method":     session.Method,
			},
		},
		RelatedIDs: map[string]string{
			"account_id": session.AccountID,
		},
	}

	if session.OAuthURL != "" {
		detail.Metadata["oauth_url"] = session.OAuthURL
	}
	if session.UserCode != "" {
		detail.Metadata["user_code"] = session.UserCode
	}

	// 从 Redis Streams 获取事件
	if h.redisStore != nil {
		events, _ := h.redisStore.GetEvents(ctx, "auth", id, "", 100)
		for _, evt := range events {
			detail.Events = append(detail.Events, WorkflowEventView{
				ID:        evt.ID,
				Type:      evt.Type,
				Seq:       int64(evt.Seq),
				Data:      evt.Data,
				Timestamp: evt.Timestamp,
				Level:     getEventLevel(evt.Type),
			})
		}
		detail.EventCount = len(detail.Events)

		// 从 Redis 获取状态
		stateData, _ := h.redisStore.GetWorkflowState(ctx, "auth", id)
		if stateData != nil {
			detail.StateData = map[string]interface{}{
				"state":        stateData.State,
				"progress":     stateData.Progress,
				"current_step": stateData.CurrentStep,
				"error":        stateData.Error,
			}
		}
	}

	return detail, nil
}

func (h *Handler) getRunWorkflowDetail(ctx context.Context, id string) (*WorkflowDetail, error) {
	run, err := h.store.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, nil
	}

	nodeID := ""
	if run.NodeID != nil {
		nodeID = *run.NodeID
	}

	detail := &WorkflowDetail{
		WorkflowSummary: WorkflowSummary{
			ID:         run.ID,
			Type:       "run",
			Name:       "任务执行: " + run.TaskID,
			State:      mapRunStatus(run.Status),
			Progress:   calculateRunProgress(run.Status),
			StartTime:  &run.CreatedAt,
			UpdateTime: &run.UpdatedAt,
			NodeID:     nodeID,
			Metadata: map[string]interface{}{
				"task_id": run.TaskID,
			},
		},
		RelatedIDs: map[string]string{
			"task_id": run.TaskID,
		},
	}

	if run.Error != nil {
		detail.Error = *run.Error
	}

	// 从 PostgreSQL 获取事件
	events, _ := h.store.GetEventsByRun(ctx, id, 0, 1000)
	for _, evt := range events {
		var data map[string]interface{}
		if evt.Payload != nil {
			json.Unmarshal(evt.Payload, &data)
		}
		detail.Events = append(detail.Events, WorkflowEventView{
			ID:        fmt.Sprintf("%d", evt.ID),
			Type:      evt.Type,
			Seq:       int64(evt.Seq),
			Data:      data,
			Timestamp: evt.Timestamp,
			Level:     getEventLevel(evt.Type),
		})
	}
	detail.EventCount = len(detail.Events)

	return detail, nil
}

func (h *Handler) getWorkflowEvents(ctx context.Context, workflowType, workflowID string) []WorkflowEventView {
	var events []WorkflowEventView

	switch workflowType {
	case "auth":
		// 从 Redis Streams 获取事件
		if h.redisStore != nil {
			redisEvents, _ := h.redisStore.GetEvents(ctx, "auth", workflowID, "", 100)
			for _, evt := range redisEvents {
				events = append(events, WorkflowEventView{
					ID:        evt.ID,
					Type:      evt.Type,
					Seq:       int64(evt.Seq),
					Data:      evt.Data,
					Timestamp: evt.Timestamp,
					Level:     getEventLevel(evt.Type),
				})
			}
		}
	case "run":
		pgEvents, _ := h.store.GetEventsByRun(ctx, workflowID, 0, 1000)
		for _, evt := range pgEvents {
			var data map[string]interface{}
			if evt.Payload != nil {
				json.Unmarshal(evt.Payload, &data)
			}
			events = append(events, WorkflowEventView{
				ID:        fmt.Sprintf("%d", evt.ID),
				Type:      evt.Type,
				Seq:       int64(evt.Seq),
				Data:      data,
				Timestamp: evt.Timestamp,
				Level:     getEventLevel(evt.Type),
			})
		}
	}

	// 按序列号排序
	sort.Slice(events, func(i, j int) bool {
		return events[i].Seq < events[j].Seq
	})

	return events
}

func (h *Handler) calculateStats(ctx context.Context) MonitorStats {
	stats := MonitorStats{
		WorkflowsByType:  make(map[string]int),
		WorkflowsByState: make(map[string]int),
	}

	today := time.Now().Truncate(24 * time.Hour)

	// 统计认证任务
	authTasks, _ := h.store.ListRecentAuthTasks(ctx, 1000)
	for _, task := range authTasks {
		stats.TotalWorkflows++
		stats.WorkflowsByType["auth"]++

		state := mapAuthTaskStatus(task.Status)
		stats.WorkflowsByState[state]++

		if state == "running" || state == "waiting" || state == "pending" {
			stats.ActiveWorkflows++
		}

		if task.UpdatedAt.After(today) {
			if state == "completed" {
				stats.CompletedToday++
			} else if state == "failed" {
				stats.FailedToday++
			}
		}
	}

	// 统计运行任务
	var runs []*model.Run
	tasks, _ := h.store.ListTasks(ctx, "", 1000, 0)
	for _, task := range tasks {
		taskRuns, _ := h.store.ListRunsByTask(ctx, task.ID)
		runs = append(runs, taskRuns...)
	}
	var totalDuration int64
	var durationCount int

	for _, run := range runs {
		stats.TotalWorkflows++
		stats.WorkflowsByType["run"]++

		state := mapRunStatus(run.Status)
		stats.WorkflowsByState[state]++

		if state == "running" || state == "pending" {
			stats.ActiveWorkflows++
		}

		if run.UpdatedAt.After(today) {
			if state == "completed" {
				stats.CompletedToday++
			} else if state == "failed" {
				stats.FailedToday++
			}
		}

		// 计算平均耗时
		if run.Status == model.RunStatusDone {
			duration := run.UpdatedAt.Sub(run.CreatedAt).Milliseconds()
			totalDuration += duration
			durationCount++
		}
	}

	if durationCount > 0 {
		stats.AvgDurationMs = totalDuration / int64(durationCount)
	}

	return stats
}

// ========== 辅助函数 ==========

func mapAuthSessionStatus(status string) string {
	switch status {
	case "pending", "assigned":
		return "pending"
	case "running":
		return "running"
	case "waiting_user":
		return "waiting"
	case "success":
		return "completed"
	case "failed", "timeout":
		return "failed"
	default:
		return "unknown"
	}
}

func calculateAuthProgressFromStatus(status string) int {
	switch status {
	case "pending":
		return 10
	case "assigned":
		return 25
	case "running":
		return 50
	case "waiting_user":
		return 75
	case "success":
		return 100
	case "failed", "timeout":
		return 100
	default:
		return 0
	}
}

func mapAuthTaskStatus(status model.AuthTaskStatus) string {
	switch status {
	case model.AuthTaskStatusPending, model.AuthTaskStatusAssigned:
		return "pending"
	case model.AuthTaskStatusRunning:
		return "running"
	case model.AuthTaskStatusWaitingUser:
		return "waiting"
	case model.AuthTaskStatusSuccess:
		return "completed"
	case model.AuthTaskStatusFailed, model.AuthTaskStatusTimeout:
		return "failed"
	default:
		return "unknown"
	}
}

func mapRunStatus(status model.RunStatus) string {
	switch status {
	case model.RunStatusQueued:
		return "pending"
	case model.RunStatusRunning:
		return "running"
	case model.RunStatusDone:
		return "completed"
	case model.RunStatusFailed, model.RunStatusCancelled, model.RunStatusTimeout:
		return "failed"
	default:
		return "unknown"
	}
}

func calculateAuthProgress(status model.AuthTaskStatus) int {
	switch status {
	case model.AuthTaskStatusPending:
		return 10
	case model.AuthTaskStatusAssigned:
		return 25
	case model.AuthTaskStatusRunning:
		return 50
	case model.AuthTaskStatusWaitingUser:
		return 75
	case model.AuthTaskStatusSuccess:
		return 100
	case model.AuthTaskStatusFailed, model.AuthTaskStatusTimeout:
		return 100
	default:
		return 0
	}
}

func calculateRunProgress(status model.RunStatus) int {
	switch status {
	case model.RunStatusQueued:
		return 10
	case model.RunStatusRunning:
		return 50
	case model.RunStatusDone:
		return 100
	case model.RunStatusFailed, model.RunStatusCancelled, model.RunStatusTimeout:
		return 100
	default:
		return 0
	}
}

func getEventLevel(eventType string) string {
	if strings.Contains(eventType, "error") || strings.Contains(eventType, "failed") {
		return "error"
	}
	if strings.Contains(eventType, "warning") || strings.Contains(eventType, "timeout") {
		return "warning"
	}
	if strings.Contains(eventType, "success") || strings.Contains(eventType, "completed") {
		return "success"
	}
	return "info"
}

func mustParseInt(s string, defaultVal int64) int64 {
	n, err := json.Number(s).Int64()
	if err != nil {
		return defaultVal
	}
	return n
}

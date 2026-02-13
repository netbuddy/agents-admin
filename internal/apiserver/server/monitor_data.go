// Package server 工作流监控内部数据聚合
//
// 本文件包含监控 API 的内部数据获取和聚合方法，从各存储层收集数据
// 并转换为统一的工作流视图模型。
//
// 文件组织：
//   - getAuthWorkflows / getRunWorkflows: 列表数据聚合
//   - getAuthWorkflowDetail / getRunWorkflowDetail: 详情数据聚合
//   - getWorkflowEvents: 事件流查询
//   - calculateStats: 统计指标计算
//   - 辅助函数：状态映射、进度计算、事件级别判断
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"agents-admin/internal/shared/model"
	"agents-admin/internal/shared/storage"
)

// ========== 数据聚合方法 ==========

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

// mapAuthSessionStatus 将 Redis 认证会话状态映射为统一工作流状态
//
// 映射规则：
//   - pending, assigned → "pending"
//   - running → "running"
//   - waiting_user → "waiting"
//   - success → "completed"
//   - failed, timeout → "failed"
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

// calculateAuthProgressFromStatus 根据认证会话状态字符串计算进度百分比
//
// 进度值：
//   - pending: 10%
//   - assigned: 25%
//   - running: 50%
//   - waiting_user: 75%
//   - success/failed/timeout: 100%
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

// mapAuthTaskStatus 将 model.AuthTaskStatus 映射为统一工作流状态
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

// mapRunStatus 将 model.RunStatus 映射为统一工作流状态
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

// calculateRunProgress 根据 Run 状态计算进度百分比
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

// getEventLevel 根据事件类型推断事件级别
//
// 级别判断规则：
//   - 包含 "error"/"failed" → "error"
//   - 包含 "warning"/"timeout" → "warning"
//   - 包含 "success"/"completed" → "success"
//   - 其他 → "info"
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

// mustParseInt 安全解析整数字符串，失败时返回默认值
func mustParseInt(s string, defaultVal int64) int64 {
	n, err := json.Number(s).Int64()
	if err != nil {
		return defaultVal
	}
	return n
}

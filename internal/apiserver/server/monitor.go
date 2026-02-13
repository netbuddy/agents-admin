// Package server 工作流监控 HTTP API
//
// 本文件提供工作流监控相关的 HTTP 处理器和类型定义。
// 内部数据聚合逻辑和辅助函数位于 monitor_data.go。
//
// API 端点：
//   - GET /api/v1/monitor/workflows          - 列出工作流
//   - GET /api/v1/monitor/workflows/{type}/{id}  - 获取工作流详情
//   - GET /api/v1/monitor/workflows/{type}/{id}/events - 获取工作流事件
//   - GET /api/v1/monitor/stats              - 获取监控统计
package server

import (
	"encoding/json"
	"net/http"
	"sort"
	"time"
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

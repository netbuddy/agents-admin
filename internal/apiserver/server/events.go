// Package handler 事件管理接口
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	openapi "agents-admin/api/generated/go"
	"agents-admin/internal/shared/model"
)

// ============================================================================
// 请求/响应类型（使用 OpenAPI 生成的类型）
// ============================================================================

// PostEventsRequest 批量上报事件的请求体（OpenAPI 生成）
type PostEventsRequest = openapi.PostEventsRequest

// EventInput 单个事件的输入结构（OpenAPI 生成）
type EventInput = openapi.EventInput

// ============================================================================
// Event 接口处理函数
// ============================================================================

// GetEvents 获取 Run 的事件列表
//
// 路由: GET /api/v1/runs/{id}/events
//
// 路径参数:
//   - id: Run ID
//
// 查询参数:
//   - from_seq: 起始序号（不包含），默认 0
//   - limit: 返回数量限制，默认 100，最大 1000
//
// 响应:
//
//	{
//	  "events": [...],
//	  "count": 10
//	}
//
// 错误响应:
//   - 500 Internal Server Error: 服务器内部错误
//
// 使用场景：
//   - 前端轮询获取新事件
//   - 断线重连后恢复事件流
func (h *Handler) GetEvents(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	fromSeq, _ := strconv.Atoi(r.URL.Query().Get("from_seq"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	events, err := h.store.GetEventsByRun(r.Context(), runID, fromSeq, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get events")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"events": events, "count": len(events)})
}

// PostEvents 批量上报事件
//
// 路由: POST /api/v1/runs/{id}/events
//
// 路径参数:
//   - id: Run ID
//
// 请求体:
//
//	{
//	  "events": [
//	    {"seq": 1, "type": "run_started", "timestamp": "...", "payload": {...}},
//	    {"seq": 2, "type": "message", "timestamp": "...", "payload": {"content": "..."}}
//	  ]
//	}
//
// 响应:
//   - 201 Created: 返回 {"created": 2}
//   - 400 Bad Request: 请求体格式错误
//   - 500 Internal Server Error: 服务器内部错误
//
// 使用场景：
//   - Node Agent 批量上报执行过程中产生的事件
//   - 支持 WebSocket 实时推送到前端
//
// 副作用：
//   - 当收到第一个事件时，更新 Task 状态为 running（表示真正开始执行）
func (h *Handler) PostEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	runID := r.PathValue("id")
	var req PostEventsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	events := make([]*model.Event, len(req.Events))
	for i, e := range req.Events {
		var payload []byte
		if e.Payload != nil {
			payload, _ = json.Marshal(*e.Payload)
		}

		events[i] = &model.Event{
			RunID:     runID,
			Seq:       e.Seq,
			Type:      e.Type,
			Timestamp: e.Timestamp,
			Payload:   payload,
			Raw:       e.Raw, // 直接使用 *string
		}
	}

	if err := h.store.CreateEvents(ctx, events); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create events")
		return
	}

	// 检查是否需要更新 Task 状态为 running
	// 当收到第一个事件（seq=1）或 run_started 事件时，表示任务真正开始执行
	h.maybeUpdateTaskToRunning(ctx, runID, req.Events)

	// 写入 DB 后，立即广播到 WebSocket 客户端（实时推送）
	for _, e := range req.Events {
		var payload map[string]interface{}
		if e.Payload != nil {
			payload = *e.Payload
		}
		h.eventGateway.Broadcast(runID, map[string]interface{}{
			"seq":       e.Seq,
			"type":      e.Type,
			"timestamp": e.Timestamp,
			"payload":   payload,
		})
	}

	writeJSON(w, http.StatusCreated, map[string]int{"created": len(events)})
}

// maybeUpdateToRunning 检查并更新 Run 和 Task 状态为 running
//
// 触发条件：
//   - 收到 seq=1 的事件（第一个事件）
//   - 或收到 type="run_started" 的事件
//
// 状态更新：
//   - Run: assigned → running（表示 NodeManager 真正开始执行）
//   - Task: pending → running（表示任务正在进行中）
//
// 此方法确保只有在 NodeManager 真正开始执行并上报事件后，状态才变为 running
func (h *Handler) maybeUpdateTaskToRunning(ctx context.Context, runID string, events []EventInput) {
	shouldUpdate := false
	for _, e := range events {
		if e.Seq == 1 || e.Type == "run_started" {
			shouldUpdate = true
			break
		}
	}

	if !shouldUpdate {
		return
	}

	// 获取 Run
	run, err := h.store.GetRun(ctx, runID)
	if err != nil || run == nil {
		return
	}

	// 更新 Run 状态：assigned → running
	if run.Status == model.RunStatusAssigned {
		h.store.UpdateRunStatus(ctx, runID, model.RunStatusRunning, nil)
	}

	// 更新 Task 状态：pending → running
	task, err := h.store.GetTask(ctx, run.TaskID)
	if err != nil || task == nil {
		return
	}

	if task.Status == model.TaskStatusPending {
		h.store.UpdateTaskStatus(ctx, run.TaskID, model.TaskStatusInProgress)
	}
}

// Package api 事件管理接口
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"agents-admin/internal/model"
)

// ============================================================================
// 请求/响应结构体
// ============================================================================

// PostEventsRequest 批量上报事件的请求体
//
// 字段说明：
//   - Events: 事件列表
type PostEventsRequest struct {
	Events []EventInput `json:"events"` // 事件列表
}

// EventInput 单个事件的输入结构
//
// 字段说明：
//   - Seq: 事件序号，Run 内递增
//   - Type: 事件类型（如 message、tool_use_start、command 等）
//   - Timestamp: 事件发生时间
//   - Payload: 事件数据（已解析）
//   - Raw: 原始 CLI 输出（用于调试和回放）
type EventInput struct {
	Seq       int                    `json:"seq"`       // 事件序号
	Type      string                 `json:"type"`      // 事件类型
	Timestamp time.Time              `json:"timestamp"` // 事件时间
	Payload   map[string]interface{} `json:"payload"`   // 事件数据
	Raw       string                 `json:"raw,omitempty"` // 原始输出
}

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
func (h *Handler) PostEvents(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	var req PostEventsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	events := make([]*model.Event, len(req.Events))
	for i, e := range req.Events {
		payload, _ := json.Marshal(e.Payload)
		
		// raw 字段：只保存真正的 agent 原始输出
		// 平台生成的事件（如 run_started, run_completed）没有 raw
		var rawPtr *string
		if e.Raw != "" {
			rawPtr = &e.Raw
		}
		
		events[i] = &model.Event{
			RunID:     runID,
			Seq:       e.Seq,
			Type:      e.Type,
			Timestamp: e.Timestamp,
			Payload:   payload,
			Raw:       rawPtr,
		}
	}

	if err := h.store.CreateEvents(r.Context(), events); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create events")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int{"created": len(events)})
}

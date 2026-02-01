// Package api 执行管理接口
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"agents-admin/internal/model"
)

// ============================================================================
// 请求/响应结构体
// ============================================================================

// UpdateRunRequest 更新 Run 的请求体
//
// 字段说明：
//   - Status: 执行状态（必填）
type UpdateRunRequest struct {
	Status string `json:"status"` // 执行状态
}

// ============================================================================
// Run 接口处理函数
// ============================================================================

// CreateRun 为任务创建一次执行
//
// 路由: POST /api/v1/tasks/{id}/runs
//
// 路径参数:
//   - id: 任务 ID
//
// 响应:
//   - 201 Created: 返回创建的 Run 对象
//   - 404 Not Found: 任务不存在
//   - 500 Internal Server Error: 服务器内部错误
//
// 业务逻辑：
//  1. 验证任务存在
//  2. 创建 Run，初始状态为 queued
//  3. 保存任务快照用于审计
//  4. 更新任务状态为 running
func (h *Handler) CreateRun(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	task, err := h.store.GetTask(r.Context(), taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get task")
		return
	}
	if task == nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	// 创建任务快照（扁平化 Task 结构的 JSON 序列化）
	taskSnapshot, _ := json.Marshal(task)

	now := time.Now()
	run := &model.Run{
		ID:        generateID("run"),
		TaskID:    taskID,
		Status:    model.RunStatusQueued,
		Snapshot:  taskSnapshot,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.store.CreateRun(r.Context(), run); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create run")
		return
	}

	h.store.UpdateTaskStatus(r.Context(), taskID, model.TaskStatusRunning)
	writeJSON(w, http.StatusCreated, run)
}

// GetRun 获取单个 Run 详情
//
// 路由: GET /api/v1/runs/{id}
//
// 路径参数:
//   - id: Run ID
//
// 响应:
//   - 200 OK: 返回 Run 对象
//   - 404 Not Found: Run 不存在
//   - 500 Internal Server Error: 服务器内部错误
func (h *Handler) GetRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	run, err := h.store.GetRun(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get run")
		return
	}
	if run == nil {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}
	writeJSON(w, http.StatusOK, run)
}

// ListRuns 列出任务的所有执行记录
//
// 路由: GET /api/v1/tasks/{id}/runs
//
// 路径参数:
//   - id: 任务 ID
//
// 响应:
//
//	{
//	  "runs": [...],
//	  "count": 5
//	}
//
// 错误响应:
//   - 500 Internal Server Error: 服务器内部错误
func (h *Handler) ListRuns(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	runs, err := h.store.ListRunsByTask(r.Context(), taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list runs")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"runs": runs, "count": len(runs)})
}

// CancelRun 取消正在执行或排队中的 Run
//
// 路由: POST /api/v1/runs/{id}/cancel
//
// 路径参数:
//   - id: Run ID
//
// 响应:
//   - 200 OK: 取消成功，返回 {"status": "cancelled"}
//   - 400 Bad Request: Run 状态不允许取消
//   - 404 Not Found: Run 不存在
//
// 业务规则：
//   - 只有 queued 和 running 状态的 Run 可以被取消
//   - 已完成（done/failed/cancelled/timeout）的 Run 不能取消
func (h *Handler) CancelRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	run, err := h.store.GetRun(r.Context(), id)
	if err != nil || run == nil {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}
	if run.Status != model.RunStatusQueued && run.Status != model.RunStatusRunning {
		writeError(w, http.StatusBadRequest, "run cannot be cancelled")
		return
	}

	h.store.UpdateRunStatus(r.Context(), id, model.RunStatusCancelled, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// UpdateRun 更新 Run 状态
//
// 路由: PATCH /api/v1/runs/{id}
//
// 路径参数:
//   - id: Run ID
//
// 请求体:
//
//	{
//	  "status": "done"
//	}
//
// 响应:
//   - 200 OK: 更新成功，返回 {"status": "done"}
//   - 400 Bad Request: 请求体格式错误
//   - 500 Internal Server Error: 服务器内部错误
//
// 允许的状态值：
//   - running: 开始执行
//   - done: 执行完成
//   - failed: 执行失败
//   - cancelled: 已取消
//   - timeout: 超时
func (h *Handler) UpdateRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req UpdateRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	status := model.RunStatus(req.Status)
	if err := h.store.UpdateRunStatus(r.Context(), id, status, nil); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update run")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": req.Status})
}

// StartScheduler 启动任务调度器
//
// 调度器会定期扫描 queued 状态的 Run，并分配到可用的 Node 执行。
//
// 参数：
//   - ctx: 上下文，用于控制调度器生命周期
func (h *Handler) StartScheduler(ctx context.Context) {
	h.scheduler.Start(ctx)
}

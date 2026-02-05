// Package run 执行领域 - HTTP 处理
package run

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	openapi "agents-admin/api/generated/go"
	"agents-admin/internal/shared/model"
	"agents-admin/internal/shared/queue"
	"agents-admin/internal/shared/storage"
)

// RunStore 定义 run handler 需要的存储接口（用于测试 mock）
type RunStore interface {
	GetTask(ctx context.Context, id string) (*model.Task, error)
	CreateRun(ctx context.Context, run *model.Run) error
	GetRun(ctx context.Context, id string) (*model.Run, error)
	ListRunsByTask(ctx context.Context, taskID string) ([]*model.Run, error)
	UpdateRunStatus(ctx context.Context, id string, status model.RunStatus, nodeID *string) error
}

// RunScheduler 定义 run handler 需要的调度队列接口
// 仅包含创建 Run 时需要的方法
type RunScheduler interface {
	ScheduleRun(ctx context.Context, runID, taskID string) (string, error)
}

// Handler 执行领域 HTTP 处理器
type Handler struct {
	store     RunStore
	scheduler RunScheduler // 调度队列（用于将 Run 加入调度）
}

// NewHandler 创建执行处理器
// scheduler 参数可选，如果为 nil 则不使用事件驱动调度（仅依赖保底轮询）
func NewHandler(store storage.PersistentStore, scheduler queue.SchedulerQueue) *Handler {
	var s RunScheduler
	if scheduler != nil {
		s = scheduler
	}
	return &Handler{store: store, scheduler: s}
}

// NewHandlerWithInterfaces 使用接口创建处理器（用于测试）
func NewHandlerWithInterfaces(store RunStore, scheduler RunScheduler) *Handler {
	return &Handler{store: store, scheduler: scheduler}
}

// RegisterRoutes 注册执行相关路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/tasks/{id}/runs", h.Create)
	mux.HandleFunc("GET /api/v1/tasks/{id}/runs", h.ListByTask)
	mux.HandleFunc("GET /api/v1/runs/{id}", h.Get)
	mux.HandleFunc("PATCH /api/v1/runs/{id}", h.Update)
	mux.HandleFunc("POST /api/v1/runs/{id}/cancel", h.Cancel)
}

// UpdateRequest 更新 Run 的请求体（使用 OpenAPI 生成的类型）
type UpdateRequest = openapi.UpdateRunRequest

// Create 为任务创建一次执行
// POST /api/v1/tasks/{id}/runs
//
// 流程：
//  1. 写入 PostgreSQL（必须成功）
//  2. 写入 Redis Streams（允许失败，有保底轮询）
//  3. 更新 Task 状态
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	taskID := r.PathValue("id")
	runID := generateID("run")

	log.Printf("[run.create.start] run_id=%s task_id=%s", runID, taskID)

	// 获取任务
	task, err := h.store.GetTask(ctx, taskID)
	if err != nil {
		log.Printf("[run.create.task.failed] run_id=%s task_id=%s error=%v", runID, taskID, err)
		writeError(w, http.StatusInternalServerError, "failed to get task")
		return
	}
	if task == nil {
		log.Printf("[run.create.task.not_found] run_id=%s task_id=%s", runID, taskID)
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	// 创建任务快照
	taskSnapshot, _ := json.Marshal(task)

	now := time.Now()
	run := &model.Run{
		ID:        runID,
		TaskID:    taskID,
		Status:    model.RunStatusQueued,
		Snapshot:  taskSnapshot,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Step 1: 写入 PostgreSQL（必须成功）
	if err := h.store.CreateRun(ctx, run); err != nil {
		log.Printf("[run.create.pg.failed] run_id=%s task_id=%s error=%v", runID, taskID, err)
		writeError(w, http.StatusInternalServerError, "failed to create run")
		return
	}
	log.Printf("[run.create.pg.success] run_id=%s task_id=%s", runID, taskID)

	// Step 2: 加入调度队列（允许失败，有保底轮询）
	if h.scheduler != nil {
		msgID, err := h.scheduler.ScheduleRun(ctx, runID, taskID)
		if err != nil {
			// 队列写入失败不是致命错误，保底轮询会处理
			log.Printf("[run.create.queue.failed] run_id=%s task_id=%s error=%v", runID, taskID, err)
		} else {
			log.Printf("[run.create.queue.success] run_id=%s task_id=%s msg_id=%s", runID, taskID, msgID)
		}
	}

	// 注意：不在这里更新 Task 状态为 running
	// Task 状态应该在 NodeManager 真正开始执行并上报事件后才变更
	// 参见 events.go PostEvents()

	log.Printf("[run.create.complete] run_id=%s task_id=%s", runID, taskID)
	writeJSON(w, http.StatusCreated, run)
}

// Get 获取单个 Run 详情
// GET /api/v1/runs/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
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

// ListByTask 列出任务的所有执行记录
// GET /api/v1/tasks/{id}/runs
func (h *Handler) ListByTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	runs, err := h.store.ListRunsByTask(r.Context(), taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list runs")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"runs": runs, "count": len(runs)})
}

// Cancel 取消正在执行或排队中的 Run
// POST /api/v1/runs/{id}/cancel
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
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

// Update 更新 Run 状态
// PATCH /api/v1/runs/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Status == nil {
		writeError(w, http.StatusBadRequest, "status is required")
		return
	}

	statusStr := string(*req.Status)
	status := model.RunStatus(statusStr)
	if err := h.store.UpdateRunStatus(r.Context(), id, status, nil); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update run")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": statusStr})
}

// ============================================================================
// 工具函数
// ============================================================================

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func generateID(prefix string) string {
	b := make([]byte, 6)
	rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}

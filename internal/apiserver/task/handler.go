// Package task 任务领域 - HTTP 处理
package task

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	openapi "agents-admin/api/generated/go"
	"agents-admin/internal/shared/model"
	"agents-admin/internal/shared/storage"
)

// Handler 任务领域 HTTP 处理器
type Handler struct {
	store storage.TaskStore // 使用接口类型
}

// NewHandler 创建任务处理器
func NewHandler(store storage.TaskStore) *Handler {
	return &Handler{store: store}
}

// RegisterRoutes 注册任务相关路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/tasks", h.List)
	mux.HandleFunc("POST /api/v1/tasks", h.Create)
	mux.HandleFunc("GET /api/v1/tasks/{id}", h.Get)
	mux.HandleFunc("DELETE /api/v1/tasks/{id}", h.Delete)
	mux.HandleFunc("GET /api/v1/tasks/{id}/subtasks", h.ListSubTasks)
	mux.HandleFunc("GET /api/v1/tasks/{id}/tree", h.GetTree)
	mux.HandleFunc("PUT /api/v1/tasks/{id}/context", h.UpdateContext)
}

// ============================================================================
// 类型别名（方便外部包使用）
// ============================================================================

// CreateRequest 创建任务的请求体（使用 OpenAPI 生成的类型）
type CreateRequest = openapi.CreateTaskRequest

// ============================================================================
// HTTP 处理函数
// ============================================================================

// Create 创建任务
// POST /api/v1/tasks
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	taskType := model.TaskTypeGeneral
	if req.Type != nil && *req.Type != "" {
		taskType = model.TaskType(*req.Type)
	}

	prompt := &model.Prompt{
		Content:    req.Prompt,
		TemplateID: req.PromptTemplateId,
	}
	if req.PromptDescription != nil {
		prompt.Description = *req.PromptDescription
	}

	now := time.Now()
	task := &model.Task{
		ID:        generateID("task"),
		ParentID:  req.ParentId,
		Name:      req.Name,
		Status:    model.TaskStatusPending,
		Type:      taskType,
		Prompt:    prompt,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// 处理可选字段
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Labels != nil {
		task.Labels = *req.Labels
	}
	if req.TemplateId != nil {
		task.TemplateID = req.TemplateId
	}
	if req.AgentId != nil {
		task.AgentID = req.AgentId
	}

	// 转换 Workspace（JSON 桥接，OpenAPI 简化版 -> model 完整版）
	if req.Workspace != nil {
		task.Workspace = jsonBridgeConvert[model.WorkspaceConfig](req.Workspace)
	}

	// 转换 Security（JSON 桥接）
	if req.Security != nil {
		task.Security = jsonBridgeConvert[model.SecurityConfig](req.Security)
	}

	// 转换 Context（openapi -> model）
	if req.Context != nil {
		task.Context = convertTaskContext(req.Context)
	}

	// 继承父任务上下文
	if req.ParentId != nil && *req.ParentId != "" {
		parentTask, err := h.store.GetTask(r.Context(), *req.ParentId)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to get parent task")
			return
		}
		if parentTask == nil {
			writeError(w, http.StatusBadRequest, "parent task not found")
			return
		}
		if parentTask.Context != nil && len(parentTask.Context.ProducedContext) > 0 {
			if task.Context == nil {
				task.Context = &model.TaskContext{}
			}
			task.Context.InheritedContext = append(task.Context.InheritedContext, parentTask.Context.ProducedContext...)
		}
	}

	if err := h.store.CreateTask(r.Context(), task); err != nil {
		log.Printf("[Task] Create error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

// Get 获取任务详情
// GET /api/v1/tasks/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	task, err := h.store.GetTask(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get task")
		return
	}
	if task == nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// List 列出任务
// GET /api/v1/tasks
//
// 支持的查询参数：
//   - status: 按状态筛选
//   - search: 按名称模糊搜索
//   - since:  创建时间下限 (ISO8601)
//   - until:  创建时间上限 (ISO8601)
//   - limit:  每页条数 (默认 20, 最大 100)
//   - offset: 偏移量
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	filter := storage.TaskFilter{
		Status: r.URL.Query().Get("status"),
		Search: r.URL.Query().Get("search"),
		Limit:  limit,
		Offset: offset,
	}
	if s := r.URL.Query().Get("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			filter.Since = t
		}
	}
	if u := r.URL.Query().Get("until"); u != "" {
		if t, err := time.Parse(time.RFC3339, u); err == nil {
			filter.Until = t
		}
	}

	tasks, total, err := h.store.ListTasksWithFilter(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tasks":    tasks,
		"count":    len(tasks),
		"total":    total,
		"has_more": offset+len(tasks) < total,
	})
}

// Delete 删除任务
// DELETE /api/v1/tasks/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.DeleteTask(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListSubTasks 列出子任务
// GET /api/v1/tasks/{id}/subtasks
func (h *Handler) ListSubTasks(w http.ResponseWriter, r *http.Request) {
	parentID := r.PathValue("id")
	tasks, err := h.store.ListSubTasks(r.Context(), parentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list subtasks")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"tasks": tasks, "count": len(tasks)})
}

// GetTree 获取任务树
// GET /api/v1/tasks/{id}/tree
func (h *Handler) GetTree(w http.ResponseWriter, r *http.Request) {
	rootID := r.PathValue("id")
	tasks, err := h.store.GetTaskTree(r.Context(), rootID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get task tree")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"tasks": tasks, "count": len(tasks)})
}

// UpdateContext 更新任务上下文
// PUT /api/v1/tasks/{id}/context
func (h *Handler) UpdateContext(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var context model.TaskContext
	if err := json.NewDecoder(r.Body).Decode(&context); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	contextJSON, err := json.Marshal(context)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to marshal context")
		return
	}

	if err := h.store.UpdateTaskContext(r.Context(), id, contextJSON); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update task context")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"message": "context updated"})
}

// Package api 任务管理接口
package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"agents-admin/internal/model"
)

// ============================================================================
// 请求/响应结构体
// ============================================================================

// CreateTaskRequest 创建任务的请求体（扁平化结构）
//
// 字段说明：
//   - Name: 任务名称，必填，用户可读的任务描述
//   - Description: 任务描述，可选
//   - Type: 任务类型，可选，默认为 general
//   - Prompt: 任务提示词内容，必填（字符串或 Prompt 对象）
//   - PromptDescription: 提示词说明，可选
//   - Workspace: 工作空间配置，可选
//   - Security: 安全配置，可选
//   - Labels: 任务标签，可选
//   - ParentID: 父任务 ID，可选，用于创建子任务
//   - TemplateID: 模板 ID，可选
//   - AgentID: Agent ID，可选
type CreateTaskRequest struct {
	Name              string                  `json:"name"`                          // 任务名称（必填）
	Description       string                  `json:"description,omitempty"`         // 任务描述（可选）
	Type              string                  `json:"type,omitempty"`                // 任务类型（可选，默认 general）
	Prompt            string                  `json:"prompt"`                        // 任务提示词内容（必填）
	PromptDescription string                  `json:"prompt_description,omitempty"`  // 提示词说明（可选）
	PromptTemplateID  *string                 `json:"prompt_template_id,omitempty"`  // 提示词模板 ID（可选）
	Workspace         *model.WorkspaceConfig  `json:"workspace,omitempty"`           // 工作空间配置（可选）
	Security          *model.SecurityConfig   `json:"security,omitempty"`            // 安全配置（可选）
	Labels            map[string]string       `json:"labels,omitempty"`              // 任务标签（可选）
	ParentID          *string                 `json:"parent_id,omitempty"`           // 父任务 ID（可选）
	TemplateID        *string                 `json:"template_id,omitempty"`         // 模板 ID（可选）
	AgentID           *string                 `json:"agent_id,omitempty"`            // Agent ID（可选）
	Context           *model.TaskContext      `json:"context,omitempty"`             // 初始上下文（可选）
}

// UpdateTaskRequest 更新任务的请求体
//
// 字段说明：
//   - Name: 任务名称，可选
//   - Status: 任务状态，可选
//   - Prompt: 任务提示词，可选
//   - Labels: 任务标签，可选
type UpdateTaskRequest struct {
	Name   *string           `json:"name,omitempty"`   // 任务名称
	Status *string           `json:"status,omitempty"` // 任务状态
	Prompt *string           `json:"prompt,omitempty"` // 任务提示词
	Labels map[string]string `json:"labels,omitempty"` // 任务标签
}

// ============================================================================
// Task 接口处理函数
// ============================================================================

// CreateTask 创建任务
//
// 路由: POST /api/v1/tasks
//
// 请求体:
//
//	{
//	  "name": "任务名称",
//	  "spec": {
//	    "prompt": "任务提示词",
//	    "agent": {"type": "gemini"}
//	  }
//	}
//
// 响应:
//   - 201 Created: 返回创建的任务对象
//   - 400 Bad Request: 请求体格式错误或缺少必填字段
//   - 500 Internal Server Error: 服务器内部错误
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
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

	// 确定任务类型
	taskType := model.TaskTypeGeneral
	if req.Type != "" {
		taskType = model.TaskType(req.Type)
	}

	// 构建结构化 Prompt
	prompt := &model.Prompt{
		Content:     req.Prompt,
		Description: req.PromptDescription,
		TemplateID:  req.PromptTemplateID,
	}

	now := time.Now()
	task := &model.Task{
		ID:          generateID("task"),
		ParentID:    req.ParentID,
		Name:        req.Name,
		Description: req.Description,
		Status:      model.TaskStatusPending,
		Type:        taskType,
		Prompt:      prompt,
		Workspace:   req.Workspace,
		Security:    req.Security,
		Labels:      req.Labels,
		Context:     req.Context,
		TemplateID:  req.TemplateID,
		AgentID:     req.AgentID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// 如果有父任务，继承父任务的上下文
	if req.ParentID != nil && *req.ParentID != "" {
		parentTask, err := h.store.GetTask(r.Context(), *req.ParentID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to get parent task")
			return
		}
		if parentTask == nil {
			writeError(w, http.StatusBadRequest, "parent task not found")
			return
		}
		// 将父任务的 produced_context 添加到子任务的 inherited_context
		if parentTask.Context != nil && len(parentTask.Context.ProducedContext) > 0 {
			if task.Context == nil {
				task.Context = &model.TaskContext{}
			}
			// 继承父任务产出的上下文
			task.Context.InheritedContext = append(task.Context.InheritedContext, parentTask.Context.ProducedContext...)
		}
	}

	if err := h.store.CreateTask(r.Context(), task); err != nil {
		log.Printf("[API] CreateTask error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

// GetTask 获取单个任务详情
//
// 路由: GET /api/v1/tasks/{id}
//
// 路径参数:
//   - id: 任务 ID
//
// 响应:
//   - 200 OK: 返回任务对象
//   - 404 Not Found: 任务不存在
//   - 500 Internal Server Error: 服务器内部错误
func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
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

// ListTasks 列出任务列表
//
// 路由: GET /api/v1/tasks
//
// 查询参数:
//   - status: 按状态筛选（可选）
//   - limit: 返回数量限制，默认 20，最大 100
//   - offset: 分页偏移量，默认 0
//
// 响应:
//
//	{
//	  "tasks": [...],
//	  "count": 10
//	}
//
// 错误响应:
//   - 500 Internal Server Error: 服务器内部错误
func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	tasks, err := h.store.ListTasks(r.Context(), status, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"tasks": tasks, "count": len(tasks)})
}

// DeleteTask 删除任务
//
// 路由: DELETE /api/v1/tasks/{id}
//
// 路径参数:
//   - id: 任务 ID
//
// 响应:
//   - 204 No Content: 删除成功
//   - 500 Internal Server Error: 服务器内部错误
//
// 注意：删除任务会级联删除相关的 Run 和 Event（由数据库外键约束处理）
func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.DeleteTask(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListSubTasks 列出子任务
//
// 路由: GET /api/v1/tasks/{id}/subtasks
//
// 路径参数:
//   - id: 父任务 ID
//
// 响应:
//
//	{
//	  "tasks": [...],
//	  "count": 10
//	}
func (h *Handler) ListSubTasks(w http.ResponseWriter, r *http.Request) {
	parentID := r.PathValue("id")
	tasks, err := h.store.ListSubTasks(r.Context(), parentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list subtasks")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"tasks": tasks, "count": len(tasks)})
}

// GetTaskTree 获取任务树
//
// 路由: GET /api/v1/tasks/{id}/tree
//
// 路径参数:
//   - id: 根任务 ID
//
// 响应:
//
//	{
//	  "tasks": [...],  // 按深度和创建时间排序
//	  "count": 10
//	}
func (h *Handler) GetTaskTree(w http.ResponseWriter, r *http.Request) {
	rootID := r.PathValue("id")
	tasks, err := h.store.GetTaskTree(r.Context(), rootID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get task tree")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"tasks": tasks, "count": len(tasks)})
}

// UpdateTaskContext 更新任务上下文
//
// 路由: PUT /api/v1/tasks/{id}/context
//
// 请求体:
//
//	{
//	  "produced_context": [...],
//	  "conversation_history": [...]
//	}
func (h *Handler) UpdateTaskContext(w http.ResponseWriter, r *http.Request) {
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

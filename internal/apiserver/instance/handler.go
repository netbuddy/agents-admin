// Package instance Agent 实例领域 - HTTP 处理（原 Instance，已重命名对齐领域模型）
package instance

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"agents-admin/internal/shared/model"
	"agents-admin/internal/shared/storage"
)

// 确保 database/sql 被使用
var _ = sql.ErrNoRows

// Handler Agent 实例 HTTP 处理器
type Handler struct {
	store storage.PersistentStore
}

// NewHandler 创建 Agent 实例处理器
func NewHandler(store storage.PersistentStore) *Handler {
	return &Handler{store: store}
}

// RegisterRoutes 注册 Agent 实例相关路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/agents", h.List)
	mux.HandleFunc("POST /api/v1/agents", h.Create)
	mux.HandleFunc("GET /api/v1/agents/{id}", h.Get)
	mux.HandleFunc("DELETE /api/v1/agents/{id}", h.Delete)
	mux.HandleFunc("POST /api/v1/agents/{id}/start", h.Start)
	mux.HandleFunc("POST /api/v1/agents/{id}/stop", h.Stop)
}

// ============================================================================
// HTTP 处理函数
// ============================================================================

// List 获取 Agent 实例列表
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	instances, err := h.store.ListAgentInstances(r.Context())
	if err != nil {
		log.Printf("[instance] Failed to list instances: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list instances")
		return
	}

	agentType := r.URL.Query().Get("agent_type")
	accountID := r.URL.Query().Get("account_id")

	var result []*model.Instance
	for _, inst := range instances {
		if agentType != "" && inst.AgentTypeID != agentType {
			continue
		}
		if accountID != "" && inst.AccountID != accountID {
			continue
		}
		result = append(result, inst)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"agents": result,
		"count":  len(result),
	})
}

// Create 创建实例
//
// 支持两种创建模式：
//   - 通用模式：提供 account_id，自动从 account 获取 agent_type_id
//   - 模板模式：提供 account_id + template_id，记录模板关联
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string  `json:"name"`
		AccountID  string  `json:"account_id"`
		NodeID     string  `json:"node_id"`
		TemplateID *string `json:"template_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.AccountID == "" {
		writeError(w, http.StatusBadRequest, "account_id is required")
		return
	}

	account, err := h.store.GetAccount(r.Context(), req.AccountID)
	if err != nil {
		log.Printf("[instance] Failed to get account %s: %v", req.AccountID, err)
		writeError(w, http.StatusInternalServerError, "failed to get account")
		return
	}
	if account == nil {
		writeError(w, http.StatusBadRequest, "account not found")
		return
	}

	if account.Status != model.AccountStatusAuthenticated {
		writeError(w, http.StatusBadRequest, "account not authenticated")
		return
	}

	if account.VolumeName == nil || *account.VolumeName == "" {
		writeError(w, http.StatusBadRequest, "account has no volume")
		return
	}

	nodeID := req.NodeID
	if nodeID == "" {
		writeError(w, http.StatusBadRequest, "node_id is required")
		return
	}

	// 如果指定了模板，验证模板存在
	if req.TemplateID != nil && *req.TemplateID != "" {
		tmpl, err := h.store.GetAgentTemplate(r.Context(), *req.TemplateID)
		if err != nil {
			log.Printf("[instance] Failed to get template %s: %v", *req.TemplateID, err)
			writeError(w, http.StatusInternalServerError, "failed to get template")
			return
		}
		if tmpl == nil {
			writeError(w, http.StatusBadRequest, "template not found")
			return
		}
	}

	agentID := generateID("agent")
	if req.Name == "" {
		req.Name = agentID
	}

	now := time.Now()
	instance := &model.Instance{
		ID:          agentID,
		Name:        req.Name,
		AccountID:   req.AccountID,
		AgentTypeID: account.AgentTypeID,
		TemplateID:  req.TemplateID,
		NodeID:      &nodeID,
		Status:      model.InstanceStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.store.CreateAgentInstance(r.Context(), instance); err != nil {
		log.Printf("[agent] Failed to create agent: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create agent")
		return
	}

	log.Printf("[agent] Agent created: %s (account=%s, template=%v)", agentID, req.AccountID, req.TemplateID)
	writeJSON(w, http.StatusCreated, instance)
}

// Get 获取 Agent 实例详情
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	instance, err := h.store.GetAgentInstance(r.Context(), id)
	if err != nil {
		log.Printf("[agent] Failed to get agent %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "failed to get agent")
		return
	}
	if instance == nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}

	writeJSON(w, http.StatusOK, instance)
}

// Delete 删除 Agent 实例
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	instance, err := h.store.GetAgentInstance(r.Context(), id)
	if err != nil {
		log.Printf("[agent] Failed to get agent %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "failed to get agent")
		return
	}
	if instance == nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}

	if err := h.store.DeleteAgentInstance(r.Context(), id); err != nil {
		log.Printf("[agent] Failed to delete agent %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "failed to delete agent")
		return
	}

	log.Printf("[agent] Agent deleted: %s", id)
	writeJSON(w, http.StatusOK, map[string]interface{}{"message": "agent deleted"})
}

// Start 启动 Agent 实例
func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	instance, err := h.store.GetAgentInstance(r.Context(), id)
	if err != nil {
		log.Printf("[agent] Failed to get agent %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "failed to get agent")
		return
	}
	if instance == nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}

	if err := h.store.UpdateAgentInstance(r.Context(), id, model.InstanceStatusPending, nil); err != nil {
		log.Printf("[agent] Failed to update agent %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "failed to start agent")
		return
	}

	log.Printf("[agent] Agent start requested: %s", id)
	writeJSON(w, http.StatusOK, map[string]interface{}{"message": "agent start requested"})
}

// Stop 停止 Agent 实例
func (h *Handler) Stop(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	instance, err := h.store.GetAgentInstance(r.Context(), id)
	if err != nil {
		log.Printf("[agent] Failed to get agent %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "failed to get agent")
		return
	}
	if instance == nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}

	if err := h.store.UpdateAgentInstance(r.Context(), id, model.InstanceStatusStopping, nil); err != nil {
		log.Printf("[agent] Failed to update agent %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "failed to stop agent")
		return
	}

	log.Printf("[agent] Agent stop requested: %s", id)
	writeJSON(w, http.StatusOK, map[string]interface{}{"message": "agent stop requested"})
}

// ============================================================================
// 工具函数
// ============================================================================

func generateID(prefix string) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-%s-%x", prefix, time.Now().Format("20060102150405"), b)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

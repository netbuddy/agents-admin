// Package instance 实例领域 - HTTP 处理
package instance

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"agents-admin/internal/shared/model"
	"agents-admin/internal/shared/storage"
)

// 确保 database/sql 被使用
var _ = sql.ErrNoRows

// Handler 实例领域 HTTP 处理器
type Handler struct {
	store storage.PersistentStore
}

// NewHandler 创建实例处理器
func NewHandler(store storage.PersistentStore) *Handler {
	return &Handler{store: store}
}

// RegisterRoutes 注册实例相关路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/instances", h.List)
	mux.HandleFunc("POST /api/v1/instances", h.Create)
	mux.HandleFunc("GET /api/v1/instances/{id}", h.Get)
	mux.HandleFunc("DELETE /api/v1/instances/{id}", h.Delete)
	mux.HandleFunc("POST /api/v1/instances/{id}/start", h.Start)
	mux.HandleFunc("POST /api/v1/instances/{id}/stop", h.Stop)
}

// ============================================================================
// HTTP 处理函数
// ============================================================================

// List 获取实例列表
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	instances, err := h.store.ListInstances(r.Context())
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
		"instances": result,
		"count":     len(result),
	})
}

// Create 创建实例
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string `json:"name"`
		AccountID string `json:"account_id"`
		NodeID    string `json:"node_id"`
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
		nodeID = account.NodeID
	}

	instanceID := generateID("inst")
	if req.Name == "" {
		req.Name = instanceID
	}

	now := time.Now()
	instance := &model.Instance{
		ID:          instanceID,
		Name:        req.Name,
		AccountID:   req.AccountID,
		AgentTypeID: account.AgentTypeID,
		NodeID:      &nodeID,
		Status:      model.InstanceStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.store.CreateInstance(r.Context(), instance); err != nil {
		log.Printf("[instance] Failed to create instance: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create instance")
		return
	}

	log.Printf("[instance] Instance created: %s (account=%s)", instanceID, req.AccountID)
	writeJSON(w, http.StatusCreated, instance)
}

// Get 获取实例详情
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	instance, err := h.store.GetInstance(r.Context(), id)
	if err != nil {
		log.Printf("[instance] Failed to get instance %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "failed to get instance")
		return
	}
	if instance == nil {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	writeJSON(w, http.StatusOK, instance)
}

// Delete 删除实例
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	instance, err := h.store.GetInstance(r.Context(), id)
	if err != nil {
		log.Printf("[instance] Failed to get instance %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "failed to get instance")
		return
	}
	if instance == nil {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	if err := h.store.DeleteInstance(r.Context(), id); err != nil {
		log.Printf("[instance] Failed to delete instance %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "failed to delete instance")
		return
	}

	log.Printf("[instance] Instance deleted: %s", id)
	writeJSON(w, http.StatusOK, map[string]interface{}{"message": "instance deleted"})
}

// Start 启动实例
func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	instance, err := h.store.GetInstance(r.Context(), id)
	if err != nil {
		log.Printf("[instance] Failed to get instance %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "failed to get instance")
		return
	}
	if instance == nil {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	if err := h.store.UpdateInstance(r.Context(), id, model.InstanceStatusPending, nil); err != nil {
		log.Printf("[instance] Failed to update instance %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "failed to start instance")
		return
	}

	log.Printf("[instance] Instance start requested: %s", id)
	writeJSON(w, http.StatusOK, map[string]interface{}{"message": "instance start requested"})
}

// Stop 停止实例
func (h *Handler) Stop(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	instance, err := h.store.GetInstance(r.Context(), id)
	if err != nil {
		log.Printf("[instance] Failed to get instance %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "failed to get instance")
		return
	}
	if instance == nil {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	if err := h.store.UpdateInstance(r.Context(), id, model.InstanceStatusStopping, nil); err != nil {
		log.Printf("[instance] Failed to update instance %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "failed to stop instance")
		return
	}

	log.Printf("[instance] Instance stop requested: %s", id)
	writeJSON(w, http.StatusOK, map[string]interface{}{"message": "instance stop requested"})
}

// ============================================================================
// 工具函数
// ============================================================================

func generateID(prefix string) string {
	b := make([]byte, 6)
	_, _ = json.Marshal(b) // 使用 json 包避免额外导入
	return prefix + "-" + time.Now().Format("20060102150405")
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

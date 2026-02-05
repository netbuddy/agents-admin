// Package instance NodeManager 回调接口
//
// 本文件包含供 NodeManager 调用的 API 端点：
//   - ListPending: 列出节点待处理的实例（NodeManager 轮询用）
//   - UpdateStatus: 更新实例状态（NodeManager 回调）
package instance

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"agents-admin/internal/shared/model"
)

// RegisterNodeManagerRoutes 注册 NodeManager 相关路由
func (h *Handler) RegisterNodeManagerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/nodes/{node_id}/instances", h.ListPending)
	mux.HandleFunc("PATCH /api/v1/instances/{id}", h.UpdateStatus)
}

// ListPending 列出节点待处理的实例（NodeManager 轮询用）
// GET /api/v1/nodes/{node_id}/instances
func (h *Handler) ListPending(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("node_id")
	if nodeID == "" {
		writeError(w, http.StatusBadRequest, "node_id is required")
		return
	}

	instances, err := h.store.ListPendingInstances(r.Context(), nodeID)
	if err != nil {
		log.Printf("[instance] Failed to list pending instances for node %s: %v", nodeID, err)
		writeError(w, http.StatusInternalServerError, "failed to list instances")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"instances": instances,
		"count":     len(instances),
	})
}

// UpdateStatus 更新实例状态（NodeManager 回调）
// PATCH /api/v1/instances/{id}
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	instanceID := r.PathValue("id")

	var req struct {
		Status        string  `json:"status"`
		ContainerName *string `json:"container_name,omitempty"`
		Error         *string `json:"error,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Status == "" {
		writeError(w, http.StatusBadRequest, "status is required")
		return
	}

	// 验证状态值
	status := model.InstanceStatus(req.Status)
	validStatuses := map[model.InstanceStatus]bool{
		model.InstanceStatusPending:  true,
		model.InstanceStatusCreating: true,
		model.InstanceStatusRunning:  true,
		model.InstanceStatusStopping: true,
		model.InstanceStatusStopped:  true,
		model.InstanceStatusError:    true,
	}
	if !validStatuses[status] {
		writeError(w, http.StatusBadRequest, "invalid status value")
		return
	}

	if err := h.store.UpdateInstance(r.Context(), instanceID, status, req.ContainerName); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "instance not found")
			return
		}
		log.Printf("[instance] Failed to update instance %s: %v", instanceID, err)
		writeError(w, http.StatusInternalServerError, "failed to update instance")
		return
	}

	log.Printf("[instance] Instance %s updated (status=%s, container=%v)",
		instanceID, req.Status, req.ContainerName)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "instance updated",
	})
}

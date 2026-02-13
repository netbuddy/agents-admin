// Package instance NodeManager 回调接口
//
// 本文件包含供 NodeManager 调用的 API 端点：
//   - ListByNode: 列出节点的 Agent 实例（支持 ?status=all 返回全部，默认仅待处理）
//   - UpdateStatus: 更新 Agent 实例状态（NodeManager 回调）
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
	mux.HandleFunc("GET /api/v1/nodes/{node_id}/agents", h.ListByNode)
	mux.HandleFunc("PATCH /api/v1/agents/{id}", h.UpdateStatus)
}

// ListByNode 列出节点的 Agent 实例（NodeManager 调用）
// GET /api/v1/nodes/{node_id}/agents          — 默认只返回待处理 Agent（pending/creating/stopping）
// GET /api/v1/nodes/{node_id}/agents?status=all — 返回该节点所有 Agent（对账用）
func (h *Handler) ListByNode(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("node_id")
	if nodeID == "" {
		writeError(w, http.StatusBadRequest, "node_id is required")
		return
	}

	statusFilter := r.URL.Query().Get("status")

	var instances []*model.Instance
	var err error
	if statusFilter == "all" {
		instances, err = h.store.ListAgentInstancesByNode(r.Context(), nodeID)
	} else {
		instances, err = h.store.ListPendingAgentInstances(r.Context(), nodeID)
	}
	if err != nil {
		log.Printf("[instance] Failed to list instances for node %s: %v", nodeID, err)
		writeError(w, http.StatusInternalServerError, "failed to list instances")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"agents": instances,
		"count":  len(instances),
	})
}

// UpdateStatus 更新 Agent 实例状态（NodeManager 回调）
// PATCH /api/v1/agents/{id}
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("id")

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

	if err := h.store.UpdateAgentInstance(r.Context(), agentID, status, req.ContainerName); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "agent not found")
			return
		}
		log.Printf("[agent] Failed to update agent %s: %v", agentID, err)
		writeError(w, http.StatusInternalServerError, "failed to update agent")
		return
	}

	log.Printf("[agent] Agent %s updated (status=%s, container=%v)",
		agentID, req.Status, req.ContainerName)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "agent updated",
	})
}

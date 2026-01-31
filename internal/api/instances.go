// Package api Instance 管理 API
//
// P2-2 重构后的实现：
//   - 状态持久化到 PostgreSQL（替代全局 map）
//   - Docker 操作声明式（API 只更新状态，Executor 执行实际操作）
//   - 支持多节点和 API Server 重启恢复
package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"agents-admin/internal/model"
)

// ListInstances 获取实例列表
func (h *Handler) ListInstances(w http.ResponseWriter, r *http.Request) {
	instances, err := h.store.ListInstances(r.Context())
	if err != nil {
		log.Printf("[Instances] Failed to list instances: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list instances")
		return
	}

	// 可选：按 agent_type 和 account_id 过滤
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

// CreateInstance 创建实例（声明式：只创建数据库记录，Executor 负责实际容器创建）
func (h *Handler) CreateInstance(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string `json:"name"`
		AccountID string `json:"account_id"`
		NodeID    string `json:"node_id"` // 指定运行节点
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.AccountID == "" {
		writeError(w, http.StatusBadRequest, "account_id is required")
		return
	}

	// 获取账号信息
	account, err := h.store.GetAccount(r.Context(), req.AccountID)
	if err != nil {
		log.Printf("[Instances] Failed to get account %s: %v", req.AccountID, err)
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

	// 检查账号是否已有 Volume
	if account.VolumeName == nil || *account.VolumeName == "" {
		writeError(w, http.StatusBadRequest, "account has no volume, please authenticate first")
		return
	}

	// 确定运行节点
	nodeID := req.NodeID
	if nodeID == "" {
		nodeID = account.NodeID // 默认使用账号绑定的节点
	}

	// 生成实例 ID
	instanceID := generateID("inst")

	if req.Name == "" {
		req.Name = instanceID
	}

	// 创建实例记录（状态为 pending，等待 Executor 创建容器）
	now := time.Now()
	instance := &model.Instance{
		ID:          instanceID,
		Name:        req.Name,
		AccountID:   req.AccountID,
		AgentTypeID: account.AgentTypeID,
		NodeID:      &nodeID,
		Status:      model.InstanceStatusPending, // 声明式：pending 状态，Executor 负责创建
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.store.CreateInstance(r.Context(), instance); err != nil {
		log.Printf("[Instances] Failed to create instance: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create instance")
		return
	}

	log.Printf("[Instances] Created instance %s (account=%s, node=%s, status=pending)",
		instanceID, req.AccountID, nodeID)

	writeJSON(w, http.StatusCreated, instance)
}

// GetInstance 获取实例详情
func (h *Handler) GetInstance(w http.ResponseWriter, r *http.Request) {
	instanceID := r.PathValue("id")

	instance, err := h.store.GetInstance(r.Context(), instanceID)
	if err != nil {
		log.Printf("[Instances] Failed to get instance %s: %v", instanceID, err)
		writeError(w, http.StatusInternalServerError, "failed to get instance")
		return
	}
	if instance == nil {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	writeJSON(w, http.StatusOK, instance)
}

// DeleteInstance 删除实例
//
// 说明：
// - 控制面仅删除数据库记录（实例会立刻从列表中消失）
// - 数据面（Executor）会在对账/GC 时清理“无对应实例记录”的孤儿容器，避免资源泄漏
func (h *Handler) DeleteInstance(w http.ResponseWriter, r *http.Request) {
	instanceID := r.PathValue("id")

	instance, err := h.store.GetInstance(r.Context(), instanceID)
	if err != nil {
		log.Printf("[Instances] Failed to get instance %s: %v", instanceID, err)
		writeError(w, http.StatusInternalServerError, "failed to get instance")
		return
	}
	if instance == nil {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	// 直接删除数据库记录
	// Executor 会在下次同步时发现容器已无对应记录，自动清理
	if err := h.store.DeleteInstance(r.Context(), instanceID); err != nil {
		log.Printf("[Instances] Failed to delete instance %s: %v", instanceID, err)
		writeError(w, http.StatusInternalServerError, "failed to delete instance")
		return
	}

	log.Printf("[Instances] Deleted instance %s", instanceID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "instance deleted",
	})
}

// StartInstance 启动实例（声明式：更新状态为 pending，Executor 负责实际启动）
func (h *Handler) StartInstance(w http.ResponseWriter, r *http.Request) {
	instanceID := r.PathValue("id")

	instance, err := h.store.GetInstance(r.Context(), instanceID)
	if err != nil {
		log.Printf("[Instances] Failed to get instance %s: %v", instanceID, err)
		writeError(w, http.StatusInternalServerError, "failed to get instance")
		return
	}
	if instance == nil {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	// 只有 stopped 或 error 状态可以启动
	if instance.Status != model.InstanceStatusStopped && instance.Status != model.InstanceStatusError {
		writeError(w, http.StatusBadRequest, "instance cannot be started in current state")
		return
	}

	// 更新状态为 pending，Executor 会处理启动
	if err := h.store.UpdateInstance(r.Context(), instanceID, model.InstanceStatusPending, nil); err != nil {
		log.Printf("[Instances] Failed to update instance %s: %v", instanceID, err)
		writeError(w, http.StatusInternalServerError, "failed to start instance")
		return
	}

	instance.Status = model.InstanceStatusPending
	log.Printf("[Instances] Instance %s marked for start (status=pending)", instanceID)

	writeJSON(w, http.StatusOK, instance)
}

// StopInstance 停止实例（声明式：更新状态为 stopping，Executor 负责实际停止）
func (h *Handler) StopInstance(w http.ResponseWriter, r *http.Request) {
	instanceID := r.PathValue("id")

	instance, err := h.store.GetInstance(r.Context(), instanceID)
	if err != nil {
		log.Printf("[Instances] Failed to get instance %s: %v", instanceID, err)
		writeError(w, http.StatusInternalServerError, "failed to get instance")
		return
	}
	if instance == nil {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	// 只有 running 状态可以停止
	if instance.Status != model.InstanceStatusRunning {
		writeError(w, http.StatusBadRequest, "instance is not running")
		return
	}

	// 更新状态为 stopping，Executor 会处理实际停止操作
	if err := h.store.UpdateInstance(r.Context(), instanceID, model.InstanceStatusStopping, nil); err != nil {
		log.Printf("[Instances] Failed to update instance %s: %v", instanceID, err)
		writeError(w, http.StatusInternalServerError, "failed to stop instance")
		return
	}

	instance.Status = model.InstanceStatusStopping
	log.Printf("[Instances] Instance %s marked for stop (status=stopping)", instanceID)

	writeJSON(w, http.StatusOK, instance)
}

// ============================================================================
// Executor API（供 Executor 调用）
// ============================================================================

// ListPendingInstances 列出节点待处理的实例（Executor 轮询用）
// GET /api/v1/nodes/{node_id}/instances
func (h *Handler) ListPendingInstances(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("node_id")
	if nodeID == "" {
		writeError(w, http.StatusBadRequest, "node_id is required")
		return
	}

	instances, err := h.store.ListPendingInstances(r.Context(), nodeID)
	if err != nil {
		log.Printf("[Instances] Failed to list pending instances for node %s: %v", nodeID, err)
		writeError(w, http.StatusInternalServerError, "failed to list instances")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"instances": instances,
		"count":     len(instances),
	})
}

// UpdateInstanceStatus 更新实例状态（Executor 回调）
// PATCH /api/v1/instances/{id}
func (h *Handler) UpdateInstanceStatus(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("[Instances] Failed to update instance %s: %v", instanceID, err)
		writeError(w, http.StatusInternalServerError, "failed to update instance")
		return
	}

	log.Printf("[Instances] Instance %s updated (status=%s, container=%v)",
		instanceID, req.Status, req.ContainerName)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "instance updated",
	})
}

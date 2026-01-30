package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"agents-admin/internal/model"
)

var (
	instances   = make(map[string]*model.Instance)
	instancesMu sync.RWMutex
)

// ListInstances 获取实例列表
func (h *Handler) ListInstances(w http.ResponseWriter, r *http.Request) {
	agentType := r.URL.Query().Get("agent_type")
	accountID := r.URL.Query().Get("account_id")

	instancesMu.RLock()
	defer instancesMu.RUnlock()

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
	})
}

// CreateInstance 创建实例
func (h *Handler) CreateInstance(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string `json:"name"`
		AccountID string `json:"account_id"`
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

	// 获取 Agent 类型
	var agentType *model.AgentType
	for _, at := range model.PredefinedAgentTypes {
		if at.ID == account.AgentTypeID {
			agentType = &at
			break
		}
	}
	if agentType == nil {
		writeError(w, http.StatusInternalServerError, "agent type not found")
		return
	}

	// 生成实例 ID 和容器名
	instanceID := fmt.Sprintf("inst_%s_%d", sanitizeName(req.AccountID), time.Now().Unix())
	containerName := fmt.Sprintf("agent_%s", instanceID)

	if req.Name == "" {
		req.Name = instanceID
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// 检查账号是否已有 Volume
	if account.VolumeName == nil || *account.VolumeName == "" {
		writeError(w, http.StatusBadRequest, "account has no volume, please authenticate first")
		return
	}

	// 创建容器
	// 使用 -t (tty) 保持容器运行，bash 需要 tty 否则会立即退出
	runCmd := exec.CommandContext(ctx, "docker", "run", "-d",
		"--name", containerName,
		"-v", fmt.Sprintf("%s:%s", *account.VolumeName, agentType.AuthDir),
		"--restart", "unless-stopped",
		"-t",           // 分配 tty，防止 bash 入口点立即退出
		"-i",           // 保持 stdin 打开
		agentType.Image)

	if err := runCmd.Run(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create container: "+err.Error())
		return
	}

	instance := &model.Instance{
		ID:          instanceID,
		Name:        req.Name,
		AccountID:   req.AccountID,
		AgentTypeID: account.AgentTypeID,
		Container:   containerName,
		Status:      model.InstanceStatusRunning,
		Node:        account.NodeID, // 实例运行在账号绑定的节点上
		CreatedAt:   time.Now(),
	}

	instancesMu.Lock()
	instances[instanceID] = instance
	instancesMu.Unlock()

	// TODO: 更新账号最后使用时间（待实现 Storage 方法）

	writeJSON(w, http.StatusCreated, instance)
}

// GetInstance 获取实例详情
func (h *Handler) GetInstance(w http.ResponseWriter, r *http.Request) {
	instanceID := r.PathValue("id")

	instancesMu.RLock()
	instance, ok := instances[instanceID]
	instancesMu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	// 更新容器状态
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	statusCmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Running}}", instance.Container)
	output, err := statusCmd.Output()
	if err != nil {
		instance.Status = model.InstanceStatusError
	} else if strings.TrimSpace(string(output)) == "true" {
		instance.Status = model.InstanceStatusRunning
	} else {
		instance.Status = model.InstanceStatusStopped
	}

	writeJSON(w, http.StatusOK, instance)
}

// DeleteInstance 删除实例
func (h *Handler) DeleteInstance(w http.ResponseWriter, r *http.Request) {
	instanceID := r.PathValue("id")

	instancesMu.Lock()
	instance, ok := instances[instanceID]
	if ok {
		delete(instances, instanceID)
	}
	instancesMu.Unlock()

	if !ok {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// 删除容器
	rmCmd := exec.CommandContext(ctx, "docker", "rm", "-f", instance.Container)
	rmCmd.Run()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "instance deleted",
	})
}

// StartInstance 启动实例
func (h *Handler) StartInstance(w http.ResponseWriter, r *http.Request) {
	instanceID := r.PathValue("id")

	instancesMu.RLock()
	instance, ok := instances[instanceID]
	instancesMu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	startCmd := exec.CommandContext(ctx, "docker", "start", instance.Container)
	if err := startCmd.Run(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start instance")
		return
	}

	instancesMu.Lock()
	instance.Status = model.InstanceStatusRunning
	instancesMu.Unlock()

	writeJSON(w, http.StatusOK, instance)
}

// StopInstance 停止实例
func (h *Handler) StopInstance(w http.ResponseWriter, r *http.Request) {
	instanceID := r.PathValue("id")

	instancesMu.RLock()
	instance, ok := instances[instanceID]
	instancesMu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	stopCmd := exec.CommandContext(ctx, "docker", "stop", instance.Container)
	if err := stopCmd.Run(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to stop instance")
		return
	}

	instancesMu.Lock()
	instance.Status = model.InstanceStatusStopped
	instancesMu.Unlock()

	writeJSON(w, http.StatusOK, instance)
}

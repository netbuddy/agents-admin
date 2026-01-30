package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"agents-admin/internal/model"
	"agents-admin/internal/storage"
)

// ListAgentTypes 获取支持的 Agent 类型列表
func (h *Handler) ListAgentTypes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"agent_types": model.PredefinedAgentTypes,
	})
}

// GetAgentType 获取指定 Agent 类型详情
func (h *Handler) GetAgentType(w http.ResponseWriter, r *http.Request) {
	typeID := r.PathValue("id")

	for _, at := range model.PredefinedAgentTypes {
		if at.ID == typeID {
			writeJSON(w, http.StatusOK, at)
			return
		}
	}

	writeError(w, http.StatusNotFound, "agent type not found")
}

// ListAccounts 获取账号列表
func (h *Handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	agentType := r.URL.Query().Get("agent_type")

	accounts, err := h.store.ListAccounts(ctx)
	if err != nil {
		log.Printf("[API] ListAccounts error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list accounts")
		return
	}

	// 过滤 agent_type
	var result []*model.Account
	for _, acc := range accounts {
		if agentType == "" || acc.AgentTypeID == agentType {
			result = append(result, acc)
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"accounts": result,
	})
}

// CreateAccount 创建新账号（声明式 API，只写数据库）
func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Name        string `json:"name"`
		AgentTypeID string `json:"agent_type"`
		NodeID      string `json:"node_id"` // 当前阶段必填，账号绑定到特定节点
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.AgentTypeID == "" || req.NodeID == "" {
		writeError(w, http.StatusBadRequest, "name, agent_type and node_id are required")
		return
	}

	// 验证 Agent 类型
	var agentType *model.AgentType
	for _, at := range model.PredefinedAgentTypes {
		if at.ID == req.AgentTypeID {
			agentType = &at
			break
		}
	}
	if agentType == nil {
		writeError(w, http.StatusBadRequest, "invalid agent_type")
		return
	}
	_ = agentType // 后续可能用到

	// 验证节点存在
	node, err := h.store.GetNode(ctx, req.NodeID)
	if err != nil || node == nil {
		writeError(w, http.StatusBadRequest, "invalid node_id: node not found")
		return
	}

	// 生成账号 ID
	accountID := generateAccountID(req.AgentTypeID, req.Name)

	now := time.Now()
	account := &model.Account{
		ID:          accountID,
		Name:        req.Name,
		AgentTypeID: req.AgentTypeID,
		NodeID:      req.NodeID, // 账号绑定到指定节点
		VolumeName:  nil,        // Volume 由 Node Agent 创建后回填
		Status:      model.AccountStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// 写入数据库
	if err := h.store.CreateAccount(ctx, account); err != nil {
		log.Printf("[API] CreateAccount error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create account")
		return
	}

	log.Printf("[API] Account created: %s (agent_type=%s)", accountID, req.AgentTypeID)
	writeJSON(w, http.StatusCreated, account)
}

// GetAccount 获取账号详情
func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID := r.PathValue("id")

	account, err := h.store.GetAccount(ctx, accountID)
	if err != nil {
		log.Printf("[API] GetAccount error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get account")
		return
	}

	if account == nil {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}

	writeJSON(w, http.StatusOK, account)
}

// DeleteAccount 删除账号
func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID := r.PathValue("id")

	// 检查账号是否存在
	account, err := h.store.GetAccount(ctx, accountID)
	if err != nil {
		log.Printf("[API] DeleteAccount error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get account")
		return
	}
	if account == nil {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}

	// 删除账号（级联删除关联的 auth_tasks）
	if err := h.store.DeleteAccount(ctx, accountID); err != nil {
		log.Printf("[API] DeleteAccount error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to delete account")
		return
	}

	log.Printf("[API] Account deleted: %s", accountID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "account deleted",
	})
}

// StartAccountAuth 发起账号认证（声明式 API，创建 AuthTask）
// 也用于重试：如果已有失败的任务，允许创建新任务
func (h *Handler) StartAccountAuth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID := r.PathValue("id")

	var req struct {
		Method  string `json:"method"`   // oauth, api_key
		ProxyID string `json:"proxy_id"` // 可选的代理 ID
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Method == "" {
		req.Method = "oauth" // 默认使用 OAuth
	}

	// 检查是否有进行中的认证任务
	existingSession, err := h.redisStore.GetAuthSessionByAccountID(ctx, accountID)
	if err != nil {
		log.Printf("[API] GetAuthSessionByAccountID error: %v", err)
	}
	if existingSession != nil {
		// 如果任务正在进行中（未执行完成），拒绝创建新任务
		isTerminal := existingSession.Status == "success" || existingSession.Status == "failed" || existingSession.Status == "timeout"
		if !isTerminal && !existingSession.Executed {
			// 任务还在等待执行或正在执行
			writeJSON(w, http.StatusConflict, map[string]interface{}{
				"error":   "authentication already in progress",
				"task_id": existingSession.TaskID,
				"status":  existingSession.Status,
			})
			return
		}
		// 如果是终态任务，允许创建新任务（重试）
		// 删除旧的会话索引，让新任务可以被正确关联
		if isTerminal {
			h.redisStore.DeleteAuthSession(ctx, existingSession.TaskID)
		}
	}

	// 检查账号是否存在
	account, err := h.store.GetAccount(ctx, accountID)
	if err != nil {
		log.Printf("[API] StartAccountAuth error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get account")
		return
	}
	if account == nil {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}

	// 验证 Agent 类型
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
	_ = agentType // 验证通过

	// 创建认证会话（使用 Redis 存储）
	now := time.Now()
	taskID := fmt.Sprintf("auth_%s_%d", accountID, now.Unix())
	session := &storage.AuthSession{
		TaskID:    taskID,
		AccountID: accountID,
		Method:    req.Method,
		NodeID:    account.NodeID, // 使用账号绑定的节点
		Status:    "assigned",
		ProxyID:   req.ProxyID, // 代理 ID（可选）
		CreatedAt: now,
		ExpiresAt: now.Add(10 * time.Minute),
	}

	// 写入 Redis
	if err := h.redisStore.CreateAuthSession(ctx, session); err != nil {
		log.Printf("[API] CreateAuthSession error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create auth session")
		return
	}

	// 更新账号状态为认证中
	if err := h.store.UpdateAccountStatus(ctx, accountID, model.AccountStatusAuthenticating); err != nil {
		log.Printf("[API] UpdateAccountStatus error: %v", err)
	}

	log.Printf("[API] AuthSession created: %s (account=%s, method=%s)", taskID, accountID, req.Method)

	// 发布初始事件到 Redis Streams
	h.publishAuthEventRedis(ctx, taskID, "auth.created", map[string]interface{}{
		"account_id": accountID,
		"method":     req.Method,
		"node_id":    session.NodeID,
		"status":     session.Status,
	})

	// 返回 API 响应
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":         taskID,
		"account_id": accountID,
		"status":     "pending", // 映射 assigned -> pending
		"expires_at": session.ExpiresAt,
	})
}

// GetAccountAuthStatus 获取认证状态
func (h *Handler) GetAccountAuthStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID := r.PathValue("id")

	// 从 Redis 获取认证会话
	session, err := h.redisStore.GetAuthSessionByAccountID(ctx, accountID)
	if err != nil {
		log.Printf("[API] GetAuthSessionByAccountID error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get auth status")
		return
	}

	if session == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status": "not_started",
		})
		return
	}

	// 映射状态
	status := session.Status
	switch status {
	case "assigned", "pending":
		status = "pending"
	case "running", "waiting_user", "waiting_oauth":
		status = "waiting"
	case "success":
		status = "success"
	case "failed", "timeout":
		status = "failed"
	}

	// 判断是否可以重试（只有终态任务可以重试）
	canRetry := status == "failed"

	// 返回 API 响应
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":            session.TaskID,
		"account_id":    session.AccountID,
		"status":        status,
		"callback_port": session.TerminalPort,
		"verify_url":    session.OAuthURL,
		"device_code":   session.UserCode,
		"message":       session.Message,
		"executed":      session.Executed,
		"executed_at":   session.ExecutedAt,
		"can_retry":     canRetry,
		"expires_at":    session.ExpiresAt,
	})
}

// GetAuthTask 获取认证任务详情（新 API）
func (h *Handler) GetAuthTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	taskID := r.PathValue("id")

	// 从 Redis 获取认证会话
	session, err := h.redisStore.GetAuthSession(ctx, taskID)
	if err != nil {
		log.Printf("[API] GetAuthSession error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get auth task")
		return
	}

	if session == nil {
		writeError(w, http.StatusNotFound, "auth task not found")
		return
	}

	writeJSON(w, http.StatusOK, session)
}

// GetNodeAuthTasks 获取节点的认证任务（Node Agent 调用）
func (h *Handler) GetNodeAuthTasks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	nodeID := r.PathValue("id")

	// 从 Redis 获取节点的认证会话
	sessions, err := h.redisStore.ListAuthSessionsByNode(ctx, nodeID)
	if err != nil {
		log.Printf("[API] ListAuthSessionsByNode error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list auth tasks")
		return
	}

	// 同时返回关联的账号信息，使用与executor匹配的字段名
	type AuthTaskForNode struct {
		ID        string           `json:"id"`
		AccountID string           `json:"account_id"`
		Method    string           `json:"method"`
		NodeID    string           `json:"node_id"`
		Status    string           `json:"status"`
		ProxyEnvs []string         `json:"proxy_envs,omitempty"` // 代理环境变量
		Account   *model.Account   `json:"account"`
		AgentType *model.AgentType `json:"agent_type"`
	}

	var result []AuthTaskForNode
	for _, session := range sessions {
		// 跳过已完成的任务（success/failed/timeout 是终态）
		if session.Status == "success" || session.Status == "failed" || session.Status == "timeout" {
			continue
		}

		// 跳过已执行的任务（每个任务只执行一次）
		if session.Executed {
			continue
		}

		account, _ := h.store.GetAccount(ctx, session.AccountID)
		var agentType *model.AgentType
		if account != nil {
			for _, at := range model.PredefinedAgentTypes {
				if at.ID == account.AgentTypeID {
					agentType = &at
					break
				}
			}
		}

		// 获取代理配置
		var proxyEnvs []string
		if session.ProxyID != "" {
			proxy, err := h.store.GetProxy(ctx, session.ProxyID)
			if err != nil {
				log.Printf("[API] GetProxy error: %v", err)
			} else if proxy != nil {
				proxyEnvs = proxy.ToEnvVars()
			}
		}

		result = append(result, AuthTaskForNode{
			ID:        session.TaskID,
			AccountID: session.AccountID,
			Method:    session.Method,
			NodeID:    session.NodeID,
			Status:    session.Status,
			ProxyEnvs: proxyEnvs,
			Account:   account,
			AgentType: agentType,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"auth_tasks": result,
	})
}

// UpdateAuthTask 更新认证任务状态（Node Agent 调用）
func (h *Handler) UpdateAuthTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	taskID := r.PathValue("id")

	var req struct {
		Status        string  `json:"status"`
		TerminalPort  *int    `json:"terminal_port,omitempty"`
		TerminalURL   *string `json:"terminal_url,omitempty"`
		ContainerName *string `json:"container_name,omitempty"`
		OAuthURL      *string `json:"oauth_url,omitempty"`
		UserCode      *string `json:"user_code,omitempty"`
		VolumeName    *string `json:"volume_name,omitempty"`
		Message       *string `json:"message,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// 从 Redis 获取现有会话
	session, err := h.redisStore.GetAuthSession(ctx, taskID)
	if err != nil {
		log.Printf("[API] GetAuthSession error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get auth task")
		return
	}
	if session == nil {
		writeError(w, http.StatusNotFound, "auth task not found")
		return
	}

	// 构建更新字段
	updates := map[string]interface{}{
		"status": req.Status,
	}

	// 当任务开始执行时，标记为已执行（防止重复执行）
	if req.Status == "running" && !session.Executed {
		updates["executed"] = true
		updates["executed_at"] = time.Now().Format(time.RFC3339)
	}

	if req.TerminalPort != nil {
		updates["terminal_port"] = *req.TerminalPort
	}
	if req.TerminalURL != nil {
		updates["terminal_url"] = *req.TerminalURL
	}
	if req.ContainerName != nil {
		updates["container_name"] = *req.ContainerName
	}
	if req.OAuthURL != nil {
		updates["oauth_url"] = *req.OAuthURL
	}
	if req.UserCode != nil {
		updates["user_code"] = *req.UserCode
	}
	if req.Message != nil {
		updates["message"] = *req.Message
	}

	// 更新 Redis 会话
	if err := h.redisStore.UpdateAuthSession(ctx, taskID, updates); err != nil {
		log.Printf("[API] UpdateAuthSession error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update auth task")
		return
	}

	// 发布事件到 Redis Streams
	h.publishAuthEventRedis(ctx, taskID, "auth."+req.Status, updates)

	// 如果提供了 Volume 名称，更新账号的 Volume
	if req.VolumeName != nil {
		if err := h.store.UpdateAccountVolume(ctx, session.AccountID, *req.VolumeName); err != nil {
			log.Printf("[API] UpdateAccountVolume error: %v", err)
		}
	}

	// 如果认证成功，更新账号状态
	if req.Status == "success" {
		if err := h.store.UpdateAccountStatus(ctx, session.AccountID, model.AccountStatusAuthenticated); err != nil {
			log.Printf("[API] UpdateAccountStatus error: %v", err)
		}
		log.Printf("[API] Account authenticated: %s", session.AccountID)
	} else if req.Status == "failed" || req.Status == "timeout" {
		// 只有当账号还未认证时，才将状态改回 pending
		// 防止已认证的账号因为后续失败的任务而被错误地重置
		account, err := h.store.GetAccount(ctx, session.AccountID)
		if err != nil {
			log.Printf("[API] GetAccount error: %v", err)
		} else if account != nil && account.Status != model.AccountStatusAuthenticated {
			if err := h.store.UpdateAccountStatus(ctx, session.AccountID, model.AccountStatusPending); err != nil {
				log.Printf("[API] UpdateAccountStatus error: %v", err)
			}
		}
	}

	log.Printf("[API] AuthSession updated: %s (status=%s)", taskID, req.Status)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "auth task updated",
	})
}

// scheduleAuthTasks 调度待处理的认证任务（在 Scheduler 中调用）
func (h *Handler) scheduleAuthTasks(ctx context.Context) {
	// 获取待调度的认证任务
	tasks, err := h.store.ListPendingAuthTasks(ctx, 10)
	if err != nil {
		log.Printf("[Scheduler] ListPendingAuthTasks error: %v", err)
		return
	}

	if len(tasks) == 0 {
		return
	}

	// 获取在线节点
	nodes, err := h.store.ListOnlineNodes(ctx)
	if err != nil {
		log.Printf("[Scheduler] ListOnlineNodes error: %v", err)
		return
	}

	if len(nodes) == 0 {
		log.Printf("[Scheduler] No online nodes available for auth tasks")
		return
	}

	// 简单轮询调度
	for i, task := range tasks {
		node := nodes[i%len(nodes)]

		if err := h.store.UpdateAuthTaskAssignment(ctx, task.ID, node.ID); err != nil {
			log.Printf("[Scheduler] UpdateAuthTaskAssignment error: %v", err)
			continue
		}

		log.Printf("[Scheduler] AuthTask %s assigned to node %s", task.ID, node.ID)
	}
}

// 辅助函数
func generateAccountID(agentType, name string) string {
	sanitized := sanitizeName(name)
	return fmt.Sprintf("%s_%s", agentType, sanitized)
}

func sanitizeName(name string) string {
	name = strings.ReplaceAll(name, "@", "_at_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, " ", "_")
	return strings.ToLower(name)
}

// publishAuthEvent 发布认证事件到 EventBus（已弃用，保留兼容）
func (h *Handler) publishAuthEvent(ctx context.Context, taskID, eventType string, data map[string]interface{}) {
	if h.eventBus == nil {
		return
	}

	event := &storage.WorkflowEvent{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
	}

	if err := h.eventBus.Publish(ctx, "auth", taskID, event); err != nil {
		log.Printf("[API] Failed to publish auth event: %v", err)
	}
}

// publishAuthEventRedis 发布认证事件到 Redis Streams
func (h *Handler) publishAuthEventRedis(ctx context.Context, taskID, eventType string, data map[string]interface{}) {
	if h.redisStore == nil {
		return
	}

	event := &storage.RedisWorkflowEvent{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
	}

	if err := h.redisStore.PublishEvent(ctx, "auth", taskID, event); err != nil {
		log.Printf("[API] Failed to publish auth event to Redis: %v", err)
	}
}

package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"agents-admin/internal/shared/model"
)

// CreateAuthOperation 创建异步认证操作（OAuth / Device Code）
//
// 由父包 operation.CreateOperation 分发调用
func (h *Handler) CreateAuthOperation(w http.ResponseWriter, r *http.Request, opType model.OperationType, config json.RawMessage, nodeID string) {
	ctx := r.Context()
	method := string(opType)

	// 解析配置
	var cfg model.OAuthConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid config")
		return
	}

	// 验证必填字段
	if cfg.Name == "" || cfg.AgentType == "" || nodeID == "" {
		writeError(w, http.StatusBadRequest, "config.name, config.agent_type, and node_id are required")
		return
	}

	// 验证 Agent 类型
	at := findAgentType(cfg.AgentType)
	if at == nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unknown agent type: %s", cfg.AgentType))
		return
	}
	if !agentTypeSupportsMethod(at, method) {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("%s not supported for agent type %s", method, cfg.AgentType))
		return
	}

	// 验证节点存在
	node, err := h.store.GetNode(ctx, nodeID)
	if err != nil {
		log.Printf("[auth] GetNode error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to check node")
		return
	}
	if node == nil {
		writeError(w, http.StatusBadRequest, "invalid node_id: node not found")
		return
	}

	// 创建 Operation + Action
	now := time.Now()
	opID := generateID("op")
	actID := generateID("act")

	op := &model.Operation{
		ID:        opID,
		Type:      opType,
		Config:    config,
		Status:    model.OperationStatusPending,
		NodeID:    nodeID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	action := &model.Action{
		ID:          actID,
		OperationID: opID,
		Status:      model.ActionStatusAssigned,
		Progress:    0,
		CreatedAt:   now,
	}

	if err := h.store.CreateOperation(ctx, op); err != nil {
		log.Printf("[auth] CreateOperation error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create operation")
		return
	}

	if err := h.store.CreateAction(ctx, action); err != nil {
		log.Printf("[auth] CreateAction error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create action")
		return
	}

	log.Printf("[auth] Created %s operation: %s (action: %s, node: %s)", method, opID, actID, nodeID)

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"operation_id": opID,
		"action_id":    actID,
		"type":         string(opType),
		"status":       "assigned",
	})
}

// CreateAPIKeyOperation 创建 API Key 认证操作（同步完成）
//
// 由父包 operation.CreateOperation 分发调用
func (h *Handler) CreateAPIKeyOperation(w http.ResponseWriter, r *http.Request, config json.RawMessage, nodeID string) {
	ctx := r.Context()

	var cfg model.APIKeyConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid config")
		return
	}

	// 验证必填字段
	if cfg.Name == "" || cfg.AgentType == "" || cfg.APIKey == "" || nodeID == "" {
		writeError(w, http.StatusBadRequest, "config.name, config.agent_type, config.api_key, and node_id are required")
		return
	}

	// 验证 Agent 类型
	at := findAgentType(cfg.AgentType)
	if at == nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unknown agent type: %s", cfg.AgentType))
		return
	}
	if !agentTypeSupportsMethod(at, "api_key") {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("api_key not supported for agent type %s", cfg.AgentType))
		return
	}

	// 验证节点存在
	node, err := h.store.GetNode(ctx, nodeID)
	if err != nil {
		log.Printf("[auth] GetNode error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to check node")
		return
	}
	if node == nil {
		writeError(w, http.StatusBadRequest, "invalid node_id: node not found")
		return
	}

	// API Key 同步完成：创建 Operation(completed) + Action(success) + Account(authenticated)
	now := time.Now()
	opID := generateID("op")
	actID := generateID("act")
	accountID := fmt.Sprintf("%s_%s", cfg.AgentType, sanitizeName(cfg.Name))
	volumeName := fmt.Sprintf("%s_%s_vol", cfg.AgentType, sanitizeName(cfg.Name))

	op := &model.Operation{
		ID:         opID,
		Type:       model.OperationTypeAPIKey,
		Config:     config,
		Status:     model.OperationStatusCompleted,
		NodeID:     nodeID,
		CreatedAt:  now,
		UpdatedAt:  now,
		FinishedAt: &now,
	}

	resultJSON, _ := json.Marshal(model.AuthActionResult{VolumeName: volumeName})
	action := &model.Action{
		ID:          actID,
		OperationID: opID,
		Status:      model.ActionStatusSuccess,
		Progress:    100,
		Result:      resultJSON,
		CreatedAt:   now,
		StartedAt:   &now,
		FinishedAt:  &now,
	}

	account := &model.Account{
		ID:          accountID,
		Name:        cfg.Name,
		AgentTypeID: cfg.AgentType,
		VolumeName:  &volumeName,
		Status:      model.AccountStatusAuthenticated,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// 持久化
	if err := h.store.CreateOperation(ctx, op); err != nil {
		log.Printf("[auth] CreateOperation error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create operation")
		return
	}
	if err := h.store.CreateAction(ctx, action); err != nil {
		log.Printf("[auth] CreateAction error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create action")
		return
	}
	if err := h.store.CreateAccount(ctx, account); err != nil {
		log.Printf("[auth] CreateAccount error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create account")
		return
	}

	log.Printf("[auth] API Key operation completed: %s → account %s", opID, accountID)

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"operation_id": opID,
		"action_id":    actID,
		"type":         "api_key",
		"status":       "success",
		"account_id":   accountID,
	})
}

// Package nodemanager AuthControllerV2 - 基于 Operation/Action 统一模型的认证控制器
//
// 使用 Docker API 直接操作容器，支持：
//   - 轮询 GET /nodes/{id}/actions 获取 assigned 状态的 Action
//   - 根据 Operation.type 分发到不同认证器
//   - 通过 PATCH /actions/{id} 上报状态
package nodemanager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"agents-admin/internal/nodemanager/auth"
	"agents-admin/internal/nodemanager/auth/qwencode"
	"agents-admin/internal/shared/model"
)

// AuthControllerV2 认证任务控制器V2（基于 Operation/Action 模型）
type AuthControllerV2 struct {
	config          Config
	httpClient      *http.Client
	containerClient *auth.Client
	authRegistry    *auth.Registry

	mu             sync.Mutex
	runningActions map[string]*runningAction
}

// runningAction 运行中的 Action
type runningAction struct {
	actionID      string
	authenticator auth.Authenticator
	cancel        context.CancelFunc
}

// NodeAction 从 API Server 获取的 Action（含 Operation 信息）
type NodeAction struct {
	ID          string          `json:"id"`
	OperationID string          `json:"operation_id"`
	Status      string          `json:"status"`
	Progress    int             `json:"progress"`
	Result      json.RawMessage `json:"result,omitempty"`
	Error       string          `json:"error,omitempty"`
	Operation   *NodeOperation  `json:"operation,omitempty"`
}

// NodeOperation 关联的 Operation 信息
type NodeOperation struct {
	ID     string          `json:"id"`
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config"`
	Status string          `json:"status"`
	NodeID string          `json:"node_id"`
}

// NewAuthControllerV2 创建认证控制器V2
func NewAuthControllerV2(cfg Config) (*AuthControllerV2, error) {
	containerClient, err := auth.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create container client: %w", err)
	}

	registry := auth.NewRegistry()
	registry.Register(qwencode.New())

	return &AuthControllerV2{
		config:          cfg,
		httpClient:      &http.Client{Timeout: 30 * time.Second},
		containerClient: containerClient,
		authRegistry:    registry,
		runningActions:  make(map[string]*runningAction),
	}, nil
}

// Close 关闭控制器
func (c *AuthControllerV2) Close() error {
	return c.containerClient.Close()
}

// Start 启动认证任务控制循环
func (c *AuthControllerV2) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.cleanupAllActions()
			return
		case <-ticker.C:
			c.pollAndExecuteActions(ctx)
		}
	}
}

// pollAndExecuteActions 轮询并执行 Action
func (c *AuthControllerV2) pollAndExecuteActions(ctx context.Context) {
	actions, err := c.fetchAssignedActions(ctx)
	if err != nil {
		log.Printf("[AuthController] Failed to fetch actions: %v", err)
		return
	}

	for _, action := range actions {
		actionID := action.ID

		c.mu.Lock()
		if _, exists := c.runningActions[actionID]; exists {
			c.mu.Unlock()
			continue
		}

		actionCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		c.runningActions[actionID] = &runningAction{
			actionID: actionID,
			cancel:   cancel,
		}
		c.mu.Unlock()

		go c.executeAction(actionCtx, action)
	}
}

// fetchAssignedActions 获取分配给本节点的 Action（含 Operation）
func (c *AuthControllerV2) fetchAssignedActions(ctx context.Context) ([]*NodeAction, error) {
	apiURL := fmt.Sprintf("%s/api/v1/nodes/%s/actions", c.config.APIServerURL, c.config.NodeID)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	var result struct {
		Actions []*NodeAction `json:"actions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Actions, nil
}

// executeAction 执行单个 Action（根据 Operation.type 分发）
func (c *AuthControllerV2) executeAction(ctx context.Context, action *NodeAction) {
	actionID := action.ID
	defer func() {
		c.mu.Lock()
		delete(c.runningActions, actionID)
		c.mu.Unlock()
	}()

	if action.Operation == nil {
		log.Printf("[AuthController] Action %s missing operation data", actionID)
		c.reportActionStatus(actionID, "failed", "", "", 0, nil, "missing operation data")
		return
	}

	opType := model.OperationType(action.Operation.Type)
	log.Printf("[AuthController] Executing action: %s (op=%s, type=%s)", actionID, action.OperationID, opType)

	switch opType {
	case model.OperationTypeOAuth, model.OperationTypeDeviceCode:
		c.executeAuthAction(ctx, action, string(opType))
	default:
		log.Printf("[AuthController] Unsupported operation type for auth controller: %s", opType)
		c.reportActionStatus(actionID, "failed", "", "", 0, nil, fmt.Sprintf("unsupported type: %s", opType))
	}
}

// executeAuthAction 执行认证类 Action
func (c *AuthControllerV2) executeAuthAction(ctx context.Context, action *NodeAction, method string) {
	actionID := action.ID

	// 解析 Operation config
	var config model.OAuthConfig
	if err := json.Unmarshal(action.Operation.Config, &config); err != nil {
		log.Printf("[AuthController] Failed to parse operation config: %v", err)
		c.reportActionStatus(actionID, "failed", "", "", 0, nil, "invalid operation config")
		return
	}

	// 查找 AgentType 配置
	agentTypeID := config.AgentType
	authenticator, ok := c.authRegistry.Get(agentTypeID)
	if !ok {
		log.Printf("[AuthController] No authenticator for agent type: %s", agentTypeID)
		c.reportActionStatus(actionID, "failed", "", "", 0, nil, fmt.Sprintf("unsupported agent type: %s", agentTypeID))
		return
	}

	// 保存认证器引用
	c.mu.Lock()
	if running, exists := c.runningActions[actionID]; exists {
		running.authenticator = authenticator
	}
	c.mu.Unlock()

	// 上报 running + initializing phase
	c.reportActionStatus(actionID, "running", "initializing", "Preparing authentication environment", 10, nil, "")

	// 构建认证任务
	volumeName := fmt.Sprintf("%s_%s_vol", agentTypeID, sanitizeForVolume(config.Name))
	agentTypeCfg := findPredefinedAgentType(agentTypeID)
	if agentTypeCfg == nil {
		c.reportActionStatus(actionID, "failed", "", "", 0, nil, "agent type config not found")
		return
	}

	authTask := &auth.AuthTask{
		ID:         actionID,
		AccountID:  fmt.Sprintf("%s_%s", agentTypeID, sanitizeForVolume(config.Name)),
		AgentType:  agentTypeID,
		Method:     method,
		Image:      agentTypeCfg.Image,
		AuthDir:    agentTypeCfg.AuthDir,
		AuthFile:   agentTypeCfg.AuthFile,
		LoginCmd:   agentTypeCfg.LoginCmd,
		VolumeName: volumeName,
	}

	// 上报 launching_container phase
	c.reportActionStatus(actionID, "running", "launching_container", "Launching authentication container", 15, nil, "")

	// 启动认证流程
	statusChan, err := authenticator.Start(ctx, authTask, c.containerClient)
	if err != nil {
		log.Printf("[AuthController] Failed to start authenticator: %v", err)
		c.reportActionStatus(actionID, "failed", "", "", 0, nil, err.Error())
		return
	}

	// 监听状态更新
	for status := range statusChan {
		c.handleAuthStatusUpdate(actionID, status, volumeName)
	}

	// 获取最终状态
	finalStatus := authenticator.GetStatus()
	switch finalStatus.State {
	case auth.AuthStateSuccess:
		log.Printf("[AuthController] Action %s completed successfully", actionID)
		result, _ := json.Marshal(map[string]string{"volume_name": volumeName})
		c.reportActionStatus(actionID, "success", "finalizing", "Authentication completed successfully", 100, result, "")
	case auth.AuthStateFailed:
		log.Printf("[AuthController] Action %s failed: %v", actionID, finalStatus.Error)
		c.reportActionStatus(actionID, "failed", "", finalStatus.Message, 0, nil, finalStatus.Message)
	case auth.AuthStateTimeout:
		log.Printf("[AuthController] Action %s timeout", actionID)
		c.reportActionStatus(actionID, "timeout", "", "Authentication timed out", 0, nil, "authentication timeout")
	}
}

// handleAuthStatusUpdate 处理认证状态更新
func (c *AuthControllerV2) handleAuthStatusUpdate(actionID string, status *auth.AuthStatus, volumeName string) {
	log.Printf("[AuthController] Action %s status: %s - %s", actionID, status.State, status.Message)

	switch status.State {
	case auth.AuthStateWaitingOAuth:
		result, _ := json.Marshal(map[string]string{
			"verify_url":  status.OAuthURL,
			"user_code":   status.UserCode,
			"volume_name": volumeName,
		})
		c.reportActionStatus(actionID, "waiting", "waiting_oauth", "Waiting for user to complete OAuth authorization", 50, result, "")
	case auth.AuthStateWaitingInput:
		c.reportActionStatus(actionID, "waiting", "waiting_input", status.Message, 50, nil, "")
	case auth.AuthStateRunning:
		c.reportActionStatus(actionID, "running", "authenticating", "Executing authentication flow", 30, nil, "")
	}
}

// reportActionStatus 上报 Action 状态到 API Server
func (c *AuthControllerV2) reportActionStatus(actionID, status, phase, message string, progress int, result json.RawMessage, errMsg string) {
	if status == "" {
		return
	}

	payload := map[string]interface{}{
		"status":   status,
		"phase":    phase,
		"message":  message,
		"progress": progress,
	}
	if result != nil {
		payload["result"] = json.RawMessage(result)
	}
	if errMsg != "" {
		payload["error"] = errMsg
	}

	body, _ := json.Marshal(payload)
	apiURL := fmt.Sprintf("%s/api/v1/actions/%s", c.config.APIServerURL, actionID)

	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "PATCH", apiURL, bytes.NewReader(body))
	if err != nil {
		log.Printf("[AuthController] Failed to create status request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[AuthController] Failed to report action status: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[AuthController] Report action status returned: %d", resp.StatusCode)
	}
}

// CancelAuthTask 取消认证任务
func (c *AuthControllerV2) CancelAuthTask(actionID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	running, exists := c.runningActions[actionID]
	if !exists {
		return fmt.Errorf("action %s not found", actionID)
	}

	if running.authenticator != nil {
		running.authenticator.Stop()
	}
	if running.cancel != nil {
		running.cancel()
	}

	delete(c.runningActions, actionID)
	return nil
}

// cleanupAllActions 清理所有运行中的 Action
func (c *AuthControllerV2) cleanupAllActions() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, running := range c.runningActions {
		if running.authenticator != nil {
			running.authenticator.Stop()
		}
		if running.cancel != nil {
			running.cancel()
		}
	}
	c.runningActions = make(map[string]*runningAction)
}

// sanitizeForVolume 将名称中的特殊字符替换（用于 volume 名称）
func sanitizeForVolume(name string) string {
	replacer := strings.NewReplacer("@", "_", ".", "_", " ", "_", "-", "_")
	return replacer.Replace(name)
}

// findPredefinedAgentType 查找预定义的 AgentType 配置
func findPredefinedAgentType(id string) *model.AgentTypeConfig {
	for _, at := range model.PredefinedAgentTypes {
		if at.ID == id {
			return &at
		}
	}
	return nil
}

// strPtr 返回字符串指针
func strPtr(s string) *string {
	return &s
}

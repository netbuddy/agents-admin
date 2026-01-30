// Package executor AuthTaskController V2 - 基于Docker API的认证任务控制器
//
// 使用Docker API直接操作容器，支持：
//   - 获取容器标准输入输出
//   - 解析输出中的OAuth URL
//   - 通过抽象认证器接口支持不同Agent
package executor

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

	"agents-admin/internal/storage"
	"agents-admin/pkg/auth"
	"agents-admin/pkg/auth/qwencode"
	"agents-admin/pkg/docker"
)

// AuthControllerV2 认证任务控制器V2
type AuthControllerV2 struct {
	config       Config
	httpClient   *http.Client
	dockerClient *docker.Client
	authRegistry *auth.Registry
	eventBus     *storage.EtcdEventBus // 事件总线（可选，用于事件驱动模式）

	mu               sync.Mutex
	runningAuthTasks map[string]*runningAuth
}

// runningAuth 运行中的认证任务
type runningAuth struct {
	taskID        string
	authenticator auth.Authenticator
	cancel        context.CancelFunc
}

// NewAuthControllerV2 创建认证控制器V2
func NewAuthControllerV2(cfg Config) (*AuthControllerV2, error) {
	dockerClient, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	// 创建认证器注册表
	registry := auth.NewRegistry()

	// 注册已实现的认证器
	registry.Register(qwencode.New())
	// 未来可以添加更多认证器:
	// registry.Register(claude.New())
	// registry.Register(openai.New())

	return &AuthControllerV2{
		config:           cfg,
		httpClient:       &http.Client{Timeout: 30 * time.Second},
		dockerClient:     dockerClient,
		authRegistry:     registry,
		runningAuthTasks: make(map[string]*runningAuth),
	}, nil
}

// Close 关闭控制器
func (c *AuthControllerV2) Close() error {
	return c.dockerClient.Close()
}

// SetEventBus 设置事件总线（用于事件驱动模式）
func (c *AuthControllerV2) SetEventBus(eventBus *storage.EtcdEventBus) {
	c.eventBus = eventBus
}

// Start 启动认证任务控制循环
func (c *AuthControllerV2) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.cleanupAllTasks()
			return
		case <-ticker.C:
			c.checkAndExecuteAuthTasks(ctx)
		}
	}
}

// checkAndExecuteAuthTasks 检查并执行认证任务
func (c *AuthControllerV2) checkAndExecuteAuthTasks(ctx context.Context) {
	tasks, err := c.fetchAssignedAuthTasks(ctx)
	if err != nil {
		log.Printf("[AuthControllerV2] Failed to fetch auth tasks: %v", err)
		return
	}

	for _, task := range tasks {
		taskID := task.ID

		c.mu.Lock()
		if _, exists := c.runningAuthTasks[taskID]; exists {
			c.mu.Unlock()
			continue
		}

		taskCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		c.runningAuthTasks[taskID] = &runningAuth{
			taskID: taskID,
			cancel: cancel,
		}
		c.mu.Unlock()

		go c.executeAuthTask(taskCtx, task)
	}
}

// fetchAssignedAuthTasks 获取分配给本节点的认证任务
func (c *AuthControllerV2) fetchAssignedAuthTasks(ctx context.Context) ([]*AuthTaskV2, error) {
	url := fmt.Sprintf("%s/api/v1/nodes/%s/auth-tasks", c.config.APIServerURL, c.config.NodeID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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
		AuthTasks []*AuthTaskV2 `json:"auth_tasks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.AuthTasks, nil
}

// AuthTaskV2 认证任务（从API Server获取）
type AuthTaskV2 struct {
	ID        string       `json:"id"`
	AccountID string       `json:"account_id"`
	Method    string       `json:"method"`
	NodeID    string       `json:"node_id"`
	Status    string       `json:"status"`
	ProxyEnvs []string     `json:"proxy_envs,omitempty"` // 代理环境变量
	Account   *AccountV2   `json:"account"`
	AgentType *AgentTypeV2 `json:"agent_type"`
}

// AccountV2 账号信息
type AccountV2 struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	AgentTypeID string  `json:"agent_type"`
	VolumeName  *string `json:"volume_name"`
	Status      string  `json:"status"`
}

// AgentTypeV2 Agent类型配置
type AgentTypeV2 struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Image    string `json:"image"`
	AuthDir  string `json:"auth_dir"`
	AuthFile string `json:"auth_file"`
	LoginCmd string `json:"login_cmd"`
}

// executeAuthTask 执行单个认证任务
func (c *AuthControllerV2) executeAuthTask(ctx context.Context, task *AuthTaskV2) {
	taskID := task.ID
	defer func() {
		c.mu.Lock()
		delete(c.runningAuthTasks, taskID)
		c.mu.Unlock()
	}()

	log.Printf("[AuthControllerV2] Executing auth task: %s (account=%s, method=%s)",
		taskID, task.AccountID, task.Method)

	// 验证任务数据
	if task.Account == nil || task.AgentType == nil {
		log.Printf("[AuthControllerV2] Task %s missing account or agent_type data", taskID)
		c.reportAuthTaskStatus(ctx, taskID, "failed", nil, nil)
		return
	}

	// 获取对应的认证器
	agentTypeID := task.AgentType.ID
	authenticator, ok := c.authRegistry.Get(agentTypeID)
	if !ok {
		log.Printf("[AuthControllerV2] No authenticator found for agent type: %s", agentTypeID)
		c.reportAuthTaskStatus(ctx, taskID, "failed", nil, strPtr(fmt.Sprintf("unsupported agent type: %s", agentTypeID)))
		return
	}

	// 保存认证器引用
	c.mu.Lock()
	if running, exists := c.runningAuthTasks[taskID]; exists {
		running.authenticator = authenticator
	}
	c.mu.Unlock()

	// 更新状态为 running
	c.reportAuthTaskStatus(ctx, taskID, "running", nil, nil)

	// 构建认证任务配置
	volumeName := fmt.Sprintf("%s_auth", strings.ReplaceAll(task.AccountID, "-", "_"))
	if task.Account.VolumeName != nil && *task.Account.VolumeName != "" {
		volumeName = *task.Account.VolumeName
	}

	authTask := &auth.AuthTask{
		ID:         taskID,
		AccountID:  task.AccountID,
		AgentType:  agentTypeID,
		Method:     task.Method,
		Image:      task.AgentType.Image,
		AuthDir:    task.AgentType.AuthDir,
		AuthFile:   task.AgentType.AuthFile,
		LoginCmd:   task.AgentType.LoginCmd,
		VolumeName: volumeName,
		ProxyEnvs:  task.ProxyEnvs, // 代理环境变量
	}

	// 启动认证流程
	statusChan, err := authenticator.Start(ctx, authTask, c.dockerClient)
	if err != nil {
		log.Printf("[AuthControllerV2] Failed to start authenticator: %v", err)
		c.reportAuthTaskStatus(ctx, taskID, "failed", nil, strPtr(err.Error()))
		return
	}

	// 上报 volume 名称
	c.reportAuthTaskVolume(ctx, taskID, volumeName)

	// 监听状态更新
	for status := range statusChan {
		c.handleStatusUpdate(ctx, taskID, status)
	}

	// 获取最终状态
	finalStatus := authenticator.GetStatus()
	if finalStatus.State == auth.AuthStateSuccess {
		log.Printf("[AuthControllerV2] Auth task %s completed successfully", taskID)
		c.reportAuthTaskStatus(ctx, taskID, "success", nil, strPtr("authentication completed"))
	} else if finalStatus.State == auth.AuthStateFailed {
		log.Printf("[AuthControllerV2] Auth task %s failed: %v", taskID, finalStatus.Error)
		c.reportAuthTaskStatus(ctx, taskID, "failed", nil, strPtr(finalStatus.Message))
	} else if finalStatus.State == auth.AuthStateTimeout {
		log.Printf("[AuthControllerV2] Auth task %s timeout", taskID)
		c.reportAuthTaskStatus(ctx, taskID, "timeout", nil, strPtr("authentication timeout"))
	}
}

// handleStatusUpdate 处理状态更新
func (c *AuthControllerV2) handleStatusUpdate(ctx context.Context, taskID string, status *auth.AuthStatus) {
	log.Printf("[AuthControllerV2] Task %s status update: %s - %s", taskID, status.State, status.Message)

	switch status.State {
	case auth.AuthStateWaitingOAuth:
		// 上报OAuth URL给前端
		c.reportAuthTaskOAuthURL(ctx, taskID, status.OAuthURL, status.UserCode)
	case auth.AuthStateWaitingInput:
		c.reportAuthTaskStatus(ctx, taskID, "waiting_user", nil, strPtr(status.Message))
	case auth.AuthStateRunning:
		c.reportAuthTaskStatus(ctx, taskID, "running", nil, strPtr(status.Message))
	}
}

// reportAuthTaskStatus 上报认证任务状态
func (c *AuthControllerV2) reportAuthTaskStatus(ctx context.Context, taskID, status string, oauthURL *string, message *string) {
	// 如果状态为空，跳过更新避免覆盖现有状态
	if status == "" {
		log.Printf("[AuthControllerV2] Skipping status update for task %s: empty status", taskID)
		return
	}

	// 如果有 EventBus，发布事件（用于实时通知）
	if c.eventBus != nil {
		c.publishAuthEvent(ctx, taskID, "auth."+status, map[string]interface{}{
			"status":  status,
			"message": message,
		})
	}

	// 始终通过 HTTP 更新 Redis 状态（确保前端轮询能获取到数据）
	payload := map[string]interface{}{
		"status": status,
	}
	if oauthURL != nil {
		payload["oauth_url"] = *oauthURL
	}
	if message != nil {
		payload["message"] = *message
	}

	body, _ := json.Marshal(payload)
	apiURL := fmt.Sprintf("%s/api/v1/auth-tasks/%s", c.config.APIServerURL, taskID)

	// 使用独立的 context 避免被取消
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "PATCH", apiURL, bytes.NewReader(body))
	if err != nil {
		log.Printf("[AuthControllerV2] Failed to create status request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[AuthControllerV2] Failed to report auth task status: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[AuthControllerV2] Report auth task status returned: %d", resp.StatusCode)
	}
}

// reportAuthTaskOAuthURL 上报OAuth URL
func (c *AuthControllerV2) reportAuthTaskOAuthURL(ctx context.Context, taskID, oauthURL, userCode string) {
	log.Printf("[AuthControllerV2] Reporting OAuth URL for task %s: %s", taskID, oauthURL)

	// 如果有 EventBus，发布事件（用于实时通知）
	if c.eventBus != nil {
		c.publishAuthEventWithState(ctx, taskID, "auth.oauth_url", map[string]interface{}{
			"oauth_url": oauthURL,
			"user_code": userCode,
			"message":   fmt.Sprintf("Please visit %s to complete authentication", oauthURL),
		}, storage.WorkflowStateWaiting)
	}

	// 始终通过 HTTP 更新 Redis 状态（确保前端轮询能获取到数据）
	payload := map[string]interface{}{
		"status":    "waiting_oauth",
		"oauth_url": oauthURL,
		"user_code": userCode,
		"message":   fmt.Sprintf("Please visit %s to complete authentication", oauthURL),
	}

	body, _ := json.Marshal(payload)
	apiURL := fmt.Sprintf("%s/api/v1/auth-tasks/%s", c.config.APIServerURL, taskID)

	// 使用独立的 context 避免被取消
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "PATCH", apiURL, bytes.NewReader(body))
	if err != nil {
		log.Printf("[AuthControllerV2] Failed to create OAuth URL request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[AuthControllerV2] Failed to report OAuth URL: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[AuthControllerV2] Report OAuth URL returned status: %d", resp.StatusCode)
	} else {
		log.Printf("[AuthControllerV2] Successfully reported OAuth URL for task %s", taskID)
	}
}

// reportAuthTaskVolume 上报Volume名称
func (c *AuthControllerV2) reportAuthTaskVolume(ctx context.Context, taskID, volumeName string) {
	payload := map[string]interface{}{
		"volume_name": volumeName,
	}

	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/api/v1/auth-tasks/%s", c.config.APIServerURL, taskID)
	req, _ := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[AuthControllerV2] Failed to report volume name: %v", err)
		return
	}
	resp.Body.Close()
}

// CancelAuthTask 取消认证任务
func (c *AuthControllerV2) CancelAuthTask(taskID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	running, exists := c.runningAuthTasks[taskID]
	if !exists {
		return fmt.Errorf("auth task %s not found", taskID)
	}

	// 停止认证器
	if running.authenticator != nil {
		running.authenticator.Stop()
	}

	// 取消上下文
	if running.cancel != nil {
		running.cancel()
	}

	delete(c.runningAuthTasks, taskID)
	return nil
}

// cleanupAllTasks 清理所有任务
func (c *AuthControllerV2) cleanupAllTasks() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, running := range c.runningAuthTasks {
		if running.authenticator != nil {
			running.authenticator.Stop()
		}
		if running.cancel != nil {
			running.cancel()
		}
	}
	c.runningAuthTasks = make(map[string]*runningAuth)
}

// strPtr 返回字符串指针
func strPtr(s string) *string {
	return &s
}

// publishAuthEvent 通过 EventBus 发布认证事件
func (c *AuthControllerV2) publishAuthEvent(ctx context.Context, taskID, eventType string, data map[string]interface{}) {
	if c.eventBus == nil {
		return
	}

	event := &storage.WorkflowEvent{
		ID:         fmt.Sprintf("evt_%s_%d", taskID, time.Now().UnixNano()),
		WorkflowID: taskID,
		Type:       eventType,
		Data:       data,
		ProducerID: c.config.NodeID,
		Timestamp:  time.Now(),
	}

	if err := c.eventBus.Publish(ctx, "auth", taskID, event); err != nil {
		log.Printf("[AuthControllerV2] Failed to publish event via EventBus: %v", err)
	}
}

// publishAuthEventWithState 通过 EventBus 发布认证事件并更新状态
func (c *AuthControllerV2) publishAuthEventWithState(ctx context.Context, taskID, eventType string, data map[string]interface{}, state storage.WorkflowState) {
	if c.eventBus == nil {
		return
	}

	event := &storage.WorkflowEvent{
		ID:         fmt.Sprintf("evt_%s_%d", taskID, time.Now().UnixNano()),
		WorkflowID: taskID,
		Type:       eventType,
		Data:       data,
		ProducerID: c.config.NodeID,
		Timestamp:  time.Now(),
	}

	stateData := &storage.WorkflowStateData{
		ID:    taskID,
		Type:  "auth",
		State: state,
		Data:  data,
	}

	if err := c.eventBus.PublishWithState(ctx, "auth", taskID, event, stateData); err != nil {
		log.Printf("[AuthControllerV2] Failed to publish event with state via EventBus: %v", err)
	}
}

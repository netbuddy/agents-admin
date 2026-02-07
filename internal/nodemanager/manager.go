// Package nodemanager 节点管理器
//
// 目录结构：
//   - manager.go:             NodeManager 主体
//   - types.go:               公共类型定义
//   - auth_controller.go:     认证控制器
//   - container_instance.go:  Instance 容器管理
//   - container_terminal.go:  Terminal 终端管理
//   - workspace_manager.go:   工作空间管理
//   - heartbeat_service.go:   心跳服务
//   - metrics_prometheus.go:  Prometheus 指标
//   - handler/:               Handler 插件框架
//   - interface.go:         Handler 接口
//   - registry.go:          Handler 注册表
//   - auth.go:              认证 Handler
//   - container.go:         容器 Handler
//   - runner.go:            Run 执行 Handler
package nodemanager

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"agents-admin/internal/nodemanager/adapter"
	"agents-admin/internal/nodemanager/handler"
	"agents-admin/internal/shared/queue"
	"agents-admin/internal/shared/storage"
)

// Config 节点管理器配置
// 包含节点标识、API 服务器地址、工作空间目录等
type Config struct {
	NodeID       string            // 节点唯一标识
	APIServerURL string            // API Server 地址
	WorkspaceDir string            // 工作空间根目录
	Labels       map[string]string // 节点标签（用于调度匹配）
	HTTPClient   *http.Client      // 自定义 HTTP 客户端（可选，用于 TLS）
}

// NodeManager 节点管理器核心结构
// 负责与 API Server 通信、执行任务、上报事件
type NodeManager struct {
	config           Config                        // 配置
	httpClient       *http.Client                  // HTTP 客户端
	adapters         *adapter.Registry             // Adapter 注册表
	mu               sync.Mutex                    // 保护 running map
	running          map[string]context.CancelFunc // 运行中的任务
	authController   *AuthControllerV2             // 认证任务控制器
	eventBus         *storage.EtcdEventBus         // 事件总线（可选，用于事件驱动模式）
	nodeQueue        queue.NodeRunQueue            // 节点队列（用于接收分配的 Run）
	instanceWorker   *InstanceWorker               // Instance 工作线程（P2-1）
	terminalWorker   *TerminalWorker               // Terminal 工作线程（P2-1）
	workspaceManager *WorkspaceManager             // Workspace 管理器

	// 新架构：Handler 注册表
	handlerRegistry *handler.Registry
}

// NewNodeManager 创建节点管理器实例
func NewNodeManager(cfg Config) (*NodeManager, error) {
	authController, err := NewAuthControllerV2(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth controller: %w", err)
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	return &NodeManager{
		config:           cfg,
		httpClient:       httpClient,
		adapters:         adapter.NewRegistry(),
		running:          make(map[string]context.CancelFunc),
		authController:   authController,
		instanceWorker:   NewInstanceWorker(cfg),                // P2-1: Instance 工作线程
		terminalWorker:   NewTerminalWorker(cfg),                // P2-1: Terminal 工作线程
		workspaceManager: NewWorkspaceManager(cfg.WorkspaceDir), // Workspace 管理器
		handlerRegistry:  handler.NewRegistry(),                 // 新架构：Handler 注册表
	}, nil
}

// RegisterHandler 注册 Handler（新架构）
func (nm *NodeManager) RegisterHandler(h handler.Handler) error {
	if nm.handlerRegistry == nil {
		nm.handlerRegistry = handler.NewRegistry()
	}
	return nm.handlerRegistry.Register(h)
}

// GetHandlerRegistry 获取 Handler 注册表
func (nm *NodeManager) GetHandlerRegistry() *handler.Registry {
	return nm.handlerRegistry
}

// RegisterAdapter 注册 Adapter
// RegisterAdapter 注册 Agent CLI 适配器
func (nm *NodeManager) RegisterAdapter(a adapter.Adapter) {
	nm.adapters.Register(a)
}

// SetEventBus 设置事件总线（用于事件驱动模式）
func (nm *NodeManager) SetEventBus(eventBus *storage.EtcdEventBus) {
	nm.eventBus = eventBus
}

// SetNodeQueue 设置节点队列（用于接收分配的 Run）
func (nm *NodeManager) SetNodeQueue(q queue.NodeRunQueue) {
	nm.nodeQueue = q
}

// Start 启动节点管理器
func (nm *NodeManager) Start(ctx context.Context) {
	log.Printf("[nodemanager] started: %s", nm.config.NodeID)
	if nm.handlerRegistry != nil {
		log.Printf("[nodemanager] registered handlers: %v", nm.handlerRegistry.List())
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		nm.heartbeatLoop(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		nm.taskLoop(ctx)
	}()

	// 认证任务控制循环
	if nm.authController != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			nm.authController.Start(ctx)
		}()
	}

	// P2-1: Instance 工作线程（处理容器创建/启动/停止）
	if nm.instanceWorker != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			nm.instanceWorker.Start(ctx)
		}()
	}

	// P2-1: Terminal 工作线程（处理终端会话启动/关闭）
	if nm.terminalWorker != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			nm.terminalWorker.Start(ctx)
		}()
	}

	// 新架构：启动所有注册的 Handler
	if nm.handlerRegistry != nil {
		nm.handlerRegistry.StartAll(ctx, &wg)
	}

	wg.Wait()
	log.Println("[nodemanager] stopped")
}

func (nm *NodeManager) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	nm.sendHeartbeat(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			nm.sendHeartbeat(ctx)
		}
	}
}

func (nm *NodeManager) sendHeartbeat(ctx context.Context) {
	payload := map[string]interface{}{
		"node_id": nm.config.NodeID,
		"status":  "online",
		"labels":  nm.config.Labels,
		"capacity": map[string]interface{}{
			"max_concurrent": 2,
			"available":      2 - len(nm.running),
		},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST",
		nm.config.APIServerURL+"/api/v1/nodes/heartbeat",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := nm.httpClient.Do(req)
	if err != nil {
		log.Printf("Heartbeat failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Heartbeat returned status: %d", resp.StatusCode)
	}
}

// taskLoop 任务获取主循环
//
// 实现两种模式：
//  1. 事件驱动模式（主路径）：消费 Redis Stream nodes:{node_id}:tasks
//  2. 保底轮询模式：每 60 秒从 API Server 获取分配的任务
//
// 如果 Redis 可用，优先使用事件驱动模式；否则只使用轮询模式。
func (nm *NodeManager) taskLoop(ctx context.Context) {
	var wg sync.WaitGroup

	// 主路径：Redis Streams 消费（如果 Redis 可用）
	if nm.nodeQueue != nil {
		// 创建消费者组
		if err := nm.nodeQueue.CreateNodeConsumerGroup(ctx, nm.config.NodeID); err != nil {
			log.Printf("[nodemanager.redis.group.failed] node_id=%s error=%v", nm.config.NodeID, err)
		} else {
			log.Printf("[nodemanager.redis.group.created] node_id=%s", nm.config.NodeID)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			nm.consumeNodeTasks(ctx)
		}()
	}

	// 保底路径：HTTP 轮询（60 秒间隔）
	wg.Add(1)
	go func() {
		defer wg.Done()
		nm.fallbackTaskPolling(ctx)
	}()

	wg.Wait()
}

// consumeNodeTasks 消费 Redis Stream 中的任务事件
func (nm *NodeManager) consumeNodeTasks(ctx context.Context) {
	log.Printf("[nodemanager.redis.start] node_id=%s", nm.config.NodeID)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[nodemanager.redis.stop] reason=context_cancelled")
			return
		default:
		}

		// 消费消息（阻塞等待 5 秒）
		messages, err := nm.nodeQueue.ConsumeNodeRuns(ctx, nm.config.NodeID, nm.config.NodeID, 10, 5*time.Second)
		if err != nil {
			log.Printf("[nodemanager.redis.consume.failed] error=%v", err)
			time.Sleep(1 * time.Second) // 出错后短暂休眠
			continue
		}

		if len(messages) == 0 {
			continue // 超时，没有新消息
		}

		log.Printf("[nodemanager.redis.received] count=%d", len(messages))

		for _, msg := range messages {
			// 获取 Run 详情并执行
			nm.processTaskMessage(ctx, msg.RunID)

			// 确认消息
			if err := nm.nodeQueue.AckNodeRun(ctx, nm.config.NodeID, msg.ID); err != nil {
				log.Printf("[nodemanager.redis.ack.failed] msg_id=%s error=%v", msg.ID, err)
			}
		}
	}
}

// fallbackTaskPolling 保底轮询：每 60 秒从 API Server 获取任务
func (nm *NodeManager) fallbackTaskPolling(ctx context.Context) {
	// 如果没有 Redis，使用更短的间隔
	interval := 60 * time.Second
	if nm.nodeQueue == nil {
		interval = 5 * time.Second // 无 Redis 时保持原有的 5 秒间隔
	}

	// 启动时立即执行一次
	nm.checkAndExecuteRuns(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			nm.checkAndExecuteRuns(ctx)
		}
	}
}

// processTaskMessage 处理任务消息
func (nm *NodeManager) processTaskMessage(ctx context.Context, runID string) {
	// 检查是否已在执行
	nm.mu.Lock()
	if _, exists := nm.running[runID]; exists {
		nm.mu.Unlock()
		log.Printf("[nodemanager.run.skip] run_id=%s reason=already_running", runID)
		return
	}
	nm.mu.Unlock()

	// 获取 Run 详情
	run, err := nm.fetchRunByID(ctx, runID)
	if err != nil {
		log.Printf("[nodemanager.run.fetch.failed] run_id=%s error=%v", runID, err)
		return
	}
	if run == nil {
		log.Printf("[nodemanager.run.not_found] run_id=%s", runID)
		return
	}

	// 启动执行
	nm.mu.Lock()
	runCtx, cancel := context.WithCancel(ctx)
	nm.running[runID] = cancel
	nm.mu.Unlock()

	go nm.executeRun(runCtx, run)
}

// fetchRunByID 根据 Run ID 获取 Run 详情
func (nm *NodeManager) fetchRunByID(ctx context.Context, runID string) (map[string]interface{}, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET",
		nm.config.APIServerURL+"/api/v1/runs/"+runID, nil)

	resp, err := nm.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func (nm *NodeManager) checkAndExecuteRuns(ctx context.Context) {
	runs, err := nm.fetchAssignedRuns(ctx)
	if err != nil {
		log.Printf("Failed to fetch runs: %v", err)
		return
	}

	for _, run := range runs {
		runID := run["id"].(string)

		nm.mu.Lock()
		if _, exists := nm.running[runID]; exists {
			nm.mu.Unlock()
			continue
		}

		runCtx, cancel := context.WithCancel(ctx)
		nm.running[runID] = cancel
		nm.mu.Unlock()

		go nm.executeRun(runCtx, run)
	}
}

func (nm *NodeManager) fetchAssignedRuns(ctx context.Context) ([]map[string]interface{}, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET",
		nm.config.APIServerURL+"/api/v1/nodes/"+nm.config.NodeID+"/runs", nil)

	resp, err := nm.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	var result struct {
		Runs []map[string]interface{} `json:"runs"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Runs, nil
}

// executeRun 执行单个 Run
// 从 snapshot 中解析 TaskSpec，调用 Adapter 构建命令并执行
func (nm *NodeManager) executeRun(ctx context.Context, run map[string]interface{}) {
	runID := run["id"].(string)
	defer func() {
		nm.mu.Lock()
		delete(nm.running, runID)
		nm.mu.Unlock()
	}()

	log.Printf("执行任务: %s", runID)

	// 解析 snapshot 中的任务配置（带类型安全检查）
	snapshot, ok := run["snapshot"].(map[string]interface{})
	if !ok || snapshot == nil {
		nm.reportError(ctx, runID, "任务快照 (snapshot) 缺失或格式错误")
		return
	}

	agentConfig, ok := snapshot["agent"].(map[string]interface{})
	if !ok || agentConfig == nil {
		nm.reportError(ctx, runID, "Agent 配置 (snapshot.agent) 缺失或格式错误")
		return
	}

	agentType, ok := agentConfig["type"].(string)
	if !ok || agentType == "" {
		nm.reportError(ctx, runID, "Agent 类型 (snapshot.agent.type) 缺失或格式错误")
		return
	}

	prompt, ok := snapshot["prompt"].(string)
	if !ok || prompt == "" {
		nm.reportError(ctx, runID, "任务提示 (snapshot.prompt) 缺失或格式错误")
		return
	}

	// 获取对应的 Adapter
	// Agent type 到 adapter name 的映射
	// 支持多种格式：qwen-code -> qwencode-v1, qwencode -> qwencode-v1
	adapterName := normalizeAdapterName(agentType)
	a, adapterOk := nm.adapters.Get(adapterName)
	if !adapterOk {
		nm.reportError(ctx, runID, fmt.Sprintf("找不到适配器: %s (原始类型: %s)", adapterName, agentType))
		return
	}

	// 构建 TaskSpec（任务描述）
	spec := &adapter.TaskSpec{
		ID:     runID,
		Prompt: prompt,
	}

	// 构建 AgentConfig（执行者配置）
	// 从 snapshot 中提取模型和参数
	model, _ := agentConfig["model"].(string)
	parameters, _ := agentConfig["parameters"].(map[string]interface{})
	if parameters == nil {
		// 兼容旧格式：直接使用 agentConfig 作为参数
		parameters = agentConfig
	}

	agent := &adapter.AgentConfig{
		Type:       agentType,
		Model:      model,
		Parameters: parameters,
	}

	// 构建运行配置
	runConfig, err := a.BuildCommand(ctx, spec, agent)
	if err != nil {
		nm.reportError(ctx, runID, fmt.Sprintf("构建命令失败: %v", err))
		return
	}

	// 准备 Workspace（如果配置了）
	var workspace *PreparedWorkspace
	wsConfig := ParseWorkspaceConfig(snapshot)
	if wsConfig != nil {
		log.Printf("任务 %s 需要准备 Workspace: type=%s", runID, wsConfig.Type)
		workspace, err = nm.workspaceManager.Prepare(ctx, runID, wsConfig)
		if err != nil {
			nm.reportError(ctx, runID, fmt.Sprintf("准备 Workspace 失败: %v", err))
			return
		}
		if workspace != nil && workspace.Cleanup != nil {
			defer workspace.Cleanup()
		}
	}

	// 优先使用 instance_id 获取容器，回退到 account_id
	instanceID, _ := agentConfig["instance_id"].(string)
	accountID, _ := agentConfig["account_id"].(string)

	var containerName string
	if instanceID != "" {
		// 直接通过 instance_id 获取容器名
		containerName, err = nm.getContainerForInstance(ctx, instanceID)
		if err != nil {
			nm.reportError(ctx, runID, fmt.Sprintf("获取实例容器失败: %v", err))
			return
		}
	} else if accountID != "" {
		// 回退：通过 account_id 查找容器
		containerName, err = nm.getContainerForAccount(ctx, accountID)
		if err != nil {
			nm.reportError(ctx, runID, fmt.Sprintf("获取容器失败: %v", err))
			return
		}
	} else {
		nm.reportError(ctx, runID, "任务缺少 instance_id 或 account_id 配置")
		return
	}

	log.Printf("任务 %s 将在容器 %s 中执行", runID, containerName)

	// 如果有 Workspace，复制到容器中
	if workspace != nil && workspace.Path != "" && wsConfig.Type == "git" {
		log.Printf("[Workspace] 复制文件到容器: %s -> %s:/workspace", workspace.Path, containerName)
		if err := nm.copyToContainer(ctx, workspace.Path, containerName, "/workspace"); err != nil {
			nm.reportError(ctx, runID, fmt.Sprintf("复制 Workspace 到容器失败: %v", err))
			return
		}
	}

	// 上报 run_started 事件
	startPayload := map[string]interface{}{
		"node_id":   nm.config.NodeID,
		"container": containerName,
	}
	if workspace != nil {
		startPayload["workspace"] = map[string]interface{}{
			"type":        wsConfig.Type,
			"path":        workspace.Path,
			"working_dir": workspace.WorkingDir,
		}
	}
	nm.reportEvent(ctx, runID, 1, "run_started", startPayload)

	// 构建 docker exec 命令
	// docker exec <container> <command> <args...>
	dockerArgs := []string{"exec"}

	// 添加环境变量
	for k, v := range runConfig.Env {
		dockerArgs = append(dockerArgs, "-e", k+"="+v)
	}

	// 设置工作目录（优先使用 Workspace 的工作目录）
	workingDir := runConfig.WorkingDir
	if workspace != nil && workspace.WorkingDir != "" {
		workingDir = workspace.WorkingDir
	}
	if workingDir != "" {
		dockerArgs = append(dockerArgs, "-w", workingDir)
	}

	dockerArgs = append(dockerArgs, containerName)
	dockerArgs = append(dockerArgs, runConfig.Command...)
	dockerArgs = append(dockerArgs, runConfig.Args...)

	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	cmd.Env = os.Environ()

	// 打印完整命令以便调试
	log.Printf("执行命令: docker %v", dockerArgs)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		nm.reportError(ctx, runID, fmt.Sprintf("启动失败: %v", err))
		return
	}

	// 异步读取 stderr 以便捕获错误信息
	var stderrBuf bytes.Buffer
	go func() {
		io.Copy(&stderrBuf, stderr)
	}()

	// 流式读取输出并解析事件
	seq := 2
	seq = nm.streamOutput(ctx, runID, stdout, a, seq)

	// 等待命令完成
	err = cmd.Wait()

	// 如果有 stderr 输出，记录日志
	if stderrBuf.Len() > 0 {
		log.Printf("任务 %s stderr 输出: %s", runID, stderrBuf.String())
	}
	status := "done"
	if err != nil {
		if ctx.Err() != nil {
			status = "cancelled"
		} else {
			status = "failed"
		}
	}

	// 上报 run_completed 事件
	nm.reportEvent(ctx, runID, seq, "run_completed", map[string]interface{}{
		"status": status,
	})

	nm.updateRunStatus(ctx, runID, status)
	log.Printf("任务 %s 完成，状态: %s", runID, status)
}

// streamOutput 流式读取命令输出并解析为事件
// 每读取一行就调用 Adapter.ParseEvent 解析，然后上报到 API Server
// 同时保存原始输出到 raw 字段，便于调试和回放
func (nm *NodeManager) streamOutput(ctx context.Context, runID string, r io.Reader, a adapter.Adapter, startSeq int) int {
	scanner := bufio.NewScanner(r)
	// 增大缓冲区以处理大行（如长 JSON）
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	seq := startSeq

	for scanner.Scan() {
		line := scanner.Text()
		event, err := a.ParseEvent(line)
		if err != nil || event == nil {
			continue
		}

		// 填充事件元数据
		event.Seq = int64(seq)
		event.RunID = runID
		event.Timestamp = time.Now()

		// 上报事件，同时传递原始行数据
		nm.reportEventWithRaw(ctx, runID, seq, string(event.Type), event.Payload, line)
		seq++
	}

	return seq
}

// reportEvent 上报事件到 API Server（不含原始数据）
func (nm *NodeManager) reportEvent(ctx context.Context, runID string, seq int, eventType string, payload map[string]interface{}) {
	nm.reportEventWithRaw(ctx, runID, seq, eventType, payload, "")
}

// reportEventWithRaw 上报事件到 API Server（含原始数据）
func (nm *NodeManager) reportEventWithRaw(ctx context.Context, runID string, seq int, eventType string, payload map[string]interface{}, raw string) {
	event := map[string]interface{}{
		"seq":       seq,
		"type":      eventType,
		"timestamp": time.Now().Format(time.RFC3339Nano),
		"payload":   payload,
	}

	// 如果有原始数据，添加到事件中
	if raw != "" {
		event["raw"] = raw
	}

	events := []map[string]interface{}{event}

	body, _ := json.Marshal(map[string]interface{}{"events": events})
	req, _ := http.NewRequestWithContext(ctx, "POST",
		nm.config.APIServerURL+"/api/v1/runs/"+runID+"/events",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := nm.httpClient.Do(req)
	if err != nil {
		log.Printf("上报事件失败: %v", err)
		return
	}
	resp.Body.Close()
}

// reportError 上报错误并更新状态为失败
func (nm *NodeManager) reportError(ctx context.Context, runID, errMsg string) {
	log.Printf("任务 %s 错误: %s", runID, errMsg)
	nm.reportEvent(ctx, runID, 1, "error", map[string]interface{}{
		"code":    "execution_error",
		"message": errMsg,
	})
	nm.updateRunStatus(ctx, runID, "failed")
}

// updateRunStatus 更新 Run 状态
func (nm *NodeManager) updateRunStatus(ctx context.Context, runID, status string) {
	body, _ := json.Marshal(map[string]string{"status": status})
	req, _ := http.NewRequestWithContext(ctx, "PATCH",
		nm.config.APIServerURL+"/api/v1/runs/"+runID,
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := nm.httpClient.Do(req)
	if err != nil {
		log.Printf("更新 Run 状态失败: %v", err)
		return
	}
	resp.Body.Close()
}

// CancelRun 取消正在执行的任务
func (nm *NodeManager) CancelRun(runID string) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if cancel, ok := nm.running[runID]; ok {
		cancel()
		log.Printf("已取消任务: %s", runID)
	}
}

// normalizeDriverName 将 agent type 转换为 driver name
// 支持多种格式的 agent type 名称
// normalizeAdapterName 将 agent type 转换为 adapter name
// 支持多种格式的 agent type 名称
func normalizeAdapterName(agentType string) string {
	// Agent type 到 adapter name 的映射
	mapping := map[string]string{
		"qwen-code": "qwencode-v1",
		"qwencode":  "qwencode-v1",
		"qwen":      "qwencode-v1",
		"gemini":    "gemini-v1",
		"claude":    "claude-v1",
	}

	if adapterName, ok := mapping[agentType]; ok {
		return adapterName
	}

	// 默认：agentType + "-v1"
	return agentType + "-v1"
}

// getContainerForInstance 通过 instance_id 获取容器名称
func (nm *NodeManager) getContainerForInstance(ctx context.Context, instanceID string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		nm.config.APIServerURL+"/api/v1/instances/"+instanceID, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := nm.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API 返回错误状态: %d", resp.StatusCode)
	}

	var instance struct {
		ID            string `json:"id"`
		ContainerName string `json:"container_name"`
		// 兼容旧字段（避免历史数据/旧 API 返回）
		Container string `json:"container"`
		Status    string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&instance); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	container := instance.ContainerName
	if container == "" {
		container = instance.Container
	}
	if container == "" {
		return "", fmt.Errorf("实例 %s 没有关联的容器", instanceID)
	}

	if instance.Status != "running" {
		log.Printf("警告: 实例 %s 状态为 %s，可能无法执行", instanceID, instance.Status)
	}

	return container, nil
}

// getContainerForAccount 根据 account_id 获取对应的容器名称
// 首先查询 API Server，如果没有则直接通过 Docker 查找
func (nm *NodeManager) getContainerForAccount(ctx context.Context, accountID string) (string, error) {
	// 方法1：查询 API Server 获取账号对应的实例
	container, err := nm.getContainerFromAPI(ctx, accountID)
	if err == nil && container != "" {
		return container, nil
	}
	log.Printf("从 API 获取实例失败: %v，尝试直接查找 Docker 容器", err)

	// 方法2：直接通过 Docker 查找匹配的容器
	container, err = nm.findContainerByAccountID(ctx, accountID)
	if err == nil && container != "" {
		return container, nil
	}

	return "", fmt.Errorf("账号 %s 没有可用的容器: %v", accountID, err)
}

// getContainerFromAPI 从 API Server 获取实例信息
func (nm *NodeManager) getContainerFromAPI(ctx context.Context, accountID string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		nm.config.APIServerURL+"/api/v1/instances?account_id="+accountID, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := nm.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API 返回错误状态: %d", resp.StatusCode)
	}

	var result struct {
		Instances []struct {
			ID            string `json:"id"`
			ContainerName string `json:"container_name"`
			// 兼容旧字段（避免历史数据/旧 API 返回）
			Container string `json:"container"`
			Status    string `json:"status"`
		} `json:"instances"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	// 查找运行中的实例
	for _, inst := range result.Instances {
		container := inst.ContainerName
		if container == "" {
			container = inst.Container
		}
		if inst.Status == "running" && container != "" {
			return container, nil
		}
	}

	// 如果没有运行中的实例，返回第一个实例
	if len(result.Instances) > 0 {
		container := result.Instances[0].ContainerName
		if container == "" {
			container = result.Instances[0].Container
		}
		if container != "" {
			return container, nil
		}
	}

	return "", fmt.Errorf("没有找到实例")
}

// findContainerByAccountID 通过 Docker 直接查找容器
// 容器命名规则：agent_inst_{sanitized_account_id}_{timestamp}
func (nm *NodeManager) findContainerByAccountID(ctx context.Context, accountID string) (string, error) {
	// 使用 docker ps 查找匹配的容器
	// 容器名包含 account_id（已 sanitize）
	sanitized := sanitizeAccountID(accountID)

	cmd := exec.CommandContext(ctx, "docker", "ps", "--format", "{{.Names}}", "--filter", "status=running")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker ps 失败: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, name := range lines {
		// 查找包含 account_id 的容器
		if strings.Contains(name, sanitized) {
			log.Printf("找到容器 %s (匹配 account_id: %s)", name, accountID)
			return name, nil
		}
	}

	return "", fmt.Errorf("没有找到匹配 %s 的运行中容器", sanitized)
}

// sanitizeAccountID 将 account_id 转换为容器名安全的格式
func sanitizeAccountID(accountID string) string {
	// 将 @ 替换为 _at_，将 . 替换为 _
	s := strings.ReplaceAll(accountID, "@", "_at_")
	s = strings.ReplaceAll(s, ".", "_")
	return s
}

// copyToContainer 将本地目录复制到容器中
func (nm *NodeManager) copyToContainer(ctx context.Context, srcPath, containerName, destPath string) error {
	// 先在容器中创建目标目录
	mkdirCmd := exec.CommandContext(ctx, "docker", "exec", containerName, "mkdir", "-p", destPath)
	if output, err := mkdirCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("创建目录失败: %w, 输出: %s", err, string(output))
	}

	// 使用 docker cp 复制文件
	// docker cp <src>/ <container>:<dest>
	// 注意：srcPath 后面加 /. 表示复制目录内容而不是目录本身
	cmd := exec.CommandContext(ctx, "docker", "cp", srcPath+"/.", containerName+":"+destPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker cp 失败: %w, 输出: %s", err, string(output))
	}

	log.Printf("[Workspace] 复制完成: %s -> %s:%s", srcPath, containerName, destPath)
	return nil
}

// copyFromContainer 从容器中复制文件到本地
func (nm *NodeManager) copyFromContainer(ctx context.Context, containerName, srcPath, destPath string) error {
	// 使用 docker cp 复制文件
	cmd := exec.CommandContext(ctx, "docker", "cp", containerName+":"+srcPath, destPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker cp 失败: %w, 输出: %s", err, string(output))
	}

	log.Printf("[Workspace] 复制完成: %s:%s -> %s", containerName, srcPath, destPath)
	return nil
}

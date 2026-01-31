// Package executor Terminal 工作线程
//
// 使用 ttyd 容器方案，单端口，同一时间只允许一个终端会话
package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	ttydContainerName = "ttyd_terminal"
	ttydPort          = 7681
	// 使用项目内构建的 ttyd 工具镜像（见 deployments/Dockerfile.ttyd）
	// 该镜像已包含 ttyd + docker-cli + bash + curl，可直接用 ttyd 运行 `docker exec ...`
	ttydImage = "tools/ttyd:latest"
)

// TerminalWorker Terminal 工作线程
type TerminalWorker struct {
	config     Config
	httpClient *http.Client
	mu         sync.Mutex

	// 当前活跃会话（同一时间只允许一个）
	activeSessionID   string
	activeContainerID string
}

// NewTerminalWorker 创建 Terminal 工作线程
func NewTerminalWorker(cfg Config) *TerminalWorker {
	return &TerminalWorker{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Start 启动 Terminal 工作线程
func (w *TerminalWorker) Start(ctx context.Context) {
	log.Printf("[TerminalWorker] 启动终端工作线程（ttyd容器模式，单端口 %d），节点: %s", ttydPort, w.config.NodeID)

	// 启动时清理可能残留的 ttyd 容器
	w.stopTTYDContainer(ctx)

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[TerminalWorker] 工作线程停止，清理终端容器...")
			w.stopTTYDContainer(context.Background())
			return
		case <-ticker.C:
			w.processPendingSessions(ctx)
			w.cleanupClosedSessions(ctx)
		}
	}
}

// processPendingSessions 处理待处理的终端会话
func (w *TerminalWorker) processPendingSessions(ctx context.Context) {
	sessions, err := w.fetchPendingSessions(ctx)
	if err != nil {
		// 静默处理，避免日志刷屏
		return
	}

	for _, session := range sessions {
		if session.Status == "pending" {
			w.startTerminal(ctx, session)
		}
	}
}

// terminalSessionInfo 终端会话信息结构
type terminalSessionInfo struct {
	ID            string `json:"id"`
	InstanceID    string `json:"instance_id"`
	ContainerName string `json:"container_name"`
	NodeID        string `json:"node_id"`
	Port          int    `json:"port"`
	URL           string `json:"url"`
	Status        string `json:"status"`
}

// fetchPendingSessions 获取待处理的终端会话列表
func (w *TerminalWorker) fetchPendingSessions(ctx context.Context) ([]terminalSessionInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		w.config.APIServerURL+"/api/v1/nodes/"+w.config.NodeID+"/terminal-sessions", nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回错误状态: %d", resp.StatusCode)
	}

	var result struct {
		Sessions []terminalSessionInfo `json:"sessions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return result.Sessions, nil
}

// startTerminal 启动终端（使用 ttyd 容器）
func (w *TerminalWorker) startTerminal(ctx context.Context, session terminalSessionInfo) {
	w.mu.Lock()
	defer w.mu.Unlock()

	log.Printf("[TerminalWorker] 启动终端: %s (容器: %s)", session.ID, session.ContainerName)

	// 更新状态为 starting
	if err := w.updateSessionStatus(ctx, session.ID, "starting", nil, nil); err != nil {
		log.Printf("[TerminalWorker] 更新状态失败: %v", err)
		return
	}

	// 检查目标容器是否运行中
	if !w.isContainerRunning(ctx, session.ContainerName) {
		log.Printf("[TerminalWorker] 目标容器未运行: %s", session.ContainerName)
		w.updateSessionStatus(ctx, session.ID, "error", nil, strPtr("目标容器未运行"))
		return
	}

	// 如果已有活跃会话，先关闭旧的 ttyd 容器
	if w.activeSessionID != "" {
		log.Printf("[TerminalWorker] 关闭旧会话: %s", w.activeSessionID)
		w.stopTTYDContainerUnlocked(ctx)
		// 更新旧会话状态为 closed
		w.updateSessionStatus(ctx, w.activeSessionID, "closed", nil, nil)
	}

	// 启动 ttyd 容器
	// docker run --rm -d --name ttyd_terminal -p 7681:7681 \
	//   -v /var/run/docker.sock:/var/run/docker.sock \
	//   tools/ttyd:latest -W -p 7681 docker exec -it <container> <bash|sh>
	targetShell := w.detectTargetShell(ctx, session.ContainerName)
	args := buildTTYDDockerRunArgs(session.ContainerName, targetShell)

	log.Printf("[TerminalWorker] 执行: docker %v", args)

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[TerminalWorker] 启动 ttyd 容器失败: %v, 输出: %s", err, string(output))
		w.updateSessionStatus(ctx, session.ID, "error", nil, strPtr("启动终端失败: "+err.Error()))
		return
	}

	containerID := strings.TrimSpace(string(output))
	log.Printf("[TerminalWorker] ttyd 容器启动成功: %s", containerID[:12])

	// 记录活跃会话
	w.activeSessionID = session.ID
	w.activeContainerID = containerID

	// 等待 ttyd 就绪
	if err := w.waitForTTYDReady(ctx, time.Now().Add(12*time.Second)); err != nil {
		log.Printf("[TerminalWorker] ttyd 未就绪: %v，查看容器日志...", err)
		logsCmd := exec.CommandContext(ctx, "docker", "logs", ttydContainerName)
		logs, _ := logsCmd.CombinedOutput()
		log.Printf("[TerminalWorker] ttyd 容器日志: %s", string(logs))
		w.updateSessionStatus(ctx, session.ID, "error", nil, strPtr("终端启动超时"))
		w.stopTTYDContainerUnlocked(ctx)
		w.activeSessionID = ""
		w.activeContainerID = ""
		return
	}

	// 构建终端 URL（直接访问 ttyd 端口）
	port := ttydPort
	terminalURL := fmt.Sprintf("/terminal/%s/", session.ID)

	// 更新状态为 running
	if err := w.updateSessionStatus(ctx, session.ID, "running", &port, &terminalURL); err != nil {
		log.Printf("[TerminalWorker] 更新状态失败: %v", err)
		return
	}

	log.Printf("[TerminalWorker] 终端 %s 启动成功，端口: %d, URL: %s", session.ID, port, terminalURL)
}

func (w *TerminalWorker) waitForTTYDReady(ctx context.Context, deadline time.Time) error {
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if w.isContainerRunning(ctx, ttydContainerName) {
			// 不依赖 curl/wget/netcat 是否存在：直接从 ttyd 日志判断是否开始监听端口
			logsCmd := exec.CommandContext(ctx, "docker", "logs", ttydContainerName)
			logs, err := logsCmd.CombinedOutput()
			if err == nil && (strings.Contains(string(logs), "Listening on port") || strings.Contains(string(logs), "Listening on port:")) {
				return nil
			}
		}

		time.Sleep(250 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for ttyd to be ready")
}

func buildTTYDDockerRunArgs(targetContainer, targetShell string) []string {
	return []string{
		"run", "-d",
		"--name", ttydContainerName,
		// tools/ttyd:latest 可能带 tini 入口（如 deployments/docker/Dockerfile.ttyd）
		// 为保证参数一致，强制使用 ttyd 作为入口
		"--entrypoint", "ttyd",
		"-p", fmt.Sprintf("%d:%d", ttydPort, ttydPort),
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
		ttydImage,
		"-W", "-p", fmt.Sprintf("%d", ttydPort),
		"docker", "exec", "-it", targetContainer, targetShell,
	}
}

// detectTargetShell 检测目标容器是否有 bash；否则回退到 sh
func (w *TerminalWorker) detectTargetShell(ctx context.Context, containerName string) string {
	// 绝大多数镜像都有 /bin/sh；bash 可能不存在
	cmd := exec.CommandContext(ctx, "docker", "exec", containerName, "sh", "-lc", "command -v bash >/dev/null 2>&1")
	if err := cmd.Run(); err == nil {
		return "bash"
	}
	return "sh"
}

// cleanupClosedSessions 清理已关闭的会话
func (w *TerminalWorker) cleanupClosedSessions(ctx context.Context) {
	w.mu.Lock()
	activeID := w.activeSessionID
	w.mu.Unlock()

	if activeID == "" {
		return
	}

	// 检查当前活跃会话是否已被关闭
	session, err := w.getSessionStatus(ctx, activeID)
	if err != nil {
		return
	}

	if session.Status == "closed" {
		log.Printf("[TerminalWorker] 检测到会话已关闭: %s，停止 ttyd 容器", activeID)
		w.mu.Lock()
		w.stopTTYDContainerUnlocked(ctx)
		w.activeSessionID = ""
		w.activeContainerID = ""
		w.mu.Unlock()
	}
}

// getSessionStatus 获取会话状态
func (w *TerminalWorker) getSessionStatus(ctx context.Context, sessionID string) (*terminalSessionInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		w.config.APIServerURL+"/api/v1/terminal-sessions/"+sessionID, nil)
	if err != nil {
		return nil, err
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回错误状态: %d", resp.StatusCode)
	}

	var session terminalSessionInfo
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, err
	}

	return &session, nil
}

// stopTTYDContainer 停止 ttyd 容器（加锁版本）
func (w *TerminalWorker) stopTTYDContainer(ctx context.Context) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.stopTTYDContainerUnlocked(ctx)
}

// stopTTYDContainerUnlocked 停止 ttyd 容器（不加锁版本，调用方需持有锁）
func (w *TerminalWorker) stopTTYDContainerUnlocked(ctx context.Context) {
	// 先尝试停止
	stopCmd := exec.CommandContext(ctx, "docker", "stop", "-t", "1", ttydContainerName)
	stopCmd.Run()

	// 再尝试删除（如果没有 --rm 的话）
	rmCmd := exec.CommandContext(ctx, "docker", "rm", "-f", ttydContainerName)
	rmCmd.Run()

	log.Printf("[TerminalWorker] ttyd 容器已停止")
}

// isContainerRunning 检查容器是否运行中
func (w *TerminalWorker) isContainerRunning(ctx context.Context, containerName string) bool {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Running}}", containerName)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "true"
}

// updateSessionStatus 更新终端会话状态
func (w *TerminalWorker) updateSessionStatus(ctx context.Context, sessionID, status string, port *int, url *string) error {
	payload := map[string]interface{}{
		"status": status,
	}
	if port != nil {
		payload["port"] = *port
	}
	if url != nil {
		payload["url"] = *url
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "PATCH",
		w.config.APIServerURL+"/api/v1/terminal-sessions/"+sessionID,
		bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API 返回错误状态: %d", resp.StatusCode)
	}

	return nil
}

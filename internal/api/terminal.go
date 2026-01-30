package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"time"

	"agents-admin/internal/model"
)

var (
	terminalSessions  = make(map[string]*model.TerminalSession)
	terminalsMu       sync.RWMutex
	nextTTYDPort      = 7681
	ttydContainerName = "ttyd_service"
)

// CreateTerminalSession 创建终端会话
func (h *Handler) CreateTerminalSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InstanceID string `json:"instance_id"`
		Container  string `json:"container"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// 如果指定了实例 ID，获取容器名
	container := req.Container
	if req.InstanceID != "" {
		instancesMu.RLock()
		instance, ok := instances[req.InstanceID]
		instancesMu.RUnlock()
		if !ok {
			writeError(w, http.StatusNotFound, "instance not found")
			return
		}
		container = instance.Container
	}

	if container == "" {
		writeError(w, http.StatusBadRequest, "container or instance_id is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// 检查目标容器是否存在且运行中
	checkCmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Running}}", container)
	output, err := checkCmd.Output()
	if err != nil || strings.TrimSpace(string(output)) != "true" {
		writeError(w, http.StatusBadRequest, "container not running")
		return
	}

	// 确保 ttyd 服务容器运行
	if err := ensureTTYDContainer(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start ttyd service: "+err.Error())
		return
	}

	// 分配端口
	terminalsMu.Lock()
	port := nextTTYDPort
	nextTTYDPort++
	if nextTTYDPort > 7780 {
		nextTTYDPort = 7681
	}
	terminalsMu.Unlock()

	// 在 ttyd 容器内启动连接目标容器的 ttyd 进程
	// ttyd 通过 docker exec 连接到目标容器
	sessionID := fmt.Sprintf("term_%s_%d", container, time.Now().Unix())

	ttydCmd := exec.CommandContext(ctx, "docker", "exec", "-d", ttydContainerName,
		"ttyd", "-W", "-o", "-p", fmt.Sprintf("%d", port),
		"docker", "exec", "-it", container, "bash")

	if err := ttydCmd.Run(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start terminal: "+err.Error())
		return
	}

	// 等待 ttyd 启动
	time.Sleep(500 * time.Millisecond)

	session := &model.TerminalSession{
		ID:         sessionID,
		InstanceID: req.InstanceID,
		Container:  container,
		Node:       "local",
		Port:       port,
		URL:        fmt.Sprintf("/terminal/%s/", sessionID),
		Status:     "running",
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(30 * time.Minute),
	}

	terminalsMu.Lock()
	terminalSessions[sessionID] = session
	terminalsMu.Unlock()

	// 启动超时清理
	go cleanupTerminalSession(sessionID, session.ExpiresAt)

	writeJSON(w, http.StatusCreated, session)
}

// GetTerminalSession 获取终端会话状态
func (h *Handler) GetTerminalSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")

	terminalsMu.RLock()
	session, ok := terminalSessions[sessionID]
	terminalsMu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	writeJSON(w, http.StatusOK, session)
}

// DeleteTerminalSession 关闭终端会话
func (h *Handler) DeleteTerminalSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")

	terminalsMu.Lock()
	session, ok := terminalSessions[sessionID]
	if ok {
		delete(terminalSessions, sessionID)
	}
	terminalsMu.Unlock()

	if !ok {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	// 终止 ttyd 进程（通过端口查找并 kill）
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	killCmd := exec.CommandContext(ctx, "docker", "exec", ttydContainerName,
		"pkill", "-f", fmt.Sprintf("ttyd.*-p %d", session.Port))
	killCmd.Run()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "session closed",
	})
}

// ProxyTerminalSession 代理终端 WebSocket 连接
func (h *Handler) ProxyTerminalSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")

	terminalsMu.RLock()
	session, ok := terminalSessions[sessionID]
	terminalsMu.RUnlock()

	if !ok || session.Status != "running" {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	// 创建反向代理到 ttyd 端口
	target, _ := url.Parse(fmt.Sprintf("http://localhost:%d", session.Port))
	proxy := httputil.NewSingleHostReverseProxy(target)

	// 移除路径前缀
	r.URL.Path = strings.TrimPrefix(r.URL.Path, fmt.Sprintf("/terminal/%s", sessionID))
	if r.URL.Path == "" {
		r.URL.Path = "/"
	}

	proxy.ServeHTTP(w, r)
}

// ensureTTYDContainer 确保 ttyd 服务容器运行
func ensureTTYDContainer(ctx context.Context) error {
	// 检查容器是否存在
	checkCmd := exec.CommandContext(ctx, "docker", "container", "inspect", ttydContainerName)
	if checkCmd.Run() == nil {
		// 容器存在，检查是否运行
		statusCmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Running}}", ttydContainerName)
		output, _ := statusCmd.Output()
		if strings.TrimSpace(string(output)) == "true" {
			return nil // 已运行
		}
		// 启动已停止的容器
		startCmd := exec.CommandContext(ctx, "docker", "start", ttydContainerName)
		return startCmd.Run()
	}

	// 创建并启动 ttyd 容器
	// 需要挂载 docker.sock 以便访问其他容器
	runCmd := exec.CommandContext(ctx, "docker", "run", "-d",
		"--name", ttydContainerName,
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
		"-p", "7681-7780:7681-7780",
		"--restart", "unless-stopped",
		"tools/ttyd:latest",
		"tail", "-f", "/dev/null") // 保持容器运行，ttyd 进程按需启动

	return runCmd.Run()
}

// cleanupTerminalSession 超时清理终端会话
func cleanupTerminalSession(sessionID string, expiresAt time.Time) {
	time.Sleep(time.Until(expiresAt))

	terminalsMu.Lock()
	defer terminalsMu.Unlock()

	session, ok := terminalSessions[sessionID]
	if !ok {
		return
	}

	// 检查是否是同一个会话
	if session.ExpiresAt == expiresAt {
		// 终止 ttyd 进程
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		killCmd := exec.CommandContext(ctx, "docker", "exec", ttydContainerName,
			"pkill", "-f", fmt.Sprintf("ttyd.*-p %d", session.Port))
		killCmd.Run()

		delete(terminalSessions, sessionID)
	}
}

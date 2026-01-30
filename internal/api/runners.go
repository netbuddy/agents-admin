package api

import (
	"bufio"
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
)

// RunnerStatus Runner 容器状态
type RunnerStatus struct {
	Account      string `json:"account"`
	Container    string `json:"container"`
	Status       string `json:"status"` // created, running, stopped, not_found
	LoggedIn     bool   `json:"logged_in"`
	TerminalPort int    `json:"terminal_port,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
}

// LoginSession 登录会话
type LoginSession struct {
	Account    string `json:"account"`
	DeviceCode string `json:"device_code,omitempty"`
	VerifyURL  string `json:"verify_url,omitempty"`
	Status     string `json:"status"` // pending, waiting, success, failed
	Message    string `json:"message,omitempty"`
	ExpiresAt  int64  `json:"expires_at,omitempty"`
}

// TerminalInfo Web 终端信息（ttyd 在容器内运行）
type TerminalInfo struct {
	Account   string `json:"account"`
	Container string `json:"container"`
	Port      int    `json:"port"`
	Status    string `json:"status"` // running, stopped
}

var (
	loginSessions    = make(map[string]*LoginSession)
	containerPorts   = make(map[string]int) // container -> host port
	sessionsMu       sync.RWMutex
	nextTerminalPort = 7681
)

// ListRunners 列出所有 Runner 容器
func (h *Handler) ListRunners(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "ps", "-a",
		"--filter", "name=qwencode_",
		"--format", "{{.Names}}|{{.Status}}|{{.CreatedAt}}")

	output, err := cmd.Output()
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"runners": []RunnerStatus{},
		})
		return
	}

	var runners []RunnerStatus
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "|")
		if len(parts) < 2 {
			continue
		}

		container := parts[0]
		statusStr := parts[1]
		createdAt := ""
		if len(parts) > 2 {
			createdAt = parts[2]
		}

		// 解析状态
		status := "stopped"
		if strings.Contains(statusStr, "Up") {
			status = "running"
		}

		// 从容器名提取账户名
		account := strings.TrimPrefix(container, "qwencode_")
		account = strings.ReplaceAll(account, "_at_", "@")
		account = strings.ReplaceAll(account, "_", ".")

		// 检查是否已登录
		loggedIn := checkRunnerLoggedIn(ctx, container)

		runners = append(runners, RunnerStatus{
			Account:   account,
			Container: container,
			Status:    status,
			LoggedIn:  loggedIn,
			CreatedAt: createdAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"runners": runners,
	})
}

// CreateRunner 创建 Runner 容器
func (h *Handler) CreateRunner(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Account string `json:"account"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Account == "" {
		writeError(w, http.StatusBadRequest, "account is required")
		return
	}

	container := accountToContainerName(req.Account)
	volume := container + "_data"

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// 检查容器是否已存在
	checkCmd := exec.CommandContext(ctx, "docker", "container", "inspect", container)
	if checkCmd.Run() == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message":   "container already exists",
			"container": container,
		})
		return
	}

	// 创建数据卷
	volCmd := exec.CommandContext(ctx, "docker", "volume", "create", volume)
	if err := volCmd.Run(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create volume")
		return
	}

	// 分配端口
	sessionsMu.Lock()
	port := nextTerminalPort
	nextTerminalPort++
	if nextTerminalPort > 7780 {
		nextTerminalPort = 7681
	}
	containerPorts[container] = port
	sessionsMu.Unlock()

	// 创建容器（映射 ttyd 端口）
	runCmd := exec.CommandContext(ctx, "docker", "run", "-d",
		"--name", container,
		"-v", volume+":/home/node/.qwen",
		"-p", fmt.Sprintf("%d:7681", port),
		"--restart", "unless-stopped",
		"runners/qwencode:latest")

	if err := runCmd.Run(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create container")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message":       "container created",
		"container":     container,
		"account":       req.Account,
		"terminal_port": port,
	})
}

// StartRunnerLogin 启动 Runner 登录流程
func (h *Handler) StartRunnerLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Account    string `json:"account"`
		DeviceAuth bool   `json:"device_auth"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	container := accountToContainerName(req.Account)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 确保容器运行
	startCmd := exec.CommandContext(ctx, "docker", "start", container)
	if err := startCmd.Run(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start container")
		return
	}

	// 检查是否已登录
	if checkRunnerLoggedIn(ctx, container) {
		writeJSON(w, http.StatusOK, &LoginSession{
			Account: req.Account,
			Status:  "success",
			Message: "already logged in",
		})
		return
	}

	// 创建登录会话
	session := &LoginSession{
		Account:   req.Account,
		Status:    "pending",
		ExpiresAt: time.Now().Add(10 * time.Minute).Unix(),
	}

	if req.DeviceAuth {
		// 设备码登录模式
		go startDeviceAuthLogin(container, session)
	} else {
		// 浏览器 OAuth 模式
		session.VerifyURL = fmt.Sprintf("http://localhost:1455/auth/callback?container=%s", container)
		session.Status = "waiting"
		session.Message = "Please complete OAuth login in browser"
	}

	sessionsMu.Lock()
	loginSessions[req.Account] = session
	sessionsMu.Unlock()

	writeJSON(w, http.StatusOK, session)
}

// GetRunnerLoginStatus 获取登录状态
func (h *Handler) GetRunnerLoginStatus(w http.ResponseWriter, r *http.Request) {
	account := r.URL.Query().Get("account")
	if account == "" {
		writeError(w, http.StatusBadRequest, "account is required")
		return
	}

	sessionsMu.RLock()
	session, exists := loginSessions[account]
	sessionsMu.RUnlock()

	if !exists {
		// 检查是否已登录
		container := accountToContainerName(account)
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if checkRunnerLoggedIn(ctx, container) {
			writeJSON(w, http.StatusOK, &LoginSession{
				Account: account,
				Status:  "success",
				Message: "logged in",
			})
			return
		}

		writeJSON(w, http.StatusOK, &LoginSession{
			Account: account,
			Status:  "not_started",
		})
		return
	}

	// 检查是否已完成登录
	if session.Status == "waiting" || session.Status == "pending" {
		container := accountToContainerName(account)
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if checkRunnerLoggedIn(ctx, container) {
			session.Status = "success"
			session.Message = "login completed"
		}
	}

	writeJSON(w, http.StatusOK, session)
}

// DeleteRunner 删除 Runner 容器
func (h *Handler) DeleteRunner(w http.ResponseWriter, r *http.Request) {
	account := r.URL.Query().Get("account")
	purge := r.URL.Query().Get("purge") == "true"

	if account == "" {
		writeError(w, http.StatusBadRequest, "account is required")
		return
	}

	container := accountToContainerName(account)
	volume := container + "_data"

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// 删除容器
	rmCmd := exec.CommandContext(ctx, "docker", "rm", "-f", container)
	rmCmd.Run() // 忽略错误

	if purge {
		// 删除数据卷
		volRmCmd := exec.CommandContext(ctx, "docker", "volume", "rm", volume)
		volRmCmd.Run()
	}

	// 清理会话
	sessionsMu.Lock()
	delete(loginSessions, account)
	sessionsMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "runner deleted",
		"purged":  purge,
	})
}

// 辅助函数

func accountToContainerName(account string) string {
	name := strings.ReplaceAll(account, "@", "_at_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "-", "_")
	return "qwencode_" + name
}

func checkRunnerLoggedIn(ctx context.Context, container string) bool {
	cmd := exec.CommandContext(ctx, "docker", "exec", container,
		"test", "-f", "/home/node/.qwen/auth.json")
	return cmd.Run() == nil
}

func startDeviceAuthLogin(container string, session *LoginSession) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "exec", container,
		"qwen", "login", "--device-auth")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		sessionsMu.Lock()
		session.Status = "failed"
		session.Message = "failed to start login process"
		sessionsMu.Unlock()
		return
	}

	if err := cmd.Start(); err != nil {
		sessionsMu.Lock()
		session.Status = "failed"
		session.Message = "failed to start login command"
		sessionsMu.Unlock()
		return
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()

		// 解析设备码和验证 URL
		if strings.Contains(line, "code:") || strings.Contains(line, "Code:") {
			// 提取设备码
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				sessionsMu.Lock()
				session.DeviceCode = strings.TrimSpace(parts[len(parts)-1])
				session.Status = "waiting"
				sessionsMu.Unlock()
			}
		}
		if strings.Contains(line, "http") {
			sessionsMu.Lock()
			session.VerifyURL = strings.TrimSpace(line)
			sessionsMu.Unlock()
		}
	}

	if err := cmd.Wait(); err != nil {
		sessionsMu.Lock()
		if session.Status != "success" {
			session.Status = "failed"
			session.Message = "login process failed"
		}
		sessionsMu.Unlock()
		return
	}

	// 检查是否成功
	checkCtx, checkCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer checkCancel()

	if checkRunnerLoggedIn(checkCtx, container) {
		sessionsMu.Lock()
		session.Status = "success"
		session.Message = "login completed"
		sessionsMu.Unlock()
	}
}

// GetTerminal 获取终端信息（ttyd 在容器内运行）
func (h *Handler) GetTerminal(w http.ResponseWriter, r *http.Request) {
	account := r.PathValue("account")
	if account == "" {
		writeError(w, http.StatusBadRequest, "account is required")
		return
	}

	container := accountToContainerName(account)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 获取容器端口映射
	portCmd := exec.CommandContext(ctx, "docker", "port", container, "7681")
	output, err := portCmd.Output()
	if err != nil {
		writeError(w, http.StatusNotFound, "container not running or port not mapped")
		return
	}

	// 解析端口: "0.0.0.0:7681" -> 7681
	portStr := strings.TrimSpace(string(output))
	parts := strings.Split(portStr, ":")
	port := 7681
	if len(parts) >= 2 {
		fmt.Sscanf(parts[len(parts)-1], "%d", &port)
	}

	// 检查容器状态
	statusCmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Running}}", container)
	statusOutput, _ := statusCmd.Output()
	status := "stopped"
	if strings.TrimSpace(string(statusOutput)) == "true" {
		status = "running"
	}

	writeJSON(w, http.StatusOK, &TerminalInfo{
		Account:   account,
		Container: container,
		Port:      port,
		Status:    status,
	})
}

// CreateTerminal 启动容器终端（确保容器运行）
func (h *Handler) CreateTerminal(w http.ResponseWriter, r *http.Request) {
	account := r.PathValue("account")
	if account == "" {
		writeError(w, http.StatusBadRequest, "account is required")
		return
	}

	container := accountToContainerName(account)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 检查容器是否存在
	checkCmd := exec.CommandContext(ctx, "docker", "container", "inspect", container)
	if err := checkCmd.Run(); err != nil {
		writeError(w, http.StatusNotFound, "container not found")
		return
	}

	// 启动容器（如果未运行）
	startCmd := exec.CommandContext(ctx, "docker", "start", container)
	if err := startCmd.Run(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start container")
		return
	}

	// 等待 ttyd 启动
	time.Sleep(1 * time.Second)

	// 获取端口映射
	portCmd := exec.CommandContext(ctx, "docker", "port", container, "7681")
	output, err := portCmd.Output()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get terminal port")
		return
	}

	// 解析端口
	portStr := strings.TrimSpace(string(output))
	parts := strings.Split(portStr, ":")
	port := 7681
	if len(parts) >= 2 {
		fmt.Sscanf(parts[len(parts)-1], "%d", &port)
	}

	writeJSON(w, http.StatusOK, &TerminalInfo{
		Account:   account,
		Container: container,
		Port:      port,
		Status:    "running",
	})
}

// DeleteTerminal 停止容器
func (h *Handler) DeleteTerminal(w http.ResponseWriter, r *http.Request) {
	account := r.PathValue("account")
	if account == "" {
		writeError(w, http.StatusBadRequest, "account is required")
		return
	}

	container := accountToContainerName(account)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 停止容器（不删除，保留认证数据）
	stopCmd := exec.CommandContext(ctx, "docker", "stop", container)
	stopCmd.Run()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "terminal stopped",
	})
}

// ProxyTerminal 代理 ttyd 请求
func (h *Handler) ProxyTerminal(w http.ResponseWriter, r *http.Request) {
	account := r.PathValue("account")
	if account == "" {
		writeError(w, http.StatusBadRequest, "account is required")
		return
	}

	container := accountToContainerName(account)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// 获取端口映射
	portCmd := exec.CommandContext(ctx, "docker", "port", container, "7681")
	output, err := portCmd.Output()
	if err != nil {
		writeError(w, http.StatusNotFound, "terminal not available")
		return
	}

	// 解析端口
	portStr := strings.TrimSpace(string(output))
	parts := strings.Split(portStr, ":")
	port := 7681
	if len(parts) >= 2 {
		fmt.Sscanf(parts[len(parts)-1], "%d", &port)
	}

	// 创建反向代理
	target, _ := url.Parse(fmt.Sprintf("http://localhost:%d", port))
	proxy := httputil.NewSingleHostReverseProxy(target)

	// 移除路径前缀
	r.URL.Path = strings.TrimPrefix(r.URL.Path, fmt.Sprintf("/ttyd/%s", account))
	if r.URL.Path == "" {
		r.URL.Path = "/"
	}

	proxy.ServeHTTP(w, r)
}

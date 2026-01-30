// Package qwencode 实现 Qwen-Code OAuth 认证器
//
// Qwen-Code 认证流程:
//  1. 启动容器运行 qwen 命令
//  2. 检测交互界面，选择 Qwen OAuth
//  3. 解析输出中的 OAuth URL
//  4. 返回 URL 给前端，等待用户完成认证
//  5. 轮询检查 auth.json 是否生成
package qwencode

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"agents-admin/pkg/auth"
	"agents-admin/pkg/docker"
)

// proxyEnvVars 需要传递给容器的代理环境变量名
var proxyEnvVars = []string{
	"HTTP_PROXY", "http_proxy",
	"HTTPS_PROXY", "https_proxy",
	"NO_PROXY", "no_proxy",
	"ALL_PROXY", "all_proxy",
}

// getProxyEnvVars 获取宿主机的代理环境变量
func getProxyEnvVars() []string {
	var envs []string
	for _, name := range proxyEnvVars {
		if val := os.Getenv(name); val != "" {
			envs = append(envs, fmt.Sprintf("%s=%s", name, val))
		}
	}
	return envs
}

var (
	oauthURLPattern   = regexp.MustCompile(`https://chat\.qwen\.ai/authorize\?[^\s]+`)
	userCodePattern   = regexp.MustCompile(`user_code=([A-Z0-9-]+)`)
	getStartedPattern = regexp.MustCompile(`Get started|How would you like to authenticate`)
	// 检测界面完整渲染的结束标记
	authMenuReadyPattern = regexp.MustCompile(`\(Use Enter to Set Auth\)|Use Enter to Set Auth`)
	// 检测认证成功的消息
	authSuccessPattern = regexp.MustCompile(`Authenticated successfully|Authentication completed|登录成功`)
)

const (
	// outputIdleTimeout 输出空闲超时，当这段时间内没有新输出时认为界面渲染完成
	outputIdleTimeout = 500 * time.Millisecond
)

// Authenticator Qwen-Code 认证器
type Authenticator struct {
	mu           sync.RWMutex
	dockerClient *docker.Client
	containerID  string
	containerIO  *docker.ContainerIO
	task         *auth.AuthTask
	status       *auth.AuthStatus
	statusChan   chan *auth.AuthStatus
	cancelFunc   context.CancelFunc
}

// New 创建 Qwen-Code 认证器
func New() *Authenticator {
	return &Authenticator{
		status: &auth.AuthStatus{
			State: auth.AuthStatePending,
		},
	}
}

// AgentType 返回支持的Agent类型
func (a *Authenticator) AgentType() string {
	return "qwen-code"
}

// SupportedMethods 返回支持的认证方法
func (a *Authenticator) SupportedMethods() []string {
	return []string{"oauth", "api_key"}
}

// Start 启动认证流程
func (a *Authenticator) Start(ctx context.Context, task *auth.AuthTask, dockerClient *docker.Client) (<-chan *auth.AuthStatus, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.task = task
	a.dockerClient = dockerClient
	a.statusChan = make(chan *auth.AuthStatus, 10)

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(ctx)
	a.cancelFunc = cancel

	// 更新状态为运行中（已持有锁，使用内部版本）
	a.updateStatusLocked(auth.AuthStateRunning, "", "", "Starting authentication...")

	// 启动认证流程
	go a.runAuthFlow(ctx)

	return a.statusChan, nil
}

// runAuthFlow 执行认证流程
func (a *Authenticator) runAuthFlow(ctx context.Context) {
	defer close(a.statusChan)
	defer a.cleanup()

	// 1. 创建数据卷
	volumeName := a.task.VolumeName
	if volumeName == "" {
		volumeName = fmt.Sprintf("auth_%s", strings.ReplaceAll(a.task.AccountID, "-", "_"))
	}

	log.Printf("[QwenCodeAuth] Creating volume: %s", volumeName)
	if err := a.dockerClient.CreateVolume(ctx, volumeName); err != nil {
		a.updateStatusWithError(auth.AuthStateFailed, fmt.Errorf("failed to create volume: %w", err))
		return
	}

	// 2. 创建容器
	containerName := fmt.Sprintf("auth_%s", a.task.ID)
	containerCfg := &docker.ContainerConfig{
		Name:       containerName,
		Image:      a.task.Image,
		Entrypoint: []string{}, // 清空entrypoint，让cmd直接执行
		Cmd:        strings.Fields(a.task.LoginCmd),
		WorkingDir: "/workspace",
		Volumes:    map[string]string{volumeName: a.task.AuthDir},
		Tty:        true,
		OpenStdin:  true,
	}

	// 添加环境变量
	for k, v := range a.task.Env {
		containerCfg.Env = append(containerCfg.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// 添加代理环境变量
	// 优先使用任务指定的代理配置，否则从宿主机继承
	proxyEnvs := a.task.ProxyEnvs
	if len(proxyEnvs) == 0 {
		proxyEnvs = getProxyEnvVars() // 兼容：如果未指定代理，尝试从宿主机继承
	}
	if len(proxyEnvs) > 0 {
		log.Printf("[QwenCodeAuth] Adding proxy env vars: %v", proxyEnvs)
		containerCfg.Env = append(containerCfg.Env, proxyEnvs...)
	}

	log.Printf("[QwenCodeAuth] Creating container: %s", containerName)
	containerID, err := a.dockerClient.CreateContainer(ctx, containerCfg)
	if err != nil {
		a.updateStatusWithError(auth.AuthStateFailed, fmt.Errorf("failed to create container: %w", err))
		return
	}
	a.containerID = containerID

	// 3. 附加到容器获取IO
	log.Printf("[QwenCodeAuth] Attaching to container: %s", containerID)
	containerIO, err := a.dockerClient.AttachContainer(ctx, containerID)
	if err != nil {
		a.updateStatusWithError(auth.AuthStateFailed, fmt.Errorf("failed to attach container: %w", err))
		return
	}
	a.containerIO = containerIO

	// 4. 启动容器
	log.Printf("[QwenCodeAuth] Starting container: %s", containerID)
	if err := a.dockerClient.StartContainer(ctx, containerID); err != nil {
		a.updateStatusWithError(auth.AuthStateFailed, fmt.Errorf("failed to start container: %w", err))
		return
	}

	// 5. 读取输出并解析
	a.processOutput(ctx, containerIO.Reader, volumeName)
}

// processOutput 处理容器输出
func (a *Authenticator) processOutput(ctx context.Context, reader io.Reader, volumeName string) {
	scanner := bufio.NewScanner(reader)
	var outputBuffer strings.Builder
	sentEnter := false
	foundOAuthURL := false
	detectedAuthMenu := false // 是否检测到认证菜单开始

	// 同时启动轮询检查认证文件
	authCheckTicker := time.NewTicker(3 * time.Second)
	defer authCheckTicker.Stop()

	// 使用goroutine读取输出
	lineChan := make(chan string, 100)
	go func() {
		defer close(lineChan)
		for scanner.Scan() {
			select {
			case lineChan <- scanner.Text():
			case <-ctx.Done():
				return
			}
		}
	}()

	// 输出空闲计时器，用于检测界面渲染完成
	var idleTimer *time.Timer
	var idleTimerChan <-chan time.Time

	for {
		select {
		case <-ctx.Done():
			a.updateStatusWithError(auth.AuthStateTimeout, fmt.Errorf("authentication timeout"))
			return

		case line, ok := <-lineChan:
			if !ok {
				// 输出流结束，检查是否成功
				if a.checkAuthSuccess(ctx, volumeName) {
					a.updateStatus(auth.AuthStateSuccess, "", "", "Authentication completed")
				} else {
					a.updateStatusWithError(auth.AuthStateFailed, fmt.Errorf("container exited without successful auth"))
				}
				return
			}

			outputBuffer.WriteString(line)
			outputBuffer.WriteString("\n")
			log.Printf("[QwenCodeAuth] Output: %s", line)

			// 检测 "Get started" 界面开始
			if !sentEnter && !detectedAuthMenu && getStartedPattern.MatchString(outputBuffer.String()) {
				log.Printf("[QwenCodeAuth] Detected auth selection screen starting...")
				detectedAuthMenu = true
			}

			// 如果已检测到认证菜单，重置空闲计时器
			// 每次有新输出时重置，直到输出空闲一段时间
			if detectedAuthMenu && !sentEnter {
				// 方法1: 检测界面结束标记（更可靠）
				if authMenuReadyPattern.MatchString(outputBuffer.String()) {
					log.Printf("[QwenCodeAuth] Auth menu fully rendered (detected end marker), sending '1' to select Qwen OAuth")
					if err := a.SendInput("1\n"); err != nil {
						log.Printf("[QwenCodeAuth] Failed to send input: %v", err)
					}
					sentEnter = true
					if idleTimer != nil {
						idleTimer.Stop()
					}
				} else {
					// 方法2: 输出空闲检测（备用）
					// 重置空闲计时器
					if idleTimer != nil {
						idleTimer.Stop()
					}
					idleTimer = time.NewTimer(outputIdleTimeout)
					idleTimerChan = idleTimer.C
				}
			}

			// 检测 OAuth URL
			if !foundOAuthURL {
				if matches := oauthURLPattern.FindString(outputBuffer.String()); matches != "" {
					log.Printf("[QwenCodeAuth] Found OAuth URL: %s", matches)

					// 提取 user_code
					userCode := ""
					if codeMatch := userCodePattern.FindStringSubmatch(matches); len(codeMatch) > 1 {
						userCode = codeMatch[1]
					}

					a.updateStatusWithOAuth(auth.AuthStateWaitingOAuth, matches, userCode,
						fmt.Sprintf("Please visit the URL to complete authentication: %s", matches))
					foundOAuthURL = true
				}
			}

			// 检测认证成功消息
			if authSuccessPattern.MatchString(line) {
				log.Printf("[QwenCodeAuth] Detected auth success message in output")
				a.updateStatus(auth.AuthStateSuccess, "", "", "Authentication completed")
				return
			}

		case <-idleTimerChan:
			// 输出空闲超时，界面渲染完成
			if detectedAuthMenu && !sentEnter {
				log.Printf("[QwenCodeAuth] Output idle detected, auth menu should be ready, sending '1' to select Qwen OAuth")
				if err := a.SendInput("1\n"); err != nil {
					log.Printf("[QwenCodeAuth] Failed to send input: %v", err)
				}
				sentEnter = true
			}
			idleTimerChan = nil

		case <-authCheckTicker.C:
			// 定期检查认证是否成功
			if a.checkAuthSuccess(ctx, volumeName) {
				log.Printf("[QwenCodeAuth] Auth file detected, authentication successful")
				a.updateStatus(auth.AuthStateSuccess, "", "", "Authentication completed")
				return
			}

			// 检查容器是否仍在运行
			running, err := a.dockerClient.IsContainerRunning(ctx, a.containerID)
			if err != nil {
				log.Printf("[QwenCodeAuth] Failed to check container status: %v", err)
				continue
			}
			if !running {
				if a.checkAuthSuccess(ctx, volumeName) {
					a.updateStatus(auth.AuthStateSuccess, "", "", "Authentication completed")
				} else {
					a.updateStatusWithError(auth.AuthStateFailed, fmt.Errorf("container stopped without successful auth"))
				}
				return
			}
		}
	}
}

// checkAuthSuccess 检查认证是否成功
func (a *Authenticator) checkAuthSuccess(ctx context.Context, volumeName string) bool {
	authFile := fmt.Sprintf("%s/%s", a.task.AuthDir, a.task.AuthFile)
	exists, err := a.dockerClient.FileExistsInVolume(ctx, volumeName, a.task.AuthDir, authFile)
	if err != nil {
		log.Printf("[QwenCodeAuth] Failed to check auth file: %v", err)
		return false
	}
	return exists
}

// SendInput 发送用户输入到容器
func (a *Authenticator) SendInput(input string) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.containerIO == nil {
		return fmt.Errorf("container IO not available")
	}

	_, err := a.containerIO.Writer.Write([]byte(input))
	return err
}

// GetStatus 获取当前认证状态
func (a *Authenticator) GetStatus() *auth.AuthStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

// Stop 停止认证并清理资源
func (a *Authenticator) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cancelFunc != nil {
		a.cancelFunc()
	}

	return a.cleanupLocked()
}

// cleanup 清理资源
func (a *Authenticator) cleanup() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cleanupLocked()
}

func (a *Authenticator) cleanupLocked() error {
	if a.containerIO != nil {
		a.containerIO.Conn.Close()
		a.containerIO = nil
	}

	if a.containerID != "" && a.dockerClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 停止并删除容器
		a.dockerClient.StopContainer(ctx, a.containerID, nil)
		a.dockerClient.RemoveContainer(ctx, a.containerID, true)
		a.containerID = ""
	}

	return nil
}

// updateStatus 更新状态
func (a *Authenticator) updateStatus(state auth.AuthState, oauthURL, userCode, message string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.updateStatusLocked(state, oauthURL, userCode, message)
}

// updateStatusLocked 更新状态（调用者已持有锁）
func (a *Authenticator) updateStatusLocked(state auth.AuthState, oauthURL, userCode, message string) {
	a.status = &auth.AuthStatus{
		State:    state,
		OAuthURL: oauthURL,
		UserCode: userCode,
		Message:  message,
	}

	// 非阻塞发送状态更新
	select {
	case a.statusChan <- a.status:
	default:
	}
}

// updateStatusWithOAuth 更新状态（包含OAuth信息）
func (a *Authenticator) updateStatusWithOAuth(state auth.AuthState, oauthURL, userCode, message string) {
	a.updateStatus(state, oauthURL, userCode, message)
}

// updateStatusWithError 更新错误状态
func (a *Authenticator) updateStatusWithError(state auth.AuthState, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.status = &auth.AuthStatus{
		State:   state,
		Message: err.Error(),
		Error:   err,
	}

	select {
	case a.statusChan <- a.status:
	default:
	}
}

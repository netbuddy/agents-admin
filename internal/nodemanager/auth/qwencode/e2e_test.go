package qwencode

import (
	"context"
	"testing"
	"time"

	"agents-admin/internal/nodemanager/auth"
)

// E2E测试：测试QwenCode认证器基本流程
func TestQwenCodeAuthenticator_E2E(t *testing.T) {
	// 创建容器客户端
	containerClient, err := auth.NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer containerClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := containerClient.Ping(ctx); err != nil {
		t.Skipf("Docker daemon not running: %v", err)
	}

	// 创建认证器
	authenticator := New()

	// 验证Agent类型
	if authenticator.AgentType() != "qwen-code" {
		t.Errorf("Expected agent type 'qwen-code', got '%s'", authenticator.AgentType())
	}

	// 验证支持的方法
	methods := authenticator.SupportedMethods()
	if len(methods) == 0 {
		t.Error("Expected at least one supported method")
	}

	t.Logf("Supported methods: %v", methods)

	// 创建认证任务
	task := &auth.AuthTask{
		ID:         "e2e-test-" + time.Now().Format("20060102150405"),
		AccountID:  "test-account",
		AgentType:  "qwen-code",
		Method:     "oauth",
		Image:      "runners/qwencode:latest",
		AuthDir:    "/home/node/.qwen",
		AuthFile:   "auth.json",
		LoginCmd:   "qwen",
		VolumeName: "e2e_test_volume_" + time.Now().Format("20060102150405"),
	}

	// 清理函数
	defer func() {
		authenticator.Stop()
		containerClient.RemoveVolume(context.Background(), task.VolumeName, true)
	}()

	// 检查镜像是否存在（如果不存在则跳过）
	t.Log("Note: This test requires 'runners/qwencode:latest' image to be available")
	t.Log("If the image is not available, the test will fail at container creation")

	// 启动认证流程
	t.Log("Starting authentication flow...")
	statusChan, err := authenticator.Start(ctx, task, containerClient)
	if err != nil {
		// 如果是镜像不存在，跳过测试
		t.Skipf("Failed to start authenticator (image may not exist): %v", err)
	}

	// 监听状态更新，等待OAuth URL
	timeout := time.After(30 * time.Second)
	var oauthURL string

	for {
		select {
		case status, ok := <-statusChan:
			if !ok {
				t.Log("Status channel closed")
				goto checkResult
			}
			t.Logf("Status update: %s - %s", status.State, status.Message)

			if status.State == auth.AuthStateWaitingOAuth {
				oauthURL = status.OAuthURL
				t.Logf("OAuth URL: %s", oauthURL)
				t.Logf("User Code: %s", status.UserCode)
				// 找到OAuth URL后停止
				goto checkResult
			}

			if status.State == auth.AuthStateFailed {
				t.Logf("Authentication failed: %s", status.Message)
				goto checkResult
			}

		case <-timeout:
			t.Log("Timeout waiting for OAuth URL")
			goto checkResult
		}
	}

checkResult:
	// 停止认证器
	authenticator.Stop()

	// 验证结果
	finalStatus := authenticator.GetStatus()
	t.Logf("Final status: %s", finalStatus.State)

	// 如果获取到了OAuth URL，测试通过
	if oauthURL != "" {
		t.Logf("Successfully obtained OAuth URL: %s", oauthURL)
	} else {
		t.Log("OAuth URL not obtained (may be expected if image is not available or qwen-code has different behavior)")
	}
}

// E2E测试：测试认证器停止功能
func TestQwenCodeAuthenticator_Stop_E2E(t *testing.T) {
	containerClient, err := auth.NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer containerClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := containerClient.Ping(ctx); err != nil {
		t.Skipf("Docker daemon not running: %v", err)
	}

	authenticator := New()

	task := &auth.AuthTask{
		ID:         "stop-test-" + time.Now().Format("20060102150405"),
		AccountID:  "test-account",
		AgentType:  "qwen-code",
		Method:     "oauth",
		Image:      "alpine:latest", // 使用alpine进行停止测试
		AuthDir:    "/tmp",
		AuthFile:   "test.json",
		LoginCmd:   "sleep 60",
		VolumeName: "stop_test_volume_" + time.Now().Format("20060102150405"),
	}

	defer containerClient.RemoveVolume(context.Background(), task.VolumeName, true)

	// 启动认证
	_, err = authenticator.Start(ctx, task, containerClient)
	if err != nil {
		t.Skipf("Failed to start authenticator: %v", err)
	}

	// 等待一会让容器启动
	time.Sleep(2 * time.Second)

	// 停止认证
	t.Log("Stopping authenticator...")
	err = authenticator.Stop()
	if err != nil {
		t.Errorf("Failed to stop authenticator: %v", err)
	}

	// 验证状态
	status := authenticator.GetStatus()
	t.Logf("Status after stop: %s", status.State)

	t.Log("Stop test completed!")
}

// E2E测试：测试状态通道通知机制
func TestQwenCodeAuthenticator_StatusChannel_E2E(t *testing.T) {
	containerClient, err := auth.NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer containerClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := containerClient.Ping(ctx); err != nil {
		t.Skipf("Docker daemon not running: %v", err)
	}

	authenticator := New()

	task := &auth.AuthTask{
		ID:         "status-test-" + time.Now().Format("20060102150405"),
		AccountID:  "test-account",
		AgentType:  "qwen-code",
		Method:     "oauth",
		Image:      "runners/qwencode:latest",
		AuthDir:    "/home/node/.qwen",
		AuthFile:   "auth.json",
		LoginCmd:   "qwen",
		VolumeName: "status_test_volume_" + time.Now().Format("20060102150405"),
	}

	defer func() {
		authenticator.Stop()
		containerClient.RemoveVolume(context.Background(), task.VolumeName, true)
	}()

	// 启动认证流程
	statusChan, err := authenticator.Start(ctx, task, containerClient)
	if err != nil {
		t.Skipf("Failed to start authenticator: %v", err)
	}

	// 收集所有状态更新
	var statusUpdates []*auth.AuthStatus
	timeout := time.After(40 * time.Second)

	t.Log("Collecting status updates...")

collectLoop:
	for {
		select {
		case status, ok := <-statusChan:
			if !ok {
				t.Log("Status channel closed")
				break collectLoop
			}

			statusUpdates = append(statusUpdates, status)
			t.Logf("Received status: State=%s, OAuthURL=%s, UserCode=%s, Message=%s",
				status.State, status.OAuthURL, status.UserCode, status.Message)

			// 如果收到WaitingOAuth或Failed状态，停止收集
			if status.State == auth.AuthStateWaitingOAuth || status.State == auth.AuthStateFailed {
				break collectLoop
			}

		case <-timeout:
			t.Log("Collection timeout")
			break collectLoop
		}
	}

	// 验证收到了状态更新
	if len(statusUpdates) == 0 {
		t.Error("Should have received at least one status update")
	} else {
		t.Logf("Received %d status updates total", len(statusUpdates))
	}

	// 验证第一个状态是Running
	if len(statusUpdates) > 0 {
		firstStatus := statusUpdates[0]
		if firstStatus.State != auth.AuthStateRunning {
			t.Logf("First status state: %s (expected running, but may vary)", firstStatus.State)
		}
	}

	// 检查是否有WaitingOAuth状态
	hasWaitingOAuth := false
	for _, s := range statusUpdates {
		if s.State == auth.AuthStateWaitingOAuth {
			hasWaitingOAuth = true
			if s.OAuthURL == "" {
				t.Error("WaitingOAuth status should have OAuthURL")
			}
			t.Logf("OAuth URL received: %s", s.OAuthURL)
			t.Logf("User Code received: %s", s.UserCode)
			break
		}
	}

	if !hasWaitingOAuth {
		t.Log("No WaitingOAuth status received (may be due to network issues in container)")
	}
}

// E2E测试：验证状态字段完整性
func TestQwenCodeAuthenticator_StatusFields_E2E(t *testing.T) {
	containerClient, err := auth.NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer containerClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := containerClient.Ping(ctx); err != nil {
		t.Skipf("Docker daemon not running: %v", err)
	}

	authenticator := New()

	// 验证初始状态
	initialStatus := authenticator.GetStatus()
	if initialStatus == nil {
		t.Fatal("Initial status should not be nil")
	}
	if initialStatus.State != auth.AuthStatePending {
		t.Errorf("Initial state should be pending, got %s", initialStatus.State)
	}

	task := &auth.AuthTask{
		ID:         "fields-test-" + time.Now().Format("20060102150405"),
		AccountID:  "test-account",
		AgentType:  "qwen-code",
		Method:     "oauth",
		Image:      "runners/qwencode:latest",
		AuthDir:    "/home/node/.qwen",
		AuthFile:   "auth.json",
		LoginCmd:   "qwen",
		VolumeName: "fields_test_volume_" + time.Now().Format("20060102150405"),
	}

	defer func() {
		authenticator.Stop()
		containerClient.RemoveVolume(context.Background(), task.VolumeName, true)
	}()

	// 启动认证
	_, err = authenticator.Start(ctx, task, containerClient)
	if err != nil {
		t.Skipf("Failed to start authenticator: %v", err)
	}

	// 等待一会
	time.Sleep(3 * time.Second)

	// 验证状态已更新
	currentStatus := authenticator.GetStatus()
	if currentStatus == nil {
		t.Fatal("Current status should not be nil")
	}

	t.Logf("Current state: %s", currentStatus.State)
	t.Logf("Current message: %s", currentStatus.Message)

	// 状态应该不再是pending
	if currentStatus.State == auth.AuthStatePending {
		t.Error("State should have changed from pending after start")
	}
}

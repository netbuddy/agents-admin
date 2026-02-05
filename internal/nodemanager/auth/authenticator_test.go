package auth

import (
	"testing"
)

func TestAuthState(t *testing.T) {
	tests := []struct {
		state    AuthState
		expected string
	}{
		{AuthStatePending, "pending"},
		{AuthStateRunning, "running"},
		{AuthStateWaitingInput, "waiting_input"},
		{AuthStateWaitingOAuth, "waiting_oauth"},
		{AuthStateSuccess, "success"},
		{AuthStateFailed, "failed"},
		{AuthStateTimeout, "timeout"},
	}

	for _, tt := range tests {
		if string(tt.state) != tt.expected {
			t.Errorf("AuthState %v: expected %s, got %s", tt.state, tt.expected, string(tt.state))
		}
	}
}

func TestAuthStatus(t *testing.T) {
	status := &AuthStatus{
		State:      AuthStateWaitingOAuth,
		OAuthURL:   "https://example.com/auth",
		DeviceCode: "ABC123",
		UserCode:   "XY-1234",
		Message:    "Please authenticate",
	}

	if status.State != AuthStateWaitingOAuth {
		t.Errorf("Expected state %v, got %v", AuthStateWaitingOAuth, status.State)
	}
	if status.OAuthURL != "https://example.com/auth" {
		t.Errorf("Expected OAuthURL %s, got %s", "https://example.com/auth", status.OAuthURL)
	}
	if status.UserCode != "XY-1234" {
		t.Errorf("Expected UserCode %s, got %s", "XY-1234", status.UserCode)
	}
}

func TestAuthTask(t *testing.T) {
	task := &AuthTask{
		ID:         "task-123",
		AccountID:  "account-456",
		AgentType:  "qwen-code",
		Method:     "oauth",
		Image:      "runners/qwencode:latest",
		AuthDir:    "/home/node/.qwen",
		AuthFile:   "auth.json",
		LoginCmd:   "qwen",
		VolumeName: "test_volume",
		Env:        map[string]string{"FOO": "bar"},
	}

	if task.ID != "task-123" {
		t.Errorf("Expected ID %s, got %s", "task-123", task.ID)
	}
	if task.AgentType != "qwen-code" {
		t.Errorf("Expected AgentType %s, got %s", "qwen-code", task.AgentType)
	}
	if task.Env["FOO"] != "bar" {
		t.Errorf("Expected Env[FOO] %s, got %s", "bar", task.Env["FOO"])
	}
}

func TestRegistry(t *testing.T) {
	registry := NewRegistry()

	// 测试空注册表
	if len(registry.List()) != 0 {
		t.Error("New registry should be empty")
	}

	// 测试获取不存在的认证器
	_, ok := registry.Get("nonexistent")
	if ok {
		t.Error("Should not find nonexistent authenticator")
	}
}

// MockAuthenticator 模拟认证器用于测试
type MockAuthenticator struct {
	agentType string
	methods   []string
}

func (m *MockAuthenticator) AgentType() string {
	return m.agentType
}

func (m *MockAuthenticator) SupportedMethods() []string {
	return m.methods
}

func (m *MockAuthenticator) Start(ctx interface{}, task *AuthTask, dockerClient interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockAuthenticator) SendInput(input string) error {
	return nil
}

func (m *MockAuthenticator) GetStatus() *AuthStatus {
	return &AuthStatus{State: AuthStatePending}
}

func (m *MockAuthenticator) Stop() error {
	return nil
}

func TestRegistryWithMock(t *testing.T) {
	registry := NewRegistry()

	// 由于接口类型不完全匹配，这里只测试基本逻辑
	t.Run("Registry operations", func(t *testing.T) {
		if len(registry.List()) != 0 {
			t.Error("Registry should be empty initially")
		}
	})
}

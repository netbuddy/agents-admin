package qwencode

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"agents-admin/pkg/auth"
)

func TestNew(t *testing.T) {
	authenticator := New()
	if authenticator == nil {
		t.Fatal("New() should return non-nil authenticator")
	}
}

func TestAgentType(t *testing.T) {
	authenticator := New()
	if authenticator.AgentType() != "qwen-code" {
		t.Errorf("Expected agent type 'qwen-code', got '%s'", authenticator.AgentType())
	}
}

func TestSupportedMethods(t *testing.T) {
	authenticator := New()
	methods := authenticator.SupportedMethods()

	if len(methods) != 2 {
		t.Errorf("Expected 2 methods, got %d", len(methods))
	}

	hasOAuth := false
	hasAPIKey := false
	for _, m := range methods {
		if m == "oauth" {
			hasOAuth = true
		}
		if m == "api_key" {
			hasAPIKey = true
		}
	}

	if !hasOAuth {
		t.Error("Should support oauth method")
	}
	if !hasAPIKey {
		t.Error("Should support api_key method")
	}
}

func TestGetStatus(t *testing.T) {
	authenticator := New()
	status := authenticator.GetStatus()

	if status == nil {
		t.Fatal("GetStatus() should return non-nil status")
	}
	if status.State != auth.AuthStatePending {
		t.Errorf("Initial state should be pending, got %s", status.State)
	}
}

func TestOAuthURLPattern(t *testing.T) {
	pattern := regexp.MustCompile(`https://chat\.qwen\.ai/authorize\?[^\s]+`)

	testCases := []struct {
		input    string
		expected bool
	}{
		{"https://chat.qwen.ai/authorize?user_code=GG-OBMFB&client=qwen-code", true},
		{"https://chat.qwen.ai/authorize?code=ABC123", true},
		{"https://other.site.com/authorize?code=ABC", false},
		{"not a url", false},
		{"", false},
	}

	for _, tc := range testCases {
		match := pattern.MatchString(tc.input)
		if match != tc.expected {
			t.Errorf("Pattern match for '%s': expected %v, got %v", tc.input, tc.expected, match)
		}
	}
}

func TestUserCodePattern(t *testing.T) {
	pattern := regexp.MustCompile(`user_code=([A-Z0-9-]+)`)

	testCases := []struct {
		input        string
		expectedCode string
	}{
		{"https://chat.qwen.ai/authorize?user_code=GG-OBMFB&client=qwen-code", "GG-OBMFB"},
		{"user_code=ABC-123", "ABC-123"},
		{"user_code=SIMPLE", "SIMPLE"},
		{"no code here", ""},
	}

	for _, tc := range testCases {
		matches := pattern.FindStringSubmatch(tc.input)
		code := ""
		if len(matches) > 1 {
			code = matches[1]
		}
		if code != tc.expectedCode {
			t.Errorf("User code extraction for '%s': expected '%s', got '%s'", tc.input, tc.expectedCode, code)
		}
	}
}

func TestGetStartedPattern(t *testing.T) {
	pattern := regexp.MustCompile(`Get started|How would you like to authenticate`)

	testCases := []struct {
		input    string
		expected bool
	}{
		{"Get started", true},
		{"How would you like to authenticate", true},
		{"How would you like to authenticate for this project?", true},
		{"Some random text", false},
		{"", false},
	}

	for _, tc := range testCases {
		match := pattern.MatchString(tc.input)
		if match != tc.expected {
			t.Errorf("Pattern match for '%s': expected %v, got %v", tc.input, tc.expected, match)
		}
	}
}

func TestSendInputWithoutIO(t *testing.T) {
	authenticator := New()
	err := authenticator.SendInput("test")
	if err == nil {
		t.Error("SendInput should fail when container IO is not available")
	}
}

func TestStopWithoutStart(t *testing.T) {
	authenticator := New()
	err := authenticator.Stop()
	if err != nil {
		t.Errorf("Stop should not fail when not started: %v", err)
	}
}

func TestUpdateStatusAndChannel(t *testing.T) {
	authenticator := New()
	authenticator.statusChan = make(chan *auth.AuthStatus, 10)

	// 测试 updateStatus 发送状态到 channel
	authenticator.updateStatus(auth.AuthStateWaitingOAuth, "https://example.com/oauth", "ABC-123", "Please authenticate")

	// 验证状态已更新
	status := authenticator.GetStatus()
	if status.State != auth.AuthStateWaitingOAuth {
		t.Errorf("Expected state waiting_oauth, got %s", status.State)
	}
	if status.OAuthURL != "https://example.com/oauth" {
		t.Errorf("Expected OAuth URL 'https://example.com/oauth', got '%s'", status.OAuthURL)
	}
	if status.UserCode != "ABC-123" {
		t.Errorf("Expected user code 'ABC-123', got '%s'", status.UserCode)
	}

	// 验证状态已发送到 channel
	select {
	case received := <-authenticator.statusChan:
		if received.State != auth.AuthStateWaitingOAuth {
			t.Errorf("Received state should be waiting_oauth, got %s", received.State)
		}
		if received.OAuthURL != "https://example.com/oauth" {
			t.Errorf("Received OAuth URL mismatch")
		}
	default:
		t.Error("Status should be sent to channel")
	}
}

func TestUpdateStatusWithOAuth(t *testing.T) {
	authenticator := New()
	authenticator.statusChan = make(chan *auth.AuthStatus, 10)

	// 测试 updateStatusWithOAuth
	authenticator.updateStatusWithOAuth(auth.AuthStateWaitingOAuth, "https://chat.qwen.ai/authorize?user_code=TEST-CODE", "TEST-CODE", "Please visit URL")

	status := authenticator.GetStatus()
	if status.State != auth.AuthStateWaitingOAuth {
		t.Errorf("Expected state waiting_oauth, got %s", status.State)
	}
	if status.OAuthURL != "https://chat.qwen.ai/authorize?user_code=TEST-CODE" {
		t.Errorf("OAuth URL mismatch")
	}
	if status.UserCode != "TEST-CODE" {
		t.Errorf("User code mismatch, expected 'TEST-CODE', got '%s'", status.UserCode)
	}
}

func TestUpdateStatusWithError(t *testing.T) {
	authenticator := New()
	authenticator.statusChan = make(chan *auth.AuthStatus, 10)

	// 测试 updateStatusWithError
	testErr := fmt.Errorf("test error message")
	authenticator.updateStatusWithError(auth.AuthStateFailed, testErr)

	status := authenticator.GetStatus()
	if status.State != auth.AuthStateFailed {
		t.Errorf("Expected state failed, got %s", status.State)
	}
	if status.Error == nil {
		t.Error("Error should not be nil")
	}
	if status.Message != "test error message" {
		t.Errorf("Message should be error message, got '%s'", status.Message)
	}
}

func TestStatusChannelNonBlocking(t *testing.T) {
	authenticator := New()
	// 创建一个容量为1的channel
	authenticator.statusChan = make(chan *auth.AuthStatus, 1)

	// 第一次更新应该成功
	authenticator.updateStatus(auth.AuthStateRunning, "", "", "First update")

	// 第二次更新不应该阻塞（channel已满时使用default分支）
	done := make(chan bool, 1)
	go func() {
		authenticator.updateStatus(auth.AuthStateWaitingOAuth, "url", "code", "Second update")
		done <- true
	}()

	select {
	case <-done:
		// 成功，没有阻塞
	case <-time.After(1 * time.Second):
		t.Error("updateStatus should not block when channel is full")
	}
}

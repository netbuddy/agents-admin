package nodemanager

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"agents-admin/internal/nodemanager/auth"
)

func TestNewAuthControllerV2(t *testing.T) {
	cfg := Config{
		APIServerURL: "http://localhost:8080",
		NodeID:       "test-node",
	}

	controller, err := NewAuthControllerV2(cfg)
	if err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}
	defer controller.Close()

	if controller.containerClient == nil {
		t.Error("Container client should not be nil")
	}
	if controller.authRegistry == nil {
		t.Error("Auth registry should not be nil")
	}
}

func TestAuthControllerV2_HandleAuthStatusUpdate(t *testing.T) {
	var receivedPayload map[string]interface{}
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PATCH" {
			mu.Lock()
			json.NewDecoder(r.Body).Decode(&receivedPayload)
			mu.Unlock()
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := Config{
		APIServerURL: server.URL,
		NodeID:       "test-node",
	}

	controller, err := NewAuthControllerV2(cfg)
	if err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}
	defer controller.Close()

	// 测试 WaitingOAuth 状态
	status := &auth.AuthStatus{
		State:    auth.AuthStateWaitingOAuth,
		OAuthURL: "https://chat.qwen.ai/authorize?user_code=TEST-123",
		UserCode: "TEST-123",
		Message:  "Please authenticate",
	}

	controller.handleAuthStatusUpdate("test-action-id", status, "test_vol")

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if receivedPayload == nil {
		t.Fatal("Should have received a payload")
	}

	if receivedPayload["status"] != "waiting" {
		t.Errorf("Expected status 'waiting', got '%v'", receivedPayload["status"])
	}
}

func TestAuthControllerV2_ReportActionStatus(t *testing.T) {
	var receivedPayloads []map[string]interface{}
	var receivedPaths []string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PATCH" {
			var payload map[string]interface{}
			json.NewDecoder(r.Body).Decode(&payload)
			mu.Lock()
			receivedPayloads = append(receivedPayloads, payload)
			receivedPaths = append(receivedPaths, r.URL.Path)
			mu.Unlock()
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := Config{
		APIServerURL: server.URL,
		NodeID:       "test-node",
	}

	controller, err := NewAuthControllerV2(cfg)
	if err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}
	defer controller.Close()

	// 测试不同状态的上报
	testCases := []struct {
		actionID string
		status   string
		phase    string
		message  string
		progress int
		errMsg   string
	}{
		{"act-1", "running", "initializing", "Starting", 30, ""},
		{"act-2", "success", "finalizing", "Done", 100, ""},
		{"act-3", "failed", "", "", 0, "auth error"},
	}

	for _, tc := range testCases {
		controller.reportActionStatus(tc.actionID, tc.status, tc.phase, tc.message, tc.progress, nil, tc.errMsg)
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(receivedPayloads) != len(testCases) {
		t.Errorf("Expected %d payloads, got %d", len(testCases), len(receivedPayloads))
	}

	// 验证路径使用 /api/v1/actions/{id}
	for _, path := range receivedPaths {
		if !strings.HasPrefix(path, "/api/v1/actions/") {
			t.Errorf("Expected path /api/v1/actions/*, got %s", path)
		}
	}
}

func TestAuthControllerV2_CancelAuthTask(t *testing.T) {
	cfg := Config{
		APIServerURL: "http://localhost:8080",
		NodeID:       "test-node",
	}

	controller, err := NewAuthControllerV2(cfg)
	if err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}
	defer controller.Close()

	// 测试取消不存在的 Action
	err = controller.CancelAuthTask("non-existent-action")
	if err == nil {
		t.Error("Should return error for non-existent action")
	}

	// 添加一个模拟的运行中 Action
	ctx, cancel := context.WithCancel(context.Background())
	controller.mu.Lock()
	controller.runningActions["test-action"] = &runningAction{
		actionID: "test-action",
		cancel:   cancel,
	}
	controller.mu.Unlock()

	// 取消 Action
	err = controller.CancelAuthTask("test-action")
	if err != nil {
		t.Errorf("Cancel should succeed: %v", err)
	}

	// 验证 context 已取消
	select {
	case <-ctx.Done():
		// 成功
	default:
		t.Error("Context should be cancelled")
	}

	// 验证 Action 已从 map 中移除
	controller.mu.Lock()
	_, exists := controller.runningActions["test-action"]
	controller.mu.Unlock()

	if exists {
		t.Error("Action should be removed from running actions")
	}
}

func TestAuthControllerV2_CleanupAllActions(t *testing.T) {
	cfg := Config{
		APIServerURL: "http://localhost:8080",
		NodeID:       "test-node",
	}

	controller, err := NewAuthControllerV2(cfg)
	if err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}
	defer controller.Close()

	// 添加多个模拟 Action
	for i := 0; i < 3; i++ {
		_, cancel := context.WithCancel(context.Background())
		controller.runningActions[string(rune('a'+i))] = &runningAction{
			actionID: string(rune('a' + i)),
			cancel:   cancel,
		}
	}

	controller.cleanupAllActions()

	if len(controller.runningActions) != 0 {
		t.Errorf("All actions should be cleaned up, got %d remaining", len(controller.runningActions))
	}
}

func TestSanitizeForVolume(t *testing.T) {
	cases := []struct{ input, want string }{
		{"test@email.com", "test_email_com"},
		{"simple", "simple"},
		{"with-dash", "with_dash"},
		{"with space", "with_space"},
	}
	for _, tc := range cases {
		got := sanitizeForVolume(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeForVolume(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFindPredefinedAgentType(t *testing.T) {
	at := findPredefinedAgentType("qwen-code")
	if at == nil {
		t.Fatal("expected to find qwen-code")
	}
	if at.ID != "qwen-code" {
		t.Errorf("expected id=qwen-code, got %s", at.ID)
	}

	at = findPredefinedAgentType("nonexistent")
	if at != nil {
		t.Error("expected nil for nonexistent agent type")
	}
}

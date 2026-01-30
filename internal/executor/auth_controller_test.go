package executor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"agents-admin/pkg/auth"
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

	if controller.dockerClient == nil {
		t.Error("Docker client should not be nil")
	}
	if controller.authRegistry == nil {
		t.Error("Auth registry should not be nil")
	}
}

func TestAuthControllerV2_HandleStatusUpdate(t *testing.T) {
	// 创建mock HTTP server
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

	ctx := context.Background()

	// 测试 WaitingOAuth 状态
	status := &auth.AuthStatus{
		State:    auth.AuthStateWaitingOAuth,
		OAuthURL: "https://chat.qwen.ai/authorize?user_code=TEST-123",
		UserCode: "TEST-123",
		Message:  "Please authenticate",
	}

	controller.handleStatusUpdate(ctx, "test-task-id", status)

	// 等待HTTP请求完成
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if receivedPayload == nil {
		t.Fatal("Should have received a payload")
	}

	if receivedPayload["status"] != "waiting_oauth" {
		t.Errorf("Expected status 'waiting_oauth', got '%v'", receivedPayload["status"])
	}
	if receivedPayload["oauth_url"] != "https://chat.qwen.ai/authorize?user_code=TEST-123" {
		t.Errorf("OAuth URL mismatch, got '%v'", receivedPayload["oauth_url"])
	}
	if receivedPayload["user_code"] != "TEST-123" {
		t.Errorf("User code mismatch, got '%v'", receivedPayload["user_code"])
	}
}

func TestAuthControllerV2_ReportAuthTaskStatus(t *testing.T) {
	var receivedPayloads []map[string]interface{}
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PATCH" {
			var payload map[string]interface{}
			json.NewDecoder(r.Body).Decode(&payload)
			mu.Lock()
			receivedPayloads = append(receivedPayloads, payload)
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

	ctx := context.Background()

	// 测试不同状态的上报
	testCases := []struct {
		status  string
		message string
	}{
		{"running", "Starting authentication..."},
		{"success", "Authentication completed"},
		{"failed", "Authentication failed"},
	}

	for _, tc := range testCases {
		controller.reportAuthTaskStatus(ctx, "task-"+tc.status, tc.status, nil, strPtr(tc.message))
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(receivedPayloads) != len(testCases) {
		t.Errorf("Expected %d payloads, got %d", len(testCases), len(receivedPayloads))
	}
}

func TestAuthControllerV2_ReportAuthTaskOAuthURL(t *testing.T) {
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

	ctx := context.Background()

	// 测试 OAuth URL 上报
	controller.reportAuthTaskOAuthURL(ctx, "test-task", "https://example.com/oauth", "USER-CODE")

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if receivedPayload == nil {
		t.Fatal("Should have received a payload")
	}

	if receivedPayload["status"] != "waiting_oauth" {
		t.Errorf("Status should be 'waiting_oauth', got '%v'", receivedPayload["status"])
	}
	if receivedPayload["oauth_url"] != "https://example.com/oauth" {
		t.Errorf("OAuth URL mismatch")
	}
	if receivedPayload["user_code"] != "USER-CODE" {
		t.Errorf("User code mismatch")
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

	// 测试取消不存在的任务
	err = controller.CancelAuthTask("non-existent-task")
	if err == nil {
		t.Error("Should return error for non-existent task")
	}

	// 添加一个模拟的运行中任务
	ctx, cancel := context.WithCancel(context.Background())
	controller.mu.Lock()
	controller.runningAuthTasks["test-task"] = &runningAuth{
		taskID: "test-task",
		cancel: cancel,
	}
	controller.mu.Unlock()

	// 取消任务
	err = controller.CancelAuthTask("test-task")
	if err != nil {
		t.Errorf("Cancel should succeed: %v", err)
	}

	// 验证context已取消
	select {
	case <-ctx.Done():
		// 成功
	default:
		t.Error("Context should be cancelled")
	}

	// 验证任务已从map中移除
	controller.mu.Lock()
	_, exists := controller.runningAuthTasks["test-task"]
	controller.mu.Unlock()

	if exists {
		t.Error("Task should be removed from running tasks")
	}
}

func TestAuthControllerV2_CleanupAllTasks(t *testing.T) {
	cfg := Config{
		APIServerURL: "http://localhost:8080",
		NodeID:       "test-node",
	}

	controller, err := NewAuthControllerV2(cfg)
	if err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}
	defer controller.Close()

	// 添加多个模拟任务
	var cancels []context.CancelFunc
	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		_ = ctx
		cancels = append(cancels, cancel)
		controller.runningAuthTasks[string(rune('a'+i))] = &runningAuth{
			taskID: string(rune('a' + i)),
			cancel: cancel,
		}
	}

	controller.cleanupAllTasks()

	if len(controller.runningAuthTasks) != 0 {
		t.Errorf("All tasks should be cleaned up, got %d remaining", len(controller.runningAuthTasks))
	}
}

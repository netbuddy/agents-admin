package storage

import (
	"context"
	"os"
	"testing"
	"time"
)

func getTestRedisAddr() string {
	if addr := os.Getenv("REDIS_ADDR"); addr != "" {
		return addr
	}
	return "localhost:6380"
}

func setupTestRedisStore(t *testing.T) *RedisStore {
	store, err := NewRedisStore(getTestRedisAddr(), "", 1) // 使用 DB 1 进行测试
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	return store
}

func TestRedisStore_AuthSession(t *testing.T) {
	store := setupTestRedisStore(t)
	defer store.Close()
	ctx := context.Background()

	// 清理测试数据
	store.client.FlushDB(ctx)

	// 测试创建认证会话
	session := &AuthSession{
		TaskID:    "test_auth_123",
		AccountID: "test_account",
		Method:    "oauth",
		NodeID:    "test-node-01",
		Status:    "assigned",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	err := store.CreateAuthSession(ctx, session)
	if err != nil {
		t.Fatalf("CreateAuthSession failed: %v", err)
	}

	// 测试获取认证会话
	got, err := store.GetAuthSession(ctx, "test_auth_123")
	if err != nil {
		t.Fatalf("GetAuthSession failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetAuthSession returned nil")
	}
	if got.TaskID != session.TaskID {
		t.Errorf("TaskID = %s, want %s", got.TaskID, session.TaskID)
	}
	if got.Status != session.Status {
		t.Errorf("Status = %s, want %s", got.Status, session.Status)
	}

	// 测试按账号 ID 获取
	got, err = store.GetAuthSessionByAccountID(ctx, "test_account")
	if err != nil {
		t.Fatalf("GetAuthSessionByAccountID failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetAuthSessionByAccountID returned nil")
	}
	if got.TaskID != session.TaskID {
		t.Errorf("TaskID = %s, want %s", got.TaskID, session.TaskID)
	}

	// 测试更新认证会话
	err = store.UpdateAuthSession(ctx, "test_auth_123", map[string]interface{}{
		"status":        "running",
		"terminal_port": 8080,
	})
	if err != nil {
		t.Fatalf("UpdateAuthSession failed: %v", err)
	}

	got, _ = store.GetAuthSession(ctx, "test_auth_123")
	if got.Status != "running" {
		t.Errorf("Status = %s, want running", got.Status)
	}
	if got.TerminalPort != 8080 {
		t.Errorf("TerminalPort = %d, want 8080", got.TerminalPort)
	}

	// 测试列出所有会话
	sessions, err := store.ListAuthSessions(ctx)
	if err != nil {
		t.Fatalf("ListAuthSessions failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("ListAuthSessions returned %d sessions, want 1", len(sessions))
	}

	// 测试删除认证会话
	err = store.DeleteAuthSession(ctx, "test_auth_123")
	if err != nil {
		t.Fatalf("DeleteAuthSession failed: %v", err)
	}

	got, _ = store.GetAuthSession(ctx, "test_auth_123")
	if got != nil {
		t.Error("Session should be deleted")
	}
}

func TestRedisStore_WorkflowEvents(t *testing.T) {
	store := setupTestRedisStore(t)
	defer store.Close()
	ctx := context.Background()

	// 清理测试数据
	store.client.FlushDB(ctx)

	wfType := "auth"
	wfID := "test_workflow_123"

	// 测试发布事件
	event1 := &RedisWorkflowEvent{
		Type:      "auth.created",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"account_id": "test_account",
			"method":     "oauth",
		},
	}
	err := store.PublishEvent(ctx, wfType, wfID, event1)
	if err != nil {
		t.Fatalf("PublishEvent failed: %v", err)
	}

	event2 := &RedisWorkflowEvent{
		Type:      "auth.running",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"message": "Starting OAuth flow",
		},
	}
	err = store.PublishEvent(ctx, wfType, wfID, event2)
	if err != nil {
		t.Fatalf("PublishEvent failed: %v", err)
	}

	// 测试获取事件列表
	events, err := store.GetEvents(ctx, wfType, wfID, "", 100)
	if err != nil {
		t.Fatalf("GetEvents failed: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("GetEvents returned %d events, want 2", len(events))
	}
	if events[0].Type != "auth.created" {
		t.Errorf("First event type = %s, want auth.created", events[0].Type)
	}
	if events[1].Type != "auth.running" {
		t.Errorf("Second event type = %s, want auth.running", events[1].Type)
	}

	// 测试获取事件数量
	count, err := store.GetEventCount(ctx, wfType, wfID)
	if err != nil {
		t.Fatalf("GetEventCount failed: %v", err)
	}
	if count != 2 {
		t.Errorf("GetEventCount = %d, want 2", count)
	}

	// 测试删除事件流
	err = store.DeleteEvents(ctx, wfType, wfID)
	if err != nil {
		t.Fatalf("DeleteEvents failed: %v", err)
	}

	count, _ = store.GetEventCount(ctx, wfType, wfID)
	if count != 0 {
		t.Errorf("Events should be deleted, got count %d", count)
	}
}

func TestRedisStore_WorkflowState(t *testing.T) {
	store := setupTestRedisStore(t)
	defer store.Close()
	ctx := context.Background()

	// 清理测试数据
	store.client.FlushDB(ctx)

	wfType := "auth"
	wfID := "test_workflow_456"

	// 测试设置状态
	state := &RedisWorkflowState{
		State:       "running",
		Progress:    50,
		CurrentStep: "waiting_oauth",
	}
	err := store.SetWorkflowState(ctx, wfType, wfID, state)
	if err != nil {
		t.Fatalf("SetWorkflowState failed: %v", err)
	}

	// 测试获取状态
	got, err := store.GetWorkflowState(ctx, wfType, wfID)
	if err != nil {
		t.Fatalf("GetWorkflowState failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetWorkflowState returned nil")
	}
	if got.State != "running" {
		t.Errorf("State = %s, want running", got.State)
	}
	if got.Progress != 50 {
		t.Errorf("Progress = %d, want 50", got.Progress)
	}

	// 测试删除状态
	err = store.DeleteWorkflowState(ctx, wfType, wfID)
	if err != nil {
		t.Fatalf("DeleteWorkflowState failed: %v", err)
	}

	got, _ = store.GetWorkflowState(ctx, wfType, wfID)
	if got != nil {
		t.Error("State should be deleted")
	}
}

func TestRedisStore_NodeHeartbeat(t *testing.T) {
	store := setupTestRedisStore(t)
	defer store.Close()
	ctx := context.Background()

	// 清理测试数据
	store.client.FlushDB(ctx)

	nodeID := "test-node-01"

	// 测试更新心跳
	status := &NodeStatus{
		Status: "online",
		Capacity: map[string]int{
			"max":       2,
			"available": 1,
		},
	}
	err := store.UpdateNodeHeartbeat(ctx, nodeID, status)
	if err != nil {
		t.Fatalf("UpdateNodeHeartbeat failed: %v", err)
	}

	// 测试获取心跳
	got, err := store.GetNodeHeartbeat(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNodeHeartbeat failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetNodeHeartbeat returned nil")
	}
	if got.Status != "online" {
		t.Errorf("Status = %s, want online", got.Status)
	}

	// 测试列出在线节点
	nodes, err := store.ListOnlineNodes(ctx)
	if err != nil {
		t.Fatalf("ListOnlineNodes failed: %v", err)
	}
	if len(nodes) != 1 {
		t.Errorf("ListOnlineNodes returned %d nodes, want 1", len(nodes))
	}
	if nodes[0] != nodeID {
		t.Errorf("Node ID = %s, want %s", nodes[0], nodeID)
	}
}

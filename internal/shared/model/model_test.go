package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTaskStatus(t *testing.T) {
	tests := []struct {
		status TaskStatus
		want   string
	}{
		{TaskStatusPending, "pending"},
		{TaskStatusInProgress, "in_progress"},
		{TaskStatusCompleted, "completed"},
		{TaskStatusFailed, "failed"},
		{TaskStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("TaskStatus = %v, want %v", tt.status, tt.want)
		}
	}
}

func TestRunStatus(t *testing.T) {
	tests := []struct {
		status RunStatus
		want   string
	}{
		{RunStatusQueued, "queued"},
		{RunStatusRunning, "running"},
		{RunStatusDone, "done"},
		{RunStatusFailed, "failed"},
		{RunStatusCancelled, "cancelled"},
		{RunStatusTimeout, "timeout"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("RunStatus = %v, want %v", tt.status, tt.want)
		}
	}
}

func TestTaskJSONSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	// 使用扁平化的 Task 结构
	task := &Task{
		ID:        "task-123",
		Name:      "Test Task",
		Status:    TaskStatusPending,
		Type:      TaskTypeGeneral,
		Prompt:    &Prompt{Content: "hello"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Failed to marshal task: %v", err)
	}

	var decoded Task
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal task: %v", err)
	}

	if decoded.ID != task.ID {
		t.Errorf("ID = %v, want %v", decoded.ID, task.ID)
	}
	if decoded.Name != task.Name {
		t.Errorf("Name = %v, want %v", decoded.Name, task.Name)
	}
	if decoded.Status != task.Status {
		t.Errorf("Status = %v, want %v", decoded.Status, task.Status)
	}
	if decoded.Prompt == nil || task.Prompt == nil {
		t.Errorf("Prompt is nil")
	} else if decoded.Prompt.Content != task.Prompt.Content {
		t.Errorf("Prompt.Content = %v, want %v", decoded.Prompt.Content, task.Prompt.Content)
	}
}

func TestRunJSONSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	nodeID := "node-001"
	run := &Run{
		ID:        "run-123",
		TaskID:    "task-123",
		Status:    RunStatusRunning,
		NodeID:    &nodeID,
		StartedAt: &now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	data, err := json.Marshal(run)
	if err != nil {
		t.Fatalf("Failed to marshal run: %v", err)
	}

	var decoded Run
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal run: %v", err)
	}

	if decoded.ID != run.ID {
		t.Errorf("ID = %v, want %v", decoded.ID, run.ID)
	}
	if decoded.TaskID != run.TaskID {
		t.Errorf("TaskID = %v, want %v", decoded.TaskID, run.TaskID)
	}
	if *decoded.NodeID != *run.NodeID {
		t.Errorf("NodeID = %v, want %v", *decoded.NodeID, *run.NodeID)
	}
}

func TestEventJSONSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	event := &Event{
		ID:        1,
		RunID:     "run-123",
		Seq:       1,
		Type:      "message",
		Timestamp: now,
		Payload:   json.RawMessage(`{"content":"hello"}`),
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	var decoded Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if decoded.RunID != event.RunID {
		t.Errorf("RunID = %v, want %v", decoded.RunID, event.RunID)
	}
	if decoded.Type != event.Type {
		t.Errorf("Type = %v, want %v", decoded.Type, event.Type)
	}
}

func TestNodeStatus(t *testing.T) {
	// 测试所有节点状态常量
	// 节点生命周期：starting → online ⇄ unhealthy → draining → offline → terminated
	tests := []struct {
		status NodeStatus
		want   string
	}{
		{NodeStatusStarting, "starting"},       // 启动中
		{NodeStatusOnline, "online"},           // 在线
		{NodeStatusUnhealthy, "unhealthy"},     // 不健康
		{NodeStatusDraining, "draining"},       // 排空中
		{NodeStatusMaintenance, "maintenance"}, // 维护中
		{NodeStatusOffline, "offline"},         // 离线
		{NodeStatusTerminated, "terminated"},   // 已终止
		{NodeStatusUnknown, "unknown"},         // 未知
	}

	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("NodeStatus = %v, want %v", tt.status, tt.want)
		}
	}
}

func TestAuthTaskStatus(t *testing.T) {
	tests := []struct {
		status AuthTaskStatus
		want   string
	}{
		{AuthTaskStatusPending, "pending"},
		{AuthTaskStatusAssigned, "assigned"},
		{AuthTaskStatusRunning, "running"},
		{AuthTaskStatusWaitingUser, "waiting_user"},
		{AuthTaskStatusSuccess, "success"},
		{AuthTaskStatusFailed, "failed"},
		{AuthTaskStatusTimeout, "timeout"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("AuthTaskStatus = %v, want %v", tt.status, tt.want)
		}
	}
}

func TestAuthTaskToAuthSession(t *testing.T) {
	now := time.Now()
	oauthURL := "https://chat.qwen.ai/authorize?user_code=TEST-123"
	userCode := "TEST-123"
	message := "Please authenticate"
	terminalPort := 7681

	task := &AuthTask{
		ID:           "auth-task-123",
		AccountID:    "account-456",
		Method:       "oauth",
		NodeID:       "node-001",
		Status:       AuthTaskStatusWaitingUser,
		TerminalPort: &terminalPort,
		OAuthURL:     &oauthURL,
		UserCode:     &userCode,
		Message:      &message,
		CreatedAt:    now,
		UpdatedAt:    now,
		ExpiresAt:    now.Add(10 * time.Minute),
	}

	session := task.ToAuthSession()

	if session.ID != task.ID {
		t.Errorf("ID = %v, want %v", session.ID, task.ID)
	}
	if session.AccountID != task.AccountID {
		t.Errorf("AccountID = %v, want %v", session.AccountID, task.AccountID)
	}
	if session.Status != "waiting" {
		t.Errorf("Status = %v, want 'waiting'", session.Status)
	}
	if session.CallbackPort != terminalPort {
		t.Errorf("CallbackPort = %v, want %v", session.CallbackPort, terminalPort)
	}
	if session.VerifyURL != oauthURL {
		t.Errorf("VerifyURL = %v, want %v", session.VerifyURL, oauthURL)
	}
	if session.DeviceCode != userCode {
		t.Errorf("DeviceCode = %v, want %v", session.DeviceCode, userCode)
	}
	if session.Message != message {
		t.Errorf("Message = %v, want %v", session.Message, message)
	}
}

func TestAuthTaskToAuthSessionStatusMapping(t *testing.T) {
	tests := []struct {
		taskStatus    AuthTaskStatus
		sessionStatus string
	}{
		{AuthTaskStatusPending, "pending"},
		{AuthTaskStatusAssigned, "pending"},
		{AuthTaskStatusRunning, "waiting"},
		{AuthTaskStatusWaitingUser, "waiting"},
		{AuthTaskStatusSuccess, "success"},
		{AuthTaskStatusFailed, "failed"},
		{AuthTaskStatusTimeout, "failed"},
	}

	for _, tt := range tests {
		task := &AuthTask{
			ID:        "test",
			AccountID: "test",
			Status:    tt.taskStatus,
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}
		session := task.ToAuthSession()
		if session.Status != tt.sessionStatus {
			t.Errorf("AuthTaskStatus %v -> SessionStatus = %v, want %v",
				tt.taskStatus, session.Status, tt.sessionStatus)
		}
	}
}

func TestAuthTaskJSONSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	oauthURL := "https://example.com/oauth"
	userCode := "ABC-123"

	task := &AuthTask{
		ID:        "auth-123",
		AccountID: "account-456",
		Method:    "oauth",
		NodeID:    "node-001",
		Status:    AuthTaskStatusWaitingUser,
		OAuthURL:  &oauthURL,
		UserCode:  &userCode,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(10 * time.Minute),
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Failed to marshal AuthTask: %v", err)
	}

	var decoded AuthTask
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal AuthTask: %v", err)
	}

	if decoded.ID != task.ID {
		t.Errorf("ID = %v, want %v", decoded.ID, task.ID)
	}
	if *decoded.OAuthURL != *task.OAuthURL {
		t.Errorf("OAuthURL = %v, want %v", *decoded.OAuthURL, *task.OAuthURL)
	}
	if *decoded.UserCode != *task.UserCode {
		t.Errorf("UserCode = %v, want %v", *decoded.UserCode, *task.UserCode)
	}
}

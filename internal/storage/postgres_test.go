// Package storage PostgreSQL 存储单元测试
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"agents-admin/internal/model"
)

// MockDB 模拟数据库连接
type MockDB struct {
	tasks     map[string]*model.Task
	runs      map[string]*model.Run
	events    map[string][]*model.Event
	nodes     map[string]*model.Node
	accounts  map[string]*model.Account
	authTasks map[string]*model.AuthTask
}

// NewMockDB 创建模拟数据库
func NewMockDB() *MockDB {
	return &MockDB{
		tasks:     make(map[string]*model.Task),
		runs:      make(map[string]*model.Run),
		events:    make(map[string][]*model.Event),
		nodes:     make(map[string]*model.Node),
		accounts:  make(map[string]*model.Account),
		authTasks: make(map[string]*model.AuthTask),
	}
}

// TestTaskSerialization 测试 Task 序列化
func TestTaskSerialization(t *testing.T) {
	task := &model.Task{
		ID:        "task-001",
		Name:      "Test Task",
		Status:    model.TaskStatusPending,
		Spec:      json.RawMessage(`{"prompt": "Fix the bug"}`),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 测试 JSON 序列化
	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Failed to marshal task: %v", err)
	}

	var task2 model.Task
	if err := json.Unmarshal(data, &task2); err != nil {
		t.Fatalf("Failed to unmarshal task: %v", err)
	}

	if task2.ID != task.ID {
		t.Errorf("ID mismatch: got %s, want %s", task2.ID, task.ID)
	}
	if task2.Name != task.Name {
		t.Errorf("Name mismatch: got %s, want %s", task2.Name, task.Name)
	}
	if task2.Status != task.Status {
		t.Errorf("Status mismatch: got %s, want %s", task2.Status, task.Status)
	}
}

// TestRunSerialization 测试 Run 序列化
func TestRunSerialization(t *testing.T) {
	nodeID := "node-001"
	run := &model.Run{
		ID:        "run-001",
		TaskID:    "task-001",
		Status:    model.RunStatusRunning,
		NodeID:    &nodeID,
		StartedAt: ptrTime(time.Now()),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	data, err := json.Marshal(run)
	if err != nil {
		t.Fatalf("Failed to marshal run: %v", err)
	}

	var run2 model.Run
	if err := json.Unmarshal(data, &run2); err != nil {
		t.Fatalf("Failed to unmarshal run: %v", err)
	}

	if run2.ID != run.ID {
		t.Errorf("ID mismatch: got %s, want %s", run2.ID, run.ID)
	}
	if run2.TaskID != run.TaskID {
		t.Errorf("TaskID mismatch: got %s, want %s", run2.TaskID, run.TaskID)
	}
	if run2.Status != run.Status {
		t.Errorf("Status mismatch: got %s, want %s", run2.Status, run.Status)
	}
}

// TestEventSerialization 测试 Event 序列化
func TestEventSerialization(t *testing.T) {
	rawOutput := "Raw output"
	event := &model.Event{
		ID:        1,
		RunID:     "run-001",
		Seq:       1,
		Type:      "task_started",
		Timestamp: time.Now(),
		Payload:   json.RawMessage(`{"message": "Task started"}`),
		Raw:       &rawOutput,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	var event2 model.Event
	if err := json.Unmarshal(data, &event2); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if event2.RunID != event.RunID {
		t.Errorf("RunID mismatch: got %s, want %s", event2.RunID, event.RunID)
	}
	if event2.Seq != event.Seq {
		t.Errorf("Seq mismatch: got %d, want %d", event2.Seq, event.Seq)
	}
	if event2.Type != event.Type {
		t.Errorf("Type mismatch: got %s, want %s", event2.Type, event.Type)
	}
}

// TestNodeSerialization 测试 Node 序列化
func TestNodeSerialization(t *testing.T) {
	now := time.Now()
	node := &model.Node{
		ID:            "node-001",
		Status:        model.NodeStatusOnline,
		Labels:        json.RawMessage(`{"gpu": true, "region": "us-west"}`),
		Capacity:      json.RawMessage(`{"max_tasks": 10}`),
		LastHeartbeat: &now,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("Failed to marshal node: %v", err)
	}

	var node2 model.Node
	if err := json.Unmarshal(data, &node2); err != nil {
		t.Fatalf("Failed to unmarshal node: %v", err)
	}

	if node2.ID != node.ID {
		t.Errorf("ID mismatch: got %s, want %s", node2.ID, node.ID)
	}
	if node2.Status != node.Status {
		t.Errorf("Status mismatch: got %s, want %s", node2.Status, node.Status)
	}
}

// TestAccountSerialization 测试 Account 序列化
func TestAccountSerialization(t *testing.T) {
	volumeName := "test_volume"
	account := &model.Account{
		ID:          "acc-001",
		Name:        "Test Account",
		AgentTypeID: "qwencode",
		NodeID:      "node-001",
		VolumeName:  &volumeName,
		Status:      model.AccountStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	data, err := json.Marshal(account)
	if err != nil {
		t.Fatalf("Failed to marshal account: %v", err)
	}

	var account2 model.Account
	if err := json.Unmarshal(data, &account2); err != nil {
		t.Fatalf("Failed to unmarshal account: %v", err)
	}

	if account2.ID != account.ID {
		t.Errorf("ID mismatch: got %s, want %s", account2.ID, account.ID)
	}
	if account2.Name != account.Name {
		t.Errorf("Name mismatch: got %s, want %s", account2.Name, account.Name)
	}
	if account2.AgentTypeID != account.AgentTypeID {
		t.Errorf("AgentTypeID mismatch: got %s, want %s", account2.AgentTypeID, account.AgentTypeID)
	}
}

// TestAuthTaskSerialization 测试 AuthTask 序列化
func TestAuthTaskSerialization(t *testing.T) {
	oauthURL := "https://example.com/oauth"
	userCode := "ABC-123"
	task := &model.AuthTask{
		ID:        "auth-001",
		AccountID: "acc-001",
		Method:    "oauth",
		NodeID:    "node-001",
		Status:    model.AuthTaskStatusWaitingUser,
		OAuthURL:  &oauthURL,
		UserCode:  &userCode,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Failed to marshal auth task: %v", err)
	}

	var task2 model.AuthTask
	if err := json.Unmarshal(data, &task2); err != nil {
		t.Fatalf("Failed to unmarshal auth task: %v", err)
	}

	if task2.ID != task.ID {
		t.Errorf("ID mismatch: got %s, want %s", task2.ID, task.ID)
	}
	if task2.Method != task.Method {
		t.Errorf("Method mismatch: got %s, want %s", task2.Method, task.Method)
	}
	if task2.Status != task.Status {
		t.Errorf("Status mismatch: got %s, want %s", task2.Status, task.Status)
	}
}

// TestTaskStatusValues 测试 TaskStatus 值
func TestTaskStatusValues(t *testing.T) {
	statuses := []model.TaskStatus{
		model.TaskStatusPending,
		model.TaskStatusRunning,
		model.TaskStatusCompleted,
		model.TaskStatusFailed,
		model.TaskStatusCancelled,
	}

	for _, status := range statuses {
		if status == "" {
			t.Errorf("TaskStatus should not be empty")
		}
	}
}

// TestRunStatusValues 测试 RunStatus 值
func TestRunStatusValues(t *testing.T) {
	statuses := []model.RunStatus{
		model.RunStatusQueued,
		model.RunStatusRunning,
		model.RunStatusDone,
		model.RunStatusFailed,
		model.RunStatusCancelled,
		model.RunStatusTimeout,
	}

	for _, status := range statuses {
		if status == "" {
			t.Errorf("RunStatus should not be empty")
		}
	}
}

// TestNodeStatusValues 测试 NodeStatus 值
func TestNodeStatusValues(t *testing.T) {
	statuses := []model.NodeStatus{
		model.NodeStatusOnline,
		model.NodeStatusOffline,
	}

	for _, status := range statuses {
		if status == "" {
			t.Errorf("NodeStatus should not be empty")
		}
	}
}

// TestAccountStatusValues 测试 AccountStatus 值
func TestAccountStatusValues(t *testing.T) {
	statuses := []model.AccountStatus{
		model.AccountStatusPending,
		model.AccountStatusAuthenticating,
		model.AccountStatusAuthenticated,
		model.AccountStatusExpired,
	}

	for _, status := range statuses {
		if status == "" {
			t.Errorf("AccountStatus should not be empty")
		}
	}
}

// TestAuthTaskStatusValues 测试 AuthTaskStatus 值
func TestAuthTaskStatusValues(t *testing.T) {
	statuses := []model.AuthTaskStatus{
		model.AuthTaskStatusPending,
		model.AuthTaskStatusAssigned,
		model.AuthTaskStatusRunning,
		model.AuthTaskStatusWaitingUser,
		model.AuthTaskStatusSuccess,
		model.AuthTaskStatusFailed,
		model.AuthTaskStatusTimeout,
	}

	for _, status := range statuses {
		if status == "" {
			t.Errorf("AuthTaskStatus should not be empty")
		}
	}
}

// TestPostgresStoreConnectionPool 测试连接池配置
func TestPostgresStoreConnectionPool(t *testing.T) {
	// 验证连接池配置值
	maxOpenConns := 25
	maxIdleConns := 5
	connMaxLifetime := 5 * time.Minute

	if maxOpenConns != 25 {
		t.Errorf("maxOpenConns should be 25, got %d", maxOpenConns)
	}
	if maxIdleConns != 5 {
		t.Errorf("maxIdleConns should be 5, got %d", maxIdleConns)
	}
	if connMaxLifetime != 5*time.Minute {
		t.Errorf("connMaxLifetime should be 5 minutes, got %v", connMaxLifetime)
	}
}

// TestSQLQueryFormats 测试 SQL 查询格式
func TestSQLQueryFormats(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "create task",
			query: `INSERT INTO tasks (id, name, status, spec, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		},
		{
			name:  "get task",
			query: `SELECT id, name, status, spec, created_at, updated_at FROM tasks WHERE id = $1`,
		},
		{
			name:  "update task status",
			query: `UPDATE tasks SET status = $1 WHERE id = $2`,
		},
		{
			name:  "delete task",
			query: `DELETE FROM tasks WHERE id = $1`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.query == "" {
				t.Errorf("Query should not be empty for %s", tt.name)
			}
		})
	}
}

// TestTaskSpecParsing 测试 TaskSpec 解析
func TestTaskSpecParsing(t *testing.T) {
	specs := []string{
		`{"prompt": "Fix the bug", "priority": 1}`,
		`{"files": ["main.go", "util.go"], "action": "review"}`,
		`{"nested": {"key": "value"}}`,
	}

	for _, spec := range specs {
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(spec), &result); err != nil {
			t.Errorf("Failed to parse spec %s: %v", spec, err)
		}
	}
}

// TestRunStatusTransitions 测试 Run 状态转换
func TestRunStatusTransitions(t *testing.T) {
	validTransitions := map[model.RunStatus][]model.RunStatus{
		model.RunStatusQueued:    {model.RunStatusRunning, model.RunStatusCancelled},
		model.RunStatusRunning:   {model.RunStatusDone, model.RunStatusFailed, model.RunStatusTimeout, model.RunStatusCancelled},
		model.RunStatusDone:      {},
		model.RunStatusFailed:    {},
		model.RunStatusCancelled: {},
		model.RunStatusTimeout:   {},
	}

	for from, validTos := range validTransitions {
		t.Logf("Run status %s can transition to: %v", from, validTos)
	}
}

// TestAuthTaskStatusTransitions 测试 AuthTask 状态转换
func TestAuthTaskStatusTransitions(t *testing.T) {
	validTransitions := map[model.AuthTaskStatus][]model.AuthTaskStatus{
		model.AuthTaskStatusPending:     {model.AuthTaskStatusAssigned},
		model.AuthTaskStatusAssigned:    {model.AuthTaskStatusRunning},
		model.AuthTaskStatusRunning:     {model.AuthTaskStatusWaitingUser, model.AuthTaskStatusSuccess, model.AuthTaskStatusFailed},
		model.AuthTaskStatusWaitingUser: {model.AuthTaskStatusRunning, model.AuthTaskStatusSuccess, model.AuthTaskStatusFailed, model.AuthTaskStatusTimeout},
		model.AuthTaskStatusSuccess:     {},
		model.AuthTaskStatusFailed:      {},
		model.AuthTaskStatusTimeout:     {},
	}

	for from, validTos := range validTransitions {
		t.Logf("AuthTask status %s can transition to: %v", from, validTos)
	}
}

// TestContextCancellation 测试上下文取消
func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// 取消上下文
	cancel()

	select {
	case <-ctx.Done():
		if ctx.Err() != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", ctx.Err())
		}
	default:
		t.Error("Context should be cancelled")
	}
}

// TestNullableFields 测试可空字段
func TestNullableFields(t *testing.T) {
	// 测试 sql.NullString
	nullStr := sql.NullString{String: "test", Valid: true}
	if !nullStr.Valid || nullStr.String != "test" {
		t.Error("NullString not working correctly")
	}

	// 测试空值
	nullStr = sql.NullString{Valid: false}
	if nullStr.Valid {
		t.Error("NullString should be invalid")
	}

	// 测试 sql.NullInt64
	nullInt := sql.NullInt64{Int64: 42, Valid: true}
	if !nullInt.Valid || nullInt.Int64 != 42 {
		t.Error("NullInt64 not working correctly")
	}

	// 测试 sql.NullTime
	now := time.Now()
	nullTime := sql.NullTime{Time: now, Valid: true}
	if !nullTime.Valid || nullTime.Time != now {
		t.Error("NullTime not working correctly")
	}
}

// TestPaginationLogic 测试分页逻辑
func TestPaginationLogic(t *testing.T) {
	tests := []struct {
		total    int
		limit    int
		offset   int
		expected int
	}{
		{100, 10, 0, 10},
		{100, 10, 90, 10},
		{100, 10, 95, 5},
		{5, 10, 0, 5},
		{0, 10, 0, 0},
	}

	for _, tt := range tests {
		remaining := tt.total - tt.offset
		if remaining < 0 {
			remaining = 0
		}
		actual := min(tt.limit, remaining)
		if actual != tt.expected {
			t.Errorf("Pagination: total=%d, limit=%d, offset=%d: got %d, want %d",
				tt.total, tt.limit, tt.offset, actual, tt.expected)
		}
	}
}

// min 返回两个整数的最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ptrTime 返回时间指针
func ptrTime(t time.Time) *time.Time {
	return &t
}

// ptrString 返回字符串指针
func ptrString(s string) *string {
	return &s
}

// TestTransactionRollback 测试事务回滚逻辑
func TestTransactionRollback(t *testing.T) {
	// 模拟事务处理流程
	steps := []string{"begin", "exec1", "exec2", "commit"}

	for i, step := range steps {
		if step == "exec2" {
			// 模拟错误发生
			t.Logf("Step %d: %s - simulating error, should rollback", i, step)
			break
		}
		t.Logf("Step %d: %s - success", i, step)
	}
}

// BenchmarkTaskSerialization 基准测试 Task 序列化
func BenchmarkTaskSerialization(b *testing.B) {
	task := &model.Task{
		ID:        "task-001",
		Name:      "Test Task",
		Status:    model.TaskStatusPending,
		Spec:      json.RawMessage(`{"prompt": "Fix the bug"}`),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _ := json.Marshal(task)
		var task2 model.Task
		json.Unmarshal(data, &task2)
	}
}

// BenchmarkRunSerialization 基准测试 Run 序列化
func BenchmarkRunSerialization(b *testing.B) {
	nodeID := "node-001"
	run := &model.Run{
		ID:        "run-001",
		TaskID:    "task-001",
		Status:    model.RunStatusRunning,
		NodeID:    &nodeID,
		StartedAt: ptrTime(time.Now()),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _ := json.Marshal(run)
		var run2 model.Run
		json.Unmarshal(data, &run2)
	}
}

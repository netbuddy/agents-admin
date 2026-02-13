package mongostore

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"agents-admin/internal/shared/model"
	"agents-admin/internal/shared/storage"
)

// testStore 创建测试用 Store，使用独立数据库避免污染
func testStore(t *testing.T) *Store {
	t.Helper()

	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	dbName := "agents_admin_test"
	s, err := NewStore(uri, dbName)
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}

	// 清空测试数据库
	ctx := context.Background()
	if err := s.db.Drop(ctx); err != nil {
		t.Fatalf("Failed to drop test database: %v", err)
	}
	// 重新创建索引
	if err := s.ensureIndexes(ctx); err != nil {
		t.Fatalf("Failed to create indexes: %v", err)
	}

	t.Cleanup(func() {
		s.db.Drop(context.Background())
		s.Close()
	})

	return s
}

// Compile-time interface check
var _ storage.PersistentStore = (*Store)(nil)

func TestTaskCRUD(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	task := &model.Task{
		ID:          "task-001",
		Name:        "Test Task",
		Description: "A test task",
		Status:      model.TaskStatusPending,
		Type:        model.TaskTypeDevelopment,
		CreatedAt:   time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt:   time.Now().UTC().Truncate(time.Millisecond),
	}

	// Create
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// Duplicate insert
	if err := s.CreateTask(ctx, task); err == nil {
		t.Fatal("Expected duplicate error")
	}

	// Get
	got, err := s.GetTask(ctx, "task-001")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.Name != "Test Task" {
		t.Errorf("Name = %q, want %q", got.Name, "Test Task")
	}

	// Get not found
	_, err = s.GetTask(ctx, "nonexistent")
	if err != storage.ErrNotFound {
		t.Errorf("GetTask(nonexistent) error = %v, want ErrNotFound", err)
	}

	// Update status
	if err := s.UpdateTaskStatus(ctx, "task-001", model.TaskStatusInProgress); err != nil {
		t.Fatalf("UpdateTaskStatus: %v", err)
	}
	got, _ = s.GetTask(ctx, "task-001")
	if got.Status != model.TaskStatusInProgress {
		t.Errorf("Status = %q, want %q", got.Status, model.TaskStatusInProgress)
	}

	// List
	tasks, err := s.ListTasks(ctx, "", 10, 0)
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("ListTasks len = %d, want 1", len(tasks))
	}

	// List with filter
	tasks, err = s.ListTasks(ctx, "in_progress", 10, 0)
	if err != nil {
		t.Fatalf("ListTasks(in_progress): %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("ListTasks(in_progress) len = %d, want 1", len(tasks))
	}

	// Delete
	if err := s.DeleteTask(ctx, "task-001"); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}
	_, err = s.GetTask(ctx, "task-001")
	if err != storage.ErrNotFound {
		t.Errorf("After delete, GetTask error = %v, want ErrNotFound", err)
	}
}

func TestRunCRUD(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	run := &model.Run{
		ID:        "run-001",
		TaskID:    "task-001",
		Status:    model.RunStatusQueued,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Millisecond),
	}

	if err := s.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	got, err := s.GetRun(ctx, "run-001")
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.TaskID != "task-001" {
		t.Errorf("TaskID = %q, want %q", got.TaskID, "task-001")
	}

	// List by task
	runs, err := s.ListRunsByTask(ctx, "task-001")
	if err != nil {
		t.Fatalf("ListRunsByTask: %v", err)
	}
	if len(runs) != 1 {
		t.Errorf("ListRunsByTask len = %d, want 1", len(runs))
	}

	// Update status
	nodeID := "node-1"
	if err := s.UpdateRunStatus(ctx, "run-001", model.RunStatusRunning, &nodeID); err != nil {
		t.Fatalf("UpdateRunStatus: %v", err)
	}
	got, _ = s.GetRun(ctx, "run-001")
	if got.Status != model.RunStatusRunning {
		t.Errorf("Status = %q, want %q", got.Status, model.RunStatusRunning)
	}
	if got.StartedAt == nil {
		t.Error("StartedAt should be set when status is running")
	}

	// Delete
	if err := s.DeleteRun(ctx, "run-001"); err != nil {
		t.Fatalf("DeleteRun: %v", err)
	}
}

func TestEventCRUD(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	events := []*model.Event{
		{RunID: "run-001", Seq: 1, Type: "message", Timestamp: time.Now().UTC()},
		{RunID: "run-001", Seq: 2, Type: "tool_use", Timestamp: time.Now().UTC()},
		{RunID: "run-001", Seq: 3, Type: "message", Timestamp: time.Now().UTC()},
	}

	if err := s.CreateEvents(ctx, events); err != nil {
		t.Fatalf("CreateEvents: %v", err)
	}

	count, err := s.CountEventsByRun(ctx, "run-001")
	if err != nil {
		t.Fatalf("CountEventsByRun: %v", err)
	}
	if count != 3 {
		t.Errorf("Count = %d, want 3", count)
	}

	// Get from seq 2
	got, err := s.GetEventsByRun(ctx, "run-001", 2, 10)
	if err != nil {
		t.Fatalf("GetEventsByRun: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("GetEventsByRun(fromSeq=2) len = %d, want 2", len(got))
	}
}

func TestNodeCRUD(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	node := &model.Node{
		ID:        "node-001",
		Status:    model.NodeStatusOnline,
		Hostname:  "test-host",
		IPs:       "192.168.1.1",
		Labels:    json.RawMessage(`{"os":"linux"}`),
		Capacity:  json.RawMessage(`{"max_concurrent":4}`),
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Millisecond),
	}

	if err := s.UpsertNode(ctx, node); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}

	got, err := s.GetNode(ctx, "node-001")
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if got.Hostname != "test-host" {
		t.Errorf("Hostname = %q, want %q", got.Hostname, "test-host")
	}

	// Upsert again (update)
	node.Hostname = "updated-host"
	if err := s.UpsertNode(ctx, node); err != nil {
		t.Fatalf("UpsertNode(update): %v", err)
	}
	got, _ = s.GetNode(ctx, "node-001")
	if got.Hostname != "updated-host" {
		t.Errorf("After upsert, Hostname = %q, want %q", got.Hostname, "updated-host")
	}

	// List
	nodes, err := s.ListAllNodes(ctx)
	if err != nil {
		t.Fatalf("ListAllNodes: %v", err)
	}
	if len(nodes) != 1 {
		t.Errorf("ListAllNodes len = %d, want 1", len(nodes))
	}

	// Delete
	if err := s.DeleteNode(ctx, "node-001"); err != nil {
		t.Fatalf("DeleteNode: %v", err)
	}
}

func TestUserCRUD(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	user := &model.User{
		ID:           "user-001",
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "hashed-password",
		Role:         "admin",
		Status:       "active",
		CreatedAt:    time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt:    time.Now().UTC().Truncate(time.Millisecond),
	}

	if err := s.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// Get by email
	got, err := s.GetUserByEmail(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if got.Username != "testuser" {
		t.Errorf("Username = %q, want %q", got.Username, "testuser")
	}

	// Get by ID
	got, err = s.GetUserByID(ctx, "user-001")
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "test@example.com")
	}

	// Update password
	if err := s.UpdateUserPassword(ctx, "user-001", "new-hash"); err != nil {
		t.Fatalf("UpdateUserPassword: %v", err)
	}
	got, _ = s.GetUserByID(ctx, "user-001")
	if got.PasswordHash != "new-hash" {
		t.Errorf("PasswordHash = %q, want %q", got.PasswordHash, "new-hash")
	}

	// Duplicate email
	user2 := &model.User{
		ID:        "user-002",
		Email:     "test@example.com",
		Username:  "duplicate",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := s.CreateUser(ctx, user2); err != storage.ErrDuplicate {
		t.Errorf("Duplicate email error = %v, want ErrDuplicate", err)
	}
}

func TestProxyCRUD(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	proxy := &model.Proxy{
		ID:        "proxy-001",
		Name:      "Test Proxy",
		Type:      "http",
		Host:      "proxy.example.com",
		Port:      8080,
		IsDefault: false,
		Status:    "active",
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Millisecond),
	}

	if err := s.CreateProxy(ctx, proxy); err != nil {
		t.Fatalf("CreateProxy: %v", err)
	}

	// Set default
	if err := s.SetDefaultProxy(ctx, "proxy-001"); err != nil {
		t.Fatalf("SetDefaultProxy: %v", err)
	}
	defaultProxy, err := s.GetDefaultProxy(ctx)
	if err != nil {
		t.Fatalf("GetDefaultProxy: %v", err)
	}
	if defaultProxy.ID != "proxy-001" {
		t.Errorf("Default proxy ID = %q, want %q", defaultProxy.ID, "proxy-001")
	}

	// Clear default
	if err := s.ClearDefaultProxy(ctx); err != nil {
		t.Fatalf("ClearDefaultProxy: %v", err)
	}
	noDefault, err := s.GetDefaultProxy(ctx)
	if err != nil {
		t.Errorf("After clear, GetDefaultProxy error = %v, want nil", err)
	}
	if noDefault != nil {
		t.Errorf("After clear, GetDefaultProxy = %+v, want nil", noDefault)
	}
}

func TestGetNotFound_ReturnsNilNil(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// 契约：Get* 不存在时必须返回 (nil, nil)，不能返回 error
	// 这与 SQL 实现的 sql.ErrNoRows → (nil, nil) 行为一致
	// 违反此契约会导致 HandleAuthSuccess 无法创建账号

	acc, err := s.GetAccount(ctx, "nonexistent")
	if err != nil {
		t.Errorf("GetAccount(nonexistent): want (nil, nil), got err=%v", err)
	}
	if acc != nil {
		t.Errorf("GetAccount(nonexistent): want nil, got %+v", acc)
	}

	node, err := s.GetNode(ctx, "nonexistent")
	if err != nil {
		t.Errorf("GetNode(nonexistent): want (nil, nil), got err=%v", err)
	}
	if node != nil {
		t.Errorf("GetNode(nonexistent): want nil, got %+v", node)
	}

	action, err := s.GetAction(ctx, "nonexistent")
	if err != nil {
		t.Errorf("GetAction(nonexistent): want (nil, nil), got err=%v", err)
	}
	if action != nil {
		t.Errorf("GetAction(nonexistent): want nil, got %+v", action)
	}

	op, err := s.GetOperation(ctx, "nonexistent")
	if err != nil {
		t.Errorf("GetOperation(nonexistent): want (nil, nil), got err=%v", err)
	}
	if op != nil {
		t.Errorf("GetOperation(nonexistent): want nil, got %+v", op)
	}

	actionWithOp, err := s.GetActionWithOperation(ctx, "nonexistent")
	if err != nil {
		t.Errorf("GetActionWithOperation(nonexistent): want (nil, nil), got err=%v", err)
	}
	if actionWithOp != nil {
		t.Errorf("GetActionWithOperation(nonexistent): want nil, got %+v", actionWithOp)
	}
}

func TestAccountAndAuthTask(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	account := &model.Account{
		ID:          "acc-001",
		Name:        "test@example.com",
		AgentTypeID: "qwen-code",
		Status:      model.AccountStatusPending,
		CreatedAt:   time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt:   time.Now().UTC().Truncate(time.Millisecond),
	}

	if err := s.CreateAccount(ctx, account); err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	got, err := s.GetAccount(ctx, "acc-001")
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got.Name != "test@example.com" {
		t.Errorf("Name = %q, want %q", got.Name, "test@example.com")
	}

	// Auth task
	authTask := &model.AuthTask{
		ID:        "auth-001",
		AccountID: "acc-001",
		Method:    "oauth",
		NodeID:    "node-001",
		Status:    model.AuthTaskStatusPending,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Millisecond),
		ExpiresAt: time.Now().Add(10 * time.Minute).UTC().Truncate(time.Millisecond),
	}

	if err := s.CreateAuthTask(ctx, authTask); err != nil {
		t.Fatalf("CreateAuthTask: %v", err)
	}

	gotAuth, err := s.GetAuthTaskByAccountID(ctx, "acc-001")
	if err != nil {
		t.Fatalf("GetAuthTaskByAccountID: %v", err)
	}
	if gotAuth.Method != "oauth" {
		t.Errorf("Method = %q, want %q", gotAuth.Method, "oauth")
	}
}

func TestHITL(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	expiresAt := time.Now().Add(10 * time.Minute).UTC().Truncate(time.Millisecond)
	req := &model.ApprovalRequest{
		ID:        "apr-001",
		RunID:     "run-001",
		Type:      "file_delete",
		Status:    model.ApprovalStatusPending,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		ExpiresAt: &expiresAt,
	}

	if err := s.CreateApprovalRequest(ctx, req); err != nil {
		t.Fatalf("CreateApprovalRequest: %v", err)
	}

	got, err := s.GetApprovalRequest(ctx, "apr-001")
	if err != nil {
		t.Fatalf("GetApprovalRequest: %v", err)
	}
	if got.Type != "file_delete" {
		t.Errorf("Type = %q, want %q", got.Type, "file_delete")
	}

	// List by status
	reqs, err := s.ListApprovalRequests(ctx, "run-001", "pending")
	if err != nil {
		t.Fatalf("ListApprovalRequests: %v", err)
	}
	if len(reqs) != 1 {
		t.Errorf("len = %d, want 1", len(reqs))
	}
}

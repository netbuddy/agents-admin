// Package repository SQLite 集成测试
//
// 使用 SQLite 内存数据库验证 repository 层所有存储接口的正确性。
// 无需外部数据库依赖，可在任何环境下运行。
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"agents-admin/internal/shared/model"
	"agents-admin/internal/shared/storage/dbutil"
	sqlitedriver "agents-admin/internal/shared/storage/driver/sqlite"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestStore 创建用于测试的 SQLite 内存数据库 Store
func newTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := sqlitedriver.Open(":memory:")
	require.NoError(t, err)
	dialect := sqlitedriver.NewDialect()
	require.NoError(t, dialect.AutoMigrate(db))
	store := NewStore(db, dialect)
	t.Cleanup(func() { store.Close() })
	return store
}

// ============================================================================
// Dialect 基础测试
// ============================================================================

func TestDialectTypes(t *testing.T) {
	d := sqlitedriver.NewDialect()
	assert.Equal(t, dbutil.DriverSQLite, d.DriverType())
	assert.Equal(t, "datetime('now')", d.CurrentTimestamp())
	assert.Equal(t, "1", d.BooleanLiteral(true))
	assert.Equal(t, "0", d.BooleanLiteral(false))
	assert.False(t, d.SupportsNullsLast())
	assert.True(t, d.SupportsRecursiveCTE())
}

func TestRebind(t *testing.T) {
	d := sqlitedriver.NewDialect()
	assert.Equal(t, "SELECT * FROM t WHERE id = ? AND name = ?",
		d.Rebind("SELECT * FROM t WHERE id = $1 AND name = $2"))
	// 应去除 PG 类型转换
	assert.Equal(t, "UPDATE t SET status = ? WHERE id = ?",
		d.Rebind("UPDATE t SET status = $1::varchar WHERE id = $2"))
}

// ============================================================================
// Task 测试
// ============================================================================

func TestTaskCRUD(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	task := &model.Task{
		ID:        "task-001",
		Name:      "Test Task",
		Status:    model.TaskStatusPending,
		Type:      "general",
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Create
	require.NoError(t, s.CreateTask(ctx, task))

	// Get
	got, err := s.GetTask(ctx, task.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, task.ID, got.ID)
	assert.Equal(t, task.Name, got.Name)
	assert.Equal(t, model.TaskStatusPending, got.Status)

	// List
	tasks, err := s.ListTasks(ctx, "", 10, 0)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)

	// List with status filter
	tasks, err = s.ListTasks(ctx, string(model.TaskStatusPending), 10, 0)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)

	tasks, err = s.ListTasks(ctx, "completed", 10, 0)
	require.NoError(t, err)
	assert.Len(t, tasks, 0)

	// Update status
	require.NoError(t, s.UpdateTaskStatus(ctx, task.ID, model.TaskStatusInProgress))
	got, _ = s.GetTask(ctx, task.ID)
	assert.Equal(t, model.TaskStatusInProgress, got.Status)

	// Update context
	taskCtx := json.RawMessage(`{"key":"value"}`)
	require.NoError(t, s.UpdateTaskContext(ctx, task.ID, taskCtx))

	// Get not found
	got, err = s.GetTask(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)

	// Delete
	require.NoError(t, s.DeleteTask(ctx, task.ID))
	got, _ = s.GetTask(ctx, task.ID)
	assert.Nil(t, got)
}

func TestTaskTree(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	parent := &model.Task{ID: "root", Name: "Root", Status: model.TaskStatusPending, Type: "general", CreatedAt: now, UpdatedAt: now}
	child1 := &model.Task{ID: "child-1", ParentID: strPtr("root"), Name: "Child 1", Status: model.TaskStatusPending, Type: "general", CreatedAt: now.Add(time.Second), UpdatedAt: now}
	child2 := &model.Task{ID: "child-2", ParentID: strPtr("root"), Name: "Child 2", Status: model.TaskStatusPending, Type: "general", CreatedAt: now.Add(2 * time.Second), UpdatedAt: now}

	require.NoError(t, s.CreateTask(ctx, parent))
	require.NoError(t, s.CreateTask(ctx, child1))
	require.NoError(t, s.CreateTask(ctx, child2))

	// ListSubTasks
	subs, err := s.ListSubTasks(ctx, "root")
	require.NoError(t, err)
	assert.Len(t, subs, 2)

	// GetTaskTree (recursive CTE)
	tree, err := s.GetTaskTree(ctx, "root")
	require.NoError(t, err)
	assert.Len(t, tree, 3)
	assert.Equal(t, "root", tree[0].ID)
}

// ============================================================================
// Run 测试
// ============================================================================

func TestRunCRUD(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// 先创建 task
	task := &model.Task{ID: "task-r1", Name: "T", Status: model.TaskStatusPending, Type: "general", CreatedAt: now, UpdatedAt: now}
	require.NoError(t, s.CreateTask(ctx, task))

	run := &model.Run{
		ID:        "run-001",
		TaskID:    "task-r1",
		Status:    model.RunStatusQueued,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Create
	require.NoError(t, s.CreateRun(ctx, run))

	// Get
	got, err := s.GetRun(ctx, run.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, model.RunStatusQueued, got.Status)

	// ListRunsByTask
	runs, err := s.ListRunsByTask(ctx, "task-r1")
	require.NoError(t, err)
	assert.Len(t, runs, 1)

	// ListQueuedRuns
	runs, err = s.ListQueuedRuns(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, runs, 1)

	// ListRunningRuns (should be empty)
	runs, err = s.ListRunningRuns(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, runs, 0)

	// UpdateRunStatus -> assigned
	nodeID := "node-1"
	require.NoError(t, s.UpdateRunStatus(ctx, run.ID, model.RunStatusAssigned, &nodeID))
	got, _ = s.GetRun(ctx, run.ID)
	assert.Equal(t, model.RunStatusAssigned, got.Status)

	// ListRunsByNode
	runs, err = s.ListRunsByNode(ctx, "node-1")
	require.NoError(t, err)
	assert.Len(t, runs, 1)

	// UpdateRunStatus -> running
	require.NoError(t, s.UpdateRunStatus(ctx, run.ID, model.RunStatusRunning, nil))

	// UpdateRunStatus -> done
	require.NoError(t, s.UpdateRunStatus(ctx, run.ID, model.RunStatusDone, nil))
	got, _ = s.GetRun(ctx, run.ID)
	assert.Equal(t, model.RunStatusDone, got.Status)

	// UpdateRunError
	run2 := &model.Run{ID: "run-002", TaskID: "task-r1", Status: model.RunStatusRunning, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, s.CreateRun(ctx, run2))
	require.NoError(t, s.UpdateRunError(ctx, "run-002", "something failed"))
	got, _ = s.GetRun(ctx, "run-002")
	assert.Equal(t, model.RunStatusFailed, got.Status)

	// Delete
	require.NoError(t, s.DeleteRun(ctx, run.ID))
	got, _ = s.GetRun(ctx, run.ID)
	assert.Nil(t, got)
}

// ============================================================================
// Event 测试
// ============================================================================

func TestEventCRUD(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	task := &model.Task{ID: "task-e1", Name: "T", Status: model.TaskStatusPending, Type: "general", CreatedAt: now, UpdatedAt: now}
	require.NoError(t, s.CreateTask(ctx, task))
	run := &model.Run{ID: "run-e1", TaskID: "task-e1", Status: model.RunStatusRunning, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, s.CreateRun(ctx, run))

	events := []*model.Event{
		{RunID: "run-e1", Seq: 1, Type: "action", Timestamp: now},
		{RunID: "run-e1", Seq: 2, Type: "observation", Timestamp: now},
	}
	require.NoError(t, s.CreateEvents(ctx, events))

	cnt, err := s.CountEventsByRun(ctx, "run-e1")
	require.NoError(t, err)
	assert.Equal(t, 2, cnt)

	evts, err := s.GetEventsByRun(ctx, "run-e1", 0, 10)
	require.NoError(t, err)
	assert.Len(t, evts, 2)

	evts, err = s.GetEventsByRun(ctx, "run-e1", 1, 10)
	require.NoError(t, err)
	assert.Len(t, evts, 1)
}

// ============================================================================
// Node 测试
// ============================================================================

func TestNodeCRUD(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	node := &model.Node{
		ID:            "node-001",
		Status:        "online",
		Labels:        json.RawMessage(`{"gpu":"true"}`),
		Capacity:      json.RawMessage(`{"slots":4}`),
		LastHeartbeat: &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Upsert
	require.NoError(t, s.UpsertNode(ctx, node))
	got, err := s.GetNode(ctx, "node-001")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, model.NodeStatus("online"), got.Status)

	// UpsertNodeHeartbeat (should not overwrite status)
	node.Status = "draining"
	require.NoError(t, s.UpsertNodeHeartbeat(ctx, node))
	got, _ = s.GetNode(ctx, "node-001")
	// 心跳 upsert 不应覆盖 status（ON CONFLICT 时不更新 status）
	assert.Equal(t, model.NodeStatus("online"), got.Status)

	// List
	nodes, err := s.ListAllNodes(ctx)
	require.NoError(t, err)
	assert.Len(t, nodes, 1)

	onlineNodes, err := s.ListOnlineNodes(ctx)
	require.NoError(t, err)
	assert.Len(t, onlineNodes, 1)

	// Delete
	require.NoError(t, s.DeleteNode(ctx, "node-001"))
	got, _ = s.GetNode(ctx, "node-001")
	assert.Nil(t, got)
}

// ============================================================================
// Account + AuthTask 测试
// ============================================================================

func TestAccountCRUD(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	account := &model.Account{
		ID:        "acc-001",
		Name:      "Test Account",
		Status:    model.AccountStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	require.NoError(t, s.CreateAccount(ctx, account))

	got, err := s.GetAccount(ctx, "acc-001")
	require.NoError(t, err)
	assert.Equal(t, "Test Account", got.Name)

	accounts, err := s.ListAccounts(ctx)
	require.NoError(t, err)
	assert.Len(t, accounts, 1)

	require.NoError(t, s.UpdateAccountStatus(ctx, "acc-001", model.AccountStatusAuthenticated))
	got, _ = s.GetAccount(ctx, "acc-001")
	assert.Equal(t, model.AccountStatusAuthenticated, got.Status)

	require.NoError(t, s.UpdateAccountVolume(ctx, "acc-001", "vol-123"))
	got, _ = s.GetAccount(ctx, "acc-001")
	assert.Equal(t, "vol-123", ptrStr(got.VolumeName))

	require.NoError(t, s.DeleteAccount(ctx, "acc-001"))
	got, _ = s.GetAccount(ctx, "acc-001")
	assert.Nil(t, got)
}

func TestAuthTaskCRUD(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	account := &model.Account{ID: "acc-auth", Name: "Auth Account", Status: model.AccountStatusPending, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, s.CreateAccount(ctx, account))

	authTask := &model.AuthTask{
		ID:        "auth-001",
		AccountID: "acc-auth",
		Method:    "browser",
		Status:    "pending",
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(time.Hour),
	}

	require.NoError(t, s.CreateAuthTask(ctx, authTask))

	got, err := s.GetAuthTask(ctx, "auth-001")
	require.NoError(t, err)
	assert.Equal(t, "browser", got.Method)

	got, err = s.GetAuthTaskByAccountID(ctx, "acc-auth")
	require.NoError(t, err)
	assert.Equal(t, "auth-001", got.ID)

	tasks, err := s.ListRecentAuthTasks(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
}

// ============================================================================
// Operation + Action 测试
// ============================================================================

func TestOperationCRUD(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	op := &model.Operation{
		ID:        "op-001",
		Type:      "deploy",
		Status:    model.OperationStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	require.NoError(t, s.CreateOperation(ctx, op))

	got, err := s.GetOperation(ctx, "op-001")
	require.NoError(t, err)
	assert.Equal(t, model.OperationType("deploy"), got.Type)

	ops, err := s.ListOperations(ctx, "deploy", "", 10, 0)
	require.NoError(t, err)
	assert.Len(t, ops, 1)

	require.NoError(t, s.UpdateOperationStatus(ctx, "op-001", model.OperationStatusCompleted))
	got, _ = s.GetOperation(ctx, "op-001")
	assert.Equal(t, model.OperationStatusCompleted, got.Status)
}

func TestActionCRUD(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	op := &model.Operation{ID: "op-act", Type: "deploy", Status: model.OperationStatusInProgress, NodeID: "node-1", CreatedAt: now, UpdatedAt: now}
	require.NoError(t, s.CreateOperation(ctx, op))

	action := &model.Action{
		ID:          "act-001",
		OperationID: "op-act",
		Status:      model.ActionStatusAssigned,
		CreatedAt:   now,
	}
	require.NoError(t, s.CreateAction(ctx, action))

	got, err := s.GetAction(ctx, "act-001")
	require.NoError(t, err)
	assert.Equal(t, model.ActionStatusAssigned, got.Status)

	// GetActionWithOperation
	got, err = s.GetActionWithOperation(ctx, "act-001")
	require.NoError(t, err)
	require.NotNil(t, got.Operation)
	assert.Equal(t, model.OperationType("deploy"), got.Operation.Type)

	// ListActionsByOperation
	actions, err := s.ListActionsByOperation(ctx, "op-act")
	require.NoError(t, err)
	assert.Len(t, actions, 1)

	// ListActionsByNode
	actions, err = s.ListActionsByNode(ctx, "node-1", "")
	require.NoError(t, err)
	assert.Len(t, actions, 1)

	// UpdateActionStatus
	require.NoError(t, s.UpdateActionStatus(ctx, "act-001", model.ActionStatusRunning, "executing", "Running", 50, nil, ""))
	got, _ = s.GetAction(ctx, "act-001")
	assert.Equal(t, model.ActionStatusRunning, got.Status)
}

// ============================================================================
// Proxy 测试
// ============================================================================

func TestProxyCRUD(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	proxy := &model.Proxy{
		ID:        "proxy-001",
		Name:      "Test Proxy",
		Type:      "http",
		Host:      "127.0.0.1",
		Port:      8080,
		IsDefault: false,
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}

	require.NoError(t, s.CreateProxy(ctx, proxy))

	got, err := s.GetProxy(ctx, "proxy-001")
	require.NoError(t, err)
	assert.Equal(t, "Test Proxy", got.Name)

	proxies, err := s.ListProxies(ctx)
	require.NoError(t, err)
	assert.Len(t, proxies, 1)

	// SetDefault
	require.NoError(t, s.SetDefaultProxy(ctx, "proxy-001"))
	def, err := s.GetDefaultProxy(ctx)
	require.NoError(t, err)
	require.NotNil(t, def)
	assert.Equal(t, "proxy-001", def.ID)

	// ClearDefault
	require.NoError(t, s.ClearDefaultProxy(ctx))
	def, _ = s.GetDefaultProxy(ctx)
	assert.Nil(t, def)

	// Update
	proxy.Name = "Updated Proxy"
	require.NoError(t, s.UpdateProxy(ctx, proxy))
	got, _ = s.GetProxy(ctx, "proxy-001")
	assert.Equal(t, "Updated Proxy", got.Name)

	// Delete
	require.NoError(t, s.DeleteProxy(ctx, "proxy-001"))
	got, _ = s.GetProxy(ctx, "proxy-001")
	assert.Nil(t, got)
}

// ============================================================================
// Instance 测试
// ============================================================================

func TestInstanceCRUD(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	inst := &model.Instance{
		ID:        "inst-001",
		Name:      "Test Instance",
		NodeID:    strPtr("node-1"),
		Status:    model.InstanceStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	require.NoError(t, s.CreateInstance(ctx, inst))

	got, err := s.GetInstance(ctx, "inst-001")
	require.NoError(t, err)
	assert.Equal(t, "Test Instance", got.Name)

	insts, err := s.ListInstances(ctx)
	require.NoError(t, err)
	assert.Len(t, insts, 1)

	insts, err = s.ListInstancesByNode(ctx, "node-1")
	require.NoError(t, err)
	assert.Len(t, insts, 1)

	insts, err = s.ListPendingInstances(ctx, "node-1")
	require.NoError(t, err)
	assert.Len(t, insts, 1)

	// Update
	cn := "container-abc"
	require.NoError(t, s.UpdateInstance(ctx, "inst-001", model.InstanceStatusRunning, &cn))
	got, _ = s.GetInstance(ctx, "inst-001")
	assert.Equal(t, model.InstanceStatusRunning, got.Status)

	// Update not found
	err = s.UpdateInstance(ctx, "nonexistent", model.InstanceStatusRunning, nil)
	assert.Equal(t, sql.ErrNoRows, err)

	// Delete
	require.NoError(t, s.DeleteInstance(ctx, "inst-001"))
}

// ============================================================================
// TerminalSession 测试
// ============================================================================

func TestTerminalSessionCRUD(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	session := &model.TerminalSession{
		ID:        "term-001",
		NodeID:    strPtr("node-1"),
		Status:    model.TerminalStatusPending,
		CreatedAt: now,
		ExpiresAt: timePtr(now.Add(time.Hour)),
	}

	require.NoError(t, s.CreateTerminalSession(ctx, session))

	got, err := s.GetTerminalSession(ctx, "term-001")
	require.NoError(t, err)
	require.NotNil(t, got)

	sessions, err := s.ListTerminalSessions(ctx)
	require.NoError(t, err)
	assert.Len(t, sessions, 1)

	sessions, err = s.ListTerminalSessionsByNode(ctx, "node-1")
	require.NoError(t, err)
	assert.Len(t, sessions, 1)

	sessions, err = s.ListPendingTerminalSessions(ctx, "node-1")
	require.NoError(t, err)
	assert.Len(t, sessions, 1)

	// Update
	port := 8080
	url := "http://localhost:8080"
	require.NoError(t, s.UpdateTerminalSession(ctx, "term-001", model.TerminalStatusRunning, &port, &url))

	// Delete
	require.NoError(t, s.DeleteTerminalSession(ctx, "term-001"))
}

// ============================================================================
// HITL 测试
// ============================================================================

func TestHITLApproval(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// Setup
	task := &model.Task{ID: "task-h1", Name: "T", Status: model.TaskStatusPending, Type: "general", CreatedAt: now, UpdatedAt: now}
	require.NoError(t, s.CreateTask(ctx, task))
	run := &model.Run{ID: "run-h1", TaskID: "task-h1", Status: model.RunStatusRunning, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, s.CreateRun(ctx, run))

	// ApprovalRequest
	req := &model.ApprovalRequest{
		ID:        "apr-001",
		RunID:     "run-h1",
		Type:      "dangerous_operation",
		Status:    model.ApprovalStatusPending,
		Operation: "delete /important",
		CreatedAt: now,
	}
	require.NoError(t, s.CreateApprovalRequest(ctx, req))

	got, err := s.GetApprovalRequest(ctx, "apr-001")
	require.NoError(t, err)
	assert.Equal(t, model.ApprovalStatusPending, got.Status)

	reqs, err := s.ListApprovalRequests(ctx, "run-h1", "")
	require.NoError(t, err)
	assert.Len(t, reqs, 1)

	reqs, err = s.ListApprovalRequests(ctx, "run-h1", string(model.ApprovalStatusPending))
	require.NoError(t, err)
	assert.Len(t, reqs, 1)

	require.NoError(t, s.UpdateApprovalRequestStatus(ctx, "apr-001", model.ApprovalStatusApproved))

	// ApprovalDecision
	decision := &model.ApprovalDecision{
		ID:        "dec-001",
		RequestID: "apr-001",
		Decision:  "approve",
		DecidedBy: "admin",
		CreatedAt: now,
	}
	require.NoError(t, s.CreateApprovalDecision(ctx, decision))
}

func TestHITLFeedback(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	task := &model.Task{ID: "task-fb", Name: "T", Status: model.TaskStatusPending, Type: "general", CreatedAt: now, UpdatedAt: now}
	require.NoError(t, s.CreateTask(ctx, task))
	run := &model.Run{ID: "run-fb", TaskID: "task-fb", Status: model.RunStatusRunning, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, s.CreateRun(ctx, run))

	fb := &model.HumanFeedback{
		ID:        "fb-001",
		RunID:     "run-fb",
		Type:      "guidance",
		Content:   "Please focus on X",
		CreatedBy: "admin",
		CreatedAt: now,
	}
	require.NoError(t, s.CreateFeedback(ctx, fb))

	fbs, err := s.ListFeedbacks(ctx, "run-fb")
	require.NoError(t, err)
	assert.Len(t, fbs, 1)

	require.NoError(t, s.MarkFeedbackProcessed(ctx, "fb-001"))
}

func TestHITLIntervention(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	task := &model.Task{ID: "task-iv", Name: "T", Status: model.TaskStatusPending, Type: "general", CreatedAt: now, UpdatedAt: now}
	require.NoError(t, s.CreateTask(ctx, task))
	run := &model.Run{ID: "run-iv", TaskID: "task-iv", Status: model.RunStatusRunning, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, s.CreateRun(ctx, run))

	iv := &model.Intervention{
		ID:        "iv-001",
		RunID:     "run-iv",
		Action:    "pause",
		Reason:    "need review",
		CreatedBy: "admin",
		CreatedAt: now,
	}
	require.NoError(t, s.CreateIntervention(ctx, iv))

	ivs, err := s.ListInterventions(ctx, "run-iv")
	require.NoError(t, err)
	assert.Len(t, ivs, 1)

	require.NoError(t, s.UpdateInterventionExecuted(ctx, "iv-001"))
}

func TestHITLConfirmation(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	task := &model.Task{ID: "task-cf", Name: "T", Status: model.TaskStatusPending, Type: "general", CreatedAt: now, UpdatedAt: now}
	require.NoError(t, s.CreateTask(ctx, task))
	run := &model.Run{ID: "run-cf", TaskID: "task-cf", Status: model.RunStatusRunning, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, s.CreateRun(ctx, run))

	cf := &model.Confirmation{
		ID:        "cf-001",
		RunID:     "run-cf",
		Type:      "deployment",
		Message:   "Deploy to production?",
		Status:    model.ConfirmStatusPending,
		Options:   []string{"yes", "no"},
		CreatedAt: now,
	}
	require.NoError(t, s.CreateConfirmation(ctx, cf))

	got, err := s.GetConfirmation(ctx, "cf-001")
	require.NoError(t, err)
	assert.Equal(t, "Deploy to production?", got.Message)
	assert.Len(t, got.Options, 2)

	cfs, err := s.ListConfirmations(ctx, "run-cf", "")
	require.NoError(t, err)
	assert.Len(t, cfs, 1)

	opt := "yes"
	require.NoError(t, s.UpdateConfirmationStatus(ctx, "cf-001", model.ConfirmStatusConfirmed, &opt))
	got, _ = s.GetConfirmation(ctx, "cf-001")
	assert.Equal(t, model.ConfirmStatusConfirmed, got.Status)
}

// ============================================================================
// Template 测试
// ============================================================================

func TestTaskTemplateCRUD(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	tmpl := &model.TaskTemplate{
		ID:          "tt-001",
		Name:        "Test Template",
		Type:        "general",
		Description: "A test template",
		Category:    "test",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	require.NoError(t, s.CreateTaskTemplate(ctx, tmpl))

	got, err := s.GetTaskTemplate(ctx, "tt-001")
	require.NoError(t, err)
	assert.Equal(t, "Test Template", got.Name)

	tmpls, err := s.ListTaskTemplates(ctx, "")
	require.NoError(t, err)
	assert.Len(t, tmpls, 1)

	tmpls, err = s.ListTaskTemplates(ctx, "test")
	require.NoError(t, err)
	assert.Len(t, tmpls, 1)

	require.NoError(t, s.DeleteTaskTemplate(ctx, "tt-001"))
	got, _ = s.GetTaskTemplate(ctx, "tt-001")
	assert.Nil(t, got)
}

func TestAgentTemplateCRUD(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	tmpl := &model.AgentTemplate{
		ID:          "at-001",
		Name:        "Test Agent",
		Type:        "assistant",
		Description: "A test agent",
		Category:    "test",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	require.NoError(t, s.CreateAgentTemplate(ctx, tmpl))

	got, err := s.GetAgentTemplate(ctx, "at-001")
	require.NoError(t, err)
	assert.Equal(t, "Test Agent", got.Name)

	// Update
	got.Name = "Updated Agent"
	got.Type = "claude"
	got.Role = "reviewer"
	got.Skills = []string{"builtin-code-review"}
	got.UpdatedAt = time.Now().Truncate(time.Second)
	require.NoError(t, s.UpdateAgentTemplate(ctx, got))

	updated, err := s.GetAgentTemplate(ctx, "at-001")
	require.NoError(t, err)
	assert.Equal(t, "Updated Agent", updated.Name)
	assert.Equal(t, model.AgentModelType("claude"), updated.Type)
	assert.Equal(t, "reviewer", updated.Role)
	assert.Equal(t, []string{"builtin-code-review"}, updated.Skills)

	require.NoError(t, s.DeleteAgentTemplate(ctx, "at-001"))
}

// ============================================================================
// Skill 测试
// ============================================================================

func TestSkillCRUD(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	skill := &model.Skill{
		ID:        "skill-001",
		Name:      "Browser",
		Category:  "automation",
		CreatedAt: now,
		UpdatedAt: now,
		Tags:      []string{"web", "browser"},
	}
	require.NoError(t, s.CreateSkill(ctx, skill))

	got, err := s.GetSkill(ctx, "skill-001")
	require.NoError(t, err)
	assert.Equal(t, "Browser", got.Name)
	assert.Len(t, got.Tags, 2)

	skills, err := s.ListSkills(ctx, "automation")
	require.NoError(t, err)
	assert.Len(t, skills, 1)

	require.NoError(t, s.DeleteSkill(ctx, "skill-001"))
}

// ============================================================================
// MCP Server 测试
// ============================================================================

func TestMCPServerCRUD(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	server := &model.MCPServer{
		ID:        "mcp-001",
		Name:      "Test MCP",
		Source:    "custom",
		Transport: "stdio",
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, s.CreateMCPServer(ctx, server))

	got, err := s.GetMCPServer(ctx, "mcp-001")
	require.NoError(t, err)
	assert.Equal(t, "Test MCP", got.Name)

	servers, err := s.ListMCPServers(ctx, "custom")
	require.NoError(t, err)
	assert.Len(t, servers, 1)

	require.NoError(t, s.DeleteMCPServer(ctx, "mcp-001"))
}

// ============================================================================
// SecurityPolicy 测试
// ============================================================================

func TestSecurityPolicyCRUD(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	policy := &model.SecurityPolicyEntity{
		ID:          "sp-001",
		Name:        "Default Policy",
		Description: "A default policy",
		Category:    "default",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	require.NoError(t, s.CreateSecurityPolicy(ctx, policy))

	got, err := s.GetSecurityPolicy(ctx, "sp-001")
	require.NoError(t, err)
	assert.Equal(t, "Default Policy", got.Name)

	policies, err := s.ListSecurityPolicies(ctx, "default")
	require.NoError(t, err)
	assert.Len(t, policies, 1)

	require.NoError(t, s.DeleteSecurityPolicy(ctx, "sp-001"))
}

// ============================================================================
// 工厂函数测试
// ============================================================================

func TestNewPersistentStoreFromDSN(t *testing.T) {
	// 测试通过 storage 包的工厂函数创建 SQLite Store
	db, err := sqlitedriver.Open(":memory:")
	require.NoError(t, err)
	dialect := sqlitedriver.NewDialect()
	require.NoError(t, dialect.AutoMigrate(db))
	store := NewStore(db, dialect)
	defer store.Close()

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// 验证基本操作
	task := &model.Task{ID: "factory-test", Name: "Factory Test", Status: model.TaskStatusPending, Type: "general", CreatedAt: now, UpdatedAt: now}
	require.NoError(t, store.CreateTask(ctx, task))
	got, err := store.GetTask(ctx, "factory-test")
	require.NoError(t, err)
	assert.Equal(t, "Factory Test", got.Name)
}

// ============================================================================
// 辅助函数
// ============================================================================

func strPtr(s string) *string { return &s }

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func timePtr(t time.Time) *time.Time { return &t }

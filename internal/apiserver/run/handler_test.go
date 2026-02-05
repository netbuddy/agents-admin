// Package run 执行领域 - Handler 单元测试
//
// 测试类型：Unit Test（使用 Mock 隔离存储层）
// 测试用例来源：docs/business-flow/20-执行管理/01-执行触发.md
package run

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"agents-admin/internal/shared/model"
)

// ============================================================================
// Mock 实现（实现 RunStore 和 RunScheduler 接口）
// ============================================================================

// mockRunStore 模拟存储（仅实现 RunStore 接口）
type mockRunStore struct {
	tasks map[string]*model.Task
	runs  map[string]*model.Run

	// 控制行为
	createRunErr error
	getTaskErr   error
	getRunErr    error
}

func newMockStore() *mockRunStore {
	return &mockRunStore{
		tasks: make(map[string]*model.Task),
		runs:  make(map[string]*model.Run),
	}
}

func (m *mockRunStore) GetTask(ctx context.Context, id string) (*model.Task, error) {
	if m.getTaskErr != nil {
		return nil, m.getTaskErr
	}
	return m.tasks[id], nil
}

func (m *mockRunStore) CreateRun(ctx context.Context, run *model.Run) error {
	if m.createRunErr != nil {
		return m.createRunErr
	}
	m.runs[run.ID] = run
	return nil
}

func (m *mockRunStore) GetRun(ctx context.Context, id string) (*model.Run, error) {
	if m.getRunErr != nil {
		return nil, m.getRunErr
	}
	return m.runs[id], nil
}

func (m *mockRunStore) ListRunsByTask(ctx context.Context, taskID string) ([]*model.Run, error) {
	var result []*model.Run
	for _, r := range m.runs {
		if r.TaskID == taskID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockRunStore) UpdateRunStatus(ctx context.Context, id string, status model.RunStatus, nodeID *string) error {
	if r, ok := m.runs[id]; ok {
		r.Status = status
		if nodeID != nil {
			r.NodeID = nodeID
		}
	}
	return nil
}

// mockRunScheduler 模拟调度队列（仅实现 RunScheduler 接口）
type mockRunScheduler struct {
	scheduledRuns []string
	scheduleErr   error
}

func (m *mockRunScheduler) ScheduleRun(ctx context.Context, runID, taskID string) (string, error) {
	if m.scheduleErr != nil {
		return "", m.scheduleErr
	}
	m.scheduledRuns = append(m.scheduledRuns, runID)
	return "mock-msg-id", nil
}

// ============================================================================
// TC-RUN-CREATE-001: 基本创建
// ============================================================================

func TestCreate_Basic(t *testing.T) {
	store := newMockStore()
	queue := &mockRunScheduler{}

	// 创建测试任务
	task := &model.Task{
		ID:     "task-test-001",
		Name:   "test task",
		Status: model.TaskStatusPending,
		Prompt: &model.Prompt{Content: "test prompt"},
	}
	store.tasks[task.ID] = task

	handler := NewHandlerWithInterfaces(store, queue)

	// 构造请求
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest("POST", "/api/v1/tasks/task-test-001/runs", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	// 验证 HTTP 状态码 = 201
	if w.Code != http.StatusCreated {
		t.Fatalf("HTTP 状态码 = %d, 期望 201, 响应: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("响应解析失败: %v", err)
	}

	// 验证响应 id 格式
	runID, ok := result["id"].(string)
	if !ok || !strings.HasPrefix(runID, "run-") {
		t.Errorf("响应 id 格式错误: %v", result["id"])
	}

	// 验证响应 task_id
	if result["task_id"] != "task-test-001" {
		t.Errorf("响应 task_id = %v, 期望 task-test-001", result["task_id"])
	}

	// 验证响应 status = queued
	if result["status"] != "queued" {
		t.Errorf("响应 status = %v, 期望 queued", result["status"])
	}

	// 验证 Run 已存储
	if len(store.runs) != 1 {
		t.Errorf("存储的 Run 数量 = %d, 期望 1", len(store.runs))
	}

	// 验证 Redis 已调度
	if len(queue.scheduledRuns) != 1 {
		t.Errorf("Redis 调度数量 = %d, 期望 1", len(queue.scheduledRuns))
	}
}

// ============================================================================
// TC-RUN-CREATE-002: 任务不存在
// ============================================================================

func TestCreate_TaskNotFound(t *testing.T) {
	store := newMockStore()
	queue := &mockRunScheduler{}
	handler := NewHandlerWithInterfaces(store, queue)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest("POST", "/api/v1/tasks/non-existent/runs", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	// 验证 HTTP 状态码 = 404
	if w.Code != http.StatusNotFound {
		t.Errorf("HTTP 状态码 = %d, 期望 404", w.Code)
	}

	// 验证错误信息
	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	if errMsg, ok := result["error"].(string); !ok || !strings.Contains(errMsg, "not found") {
		t.Errorf("错误信息 = %v, 期望包含 'not found'", result["error"])
	}

	// 验证没有创建 Run
	if len(store.runs) != 0 {
		t.Errorf("存储的 Run 数量 = %d, 期望 0", len(store.runs))
	}
}

// ============================================================================
// TC-RUN-CREATE-003: PostgreSQL 写入失败
// ============================================================================

func TestCreate_PostgresFailed(t *testing.T) {
	store := newMockStore()
	store.createRunErr = errors.New("database error")

	queue := &mockRunScheduler{}

	// 创建测试任务
	task := &model.Task{ID: "task-test-002", Name: "test", Status: model.TaskStatusPending}
	store.tasks[task.ID] = task

	handler := NewHandlerWithInterfaces(store, queue)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest("POST", "/api/v1/tasks/task-test-002/runs", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	// 验证 HTTP 状态码 = 500
	if w.Code != http.StatusInternalServerError {
		t.Errorf("HTTP 状态码 = %d, 期望 500", w.Code)
	}

	// 验证 Redis 没有调度（因为 PG 失败了）
	if len(queue.scheduledRuns) != 0 {
		t.Errorf("Redis 调度数量 = %d, 期望 0", len(queue.scheduledRuns))
	}
}

// ============================================================================
// TC-RUN-CREATE-006: Redis 不可用时仍能创建
// ============================================================================

func TestCreate_RedisFailed(t *testing.T) {
	store := newMockStore()
	queue := &mockRunScheduler{scheduleErr: errors.New("queue error")}

	// 创建测试任务
	task := &model.Task{ID: "task-test-003", Name: "test", Status: model.TaskStatusPending}
	store.tasks[task.ID] = task

	handler := NewHandlerWithInterfaces(store, queue)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest("POST", "/api/v1/tasks/task-test-003/runs", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	// 验证 HTTP 状态码 = 201（Redis 失败不影响创建）
	if w.Code != http.StatusCreated {
		t.Fatalf("HTTP 状态码 = %d, 期望 201", w.Code)
	}

	// 验证 Run 已存储
	if len(store.runs) != 1 {
		t.Errorf("存储的 Run 数量 = %d, 期望 1", len(store.runs))
	}
}

// ============================================================================
// TC-RUN-CREATE-007: Redis 为 nil 时仍能创建
// ============================================================================

func TestCreate_NilRedis(t *testing.T) {
	store := newMockStore()

	// 创建测试任务
	task := &model.Task{ID: "task-test-004", Name: "test", Status: model.TaskStatusPending}
	store.tasks[task.ID] = task

	// Redis 为 nil
	handler := NewHandlerWithInterfaces(store, nil)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest("POST", "/api/v1/tasks/task-test-004/runs", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	// 验证 HTTP 状态码 = 201
	if w.Code != http.StatusCreated {
		t.Fatalf("HTTP 状态码 = %d, 期望 201", w.Code)
	}

	// 验证 Run 已存储
	if len(store.runs) != 1 {
		t.Errorf("存储的 Run 数量 = %d, 期望 1", len(store.runs))
	}
}

// ============================================================================
// TestGet: 获取 Run
// ============================================================================

func TestGet_Basic(t *testing.T) {
	store := newMockStore()
	run := &model.Run{
		ID:     "run-test-001",
		TaskID: "task-001",
		Status: model.RunStatusQueued,
	}
	store.runs[run.ID] = run

	handler := NewHandlerWithInterfaces(store, nil)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/runs/run-test-001", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态码 = %d, 期望 200", w.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	if result["id"] != "run-test-001" {
		t.Errorf("响应 id = %v, 期望 run-test-001", result["id"])
	}
}

func TestGet_NotFound(t *testing.T) {
	store := newMockStore()
	handler := NewHandlerWithInterfaces(store, nil)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/runs/non-existent", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("HTTP 状态码 = %d, 期望 404", w.Code)
	}
}

// ============================================================================
// TestListByTask: 列出任务的所有执行记录
// ============================================================================

func TestListByTask_Basic(t *testing.T) {
	store := newMockStore()
	store.runs["run-1"] = &model.Run{ID: "run-1", TaskID: "task-001", Status: model.RunStatusQueued}
	store.runs["run-2"] = &model.Run{ID: "run-2", TaskID: "task-001", Status: model.RunStatusDone}
	store.runs["run-3"] = &model.Run{ID: "run-3", TaskID: "task-002", Status: model.RunStatusQueued}

	// 创建任务
	store.tasks["task-001"] = &model.Task{ID: "task-001", Name: "test"}

	handler := NewHandlerWithInterfaces(store, nil)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/tasks/task-001/runs", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态码 = %d, 期望 200", w.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	count := int(result["count"].(float64))
	if count != 2 {
		t.Errorf("count = %d, 期望 2", count)
	}
}

// ============================================================================
// TestCancel: 取消 Run
// ============================================================================

func TestCancel_Queued(t *testing.T) {
	store := newMockStore()
	store.runs["run-cancel-1"] = &model.Run{
		ID:     "run-cancel-1",
		TaskID: "task-001",
		Status: model.RunStatusQueued,
	}

	handler := NewHandlerWithInterfaces(store, nil)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest("POST", "/api/v1/runs/run-cancel-1/cancel", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态码 = %d, 期望 200", w.Code)
	}

	// 验证状态已更新
	if store.runs["run-cancel-1"].Status != model.RunStatusCancelled {
		t.Errorf("Run 状态 = %s, 期望 cancelled", store.runs["run-cancel-1"].Status)
	}
}

func TestCancel_AlreadyDone(t *testing.T) {
	store := newMockStore()
	store.runs["run-done-1"] = &model.Run{
		ID:     "run-done-1",
		TaskID: "task-001",
		Status: model.RunStatusDone, // 已完成，不能取消
	}

	handler := NewHandlerWithInterfaces(store, nil)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest("POST", "/api/v1/runs/run-done-1/cancel", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	// 已完成的 Run 不能取消
	if w.Code != http.StatusBadRequest {
		t.Errorf("HTTP 状态码 = %d, 期望 400", w.Code)
	}
}

// ============================================================================
// TestUpdate: 更新 Run 状态
// ============================================================================

func TestUpdate_Basic(t *testing.T) {
	store := newMockStore()
	store.runs["run-update-1"] = &model.Run{
		ID:     "run-update-1",
		TaskID: "task-001",
		Status: model.RunStatusQueued,
	}

	handler := NewHandlerWithInterfaces(store, nil)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	body := strings.NewReader(`{"status": "running"}`)
	req := httptest.NewRequest("PATCH", "/api/v1/runs/run-update-1", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态码 = %d, 期望 200, 响应: %s", w.Code, w.Body.String())
	}

	// 验证状态已更新
	if store.runs["run-update-1"].Status != model.RunStatusRunning {
		t.Errorf("Run 状态 = %s, 期望 running", store.runs["run-update-1"].Status)
	}
}

func TestUpdate_MissingStatus(t *testing.T) {
	store := newMockStore()
	store.runs["run-update-2"] = &model.Run{
		ID:     "run-update-2",
		TaskID: "task-001",
		Status: model.RunStatusQueued,
	}

	handler := NewHandlerWithInterfaces(store, nil)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	body := strings.NewReader(`{}`)
	req := httptest.NewRequest("PATCH", "/api/v1/runs/run-update-2", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	// 缺少 status 应返回 400
	if w.Code != http.StatusBadRequest {
		t.Errorf("HTTP 状态码 = %d, 期望 400", w.Code)
	}
}

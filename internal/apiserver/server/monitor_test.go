// Package server 工作流监控单元测试
//
// 本文件测试 monitor.go 和 monitor_data.go 中的功能：
//
// # 测试分组
//
// ## 纯函数测试（无需 mock，直接调用）
//   - TestMapAuthSessionStatus: 认证会话状态映射（6 种输入 → 5 种输出）
//   - TestCalculateAuthProgressFromStatus: 认证进度计算（7 种状态 → 对应百分比）
//   - TestMapAuthTaskStatus: model.AuthTaskStatus 状态映射
//   - TestMapRunStatus: model.RunStatus 状态映射
//   - TestCalculateRunProgress: Run 进度计算
//   - TestGetEventLevel: 事件级别推断（error/warning/success/info）
//   - TestMustParseInt: 整数解析（正常/异常/默认值）
//
// ## HTTP 处理器测试（使用 mockMonitorStore + mockCacheStore）
//   - TestListWorkflows_Empty: 无数据时返回空列表
//   - TestListWorkflows_WithRuns: 包含 Run 数据时的聚合
//   - TestListWorkflows_TypeFilter: type 参数过滤
//   - TestListWorkflows_Pagination: limit/offset 分页
//   - TestGetWorkflow_Run: 获取 Run 类型工作流详情
//   - TestGetWorkflow_NotFound: 工作流不存在返回 404
//   - TestGetWorkflow_BadType: 不支持的类型返回 400
//   - TestGetMonitorStats: 统计指标计算
//   - TestGetWorkflowEvents_Run: 获取 Run 事件流
//   - TestGetWorkflowEvents_MissingParams: 缺少参数返回 400
//
// # 使用的 Mock
//   - mockMonitorStore: 实现 PersistentStore 中 monitor 所需的子集
//   - mockCacheStore: 实现 CacheStore 接口（返回空数据）
//
// # 运行方式
//
//	go test -v -run TestMapAuthSessionStatus ./internal/apiserver/server/
//	go test -v -run TestListWorkflows ./internal/apiserver/server/
//	go test -v -run TestGetWorkflow ./internal/apiserver/server/
//	go test -v -run TestGetMonitorStats ./internal/apiserver/server/
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agents-admin/internal/shared/cache"
	"agents-admin/internal/shared/eventbus"
	"agents-admin/internal/shared/model"
	"agents-admin/internal/shared/queue"
	"agents-admin/internal/shared/storage"
)

// ============================================================================
// Mock 实现
// ============================================================================

// mockMonitorStore 模拟 PersistentStore，只实现 monitor 所需方法
//
// 字段说明：
//   - Tasks: ListTasks 返回的任务列表
//   - Runs: 按 TaskID 索引的 Run 列表（ListRunsByTask 使用）
//   - RunByID: 按 RunID 索引的 Run（GetRun 使用）
//   - Events: 按 RunID 索引的事件列表（GetEventsByRun 使用）
//   - AuthTasks: ListRecentAuthTasks 返回的认证任务列表
type mockMonitorStore struct {
	storage.PersistentStore // 嵌入接口，未实现的方法会 panic（测试中不应调用）

	Tasks     []*model.Task
	Runs      map[string][]*model.Run   // key: taskID
	RunByID   map[string]*model.Run     // key: runID
	Events    map[string][]*model.Event // key: runID
	AuthTasks []*model.AuthTask
}

func (m *mockMonitorStore) ListTasks(_ context.Context, _ string, _, _ int) ([]*model.Task, error) {
	return m.Tasks, nil
}

func (m *mockMonitorStore) ListRunsByTask(_ context.Context, taskID string) ([]*model.Run, error) {
	return m.Runs[taskID], nil
}

func (m *mockMonitorStore) GetRun(_ context.Context, id string) (*model.Run, error) {
	if r, ok := m.RunByID[id]; ok {
		return r, nil
	}
	return nil, nil
}

func (m *mockMonitorStore) GetEventsByRun(_ context.Context, runID string, fromSeq int, limit int) ([]*model.Event, error) {
	events := m.Events[runID]
	// 模拟 fromSeq 过滤
	var filtered []*model.Event
	for _, e := range events {
		if e.Seq > fromSeq {
			filtered = append(filtered, e)
		}
	}
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, nil
}

func (m *mockMonitorStore) ListRecentAuthTasks(_ context.Context, _ int) ([]*model.AuthTask, error) {
	return m.AuthTasks, nil
}

// mockCacheStore 实现 CacheStore 接口，返回空数据
//
// monitor 模块通过 h.redisStore 访问缓存数据。
// 此 mock 组合了 NoOpCache 和 NoOpEventBus，并添加 Queue 空实现。
type mockCacheStore struct {
	cache.NoOpCache
	eventbus.NoOpEventBus
}

// Queue 接口方法 — SchedulerQueue
func (m *mockCacheStore) Close() error { return nil }
func (m *mockCacheStore) ScheduleRun(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (m *mockCacheStore) CreateSchedulerConsumerGroup(_ context.Context) error { return nil }
func (m *mockCacheStore) ConsumeSchedulerRuns(_ context.Context, _ string, _ int64, _ time.Duration) ([]*queue.SchedulerMessage, error) {
	return nil, nil
}
func (m *mockCacheStore) AckSchedulerRun(_ context.Context, _ string) error { return nil }
func (m *mockCacheStore) GetSchedulerQueueLength(_ context.Context) (int64, error) {
	return 0, nil
}
func (m *mockCacheStore) GetSchedulerPendingCount(_ context.Context) (int64, error) {
	return 0, nil
}

// Queue 接口方法 — NodeRunQueue
func (m *mockCacheStore) PublishRunToNode(_ context.Context, _, _, _ string) (string, error) {
	return "", nil
}
func (m *mockCacheStore) CreateNodeConsumerGroup(_ context.Context, _ string) error { return nil }
func (m *mockCacheStore) ConsumeNodeRuns(_ context.Context, _, _ string, _ int64, _ time.Duration) ([]*queue.NodeRunMessage, error) {
	return nil, nil
}
func (m *mockCacheStore) AckNodeRun(_ context.Context, _, _ string) error { return nil }
func (m *mockCacheStore) GetNodeRunsQueueLength(_ context.Context, _ string) (int64, error) {
	return 0, nil
}
func (m *mockCacheStore) GetNodeRunsPendingCount(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

// 确保 mockCacheStore 满足 CacheStore 接口
var _ storage.CacheStore = (*mockCacheStore)(nil)

// testMetrics 全局共享的 Metrics 实例（避免 Prometheus 重复注册 panic）
var testMetrics = NewMetrics("monitor_test")

// newTestHandler 创建用于测试的 Handler
//
// 注意：不使用 NewHandler 以避免 Prometheus 全局指标重复注册。
// 直接构造 Handler，只设置 monitor 测试所需的字段。
//
// 参数：
//   - store: mock 持久化存储
func newTestHandler(store storage.PersistentStore) *Handler {
	cs := &mockCacheStore{}
	return &Handler{
		store:      store,
		redisStore: cs,
		metrics:    testMetrics,
	}
}

// ============================================================================
// 纯函数测试
// ============================================================================

// TestMapAuthSessionStatus 测试认证会话状态映射
//
// 映射规则：
//   - pending, assigned → "pending"
//   - running → "running"
//   - waiting_user → "waiting"
//   - success → "completed"
//   - failed, timeout → "failed"
//   - 其他 → "unknown"
func TestMapAuthSessionStatus(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"pending", "pending"},
		{"assigned", "pending"},
		{"running", "running"},
		{"waiting_user", "waiting"},
		{"success", "completed"},
		{"failed", "failed"},
		{"timeout", "failed"},
		{"unknown_status", "unknown"},
		{"", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapAuthSessionStatus(tt.input)
			if got != tt.want {
				t.Errorf("mapAuthSessionStatus(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestCalculateAuthProgressFromStatus 测试认证进度百分比计算
//
// 进度值对照：
//   - pending: 10, assigned: 25, running: 50, waiting_user: 75
//   - success/failed/timeout: 100, 未知: 0
func TestCalculateAuthProgressFromStatus(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"pending", 10},
		{"assigned", 25},
		{"running", 50},
		{"waiting_user", 75},
		{"success", 100},
		{"failed", 100},
		{"timeout", 100},
		{"unknown", 0},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := calculateAuthProgressFromStatus(tt.input)
			if got != tt.want {
				t.Errorf("calculateAuthProgressFromStatus(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

// TestMapAuthTaskStatus 测试 model.AuthTaskStatus → 统一工作流状态映射
func TestMapAuthTaskStatus(t *testing.T) {
	tests := []struct {
		name  string
		input model.AuthTaskStatus
		want  string
	}{
		{"pending", model.AuthTaskStatusPending, "pending"},
		{"assigned", model.AuthTaskStatusAssigned, "pending"},
		{"running", model.AuthTaskStatusRunning, "running"},
		{"waiting_user", model.AuthTaskStatusWaitingUser, "waiting"},
		{"success", model.AuthTaskStatusSuccess, "completed"},
		{"failed", model.AuthTaskStatusFailed, "failed"},
		{"timeout", model.AuthTaskStatusTimeout, "failed"},
		{"unknown", model.AuthTaskStatus("custom"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapAuthTaskStatus(tt.input)
			if got != tt.want {
				t.Errorf("mapAuthTaskStatus(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestMapRunStatus 测试 model.RunStatus → 统一工作流状态映射
//
// 映射规则：
//   - queued → "pending"
//   - running → "running"
//   - done → "completed"
//   - failed/cancelled/timeout → "failed"
func TestMapRunStatus(t *testing.T) {
	tests := []struct {
		name  string
		input model.RunStatus
		want  string
	}{
		{"queued", model.RunStatusQueued, "pending"},
		{"running", model.RunStatusRunning, "running"},
		{"done", model.RunStatusDone, "completed"},
		{"failed", model.RunStatusFailed, "failed"},
		{"cancelled", model.RunStatusCancelled, "failed"},
		{"timeout", model.RunStatusTimeout, "failed"},
		{"unknown", model.RunStatus("custom"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapRunStatus(tt.input)
			if got != tt.want {
				t.Errorf("mapRunStatus(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestCalculateRunProgress 测试 Run 进度百分比计算
func TestCalculateRunProgress(t *testing.T) {
	tests := []struct {
		name  string
		input model.RunStatus
		want  int
	}{
		{"queued", model.RunStatusQueued, 10},
		{"running", model.RunStatusRunning, 50},
		{"done", model.RunStatusDone, 100},
		{"failed", model.RunStatusFailed, 100},
		{"cancelled", model.RunStatusCancelled, 100},
		{"timeout", model.RunStatusTimeout, 100},
		{"unknown", model.RunStatus("custom"), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateRunProgress(tt.input)
			if got != tt.want {
				t.Errorf("calculateRunProgress(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

// TestGetEventLevel 测试事件级别推断
//
// 规则：
//   - 包含 "error"/"failed" → "error"
//   - 包含 "warning"/"timeout" → "warning"
//   - 包含 "success"/"completed" → "success"
//   - 其他 → "info"
func TestGetEventLevel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// error 级别
		{"run_error", "error"},
		{"task_failed", "error"},
		{"auth_error_occurred", "error"},
		// warning 级别
		{"connection_warning", "warning"},
		{"run_timeout", "warning"},
		// success 级别
		{"auth_success", "success"},
		{"run_completed", "success"},
		// info 级别（默认）
		{"run_started", "info"},
		{"message", "info"},
		{"heartbeat", "info"},
		{"", "info"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := getEventLevel(tt.input)
			if got != tt.want {
				t.Errorf("getEventLevel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestMustParseInt 测试安全整数解析
//
// 验证：
//   - 有效数字字符串正确解析
//   - 无效字符串返回默认值
//   - 空字符串返回默认值
func TestMustParseInt(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		defaultVal int64
		want       int64
	}{
		{"正数", "42", 0, 42},
		{"零", "0", 10, 0},
		{"负数", "-5", 0, -5},
		{"大数", "1000000", 0, 1000000},
		{"无效字符串", "abc", 50, 50},
		{"空字符串", "", 100, 100},
		{"浮点数", "3.14", 0, 0}, // json.Number.Int64() 无法解析浮点
		{"带空格", " 10 ", 0, 0}, // json.Number 不容忍空格
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mustParseInt(tt.input, tt.defaultVal)
			if got != tt.want {
				t.Errorf("mustParseInt(%q, %d) = %d, want %d", tt.input, tt.defaultVal, got, tt.want)
			}
		})
	}
}

// ============================================================================
// HTTP 处理器测试
// ============================================================================

// TestListWorkflows_Empty 无数据时返回空列表
func TestListWorkflows_Empty(t *testing.T) {
	store := &mockMonitorStore{
		Tasks: []*model.Task{},
		Runs:  map[string][]*model.Run{},
	}
	h := newTestHandler(store)

	req := httptest.NewRequest("GET", "/api/v1/monitor/workflows", nil)
	w := httptest.NewRecorder()
	h.ListWorkflows(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	workflows, _ := resp["workflows"].([]interface{})
	// 可能为 nil（空列表 JSON 序列化）
	if workflows != nil && len(workflows) != 0 {
		t.Errorf("expected 0 workflows, got %d", len(workflows))
	}
	if resp["total"].(float64) != 0 {
		t.Errorf("total = %v, want 0", resp["total"])
	}
}

// TestListWorkflows_WithRuns 包含 Run 数据时的聚合
//
// 验证：
//   - Run 被正确转换为 WorkflowSummary
//   - 状态映射正确
//   - 按更新时间降序排列
func TestListWorkflows_WithRuns(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	nodeID := "node-1"

	store := &mockMonitorStore{
		Tasks: []*model.Task{
			{ID: "task-1", Status: model.TaskStatusInProgress},
		},
		Runs: map[string][]*model.Run{
			"task-1": {
				{
					ID:        "run-1",
					TaskID:    "task-1",
					Status:    model.RunStatusDone,
					NodeID:    &nodeID,
					CreatedAt: earlier,
					UpdatedAt: now,
				},
				{
					ID:        "run-2",
					TaskID:    "task-1",
					Status:    model.RunStatusRunning,
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		},
		Events: map[string][]*model.Event{
			"run-1": {{ID: 1, Seq: 1}, {ID: 2, Seq: 2}},
			"run-2": {},
		},
	}
	h := newTestHandler(store)

	req := httptest.NewRequest("GET", "/api/v1/monitor/workflows?type=run", nil)
	w := httptest.NewRecorder()
	h.ListWorkflows(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	total := int(resp["total"].(float64))
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}

	workflows := resp["workflows"].([]interface{})
	if len(workflows) != 2 {
		t.Fatalf("workflows count = %d, want 2", len(workflows))
	}

	// 验证第一个（最新的）
	wf0 := workflows[0].(map[string]interface{})
	if wf0["type"] != "run" {
		t.Errorf("type = %v, want 'run'", wf0["type"])
	}
}

// TestListWorkflows_TypeFilter type 参数过滤
//
// 验证 type=run 时只返回 Run 类型工作流，不包含 auth。
func TestListWorkflows_TypeFilter(t *testing.T) {
	store := &mockMonitorStore{
		Tasks:     []*model.Task{{ID: "task-1"}},
		Runs:      map[string][]*model.Run{"task-1": {{ID: "run-1", TaskID: "task-1", Status: model.RunStatusQueued}}},
		Events:    map[string][]*model.Event{"run-1": {}},
		AuthTasks: []*model.AuthTask{},
	}
	h := newTestHandler(store)

	// 请求 type=run
	req := httptest.NewRequest("GET", "/api/v1/monitor/workflows?type=run", nil)
	w := httptest.NewRecorder()
	h.ListWorkflows(w, req)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	total := int(resp["total"].(float64))
	if total != 1 {
		t.Errorf("total = %d, want 1 (only runs)", total)
	}
}

// TestListWorkflows_Pagination limit/offset 分页
func TestListWorkflows_Pagination(t *testing.T) {
	now := time.Now()
	store := &mockMonitorStore{
		Tasks: []*model.Task{{ID: "task-1"}},
		Runs: map[string][]*model.Run{
			"task-1": {
				{ID: "run-1", TaskID: "task-1", Status: model.RunStatusDone, CreatedAt: now, UpdatedAt: now.Add(-3 * time.Hour)},
				{ID: "run-2", TaskID: "task-1", Status: model.RunStatusDone, CreatedAt: now, UpdatedAt: now.Add(-2 * time.Hour)},
				{ID: "run-3", TaskID: "task-1", Status: model.RunStatusDone, CreatedAt: now, UpdatedAt: now.Add(-1 * time.Hour)},
			},
		},
		Events: map[string][]*model.Event{"run-1": {}, "run-2": {}, "run-3": {}},
	}
	h := newTestHandler(store)

	// 请求 limit=1, offset=1
	req := httptest.NewRequest("GET", "/api/v1/monitor/workflows?type=run&limit=1&offset=1", nil)
	w := httptest.NewRecorder()
	h.ListWorkflows(w, req)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	total := int(resp["total"].(float64))
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}

	workflows := resp["workflows"].([]interface{})
	if len(workflows) != 1 {
		t.Errorf("page size = %d, want 1", len(workflows))
	}

	limit := int(resp["limit"].(float64))
	offset := int(resp["offset"].(float64))
	if limit != 1 || offset != 1 {
		t.Errorf("limit=%d offset=%d, want limit=1 offset=1", limit, offset)
	}
}

// TestGetWorkflow_Run 获取 Run 类型工作流详情
func TestGetWorkflow_Run(t *testing.T) {
	now := time.Now()
	errMsg := "something failed"
	store := &mockMonitorStore{
		RunByID: map[string]*model.Run{
			"run-1": {
				ID:        "run-1",
				TaskID:    "task-1",
				Status:    model.RunStatusFailed,
				Error:     &errMsg,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		Events: map[string][]*model.Event{
			"run-1": {
				{ID: 1, RunID: "run-1", Seq: 1, Type: "run_started", Timestamp: now, Payload: []byte(`{"key":"val"}`)},
				{ID: 2, RunID: "run-1", Seq: 2, Type: "run_failed", Timestamp: now},
			},
		},
	}
	h := newTestHandler(store)

	// Go 1.22 mux 中 PathValue 需要通过真实路由
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/monitor/workflows/{type}/{id}", h.GetWorkflow)

	req := httptest.NewRequest("GET", "/api/v1/monitor/workflows/run/run-1", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var detail WorkflowDetail
	json.NewDecoder(w.Body).Decode(&detail)

	if detail.ID != "run-1" {
		t.Errorf("ID = %q, want 'run-1'", detail.ID)
	}
	if detail.Type != "run" {
		t.Errorf("Type = %q, want 'run'", detail.Type)
	}
	if detail.State != "failed" {
		t.Errorf("State = %q, want 'failed'", detail.State)
	}
	if detail.Error != errMsg {
		t.Errorf("Error = %q, want %q", detail.Error, errMsg)
	}
	if detail.EventCount != 2 {
		t.Errorf("EventCount = %d, want 2", detail.EventCount)
	}
	if len(detail.Events) != 2 {
		t.Errorf("Events count = %d, want 2", len(detail.Events))
	}
	// 验证事件级别推断
	if detail.Events[0].Level != "info" {
		t.Errorf("event[0].Level = %q, want 'info'", detail.Events[0].Level)
	}
	if detail.Events[1].Level != "error" {
		t.Errorf("event[1].Level = %q, want 'error'", detail.Events[1].Level)
	}
	// 验证关联 ID
	if detail.RelatedIDs["task_id"] != "task-1" {
		t.Errorf("RelatedIDs[task_id] = %q, want 'task-1'", detail.RelatedIDs["task_id"])
	}
}

// TestGetWorkflow_NotFound 工作流不存在返回 404
func TestGetWorkflow_NotFound(t *testing.T) {
	store := &mockMonitorStore{
		RunByID: map[string]*model.Run{},
	}
	h := newTestHandler(store)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/monitor/workflows/{type}/{id}", h.GetWorkflow)

	req := httptest.NewRequest("GET", "/api/v1/monitor/workflows/run/non-existent", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// TestGetWorkflow_BadType 不支持的工作流类型返回 400
func TestGetWorkflow_BadType(t *testing.T) {
	h := newTestHandler(&mockMonitorStore{})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/monitor/workflows/{type}/{id}", h.GetWorkflow)

	req := httptest.NewRequest("GET", "/api/v1/monitor/workflows/invalid/some-id", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestGetMonitorStats 测试统计指标计算
//
// 验证：
//   - 总工作流数 = auth + run
//   - 活跃工作流计数正确
//   - 今日完成/失败计数
//   - 平均耗时计算
func TestGetMonitorStats(t *testing.T) {
	now := time.Now()
	store := &mockMonitorStore{
		Tasks: []*model.Task{
			{ID: "task-1"},
		},
		Runs: map[string][]*model.Run{
			"task-1": {
				{ID: "run-1", TaskID: "task-1", Status: model.RunStatusDone, CreatedAt: now.Add(-10 * time.Second), UpdatedAt: now},
				{ID: "run-2", TaskID: "task-1", Status: model.RunStatusRunning, CreatedAt: now, UpdatedAt: now},
				{ID: "run-3", TaskID: "task-1", Status: model.RunStatusFailed, CreatedAt: now, UpdatedAt: now},
			},
		},
		AuthTasks: []*model.AuthTask{
			{ID: "auth-1", Status: model.AuthTaskStatusSuccess, UpdatedAt: now},
			{ID: "auth-2", Status: model.AuthTaskStatusRunning, UpdatedAt: now},
		},
	}
	h := newTestHandler(store)

	req := httptest.NewRequest("GET", "/api/v1/monitor/stats", nil)
	w := httptest.NewRecorder()
	h.GetMonitorStats(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var stats MonitorStats
	json.NewDecoder(w.Body).Decode(&stats)

	// 总工作流数 = 2 auth + 3 run = 5
	if stats.TotalWorkflows != 5 {
		t.Errorf("TotalWorkflows = %d, want 5", stats.TotalWorkflows)
	}

	// 活跃 = 1 running auth + 1 running run + 0 pending = 2
	// (auth running=1, run running=1)
	if stats.ActiveWorkflows != 2 {
		t.Errorf("ActiveWorkflows = %d, want 2", stats.ActiveWorkflows)
	}

	// 今日完成 = 1 auth success + 1 run done = 2
	if stats.CompletedToday != 2 {
		t.Errorf("CompletedToday = %d, want 2", stats.CompletedToday)
	}

	// 今日失败 = 0 auth failed + 1 run failed = 1
	if stats.FailedToday != 1 {
		t.Errorf("FailedToday = %d, want 1", stats.FailedToday)
	}

	// 平均耗时 = 10s (只有 run-1 done)
	expectedAvgMs := int64(10000) // 10 seconds
	// 允许一点误差（测试执行时间）
	if stats.AvgDurationMs < expectedAvgMs-100 || stats.AvgDurationMs > expectedAvgMs+100 {
		t.Errorf("AvgDurationMs = %d, want ~%d", stats.AvgDurationMs, expectedAvgMs)
	}

	// 按类型
	if stats.WorkflowsByType["auth"] != 2 {
		t.Errorf("WorkflowsByType[auth] = %d, want 2", stats.WorkflowsByType["auth"])
	}
	if stats.WorkflowsByType["run"] != 3 {
		t.Errorf("WorkflowsByType[run] = %d, want 3", stats.WorkflowsByType["run"])
	}
}

// TestGetWorkflowEvents_Run 获取 Run 类型的事件流
func TestGetWorkflowEvents_Run(t *testing.T) {
	now := time.Now()
	store := &mockMonitorStore{
		Events: map[string][]*model.Event{
			"run-1": {
				{ID: 1, RunID: "run-1", Seq: 1, Type: "run_started", Timestamp: now},
				{ID: 2, RunID: "run-1", Seq: 2, Type: "message", Timestamp: now, Payload: []byte(`{"content":"hello"}`)},
				{ID: 3, RunID: "run-1", Seq: 3, Type: "run_completed", Timestamp: now},
			},
		},
	}
	h := newTestHandler(store)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/monitor/workflows/{type}/{id}/events", h.GetWorkflowEvents)

	req := httptest.NewRequest("GET", "/api/v1/monitor/workflows/run/run-1/events", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	total := int(resp["total"].(float64))
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}

	events := resp["events"].([]interface{})
	if len(events) != 3 {
		t.Fatalf("events count = %d, want 3", len(events))
	}

	// 验证按 seq 排序
	e0 := events[0].(map[string]interface{})
	e2 := events[2].(map[string]interface{})
	if e0["seq"].(float64) >= e2["seq"].(float64) {
		t.Error("events should be sorted by seq ascending")
	}

	// 验证事件级别
	if e0["level"] != "info" {
		t.Errorf("event[0].level = %v, want 'info'", e0["level"])
	}
	if e2["level"] != "success" {
		t.Errorf("event[2].level = %v, want 'success'", e2["level"])
	}
}

// TestGetWorkflowEvents_MissingParams 缺少参数返回 400
func TestGetWorkflowEvents_MissingParams(t *testing.T) {
	h := newTestHandler(&mockMonitorStore{})

	// 不通过路由注册，直接调用（PathValue 返回空字符串）
	req := httptest.NewRequest("GET", "/api/v1/monitor/workflows///events", nil)
	w := httptest.NewRecorder()
	h.GetWorkflowEvents(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

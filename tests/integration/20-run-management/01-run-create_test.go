// Package runmanagement 执行管理集成测试
//
// 测试用例来源：docs/business-flow/20-执行管理/01-执行触发.md
// 目录命名规则：与文档目录保持一致（中文翻译为英文）
//
// 测试架构：
//
//	测试代码 ──HTTP请求──→ httptest.Server ──→ Handler.Create() ──→ PostgreSQL + Redis
//	                           ↑
//	                      真实的HTTP服务器（监听端口）
package runmanagement

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"agents-admin/internal/apiserver/server"
	"agents-admin/internal/shared/infra"
	"agents-admin/internal/shared/model"
	"agents-admin/internal/shared/queue"
	"agents-admin/internal/shared/storage"
)

var testStore *storage.PostgresStore
var testRedis *infra.RedisInfra
var testHandler *server.Handler
var testServer *httptest.Server

func TestMain(m *testing.M) {
	// 加载 .env 文件
	envPaths := []string{".env", "../../../.env", "../../../../.env"}
	for _, p := range envPaths {
		if err := godotenv.Load(p); err == nil {
			break
		}
	}

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		dbURL = "postgres://agents:agents_dev_password@localhost:5432/agents_admin?sslmode=disable"
	}

	var err error
	testStore, err = storage.NewPostgresStore(dbURL)
	if err != nil {
		os.Exit(0)
	}

	// 初始化 Redis（可选）
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}
	testRedis, _ = infra.NewRedisInfra(redisURL)

	// 如果 Redis 不可用，使用 NoOpCacheStore
	var cacheStore storage.CacheStore
	if testRedis != nil {
		cacheStore = testRedis
	} else {
		cacheStore = storage.NewNoOpCacheStore()
	}

	testHandler = server.NewHandler(testStore, cacheStore)
	testServer = httptest.NewServer(testHandler.Router())

	code := m.Run()

	testServer.Close()
	if testRedis != nil {
		testRedis.Close()
	}
	testStore.Close()

	os.Exit(code)
}

// createTestTask 创建测试任务，返回 task ID
func createTestTask(t *testing.T, name string) string {
	ctx := context.Background()
	task := &model.Task{
		ID:        "task-" + time.Now().Format("150405"),
		Name:      name,
		Status:    model.TaskStatusPending,
		Type:      model.TaskTypeGeneral,
		Prompt:    &model.Prompt{Content: "test prompt"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := testStore.CreateTask(ctx, task); err != nil {
		t.Fatalf("创建测试任务失败: %v", err)
	}
	return task.ID
}

// cleanupTask 清理测试任务及其关联的 Run
func cleanupTask(taskID string) {
	ctx := context.Background()
	// 先删除关联的 runs
	runs, _ := testStore.ListRunsByTask(ctx, taskID)
	for _, run := range runs {
		testStore.DeleteRun(ctx, run.ID)
	}
	testStore.DeleteTask(ctx, taskID)
}

func getRedisValueAsString(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	default:
		return fmt.Sprint(v)
	}
}

// ============================================================================
// TC-RUN-CREATE-001: 基本创建
// ============================================================================

func TestRunCreate_Basic(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	taskID := createTestTask(t, "run-create-basic-test")
	defer cleanupTask(taskID)

	// 发送 POST /api/v1/tasks/{id}/runs
	resp, err := http.Post(
		testServer.URL+"/api/v1/tasks/"+taskID+"/runs",
		"application/json",
		nil,
	)
	if err != nil {
		t.Fatalf("TC-RUN-CREATE-001: HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证 HTTP 状态码 = 201
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("TC-RUN-CREATE-001: HTTP 状态码 = %d, 期望 201, 响应: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("TC-RUN-CREATE-001: 响应解析失败: %v", err)
	}

	// 验证响应 id 非空，格式 run-xxx
	runID, ok := result["id"].(string)
	if !ok || runID == "" {
		t.Errorf("TC-RUN-CREATE-001: 响应 id 为空")
	}
	if !strings.HasPrefix(runID, "run-") {
		t.Errorf("TC-RUN-CREATE-001: 响应 id 格式错误: %s, 期望 run-xxx", runID)
	}

	// 验证响应 task_id
	if result["task_id"] != taskID {
		t.Errorf("TC-RUN-CREATE-001: 响应 task_id = %v, 期望 %s", result["task_id"], taskID)
	}

	// 验证响应 status = queued
	if result["status"] != "queued" {
		t.Errorf("TC-RUN-CREATE-001: 响应 status = %v, 期望 queued", result["status"])
	}

	// 验证响应 snapshot 非空
	if result["snapshot"] == nil {
		t.Errorf("TC-RUN-CREATE-001: 响应 snapshot 为空")
	}

	// 验证 DB: runs 表存在记录
	ctx := context.Background()
	run, err := testStore.GetRun(ctx, runID)
	if err != nil {
		t.Fatalf("TC-RUN-CREATE-001: 查询数据库失败: %v", err)
	}
	if run == nil {
		t.Errorf("TC-RUN-CREATE-001: DB 中不存在 id=%s 的记录", runID)
	}

	// 验证 DB: runs.status = queued
	if run != nil && string(run.Status) != "queued" {
		t.Errorf("TC-RUN-CREATE-001: DB runs.status = %s, 期望 queued", run.Status)
	}

	// 验证 DB: runs.node_id = NULL
	if run != nil && run.NodeID != nil && *run.NodeID != "" {
		t.Errorf("TC-RUN-CREATE-001: DB runs.node_id = %s, 期望 NULL", *run.NodeID)
	}

	// 验证 DB: tasks.status = pending（不变）
	task, _ := testStore.GetTask(ctx, taskID)
	if task != nil && string(task.Status) != "pending" {
		t.Errorf("TC-RUN-CREATE-001: DB tasks.status = %s, 期望 pending（不变）", task.Status)
	}

	// 验证 Redis: scheduler:runs 包含该 run_id 的消息（Step 5）
	if testRedis != nil {
		msgs, err := testRedis.Client().XRevRangeN(ctx, queue.KeySchedulerRuns, "+", "-", 100).Result()
		if err != nil {
			t.Errorf("TC-RUN-CREATE-001: XREVRANGE 失败: %v", err)
		} else {
			found := false
			for _, m := range msgs {
				if getRedisValueAsString(m.Values["run_id"]) == runID {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("TC-RUN-CREATE-001: Redis scheduler:runs 未找到 run_id=%s 的消息", runID)
			}
		}
	}

	t.Logf("TC-RUN-CREATE-001: 测试通过，Run ID: %s, Task ID: %s", runID, taskID)
}

// ============================================================================
// TC-RUN-CREATE-002: 任务不存在
// ============================================================================

func TestRunCreate_TaskNotFound(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	// 获取当前 runs 数量
	ctx := context.Background()
	runsBefore, _ := testStore.ListRunsByTask(ctx, "not-exist-task")
	countBefore := len(runsBefore)

	// 发送 POST /api/v1/tasks/not-exist/runs
	resp, err := http.Post(
		testServer.URL+"/api/v1/tasks/not-exist-task/runs",
		"application/json",
		nil,
	)
	if err != nil {
		t.Fatalf("TC-RUN-CREATE-002: HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证 HTTP 状态码 = 404
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("TC-RUN-CREATE-002: HTTP 状态码 = %d, 期望 404", resp.StatusCode)
	}

	// 验证 DB: runs 表无新增记录
	runsAfter, _ := testStore.ListRunsByTask(ctx, "not-exist-task")
	if len(runsAfter) != countBefore {
		t.Errorf("TC-RUN-CREATE-002: runs 数量变化 %d -> %d, 期望不变", countBefore, len(runsAfter))
	}
}

// ============================================================================
// TC-RUN-CREATE-003: 快照包含任务完整信息
// ============================================================================

func TestRunCreate_SnapshotContainsTask(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	// 创建带 workspace 和 labels 的任务
	ctx := context.Background()
	task := &model.Task{
		ID:     "task-snap" + time.Now().Format("150405"),
		Name:   "snapshot-test",
		Status: model.TaskStatusPending,
		Type:   model.TaskTypeDevelopment,
		Prompt: &model.Prompt{Content: "snapshot test prompt"},
		Workspace: &model.WorkspaceConfig{
			Type: "git",
			Git:  &model.GitConfig{URL: "https://github.com/test/repo.git", Branch: "main"},
		},
		Labels:    map[string]string{"env": "test", "priority": "high"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := testStore.CreateTask(ctx, task); err != nil {
		t.Fatalf("创建测试任务失败: %v", err)
	}
	defer cleanupTask(task.ID)

	// 创建 Run
	resp, err := http.Post(
		testServer.URL+"/api/v1/tasks/"+task.ID+"/runs",
		"application/json",
		nil,
	)
	if err != nil {
		t.Fatalf("TC-RUN-CREATE-003: HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("TC-RUN-CREATE-003: HTTP 状态码 = %d, 期望 201, 响应: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	runID := result["id"].(string)

	// 获取 Run 并解析 snapshot
	run, _ := testStore.GetRun(ctx, runID)
	if run == nil || run.Snapshot == nil {
		t.Fatalf("TC-RUN-CREATE-003: Run 或 snapshot 为空")
	}

	var snapshot map[string]interface{}
	if err := json.Unmarshal(run.Snapshot, &snapshot); err != nil {
		t.Fatalf("TC-RUN-CREATE-003: snapshot 解析失败: %v", err)
	}

	// 验证 snapshot 包含 Task 信息
	if snapshot["name"] != task.Name {
		t.Errorf("TC-RUN-CREATE-003: snapshot.name = %v, 期望 %s", snapshot["name"], task.Name)
	}
	if snapshot["type"] != string(task.Type) {
		t.Errorf("TC-RUN-CREATE-003: snapshot.type = %v, 期望 %s", snapshot["type"], task.Type)
	}

	// 验证 prompt
	prompt, ok := snapshot["prompt"].(map[string]interface{})
	if !ok {
		t.Errorf("TC-RUN-CREATE-003: snapshot.prompt 不存在")
	} else if prompt["content"] != task.Prompt.Content {
		t.Errorf("TC-RUN-CREATE-003: snapshot.prompt.content = %v, 期望 %s", prompt["content"], task.Prompt.Content)
	}

	// 验证 workspace
	ws, ok := snapshot["workspace"].(map[string]interface{})
	if !ok {
		t.Errorf("TC-RUN-CREATE-003: snapshot.workspace 不存在")
	} else if ws["type"] != "git" {
		t.Errorf("TC-RUN-CREATE-003: snapshot.workspace.type = %v, 期望 git", ws["type"])
	}

	// 验证 labels
	labels, ok := snapshot["labels"].(map[string]interface{})
	if !ok {
		t.Errorf("TC-RUN-CREATE-003: snapshot.labels 不存在")
	} else {
		if labels["env"] != "test" {
			t.Errorf("TC-RUN-CREATE-003: snapshot.labels.env = %v, 期望 test", labels["env"])
		}
	}
}

// ============================================================================
// TC-RUN-CREATE-004: 多次创建 Run
// ============================================================================

func TestRunCreate_Multiple(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	taskID := createTestTask(t, "multiple-runs-test")
	defer cleanupTask(taskID)

	ctx := context.Background()

	// 创建第一个 Run
	resp1, _ := http.Post(testServer.URL+"/api/v1/tasks/"+taskID+"/runs", "application/json", nil)
	var result1 map[string]interface{}
	json.NewDecoder(resp1.Body).Decode(&result1)
	resp1.Body.Close()
	runID1 := result1["id"].(string)

	// 创建第二个 Run
	resp2, _ := http.Post(testServer.URL+"/api/v1/tasks/"+taskID+"/runs", "application/json", nil)
	var result2 map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&result2)
	resp2.Body.Close()
	runID2 := result2["id"].(string)

	// 验证 id1 ≠ id2
	if runID1 == runID2 {
		t.Errorf("TC-RUN-CREATE-004: run ID 相同: %s", runID1)
	}

	// 验证 DB 存在 2 条记录
	runs, _ := testStore.ListRunsByTask(ctx, taskID)
	if len(runs) != 2 {
		t.Errorf("TC-RUN-CREATE-004: runs 数量 = %d, 期望 2", len(runs))
	}

	// 验证两条记录的 task_id 相同
	for _, run := range runs {
		if run.TaskID != taskID {
			t.Errorf("TC-RUN-CREATE-004: run.task_id = %s, 期望 %s", run.TaskID, taskID)
		}
	}
}

// ============================================================================
// TC-RUN-CREATE-007: 验证时间戳
// ============================================================================

func TestRunCreate_Timestamps(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	taskID := createTestTask(t, "timestamp-test")
	defer cleanupTask(taskID)

	ctx := context.Background()

	// 记录时间 T1
	t1 := time.Now().Add(-time.Second)

	// 创建 Run
	resp, _ := http.Post(testServer.URL+"/api/v1/tasks/"+taskID+"/runs", "application/json", nil)
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()
	runID := result["id"].(string)

	// 记录时间 T2
	t2 := time.Now().Add(time.Second)

	// 验证 DB: runs.created_at
	run, _ := testStore.GetRun(ctx, runID)
	if run == nil {
		t.Fatalf("TC-RUN-CREATE-007: Run 不存在")
	}

	if run.CreatedAt.Before(t1) || run.CreatedAt.After(t2) {
		t.Errorf("TC-RUN-CREATE-007: runs.created_at = %v, 期望在 [%v, %v] 之间", run.CreatedAt, t1, t2)
	}

	// 验证 DB: runs.started_at = NULL
	if run.StartedAt != nil {
		t.Errorf("TC-RUN-CREATE-007: runs.started_at = %v, 期望 NULL", run.StartedAt)
	}

	// 验证 DB: runs.finished_at = NULL
	if run.FinishedAt != nil {
		t.Errorf("TC-RUN-CREATE-007: runs.finished_at = %v, 期望 NULL", run.FinishedAt)
	}
}

// ============================================================================
// TC-RUN-CREATE-005: Redis 消息验证（可选，Redis 不可用时跳过）
// ============================================================================

func TestRunCreate_RedisMessage(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	taskID := createTestTask(t, "redis-test")
	defer cleanupTask(taskID)

	ctx := context.Background()

	// 记录 Stream 长度 L1
	l1, err := testRedis.GetSchedulerQueueLength(ctx)
	if err != nil {
		t.Skipf("无法获取 Redis Stream 长度: %v", err)
	}

	// 创建 Run
	resp, err := http.Post(testServer.URL+"/api/v1/tasks/"+taskID+"/runs", "application/json", nil)
	if err != nil {
		t.Fatalf("TC-RUN-CREATE-005: HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("TC-RUN-CREATE-005: HTTP 状态码 = %d, 期望 201", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("TC-RUN-CREATE-005: 响应解析失败: %v", err)
	}
	createRunID, _ := result["id"].(string)
	if createRunID == "" {
		t.Fatalf("TC-RUN-CREATE-005: 响应 run_id 为空")
	}
	if result["task_id"] != taskID {
		t.Fatalf("TC-RUN-CREATE-005: 响应 task_id = %v, 期望 %s", result["task_id"], taskID)
	}

	// 记录 Stream 长度 L2
	l2, _ := testRedis.GetSchedulerQueueLength(ctx)

	// 验证 L2 - L1 >= 1
	if l2-l1 < 1 {
		t.Fatalf("TC-RUN-CREATE-005: Stream 长度变化 = %d, 期望 >= 1", l2-l1)
	}

	// 验证 Redis: 最新消息包含 run_id 和 task_id
	msgs, err := testRedis.Client().XRevRangeN(ctx, queue.KeySchedulerRuns, "+", "-", 100).Result()
	if err != nil {
		t.Fatalf("TC-RUN-CREATE-005: XREVRANGE 失败: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatalf("TC-RUN-CREATE-005: scheduler:runs 无消息")
	}

	found := false
	for _, m := range msgs {
		runID := getRedisValueAsString(m.Values["run_id"])
		tID := getRedisValueAsString(m.Values["task_id"])
		if runID == createRunID {
			found = true
			if tID != taskID {
				t.Fatalf("TC-RUN-CREATE-005: scheduler:runs 消息 task_id=%s, 期望 %s", tID, taskID)
			}
			break
		}
	}
	if !found {
		t.Fatalf("TC-RUN-CREATE-005: scheduler:runs 未找到 run_id=%s 的消息", createRunID)
	}
}

type failingSchedulerCacheStore struct {
	storage.CacheStore
}

func (s failingSchedulerCacheStore) ScheduleRun(ctx context.Context, runID, taskID string) (string, error) {
	return "", fmt.Errorf("forced ScheduleRun failure")
}

func TestRunCreate_RedisUnavailableStillCreates(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	// 使用“强制入队失败”的 CacheStore 构造 Handler
	failingStore := failingSchedulerCacheStore{CacheStore: testRedis}
	h := server.NewHandler(testStore, failingStore)
	srv := httptest.NewServer(h.Router())
	defer srv.Close()

	// 捕获日志，验证包含 queue.failed
	var buf bytes.Buffer
	prevOut := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prevOut)

	taskID := createTestTask(t, "run-create-redis-down-test")
	defer cleanupTask(taskID)

	resp, err := http.Post(srv.URL+"/api/v1/tasks/"+taskID+"/runs", "application/json", nil)
	if err != nil {
		t.Fatalf("TC-RUN-CREATE-006: HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("TC-RUN-CREATE-006: HTTP 状态码=%d, 期望 201, 响应=%s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("TC-RUN-CREATE-006: 响应解析失败: %v", err)
	}
	runID, _ := result["id"].(string)
	if runID == "" {
		t.Fatalf("TC-RUN-CREATE-006: 响应 run_id 为空")
	}

	// DB: runs 表存在新记录，且 status=queued
	run, err := testStore.GetRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("TC-RUN-CREATE-006: 查询 Run 失败: %v", err)
	}
	if run == nil {
		t.Fatalf("TC-RUN-CREATE-006: DB 中不存在 run_id=%s", runID)
	}
	if run.TaskID != taskID {
		t.Fatalf("TC-RUN-CREATE-006: DB run.task_id=%s, 期望 %s", run.TaskID, taskID)
	}
	if run.Status != model.RunStatusQueued {
		t.Fatalf("TC-RUN-CREATE-006: DB run.status=%s, 期望 queued", run.Status)
	}

	// Task 状态不变
	task, err := testStore.GetTask(context.Background(), taskID)
	if err != nil {
		t.Fatalf("TC-RUN-CREATE-006: 查询 Task 失败: %v", err)
	}
	if task == nil {
		t.Fatalf("TC-RUN-CREATE-006: DB 中不存在 task_id=%s", taskID)
	}
	if task.Status != model.TaskStatusPending {
		t.Fatalf("TC-RUN-CREATE-006: DB task.status=%s, 期望 pending", task.Status)
	}

	if !strings.Contains(buf.String(), "[run.create.queue.failed]") {
		t.Fatalf("TC-RUN-CREATE-006: 日志未包含 [run.create.queue.failed]，实际日志=%s", buf.String())
	}
}

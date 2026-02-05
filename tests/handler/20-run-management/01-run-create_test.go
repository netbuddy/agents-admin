// Package runmanagement 执行管理 Handler 单元测试
//
// 测试类型：Handler Unit Test（处理器单元测试）
// 测试用例来源：docs/business-flow/20-执行管理/01-执行触发.md
//
// 使用 ServeHTTP 直接调用 Handler，跳过网络层，在内存中完成测试。
package runmanagement

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"agents-admin/internal/apiserver/server"
	"agents-admin/internal/shared/model"
	"agents-admin/internal/shared/storage"
)

var testStore *storage.PostgresStore
var testHandler *server.Handler
var testMux http.Handler

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

	testHandler = server.NewHandler(testStore, storage.NewNoOpCacheStore())
	testMux = testHandler.Router()

	code := m.Run()

	testStore.Close()
	os.Exit(code)
}

// createTestTask 创建测试任务
func createTestTask(t *testing.T, name string) string {
	ctx := context.Background()
	task := &model.Task{
		ID:        "task-h" + time.Now().Format("150405"),
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

// cleanupTask 清理测试任务
func cleanupTask(taskID string) {
	ctx := context.Background()
	runs, _ := testStore.ListRunsByTask(ctx, taskID)
	for _, run := range runs {
		testStore.DeleteRun(ctx, run.ID)
	}
	testStore.DeleteTask(ctx, taskID)
}

// ============================================================================
// TC-RUN-CREATE-001: 基本创建（Handler 单元测试）
// ============================================================================

func TestRunCreate_Basic_Handler(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	taskID := createTestTask(t, "handler-run-create-basic")
	defer cleanupTask(taskID)

	// 构造请求
	req := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// 直接调用 ServeHTTP
	testMux.ServeHTTP(w, req)

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
	if result["task_id"] != taskID {
		t.Errorf("响应 task_id = %v, 期望 %s", result["task_id"], taskID)
	}

	// 验证响应 status = queued
	if result["status"] != "queued" {
		t.Errorf("响应 status = %v, 期望 queued", result["status"])
	}

	t.Logf("Handler Unit Test 通过，Run ID: %s", runID)
}

// ============================================================================
// TC-RUN-CREATE-002: 任务不存在（Handler 单元测试）
// ============================================================================

func TestRunCreate_TaskNotFound_Handler(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	// 构造请求
	req := httptest.NewRequest("POST", "/api/v1/tasks/non-existent-task/runs", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	testMux.ServeHTTP(w, req)

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
}

// ============================================================================
// 验证 Task 状态不变
// ============================================================================

func TestRunCreate_TaskStatusUnchanged_Handler(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	taskID := createTestTask(t, "handler-task-status-test")
	defer cleanupTask(taskID)

	ctx := context.Background()

	// 获取创建前的 Task 状态
	taskBefore, _ := testStore.GetTask(ctx, taskID)
	if taskBefore == nil {
		t.Fatalf("Task 不存在")
	}

	// 创建 Run
	req := httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
	w := httptest.NewRecorder()
	testMux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("HTTP 状态码 = %d, 期望 201", w.Code)
	}

	// 获取创建后的 Task 状态
	taskAfter, _ := testStore.GetTask(ctx, taskID)
	if taskAfter == nil {
		t.Fatalf("Task 不存在")
	}

	// 验证 Task 状态未变
	if taskBefore.Status != taskAfter.Status {
		t.Errorf("Task 状态从 %s 变为 %s, 期望不变", taskBefore.Status, taskAfter.Status)
	}

	t.Logf("Task 状态验证通过: %s（保持不变）", taskAfter.Status)
}

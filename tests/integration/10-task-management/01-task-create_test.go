// Package taskmanagement 任务管理集成测试
//
// 测试用例来源：docs/business-flow/10-任务管理/01-任务创建.md
// 目录命名规则：与文档目录保持一致（中文翻译为英文）
//
// 测试架构：
//
//	测试代码 ──HTTP请求──→ httptest.Server ──→ Handler.Create() ──→ PostgreSQL
//	                           ↑
//	                      真实的HTTP服务器（监听端口）
package taskmanagement

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"agents-admin/internal/apiserver/server"
	"agents-admin/internal/config"
	"agents-admin/internal/shared/storage"
)

var testStore *storage.PostgresStore
var testHandler *server.Handler
var testServer *httptest.Server // 真正的 HTTP 服务器

func TestMain(m *testing.M) {
	// 强制使用测试环境（加载 configs/test.yaml）
	os.Setenv("APP_ENV", "test")
	cfg := config.Load()

	var err error
	testStore, err = storage.NewPostgresStore(cfg.DatabaseURL)
	if err != nil {
		// 如果无法连接数据库，跳过集成测试
		os.Exit(0)
	}

	testHandler = server.NewHandler(testStore, storage.NewNoOpCacheStore())

	// 启动真正的 HTTP 服务器（使用随机端口）
	// httptest.NewServer 会启动一个真实的 HTTP 服务器，监听一个随机端口
	// 这与生产环境中 http.ListenAndServe 的行为完全一致
	testServer = httptest.NewServer(testHandler.Router())

	code := m.Run()

	// 清理资源
	testServer.Close()
	testStore.Close()

	os.Exit(code)
}

// ============================================================================
// TC-TASK-CREATE-001: 基本创建
// ============================================================================

func TestTaskCreate_Basic(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()

	// 步骤：发送 POST /api/v1/tasks
	// 注意：这里使用 http.Post 发送真实的 HTTP 请求到 testServer
	createBody := `{"name": "test", "prompt": "test prompt"}`
	resp, err := http.Post(
		testServer.URL+"/api/v1/tasks",
		"application/json",
		bytes.NewBufferString(createBody),
	)
	if err != nil {
		t.Fatalf("TC-TASK-CREATE-001: HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证 HTTP 状态码 = 201
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("TC-TASK-CREATE-001: HTTP 状态码 = %d, 期望 201, 响应: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("TC-TASK-CREATE-001: 响应解析失败: %v", err)
	}

	// 验证响应 id 非空，格式 task-xxx
	taskID, ok := result["id"].(string)
	if !ok || taskID == "" {
		t.Errorf("TC-TASK-CREATE-001: 响应 id 为空")
	}
	if !strings.HasPrefix(taskID, "task-") {
		t.Errorf("TC-TASK-CREATE-001: 响应 id 格式错误: %s, 期望 task-xxx", taskID)
	}

	// 验证响应 status = pending
	if result["status"] != "pending" {
		t.Errorf("TC-TASK-CREATE-001: 响应 status = %v, 期望 pending", result["status"])
	}

	// 验证响应 type = general（默认值）
	if result["type"] != "general" {
		t.Errorf("TC-TASK-CREATE-001: 响应 type = %v, 期望 general", result["type"])
	}

	// 验证 DB: tasks 表存在对应记录
	task, err := testStore.GetTask(ctx, taskID)
	if err != nil {
		t.Fatalf("TC-TASK-CREATE-001: 查询数据库失败: %v", err)
	}
	if task == nil {
		t.Errorf("TC-TASK-CREATE-001: DB 中不存在 id=%s 的记录", taskID)
	}

	// 验证 DB: tasks.status = pending
	if task != nil && string(task.Status) != "pending" {
		t.Errorf("TC-TASK-CREATE-001: DB tasks.status = %s, 期望 pending", task.Status)
	}

	// 验证 DB: tasks.name = test
	if task != nil && task.Name != "test" {
		t.Errorf("TC-TASK-CREATE-001: DB tasks.name = %s, 期望 test", task.Name)
	}

	t.Logf("TC-TASK-CREATE-001: 测试通过，Server URL: %s, Task ID: %s", testServer.URL, taskID)

	// 清理
	_ = testStore.DeleteTask(ctx, taskID)
}

// ============================================================================
// TC-TASK-CREATE-002: 指定类型创建
// ============================================================================

func TestTaskCreate_WithType(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()

	// 步骤：发送请求，type = development
	createBody := `{"name": "typed task", "prompt": "test prompt", "type": "development"}`
	resp, err := http.Post(
		testServer.URL+"/api/v1/tasks",
		"application/json",
		bytes.NewBufferString(createBody),
	)
	if err != nil {
		t.Fatalf("TC-TASK-CREATE-002: HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证 HTTP 状态码 = 201
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("TC-TASK-CREATE-002: HTTP 状态码 = %d, 期望 201", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	taskID := result["id"].(string)

	// 验证响应 type = development
	if result["type"] != "development" {
		t.Errorf("TC-TASK-CREATE-002: 响应 type = %v, 期望 development", result["type"])
	}

	// 验证 DB: tasks.type = development
	task, _ := testStore.GetTask(ctx, taskID)
	if task != nil && string(task.Type) != "development" {
		t.Errorf("TC-TASK-CREATE-002: DB tasks.type = %s, 期望 development", task.Type)
	}

	// 清理
	_ = testStore.DeleteTask(ctx, taskID)
}

// ============================================================================
// TC-TASK-CREATE-003: 缺少必填字段 - name
// ============================================================================

func TestTaskCreate_MissingName(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	// 获取当前任务数量
	tasksBefore, _ := testStore.ListTasks(context.Background(), "", 1000, 0)
	countBefore := len(tasksBefore)

	// 步骤：发送请求，不包含 name
	createBody := `{"prompt": "test prompt"}`
	resp, err := http.Post(
		testServer.URL+"/api/v1/tasks",
		"application/json",
		bytes.NewBufferString(createBody),
	)
	if err != nil {
		t.Fatalf("TC-TASK-CREATE-003: HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证 HTTP 状态码 = 400
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("TC-TASK-CREATE-003: HTTP 状态码 = %d, 期望 400", resp.StatusCode)
	}

	// 验证错误信息包含 "name"
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	errMsg, _ := result["error"].(string)
	if !strings.Contains(errMsg, "name") {
		t.Errorf("TC-TASK-CREATE-003: 错误信息 = %s, 期望包含 'name'", errMsg)
	}

	// 验证 DB: tasks 表无新增记录
	tasksAfter, _ := testStore.ListTasks(context.Background(), "", 1000, 0)
	if len(tasksAfter) != countBefore {
		t.Errorf("TC-TASK-CREATE-003: 任务数量变化 %d -> %d, 期望不变", countBefore, len(tasksAfter))
	}
}

// ============================================================================
// TC-TASK-CREATE-004: 缺少必填字段 - prompt
// ============================================================================

func TestTaskCreate_MissingPrompt(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	// 获取当前任务数量
	tasksBefore, _ := testStore.ListTasks(context.Background(), "", 1000, 0)
	countBefore := len(tasksBefore)

	// 步骤：发送请求，不包含 prompt
	createBody := `{"name": "test task"}`
	resp, err := http.Post(
		testServer.URL+"/api/v1/tasks",
		"application/json",
		bytes.NewBufferString(createBody),
	)
	if err != nil {
		t.Fatalf("TC-TASK-CREATE-004: HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证 HTTP 状态码 = 400
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("TC-TASK-CREATE-004: HTTP 状态码 = %d, 期望 400", resp.StatusCode)
	}

	// 验证错误信息包含 "prompt"
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	errMsg, _ := result["error"].(string)
	if !strings.Contains(errMsg, "prompt") {
		t.Errorf("TC-TASK-CREATE-004: 错误信息 = %s, 期望包含 'prompt'", errMsg)
	}

	// 验证 DB: tasks 表无新增记录
	tasksAfter, _ := testStore.ListTasks(context.Background(), "", 1000, 0)
	if len(tasksAfter) != countBefore {
		t.Errorf("TC-TASK-CREATE-004: 任务数量变化 %d -> %d, 期望不变", countBefore, len(tasksAfter))
	}
}

// ============================================================================
// TC-TASK-CREATE-005: 带工作空间配置
// ============================================================================

func TestTaskCreate_WithWorkspace(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()

	// 步骤：发送请求，包含 workspace 配置
	// 使用新的 OpenAPI 格式（与 model.WorkspaceConfig 一致）
	createBody := `{
		"name": "test",
		"prompt": "test",
		"workspace": {
			"type": "git",
			"git": {"url": "https://github.com/example/repo.git", "branch": "main"}
		}
	}`
	resp, err := http.Post(
		testServer.URL+"/api/v1/tasks",
		"application/json",
		bytes.NewBufferString(createBody),
	)
	if err != nil {
		t.Fatalf("TC-TASK-CREATE-005: HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证 HTTP 状态码 = 201
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("TC-TASK-CREATE-005: HTTP 状态码 = %d, 期望 201, 响应: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	taskID := result["id"].(string)

	// 验证响应 workspace.type = git
	workspace, ok := result["workspace"].(map[string]interface{})
	if !ok {
		t.Errorf("TC-TASK-CREATE-005: 响应 workspace 不存在")
	} else if workspace["type"] != "git" {
		t.Errorf("TC-TASK-CREATE-005: 响应 workspace.type = %v, 期望 git", workspace["type"])
	}

	// 验证 DB: tasks.workspace 包含 git 配置
	task, _ := testStore.GetTask(ctx, taskID)
	if task == nil {
		t.Errorf("TC-TASK-CREATE-005: DB 中不存在记录")
	} else if task.Workspace == nil {
		t.Errorf("TC-TASK-CREATE-005: DB tasks.workspace 为空")
	} else if string(task.Workspace.Type) != "git" {
		t.Errorf("TC-TASK-CREATE-005: DB tasks.workspace.type = %s, 期望 git", task.Workspace.Type)
	}

	// 清理
	_ = testStore.DeleteTask(ctx, taskID)
}

// ============================================================================
// TC-TASK-CREATE-006: 带标签创建
// ============================================================================

func TestTaskCreate_WithLabels(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()

	// 步骤：发送请求，包含 labels
	createBody := `{
		"name": "labeled task",
		"prompt": "test prompt",
		"labels": {"env": "prod", "priority": "high"}
	}`
	resp, err := http.Post(
		testServer.URL+"/api/v1/tasks",
		"application/json",
		bytes.NewBufferString(createBody),
	)
	if err != nil {
		t.Fatalf("TC-TASK-CREATE-006: HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证 HTTP 状态码 = 201
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("TC-TASK-CREATE-006: HTTP 状态码 = %d, 期望 201", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	taskID := result["id"].(string)

	// 验证响应 labels
	labels, ok := result["labels"].(map[string]interface{})
	if !ok {
		t.Errorf("TC-TASK-CREATE-006: 响应 labels 不存在")
	} else {
		if labels["env"] != "prod" {
			t.Errorf("TC-TASK-CREATE-006: 响应 labels.env = %v, 期望 prod", labels["env"])
		}
		if labels["priority"] != "high" {
			t.Errorf("TC-TASK-CREATE-006: 响应 labels.priority = %v, 期望 high", labels["priority"])
		}
	}

	// 验证 DB: tasks.labels 包含正确的 JSON
	task, _ := testStore.GetTask(ctx, taskID)
	if task == nil {
		t.Errorf("TC-TASK-CREATE-006: DB 中不存在记录")
	} else if task.Labels == nil {
		t.Errorf("TC-TASK-CREATE-006: DB tasks.labels 为空")
	} else {
		if task.Labels["env"] != "prod" {
			t.Errorf("TC-TASK-CREATE-006: DB tasks.labels[env] = %s, 期望 prod", task.Labels["env"])
		}
		if task.Labels["priority"] != "high" {
			t.Errorf("TC-TASK-CREATE-006: DB tasks.labels[priority] = %s, 期望 high", task.Labels["priority"])
		}
	}

	// 清理
	_ = testStore.DeleteTask(ctx, taskID)
}

// ============================================================================
// TC-TASK-CREATE-007: 验证时间戳
// ============================================================================

func TestTaskCreate_Timestamps(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()

	// 步骤 1: 记录当前时间 T1
	t1 := time.Now().Add(-time.Second) // 留 1 秒缓冲

	// 步骤 2: 创建 Task
	createBody := `{"name": "timestamp test", "prompt": "test prompt"}`
	resp, err := http.Post(
		testServer.URL+"/api/v1/tasks",
		"application/json",
		bytes.NewBufferString(createBody),
	)
	if err != nil {
		t.Fatalf("TC-TASK-CREATE-007: HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 步骤 3: 记录当前时间 T2
	t2 := time.Now().Add(time.Second) // 留 1 秒缓冲

	// 验证 HTTP 状态码 = 201
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("TC-TASK-CREATE-007: HTTP 状态码 = %d, 期望 201", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	taskID := result["id"].(string)

	// 验证响应 created_at: T1 ≤ created_at ≤ T2
	createdAtStr, ok := result["created_at"].(string)
	if !ok {
		t.Errorf("TC-TASK-CREATE-007: 响应 created_at 不存在")
	} else {
		createdAt, err := time.Parse(time.RFC3339Nano, createdAtStr)
		if err != nil {
			createdAt, err = time.Parse(time.RFC3339, createdAtStr)
		}
		if err != nil {
			t.Errorf("TC-TASK-CREATE-007: 响应 created_at 格式错误: %s", createdAtStr)
		} else {
			if createdAt.Before(t1) || createdAt.After(t2) {
				t.Errorf("TC-TASK-CREATE-007: 响应 created_at = %v, 期望在 [%v, %v] 之间", createdAt, t1, t2)
			}
		}
	}

	// 验证 DB: tasks.created_at 和 tasks.updated_at
	task, _ := testStore.GetTask(ctx, taskID)
	if task == nil {
		t.Errorf("TC-TASK-CREATE-007: DB 中不存在记录")
	} else {
		// T1 ≤ created_at ≤ T2
		if task.CreatedAt.Before(t1) || task.CreatedAt.After(t2) {
			t.Errorf("TC-TASK-CREATE-007: DB tasks.created_at = %v, 期望在 [%v, %v] 之间", task.CreatedAt, t1, t2)
		}

		// updated_at = created_at
		if !task.UpdatedAt.Equal(task.CreatedAt) {
			// 允许微小差异（毫秒级）
			diff := task.UpdatedAt.Sub(task.CreatedAt)
			if diff > time.Second || diff < -time.Second {
				t.Errorf("TC-TASK-CREATE-007: DB tasks.updated_at = %v, 期望等于 created_at = %v", task.UpdatedAt, task.CreatedAt)
			}
		}
	}

	// 清理
	_ = testStore.DeleteTask(ctx, taskID)
}

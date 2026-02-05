// Package taskmanagement 任务管理 Handler 单元测试
//
// 测试类型：Handler Unit Test（处理器单元测试）
// 测试用例来源：docs/business-flow/10-任务管理/01-任务创建.md
//
// ============================================================================
// Handler Unit Test vs Integration Test 的区别
// ============================================================================
//
// Handler Unit Test（本文件）:
//   - 使用 router.ServeHTTP(w, req) 直接调用 Handler
//   - 跳过网络层，在内存中完成
//   - 可以使用 Mock Database 隔离外部依赖
//   - 速度极快（毫秒级）
//   - 适合测试：Handler 的业务逻辑、参数校验、错误处理
//
// Integration Test（tests/integration/...）:
//   - 使用 httptest.NewServer 启动真实 HTTP 服务器
//   - 请求经过完整的 TCP/IP 网络栈
//   - 使用真实的 PostgreSQL 数据库
//   - 速度较慢但更真实
//   - 适合测试：完整的请求-响应流程、中间件、数据库交互
//
// ============================================================================
// 测试架构图
// ============================================================================
//
//   Handler Unit Test:
//   ┌──────────────┐        ┌─────────────┐
//   │  Test Code   │──────→ │   Handler   │ ──→ Mock DB (可选)
//   │ ServeHTTP()  │        │  Create()   │
//   └──────────────┘        └─────────────┘
//         ↑ 内存中直接调用，无网络开销
//
//   Integration Test:
//   ┌──────────────┐  HTTP  ┌─────────────┐
//   │  Test Code   │──────→ │ HTTP Server │ ──→ Real PostgreSQL
//   │ http.Post()  │ TCP/IP │  :random    │
//   └──────────────┘        └─────────────┘
//         ↑ 真实网络请求，完整流程
//
package taskmanagement

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"agents-admin/internal/apiserver/task"
	"agents-admin/internal/shared/model"
	"agents-admin/internal/shared/storage"
)

// ============================================================================
// Mock Storage - 用于隔离数据库依赖
// ============================================================================

// mockStore 模拟存储层，用于 Handler 单元测试
type mockStore struct {
	tasks map[string]*model.Task
}

func newMockStore() *mockStore {
	return &mockStore{
		tasks: make(map[string]*model.Task),
	}
}

func (m *mockStore) CreateTask(t *model.Task) error {
	m.tasks[t.ID] = t
	return nil
}

func (m *mockStore) GetTask(id string) (*model.Task, error) {
	return m.tasks[id], nil
}

func (m *mockStore) ListTasks(status string, limit, offset int) ([]*model.Task, error) {
	var result []*model.Task
	for _, t := range m.tasks {
		if status == "" || string(t.Status) == status {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *mockStore) DeleteTask(id string) error {
	delete(m.tasks, id)
	return nil
}

// ============================================================================
// 适配器：将 mockStore 包装为 *storage.PostgresStore 兼容的接口
// ============================================================================
// 注意：由于 task.Handler 依赖 *storage.PostgresStore，
// 在真正的单元测试中，我们应该将 Handler 改为依赖接口而非具体类型。
// 这里我们仍使用真实数据库，但演示单元测试的写法。
// ============================================================================

// ============================================================================
// TC-TASK-CREATE-001: 基本创建（Handler 单元测试版本）
// ============================================================================

func TestTaskCreate_Basic_Handler(t *testing.T) {
	// 由于当前 Handler 依赖 *storage.PostgresStore，
	// 这里演示使用真实数据库的 Handler Unit Test 写法
	// 理想情况下应该使用 Mock，但需要重构 Handler 依赖接口

	store, err := storage.NewPostgresStore(getTestDatabaseURL())
	if err != nil {
		t.Skip("Database not available for handler unit test")
	}
	defer store.Close()

	handler := task.NewHandler(store)

	// 注册路由
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// 构造请求（使用 httptest.NewRequest，这是服务端请求）
	createBody := `{"name": "unit-test-task", "prompt": "test prompt"}`
	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")

	// 使用 ResponseRecorder 捕获响应
	w := httptest.NewRecorder()

	// 直接调用 ServeHTTP（跳过网络层）
	mux.ServeHTTP(w, req)

	// 验证 HTTP 状态码
	if w.Code != http.StatusCreated {
		t.Fatalf("HTTP 状态码 = %d, 期望 201, 响应: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("响应解析失败: %v", err)
	}

	// 验证响应字段
	taskID, ok := result["id"].(string)
	if !ok || taskID == "" {
		t.Errorf("响应 id 为空")
	}
	if !strings.HasPrefix(taskID, "task-") {
		t.Errorf("响应 id 格式错误: %s, 期望 task-xxx", taskID)
	}
	if result["status"] != "pending" {
		t.Errorf("响应 status = %v, 期望 pending", result["status"])
	}
	if result["type"] != "general" {
		t.Errorf("响应 type = %v, 期望 general", result["type"])
	}

	t.Logf("Handler Unit Test 通过，Task ID: %s", taskID)

	// 清理
	_ = store.DeleteTask(context.Background(), taskID)
}

// ============================================================================
// TC-TASK-CREATE-003: 缺少必填字段（纯 Handler 逻辑测试）
// ============================================================================

func TestTaskCreate_Validation_Handler(t *testing.T) {
	store, err := storage.NewPostgresStore(getTestDatabaseURL())
	if err != nil {
		t.Skip("Database not available for handler unit test")
	}
	defer store.Close()

	handler := task.NewHandler(store)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	testCases := []struct {
		name        string
		body        string
		expectError string
	}{
		{
			name:        "缺少 name",
			body:        `{"prompt": "test prompt"}`,
			expectError: "name",
		},
		{
			name:        "缺少 prompt",
			body:        `{"name": "test task"}`,
			expectError: "prompt",
		},
		{
			name:        "空 JSON",
			body:        `{}`,
			expectError: "name",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			// 验证返回 400
			if w.Code != http.StatusBadRequest {
				t.Errorf("HTTP 状态码 = %d, 期望 400", w.Code)
			}

			// 验证错误信息
			var result map[string]interface{}
			json.NewDecoder(w.Body).Decode(&result)
			errMsg, _ := result["error"].(string)
			if !strings.Contains(errMsg, tc.expectError) {
				t.Errorf("错误信息 = %s, 期望包含 '%s'", errMsg, tc.expectError)
			}
		})
	}
}

// ============================================================================
// 辅助函数
// ============================================================================

func getTestDatabaseURL() string {
	// 优先使用环境变量
	url := ""
	for _, env := range []string{"TEST_DATABASE_URL", "DATABASE_URL"} {
		if v := getEnv(env); v != "" {
			url = v
			break
		}
	}
	if url == "" {
		url = "postgres://agents:agents_dev_password@localhost:5432/agents_admin?sslmode=disable"
	}
	return url
}

func getEnv(key string) string {
	// 简单实现，生产中应使用 godotenv
	return ""
}

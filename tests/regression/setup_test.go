// Package regression 回归测试用例集
//
// 本包包含项目的核心功能回归测试，用于：
//   - 架构重构前后的功能验证
//   - 持续集成中的功能回归检查
//   - 新功能开发后的全量验证
//
// 测试文件组织：
//   - setup_test.go     - 测试基础设施和初始化
//   - task_test.go      - Task 生命周期测试
//   - run_test.go       - Run 执行管理测试
//   - event_test.go     - Event 事件流测试
//   - node_test.go      - Node 节点管理测试
//   - account_test.go   - Account 账号管理测试
//   - proxy_test.go     - Proxy 代理管理测试
//   - instance_test.go  - Instance 实例管理测试
//   - terminal_test.go  - Terminal 终端会话测试
//   - health_test.go    - 健康检查和监控测试
//   - error_test.go     - 错误处理测试
//   - consistency_test.go - 数据一致性测试
//
// 运行方式：
//   go test -v ./tests/regression/...
//
// 环境要求：
//   - PostgreSQL 数据库可用
//   - Redis 可用（可选）
package regression

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"agents-admin/internal/api"
	"agents-admin/internal/storage"
)

// ============================================================================
// 全局测试基础设施
// ============================================================================

var (
	testStore   *storage.PostgresStore
	testRedis   *storage.RedisStore
	testHandler *api.Handler
	testRouter  http.Handler
)

// TestMain 测试入口，初始化测试环境
func TestMain(m *testing.M) {
	// 从环境变量获取数据库连接
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://agents:agents_dev_password@localhost:5432/agents_admin?sslmode=disable"
	}

	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6380/0"
	}

	var err error

	// 初始化 PostgreSQL
	testStore, err = storage.NewPostgresStore(dbURL)
	if err != nil {
		// 数据库不可用时跳过测试
		os.Exit(0)
	}
	defer testStore.Close()

	// 初始化 Redis（可选）
	testRedis, _ = storage.NewRedisStoreFromURL(redisURL)
	if testRedis != nil {
		defer testRedis.Close()
	}

	// 初始化 Handler
	testHandler = api.NewHandler(testStore, testRedis, nil)
	testRouter = testHandler.Router()

	os.Exit(m.Run())
}

// ============================================================================
// 测试辅助函数
// ============================================================================

// makeRequest 创建并执行 HTTP 请求
func makeRequest(method, path string, body interface{}) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	return w
}

// makeRequestWithString 使用字符串 body 创建请求
func makeRequestWithString(method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	return w
}

// parseJSONResponse 解析 JSON 响应
func parseJSONResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	return resp
}

// skipIfNoDatabase 如果数据库不可用则跳过测试
func skipIfNoDatabase(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}
}

// skipIfNoRedis 如果 Redis 不可用则跳过测试
func skipIfNoRedis(t *testing.T) {
	if testRedis == nil {
		t.Skip("Redis not available")
	}
}

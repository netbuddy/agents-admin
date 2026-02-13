// Package testutil 提供测试共享基础设施
//
// 包含两类工具：
//   - InProcEnv: 进程内测试环境（用于 handler / integration 测试）
//   - E2EClient: 外部 HTTP 客户端（用于 E2E 验收测试，见 e2e.go）
package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"agents-admin/internal/apiserver/server"
	"agents-admin/internal/shared/infra"
	"agents-admin/internal/shared/storage"
	"agents-admin/internal/shared/storage/mongostore"
)

// InProcEnv 进程内测试环境（httptest + 真实 DB）
type InProcEnv struct {
	Store   storage.PersistentStore
	Redis   *infra.RedisInfra
	Handler *server.Handler
	Router  http.Handler
	Driver  string
}

// SetupInProcEnv 初始化进程内测试环境
// 返回 error 表示数据库不可用，调用者应 os.Exit(0) 跳过测试
func SetupInProcEnv() (*InProcEnv, error) {
	driver := os.Getenv("TEST_DB_DRIVER")
	if driver == "" {
		driver = "mongodb"
	}

	var store storage.PersistentStore
	var err error

	switch driver {
	case "mongodb":
		mongoURI := os.Getenv("TEST_MONGO_URI")
		if mongoURI == "" {
			mongoURI = "mongodb://localhost:27017"
		}
		mongoDBName := os.Getenv("TEST_MONGO_DB")
		if mongoDBName == "" {
			mongoDBName = "agents_admin_test"
		}
		store, err = mongostore.NewStore(mongoURI, mongoDBName)
	case "postgres":
		dbURL := os.Getenv("TEST_DATABASE_URL")
		if dbURL == "" {
			dbURL = "postgres://agents:agents_dev_password@localhost:5432/agents_admin?sslmode=disable"
		}
		store, err = storage.NewPostgresStore(dbURL)
	default:
		return nil, fmt.Errorf("unsupported TEST_DB_DRIVER: %s", driver)
	}

	if err != nil {
		return nil, fmt.Errorf("database init failed (%s): %w", driver, err)
	}

	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6380/0"
	}
	redisInfra, _ := infra.NewRedisInfra(redisURL)

	var cacheStore storage.CacheStore
	if redisInfra != nil {
		cacheStore = redisInfra
	} else {
		cacheStore = storage.NewNoOpCacheStore()
	}

	handler := server.NewHandler(store, cacheStore)
	router := handler.Router()

	fmt.Fprintf(os.Stderr, "test env: driver=%s\n", driver)

	return &InProcEnv{
		Store:   store,
		Redis:   redisInfra,
		Handler: handler,
		Router:  router,
		Driver:  driver,
	}, nil
}

// Close 关闭测试环境资源
func (e *InProcEnv) Close() {
	if e.Store != nil {
		e.Store.Close()
	}
	if e.Redis != nil {
		e.Redis.Close()
	}
}

// SkipIfNoDatabase 如果数据库不可用则跳过测试
func (e *InProcEnv) SkipIfNoDatabase(t *testing.T) {
	t.Helper()
	if e == nil || e.Store == nil {
		t.Skip("Database not available")
	}
}

// MakeRequest 创建并执行 HTTP 请求
func (e *InProcEnv) MakeRequest(method, path string, body interface{}) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	e.Router.ServeHTTP(w, req)
	return w
}

// MakeRequestWithString 使用字符串 body 创建请求
func (e *InProcEnv) MakeRequestWithString(method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.Router.ServeHTTP(w, req)
	return w
}

// ParseJSONResponse 解析 httptest JSON 响应
func ParseJSONResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	return resp
}

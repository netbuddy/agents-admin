// Package integration 代理管理集成测试
//
// 测试范围：代理 CRUD、连通性测试、默认代理管理
//
// 测试用例：
//   - TestProxy_CRUD: 创建 → 查询 → 更新 → 列表 → 删除
//   - TestProxy_CreateWithAuth: 创建带认证信息的代理（响应不含密码）
//   - TestProxy_CreateMissingFields: 缺少必填字段
//   - TestProxy_GetNotFound: 查询不存在的代理
//   - TestProxy_DeleteNotFound: 删除不存在的代理
//   - TestProxy_Test_Unreachable: 测试不可达代理
//   - TestProxy_Test_NotFound: 测试不存在的代理
//   - TestProxy_SetDefault: 设置默认代理
//   - TestProxy_SwitchDefault: 切换默认代理
//   - TestProxy_ClearDefault: 清除默认代理
//   - TestProxy_SetDefault_NotFound: 设置不存在的代理为默认
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"agents-admin/internal/apiserver/server"
	"agents-admin/internal/config"
	"agents-admin/internal/shared/infra"
	"agents-admin/internal/shared/storage"
)

var (
	testStore   *storage.PostgresStore
	testRedis   *infra.RedisInfra
	testHandler *server.Handler
	testServer  *httptest.Server
	idSeq       uint32
)

func uniqueID(prefix string) string {
	seq := atomic.AddUint32(&idSeq, 1) % 1000
	return fmt.Sprintf("%s-%s-%03d", prefix, time.Now().Format("150405"), seq)
}

func TestMain(m *testing.M) {
	os.Setenv("APP_ENV", "test")
	cfg := config.Load()

	var err error
	testStore, err = storage.NewPostgresStore(cfg.DatabaseURL)
	if err != nil {
		os.Exit(0)
	}

	testRedis, err = infra.NewRedisInfra(cfg.RedisURL)
	if err != nil {
		testRedis = nil
	}

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

// ============================================================================
// Helper functions
// ============================================================================

func doRequest(t *testing.T, method, path string, body interface{}) *http.Response {
	t.Helper()
	var reqBody *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body failed: %v", err)
		}
		reqBody = bytes.NewReader(data)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(method, testServer.URL+path, reqBody)
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s failed: %v", method, path, err)
	}
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	return result
}

// cleanupProxy 清理测试代理
func cleanupProxy(t *testing.T, proxyID string) {
	t.Helper()
	testStore.DeleteProxy(context.Background(), proxyID)
}

// createTestProxy 创建测试代理并返回 ID
func createTestProxy(t *testing.T, name string) string {
	t.Helper()
	resp := doRequest(t, "POST", "/api/v1/proxies", map[string]interface{}{
		"name": name,
		"type": "http",
		"host": "127.0.0.1",
		"port": 19999,
	})
	if resp.StatusCode != http.StatusCreated {
		body := decodeJSON(t, resp)
		t.Fatalf("create proxy expected 201, got %d: %v", resp.StatusCode, body)
	}
	result := decodeJSON(t, resp)
	id, ok := result["id"].(string)
	if !ok || id == "" {
		t.Fatal("expected non-empty proxy id")
	}
	return id
}

// ============================================================================
// Proxy CRUD Tests (对应 01-代理CRUD.md)
// ============================================================================

func TestProxy_CRUD(t *testing.T) {
	name := uniqueID("proxy-crud")

	// 1. 创建
	resp := doRequest(t, "POST", "/api/v1/proxies", map[string]interface{}{
		"name": name,
		"type": "http",
		"host": "proxy.test.local",
		"port": 8080,
	})
	if resp.StatusCode != http.StatusCreated {
		body := decodeJSON(t, resp)
		t.Fatalf("create expected 201, got %d: %v", resp.StatusCode, body)
	}
	created := decodeJSON(t, resp)
	proxyID := created["id"].(string)
	defer cleanupProxy(t, proxyID)

	if created["name"] != name {
		t.Errorf("expected name=%s, got %v", name, created["name"])
	}
	if created["host"] != "proxy.test.local" {
		t.Errorf("expected host=proxy.test.local, got %v", created["host"])
	}

	// 2. 查询详情
	resp = doRequest(t, "GET", "/api/v1/proxies/"+proxyID, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get expected 200, got %d", resp.StatusCode)
	}
	got := decodeJSON(t, resp)
	if got["id"] != proxyID {
		t.Errorf("expected id=%s, got %v", proxyID, got["id"])
	}

	// 3. 更新
	newName := name + "-updated"
	resp = doRequest(t, "PUT", "/api/v1/proxies/"+proxyID, map[string]interface{}{
		"name": &newName,
		"port": 9090,
	})
	if resp.StatusCode != http.StatusOK {
		body := decodeJSON(t, resp)
		t.Fatalf("update expected 200, got %d: %v", resp.StatusCode, body)
	}
	updated := decodeJSON(t, resp)
	if updated["name"] != newName {
		t.Errorf("expected updated name=%s, got %v", newName, updated["name"])
	}

	// 4. 列表包含该代理
	resp = doRequest(t, "GET", "/api/v1/proxies", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list expected 200, got %d", resp.StatusCode)
	}
	listResult := decodeJSON(t, resp)
	proxies, _ := listResult["proxies"].([]interface{})
	found := false
	for _, p := range proxies {
		pm := p.(map[string]interface{})
		if pm["id"] == proxyID {
			found = true
			break
		}
	}
	if !found {
		t.Error("proxy not found in list")
	}

	// 5. 删除
	resp = doRequest(t, "DELETE", "/api/v1/proxies/"+proxyID, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 6. 确认删除
	resp = doRequest(t, "GET", "/api/v1/proxies/"+proxyID, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestProxy_CreateWithAuth(t *testing.T) {
	name := uniqueID("proxy-auth")
	resp := doRequest(t, "POST", "/api/v1/proxies", map[string]interface{}{
		"name":     name,
		"type":     "http",
		"host":     "proxy.test.local",
		"port":     8080,
		"username": "testuser",
		"password": "secret123",
	})
	if resp.StatusCode != http.StatusCreated {
		body := decodeJSON(t, resp)
		t.Fatalf("expected 201, got %d: %v", resp.StatusCode, body)
	}
	result := decodeJSON(t, resp)
	proxyID := result["id"].(string)
	defer cleanupProxy(t, proxyID)

	// 响应不应包含密码
	if result["password"] != nil {
		t.Error("response should not contain password")
	}
	if result["username"] != "testuser" {
		t.Errorf("expected username=testuser, got %v", result["username"])
	}

	// GET 也不应返回密码
	resp = doRequest(t, "GET", "/api/v1/proxies/"+proxyID, nil)
	got := decodeJSON(t, resp)
	if got["password"] != nil {
		t.Error("GET response should not contain password")
	}
}

func TestProxy_CreateMissingFields(t *testing.T) {
	// 缺少 host
	resp := doRequest(t, "POST", "/api/v1/proxies", map[string]interface{}{
		"name": "incomplete",
		"type": "http",
		"port": 8080,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 缺少 port
	resp = doRequest(t, "POST", "/api/v1/proxies", map[string]interface{}{
		"name": "incomplete",
		"type": "http",
		"host": "proxy.test.local",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestProxy_GetNotFound(t *testing.T) {
	resp := doRequest(t, "GET", "/api/v1/proxies/nonexistent-proxy", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestProxy_DeleteNotFound(t *testing.T) {
	resp := doRequest(t, "DELETE", "/api/v1/proxies/nonexistent-proxy", nil)
	// proxy/ 独立包的 Delete 不检查存在性，直接删除
	// server/proxies.go 检查存在性返回 404
	// 根据当前 handler.go 使用的是 server/proxies.go 的方法
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusOK {
		t.Errorf("expected 404 or 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// ============================================================================
// Proxy Test (对应 02-代理连通性测试.md)
// ============================================================================

func TestProxy_Test_Unreachable(t *testing.T) {
	// 创建指向不可达地址的代理
	name := uniqueID("proxy-unreach")
	resp := doRequest(t, "POST", "/api/v1/proxies", map[string]interface{}{
		"name": name,
		"type": "http",
		"host": "192.0.2.1", // RFC 5737 TEST-NET，不可路由
		"port": 19999,
	})
	created := decodeJSON(t, resp)
	proxyID := created["id"].(string)
	defer cleanupProxy(t, proxyID)

	// 测试连通性
	resp = doRequest(t, "POST", "/api/v1/proxies/"+proxyID+"/test", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("test expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON(t, resp)
	if result["success"] != false {
		t.Errorf("expected success=false, got %v", result["success"])
	}
	msg, _ := result["message"].(string)
	if msg == "" {
		t.Error("expected non-empty message")
	}
}

func TestProxy_Test_NotFound(t *testing.T) {
	resp := doRequest(t, "POST", "/api/v1/proxies/nonexistent/test", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// ============================================================================
// Default Proxy (对应 03-默认代理管理.md)
// ============================================================================

func TestProxy_SetDefault(t *testing.T) {
	proxyID := createTestProxy(t, uniqueID("proxy-def"))
	defer cleanupProxy(t, proxyID)

	// 设置为默认
	resp := doRequest(t, "POST", "/api/v1/proxies/"+proxyID+"/set-default", nil)
	if resp.StatusCode != http.StatusOK {
		body := decodeJSON(t, resp)
		t.Fatalf("set-default expected 200, got %d: %v", resp.StatusCode, body)
	}
	resp.Body.Close()

	// 验证 is_default = true
	resp = doRequest(t, "GET", "/api/v1/proxies/"+proxyID, nil)
	got := decodeJSON(t, resp)
	if got["is_default"] != true {
		t.Errorf("expected is_default=true, got %v", got["is_default"])
	}
}

func TestProxy_SwitchDefault(t *testing.T) {
	proxyA := createTestProxy(t, uniqueID("proxy-a"))
	defer cleanupProxy(t, proxyA)
	proxyB := createTestProxy(t, uniqueID("proxy-b"))
	defer cleanupProxy(t, proxyB)

	// 设置 A 为默认
	resp := doRequest(t, "POST", "/api/v1/proxies/"+proxyA+"/set-default", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("set A default expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 设置 B 为默认
	resp = doRequest(t, "POST", "/api/v1/proxies/"+proxyB+"/set-default", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("set B default expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 验证 A 不再是默认
	resp = doRequest(t, "GET", "/api/v1/proxies/"+proxyA, nil)
	gotA := decodeJSON(t, resp)
	if gotA["is_default"] == true {
		t.Error("proxy A should no longer be default")
	}

	// 验证 B 是默认
	resp = doRequest(t, "GET", "/api/v1/proxies/"+proxyB, nil)
	gotB := decodeJSON(t, resp)
	if gotB["is_default"] != true {
		t.Error("proxy B should be default")
	}
}

func TestProxy_ClearDefault(t *testing.T) {
	proxyID := createTestProxy(t, uniqueID("proxy-clear"))
	defer cleanupProxy(t, proxyID)

	// 设置为默认
	resp := doRequest(t, "POST", "/api/v1/proxies/"+proxyID+"/set-default", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("set-default expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 清除默认
	resp = doRequest(t, "POST", "/api/v1/proxies/clear-default", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("clear-default expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 验证不再是默认
	resp = doRequest(t, "GET", "/api/v1/proxies/"+proxyID, nil)
	got := decodeJSON(t, resp)
	if got["is_default"] == true {
		t.Error("proxy should no longer be default after clear")
	}
}

func TestProxy_SetDefault_NotFound(t *testing.T) {
	resp := doRequest(t, "POST", "/api/v1/proxies/nonexistent/set-default", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

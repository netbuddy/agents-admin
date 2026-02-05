// Package integration 系统操作集成测试
//
// 测试范围：Operation/Action 统一 API、Agent 类型查询、Account 只读操作
//
// 测试用例：
//   - TestAgentTypes_List: 列出 Agent 类型
//   - TestAgentTypes_Get: 获取指定 Agent 类型
//   - TestOperation_APIKey: API Key 同步操作（自动创建 Account）
//   - TestAccount_Get: 获取账号详情
//   - TestAccount_List: 列出账号
//   - TestAccount_Delete: 删除账号
//   - TestOperation_InvalidType: 无效操作类型
//   - TestOperation_InvalidNode: 无效节点
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
	"agents-admin/internal/shared/cache"
	"agents-admin/internal/shared/infra"
	"agents-admin/internal/shared/model"
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

// ensureTestNode 确保测试节点存在
func ensureTestNode(t *testing.T, nodeID string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	testStore.UpsertNode(ctx, &model.Node{
		ID:            nodeID,
		Status:        model.NodeStatusOnline,
		Labels:        json.RawMessage(`{}`),
		Capacity:      json.RawMessage(`{}`),
		LastHeartbeat: &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if testRedis != nil {
		testRedis.UpdateNodeHeartbeat(ctx, nodeID, &cache.NodeStatus{
			Status:    "online",
			UpdatedAt: now,
		})
	}
}

// cleanupTestNode 清理测试节点
func cleanupTestNode(t *testing.T, nodeID string) {
	t.Helper()
	ctx := context.Background()
	testStore.DeleteNode(ctx, nodeID)
	if testRedis != nil {
		testRedis.Client().Del(ctx, cache.KeyNodeHeartbeat+nodeID).Err()
		testRedis.Client().SRem(ctx, cache.KeyOnlineNodes, nodeID).Err()
	}
}

// cleanupAccount 清理测试账号
func cleanupAccount(t *testing.T, accountID string) {
	t.Helper()
	ctx := context.Background()
	testStore.DeleteAccount(ctx, accountID)
}

// ============================================================================
// Agent Types Tests
// ============================================================================

func TestAgentTypes_List(t *testing.T) {
	resp := doRequest(t, "GET", "/api/v1/agent-types", nil)
	result := decodeJSON(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	agentTypes, ok := result["agent_types"].([]interface{})
	if !ok || len(agentTypes) == 0 {
		t.Fatal("expected non-empty agent_types list")
	}

	foundQwenCode := false
	for _, at := range agentTypes {
		atMap := at.(map[string]interface{})
		if atMap["id"] == "qwen-code" {
			foundQwenCode = true
			if atMap["name"] != "Qwen-Code" {
				t.Errorf("expected name=Qwen-Code, got %v", atMap["name"])
			}
		}
	}
	if !foundQwenCode {
		t.Error("expected qwen-code in agent_types list")
	}
}

func TestAgentTypes_Get(t *testing.T) {
	resp := doRequest(t, "GET", "/api/v1/agent-types/qwen-code", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON(t, resp)
	if result["id"] != "qwen-code" {
		t.Errorf("expected id=qwen-code, got %v", result["id"])
	}

	resp = doRequest(t, "GET", "/api/v1/agent-types/nonexistent", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// ============================================================================
// Operation/Action Tests — API Key（同步）
// ============================================================================

func TestOperation_APIKey_Success(t *testing.T) {
	nodeID := uniqueID("op-node")
	ensureTestNode(t, nodeID)
	defer cleanupTestNode(t, nodeID)

	accName := uniqueID("apikey")
	resp := doRequest(t, "POST", "/api/v1/operations", map[string]interface{}{
		"type":    "api_key",
		"node_id": nodeID,
		"config": map[string]interface{}{
			"name":       accName,
			"agent_type": "qwen-code",
			"api_key":    "sk-test-123",
		},
	})

	if resp.StatusCode != http.StatusCreated {
		body := decodeJSON(t, resp)
		t.Fatalf("expected 201, got %d, body: %v", resp.StatusCode, body)
	}

	result := decodeJSON(t, resp)

	if result["status"] != "success" {
		t.Errorf("expected status=success, got %v", result["status"])
	}
	accountID, ok := result["account_id"].(string)
	if !ok || accountID == "" {
		t.Fatal("expected non-empty account_id")
	}
	defer cleanupAccount(t, accountID)

	// 验证 Account 已创建
	resp = doRequest(t, "GET", "/api/v1/accounts/"+accountID, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	acc := decodeJSON(t, resp)
	if acc["status"] != "authenticated" {
		t.Errorf("expected account status=authenticated, got %v", acc["status"])
	}

	// 验证 Operation 可查询
	opID := result["operation_id"].(string)
	resp = doRequest(t, "GET", "/api/v1/operations/"+opID, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	op := decodeJSON(t, resp)
	if op["status"] != "completed" {
		t.Errorf("expected operation status=completed, got %v", op["status"])
	}
}

// ============================================================================
// Account Read/Delete Tests
// ============================================================================

func TestAccount_Get(t *testing.T) {
	nodeID := uniqueID("op-node")
	ensureTestNode(t, nodeID)
	defer cleanupTestNode(t, nodeID)

	accName := uniqueID("acc-get")
	resp := doRequest(t, "POST", "/api/v1/operations", map[string]interface{}{
		"type": "api_key", "node_id": nodeID,
		"config": map[string]interface{}{
			"name": accName, "agent_type": "qwen-code", "api_key": "sk-123",
		},
	})
	created := decodeJSON(t, resp)
	accountID := created["account_id"].(string)
	defer cleanupAccount(t, accountID)

	resp = doRequest(t, "GET", "/api/v1/accounts/"+accountID, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON(t, resp)
	if result["id"] != accountID {
		t.Errorf("expected id=%s, got %v", accountID, result["id"])
	}

	resp = doRequest(t, "GET", "/api/v1/accounts/nonexistent-id", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestAccount_List(t *testing.T) {
	nodeID := uniqueID("op-node")
	ensureTestNode(t, nodeID)
	defer cleanupTestNode(t, nodeID)

	acc1Name := uniqueID("acc-list1")
	acc2Name := uniqueID("acc-list2")

	resp1 := doRequest(t, "POST", "/api/v1/operations", map[string]interface{}{
		"type": "api_key", "node_id": nodeID,
		"config": map[string]interface{}{
			"name": acc1Name, "agent_type": "qwen-code", "api_key": "sk-1",
		},
	})
	c1 := decodeJSON(t, resp1)
	defer cleanupAccount(t, c1["account_id"].(string))

	resp2 := doRequest(t, "POST", "/api/v1/operations", map[string]interface{}{
		"type": "api_key", "node_id": nodeID,
		"config": map[string]interface{}{
			"name": acc2Name, "agent_type": "qwen-code", "api_key": "sk-2",
		},
	})
	c2 := decodeJSON(t, resp2)
	defer cleanupAccount(t, c2["account_id"].(string))

	resp := doRequest(t, "GET", "/api/v1/accounts", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON(t, resp)
	accounts := result["accounts"].([]interface{})
	if len(accounts) < 2 {
		t.Errorf("expected at least 2 accounts, got %d", len(accounts))
	}
}

func TestAccount_Delete(t *testing.T) {
	nodeID := uniqueID("op-node")
	ensureTestNode(t, nodeID)
	defer cleanupTestNode(t, nodeID)

	accName := uniqueID("acc-del")
	resp := doRequest(t, "POST", "/api/v1/operations", map[string]interface{}{
		"type": "api_key", "node_id": nodeID,
		"config": map[string]interface{}{
			"name": accName, "agent_type": "qwen-code", "api_key": "sk-del",
		},
	})
	created := decodeJSON(t, resp)
	accountID := created["account_id"].(string)

	resp = doRequest(t, "DELETE", "/api/v1/accounts/"+accountID, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = doRequest(t, "GET", "/api/v1/accounts/"+accountID, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = doRequest(t, "DELETE", "/api/v1/accounts/nonexistent-id", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// ============================================================================
// Operation Validation Tests
// ============================================================================

func TestOperation_InvalidType(t *testing.T) {
	resp := doRequest(t, "POST", "/api/v1/operations", map[string]interface{}{
		"type":    "unknown_type",
		"node_id": "node-001",
		"config":  map[string]interface{}{},
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestOperation_InvalidNode(t *testing.T) {
	resp := doRequest(t, "POST", "/api/v1/operations", map[string]interface{}{
		"type":    "oauth",
		"node_id": "nonexistent-node",
		"config": map[string]interface{}{
			"name":       "test",
			"agent_type": "qwen-code",
		},
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

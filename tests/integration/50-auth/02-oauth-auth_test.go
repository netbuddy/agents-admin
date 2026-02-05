// Package integration Operation/Action 认证流程集成测试
//
// 测试范围：OAuth/DeviceCode 异步操作完整生命周期
//
// 测试用例：
//   - TestOperation_OAuth_Success: 创建 OAuth 操作 → running → waiting → success（创建 Account）
//   - TestOperation_OAuth_Failed: OAuth 操作失败（不创建 Account）
//   - TestOperation_DeviceCode_Success: DeviceCode 操作成功
//   - TestAction_NotFound: Action 不存在
//   - TestNodeActions_List: 节点 Action 轮询
//   - TestOperation_List: Operation 列表查询
package integration

import (
	"net/http"
	"testing"
)

// ============================================================================
// OAuth Operation — 完整生命周期
// ============================================================================

func TestOperation_OAuth_Success(t *testing.T) {
	nodeID := uniqueID("op-node")
	ensureTestNode(t, nodeID)
	defer cleanupTestNode(t, nodeID)

	accName := uniqueID("oauth-ok")
	resp := doRequest(t, "POST", "/api/v1/operations", map[string]interface{}{
		"type":    "oauth",
		"node_id": nodeID,
		"config": map[string]interface{}{
			"name":       accName,
			"agent_type": "qwen-code",
		},
	})
	if resp.StatusCode != http.StatusCreated {
		body := decodeJSON(t, resp)
		t.Fatalf("expected 201, got %d, body: %v", resp.StatusCode, body)
	}
	created := decodeJSON(t, resp)
	opID := created["operation_id"].(string)
	actID := created["action_id"].(string)

	if created["status"] != "assigned" {
		t.Errorf("expected status=assigned, got %v", created["status"])
	}

	// 模拟节点管理器上报 running + initializing phase
	resp = doRequest(t, "PATCH", "/api/v1/actions/"+actID, map[string]interface{}{
		"status":   "running",
		"phase":    "initializing",
		"message":  "Preparing authentication environment",
		"progress": 10,
	})
	if resp.StatusCode != http.StatusOK {
		body := decodeJSON(t, resp)
		t.Fatalf("running update expected 200, got %d, body: %v", resp.StatusCode, body)
	}
	runResp := decodeJSON(t, resp)
	if runResp["phase"] != "initializing" {
		t.Errorf("expected phase=initializing, got %v", runResp["phase"])
	}
	if runResp["message"] != "Preparing authentication environment" {
		t.Errorf("expected message, got %v", runResp["message"])
	}

	// 模拟上报 launching_container phase（同一 running status，不同 phase）
	resp = doRequest(t, "PATCH", "/api/v1/actions/"+actID, map[string]interface{}{
		"status":   "running",
		"phase":    "launching_container",
		"message":  "Launching authentication container",
		"progress": 15,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("launching_container update expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 模拟上报 authenticating phase
	resp = doRequest(t, "PATCH", "/api/v1/actions/"+actID, map[string]interface{}{
		"status":   "running",
		"phase":    "authenticating",
		"message":  "Executing OAuth flow",
		"progress": 30,
		"result":   map[string]interface{}{"container_name": "auth-ctr-1"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("authenticating update expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 验证 Operation 状态为 in_progress
	resp = doRequest(t, "GET", "/api/v1/operations/"+opID, nil)
	op := decodeJSON(t, resp)
	if op["status"] != "in_progress" {
		t.Errorf("expected operation status=in_progress, got %v", op["status"])
	}

	// 模拟上报 waiting + waiting_oauth phase
	resp = doRequest(t, "PATCH", "/api/v1/actions/"+actID, map[string]interface{}{
		"status":   "waiting",
		"phase":    "waiting_oauth",
		"message":  "Waiting for user to complete OAuth authorization",
		"progress": 50,
		"result":   map[string]interface{}{"verify_url": "https://oauth.example.com/auth"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("waiting update expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 模拟上报 success + finalizing phase
	volumeName := "qwen-code_" + accName + "_vol"
	resp = doRequest(t, "PATCH", "/api/v1/actions/"+actID, map[string]interface{}{
		"status":   "success",
		"phase":    "finalizing",
		"message":  "Authentication completed successfully",
		"progress": 100,
		"result":   map[string]interface{}{"volume_name": volumeName},
	})
	if resp.StatusCode != http.StatusOK {
		body := decodeJSON(t, resp)
		t.Fatalf("success update expected 200, got %d, body: %v", resp.StatusCode, body)
	}
	resp.Body.Close()

	// 验证 Operation 状态为 completed
	resp = doRequest(t, "GET", "/api/v1/operations/"+opID, nil)
	op = decodeJSON(t, resp)
	if op["status"] != "completed" {
		t.Errorf("expected operation status=completed, got %v", op["status"])
	}

	// 验证 Account 已自动创建
	expectedAccID := "qwen-code_" + accName
	resp = doRequest(t, "GET", "/api/v1/accounts/"+expectedAccID, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected account created, got %d", resp.StatusCode)
	}
	acc := decodeJSON(t, resp)
	defer cleanupAccount(t, expectedAccID)

	if acc["status"] != "authenticated" {
		t.Errorf("expected account status=authenticated, got %v", acc["status"])
	}
}

func TestOperation_OAuth_Failed(t *testing.T) {
	nodeID := uniqueID("op-node")
	ensureTestNode(t, nodeID)
	defer cleanupTestNode(t, nodeID)

	accName := uniqueID("oauth-fail")
	resp := doRequest(t, "POST", "/api/v1/operations", map[string]interface{}{
		"type":    "oauth",
		"node_id": nodeID,
		"config": map[string]interface{}{
			"name":       accName,
			"agent_type": "qwen-code",
		},
	})
	created := decodeJSON(t, resp)
	opID := created["operation_id"].(string)
	actID := created["action_id"].(string)

	// 模拟失败
	resp = doRequest(t, "PATCH", "/api/v1/actions/"+actID, map[string]interface{}{
		"status":   "failed",
		"phase":    "",
		"message":  "Authentication failed: user cancelled",
		"progress": 0,
		"error":    "user cancelled",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("failed update expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 验证 Operation 状态为 failed
	resp = doRequest(t, "GET", "/api/v1/operations/"+opID, nil)
	op := decodeJSON(t, resp)
	if op["status"] != "failed" {
		t.Errorf("expected operation status=failed, got %v", op["status"])
	}

	// 不应创建 Account
	expectedAccID := "qwen-code_" + accName
	resp = doRequest(t, "GET", "/api/v1/accounts/"+expectedAccID, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected no account created on failure, got %d", resp.StatusCode)
		cleanupAccount(t, expectedAccID)
	}
	resp.Body.Close()
}

// ============================================================================
// DeviceCode Operation
// ============================================================================

func TestOperation_DeviceCode_Success(t *testing.T) {
	nodeID := uniqueID("op-node")
	ensureTestNode(t, nodeID)
	defer cleanupTestNode(t, nodeID)

	accName := uniqueID("dc-ok")
	resp := doRequest(t, "POST", "/api/v1/operations", map[string]interface{}{
		"type":    "device_code",
		"node_id": nodeID,
		"config": map[string]interface{}{
			"name":       accName,
			"agent_type": "openai-codex",
		},
	})
	if resp.StatusCode != http.StatusCreated {
		body := decodeJSON(t, resp)
		t.Fatalf("expected 201, got %d, body: %v", resp.StatusCode, body)
	}
	created := decodeJSON(t, resp)
	actID := created["action_id"].(string)

	// 模拟 success + finalizing phase
	volumeName := "openai-codex_" + accName + "_vol"
	resp = doRequest(t, "PATCH", "/api/v1/actions/"+actID, map[string]interface{}{
		"status":   "success",
		"phase":    "finalizing",
		"message":  "Device code authentication completed",
		"progress": 100,
		"result":   map[string]interface{}{"volume_name": volumeName},
	})
	if resp.StatusCode != http.StatusOK {
		body := decodeJSON(t, resp)
		t.Fatalf("success update expected 200, got %d, body: %v", resp.StatusCode, body)
	}
	resp.Body.Close()

	// 验证 Account 已创建
	expectedAccID := "openai-codex_" + accName
	resp = doRequest(t, "GET", "/api/v1/accounts/"+expectedAccID, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected account created, got %d", resp.StatusCode)
	}
	acc := decodeJSON(t, resp)
	defer cleanupAccount(t, expectedAccID)

	if acc["status"] != "authenticated" {
		t.Errorf("expected account status=authenticated, got %v", acc["status"])
	}
}

// ============================================================================
// Action Queries
// ============================================================================

func TestAction_NotFound(t *testing.T) {
	resp := doRequest(t, "PATCH", "/api/v1/actions/nonexistent-action", map[string]interface{}{
		"status": "success",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestAction_AlreadyTerminal(t *testing.T) {
	nodeID := uniqueID("op-node")
	ensureTestNode(t, nodeID)
	defer cleanupTestNode(t, nodeID)

	accName := uniqueID("term-test")
	resp := doRequest(t, "POST", "/api/v1/operations", map[string]interface{}{
		"type":    "api_key",
		"node_id": nodeID,
		"config": map[string]interface{}{
			"name": accName, "agent_type": "qwen-code", "api_key": "sk-test",
		},
	})
	created := decodeJSON(t, resp)
	actID := created["action_id"].(string)
	defer cleanupAccount(t, created["account_id"].(string))

	// API Key 的 Action 已经是 success，再次更新应冲突
	resp = doRequest(t, "PATCH", "/api/v1/actions/"+actID, map[string]interface{}{
		"status": "failed",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409 for terminal action, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// ============================================================================
// Node Actions — 统一轮询
// ============================================================================

func TestNodeActions_List(t *testing.T) {
	nodeID := uniqueID("poll-node")
	ensureTestNode(t, nodeID)
	defer cleanupTestNode(t, nodeID)

	// 创建 OAuth 操作
	accName := uniqueID("poll-acc")
	resp := doRequest(t, "POST", "/api/v1/operations", map[string]interface{}{
		"type":    "oauth",
		"node_id": nodeID,
		"config": map[string]interface{}{
			"name":       accName,
			"agent_type": "qwen-code",
		},
	})
	if resp.StatusCode != http.StatusCreated {
		body := decodeJSON(t, resp)
		t.Fatalf("expected 201, got %d, body: %v", resp.StatusCode, body)
	}
	resp.Body.Close()

	// 轮询节点的 assigned actions
	resp = doRequest(t, "GET", "/api/v1/nodes/"+nodeID+"/actions", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON(t, resp)
	actions, ok := result["actions"].([]interface{})
	if !ok || len(actions) == 0 {
		t.Fatal("expected at least one assigned action")
	}

	// 验证 action 包含 operation 信息
	actMap := actions[0].(map[string]interface{})
	if actMap["operation"] == nil {
		t.Error("expected operation info in action")
	}
	opMap := actMap["operation"].(map[string]interface{})
	if opMap["type"] != "oauth" {
		t.Errorf("expected operation type=oauth, got %v", opMap["type"])
	}
}

// ============================================================================
// Operation List
// ============================================================================

func TestOperation_List(t *testing.T) {
	nodeID := uniqueID("list-node")
	ensureTestNode(t, nodeID)
	defer cleanupTestNode(t, nodeID)

	// 创建一个操作
	resp := doRequest(t, "POST", "/api/v1/operations", map[string]interface{}{
		"type":    "api_key",
		"node_id": nodeID,
		"config": map[string]interface{}{
			"name": uniqueID("list-acc"), "agent_type": "qwen-code", "api_key": "sk-list",
		},
	})
	created := decodeJSON(t, resp)
	defer cleanupAccount(t, created["account_id"].(string))

	resp = doRequest(t, "GET", "/api/v1/operations", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON(t, resp)
	if result["operations"] == nil {
		t.Error("expected operations in response")
	}

	// 按类型过滤
	resp = doRequest(t, "GET", "/api/v1/operations?type=api_key", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

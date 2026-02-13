package operation

import (
	"net/http"
	"testing"

	"agents-admin/tests/testutil"
)

// ============================================================================
// Agent 类型 API
// ============================================================================

// TestAgentType_List 验证 Agent 类型列表
func TestAgentType_List(t *testing.T) {
	resp, err := c.Get("/api/v1/agent-types")
	if err != nil {
		t.Fatalf("List agent types failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	result := testutil.ReadJSON(resp)
	types, _ := result["agent_types"].([]interface{})
	if len(types) == 0 {
		t.Error("expected at least 1 agent type")
	}
}

// ============================================================================
// 账号 API
// ============================================================================

// TestAccount_List 验证账号列表
func TestAccount_List(t *testing.T) {
	resp, err := c.Get("/api/v1/accounts")
	if err != nil {
		t.Fatalf("List accounts failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Operation/Action API
// ============================================================================

// TestOperation_List 验证操作列表查询
func TestOperation_List(t *testing.T) {
	resp, err := c.Get("/api/v1/operations")
	if err != nil {
		t.Fatalf("List operations failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// TestOperation_Create_InvalidType 验证不支持的操作类型
func TestOperation_Create_InvalidType(t *testing.T) {
	payload := map[string]interface{}{
		"type":    "invalid_type",
		"config":  map[string]interface{}{"name": "test", "agent_type": "qwen-code"},
		"node_id": "node-001",
	}
	resp, err := c.Post("/api/v1/operations", payload)
	if err != nil {
		t.Fatalf("Create operation failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// TC-OAUTH-INIT-002: Agent 类型不支持 OAuth
func TestOperation_Create_OAuth_UnsupportedMethod(t *testing.T) {
	// gemini 不支持 oauth（只支持 api_key + device_code）
	payload := map[string]interface{}{
		"type":    "oauth",
		"config":  map[string]interface{}{"name": "test", "agent_type": "gemini"},
		"node_id": "node-001",
	}
	resp, err := c.Post("/api/v1/operations", payload)
	if err != nil {
		t.Fatalf("Create operation failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	result := testutil.ReadJSON(resp)
	errMsg, _ := result["error"].(string)
	if errMsg == "" {
		t.Error("expected error message about unsupported method")
	}
	t.Logf("Error: %s", errMsg)
}

// TC-OAUTH-INIT-003: 节点不存在
func TestOperation_Create_OAuth_NodeNotFound(t *testing.T) {
	payload := map[string]interface{}{
		"type":    "oauth",
		"config":  map[string]interface{}{"name": "test", "agent_type": "qwen-code"},
		"node_id": "nonexistent-node",
	}
	resp, err := c.Post("/api/v1/operations", payload)
	if err != nil {
		t.Fatalf("Create operation failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	result := testutil.ReadJSON(resp)
	errMsg, _ := result["error"].(string)
	if errMsg == "" {
		t.Error("expected error about node not found")
	}
	t.Logf("Error: %s", errMsg)
}

// TC-OAUTH-INIT-001: 发起 OAuth 认证成功（需要在线节点）
func TestOperation_Create_OAuth_Success(t *testing.T) {
	// 先查在线节点
	nodesResp, err := c.Get("/api/v1/nodes")
	if err != nil {
		t.Fatalf("List nodes failed: %v", err)
	}
	nodesResult := testutil.ReadJSON(nodesResp)
	nodes, _ := nodesResult["nodes"].([]interface{})

	var nodeID string
	for _, n := range nodes {
		nm, _ := n.(map[string]interface{})
		if nm["status"] == "online" {
			nodeID, _ = nm["id"].(string)
			break
		}
	}
	if nodeID == "" {
		t.Skip("No online node available, skipping OAuth operation test")
	}

	// 创建 OAuth Operation
	payload := map[string]interface{}{
		"type":    "oauth",
		"config":  map[string]interface{}{"name": "e2e-test", "agent_type": "qwen-code"},
		"node_id": nodeID,
	}
	resp, err := c.Post("/api/v1/operations", payload)
	if err != nil {
		t.Fatalf("Create operation failed: %v", err)
	}
	result := testutil.ReadJSON(resp)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %v", resp.StatusCode, result)
	}

	opID, _ := result["operation_id"].(string)
	actID, _ := result["action_id"].(string)
	status, _ := result["status"].(string)

	if opID == "" {
		t.Error("response missing operation_id")
	}
	if actID == "" {
		t.Error("response missing action_id")
	}
	if status != "assigned" {
		t.Errorf("expected status=assigned, got %s", status)
	}

	t.Logf("Created OAuth operation: %s (action: %s)", opID, actID)

	// 验证 Operation 详情
	opResp, err := c.Get("/api/v1/operations/" + opID)
	if err != nil {
		t.Fatalf("Get operation failed: %v", err)
	}
	if opResp.StatusCode != http.StatusOK {
		t.Errorf("Get operation returned %d", opResp.StatusCode)
	}

	// 验证 Action 详情
	actResp, err := c.Get("/api/v1/actions/" + actID)
	if err != nil {
		t.Fatalf("Get action failed: %v", err)
	}
	actResult := testutil.ReadJSON(actResp)
	if actResp.StatusCode != http.StatusOK {
		t.Errorf("Get action returned %d", actResp.StatusCode)
	}
	actStatus, _ := actResult["status"].(string)
	t.Logf("Action status: %s", actStatus)
}

// TC-OAUTH-CALLBACK-001: 模拟认证成功回调（通过 PATCH /actions/{id}）
func TestOperation_AuthCallback_Success(t *testing.T) {
	// 先查在线节点
	nodesResp, err := c.Get("/api/v1/nodes")
	if err != nil {
		t.Fatalf("List nodes failed: %v", err)
	}
	nodesResult := testutil.ReadJSON(nodesResp)
	nodes, _ := nodesResult["nodes"].([]interface{})

	var nodeID string
	for _, n := range nodes {
		nm, _ := n.(map[string]interface{})
		if nm["status"] == "online" {
			nodeID, _ = nm["id"].(string)
			break
		}
	}
	if nodeID == "" {
		t.Skip("No online node available, skipping callback test")
	}

	// 1. 创建 OAuth Operation
	payload := map[string]interface{}{
		"type":    "oauth",
		"config":  map[string]interface{}{"name": "callback-test", "agent_type": "qwen-code"},
		"node_id": nodeID,
	}
	resp, err := c.Post("/api/v1/operations", payload)
	if err != nil {
		t.Fatalf("Create operation failed: %v", err)
	}
	result := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %v", resp.StatusCode, result)
	}
	actID, _ := result["action_id"].(string)

	// 2. 模拟节点管理器上报 success（含 volume_name）
	callbackPayload := map[string]interface{}{
		"status":   "success",
		"phase":    "finalizing",
		"message":  "Authentication completed",
		"progress": 100,
		"result":   map[string]interface{}{"volume_name": "qwen-code_callback_test_vol"},
	}
	callbackResp, err := c.Patch("/api/v1/actions/"+actID, callbackPayload)
	if err != nil {
		t.Fatalf("Callback failed: %v", err)
	}
	callbackResult := testutil.ReadJSON(callbackResp)
	if callbackResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %v", callbackResp.StatusCode, callbackResult)
	}

	// 3. 验证账号已自动创建
	accResp, err := c.Get("/api/v1/accounts/qwen-code_callback_test")
	if err != nil {
		t.Fatalf("Get account failed: %v", err)
	}
	if accResp.StatusCode != http.StatusOK {
		t.Fatalf("Account should be auto-created, got %d", accResp.StatusCode)
	}
	accResult := testutil.ReadJSON(accResp)
	accStatus, _ := accResult["status"].(string)
	if accStatus != "authenticated" {
		t.Errorf("expected account status=authenticated, got %s", accStatus)
	}
	t.Logf("Account auto-created: qwen-code_callback_test (status=%s)", accStatus)

	// 4. 清理：删除测试账号
	c.Delete("/api/v1/accounts/qwen-code_callback_test?purge=true")
}

// TC-OAUTH-CALLBACK-002: Action 不存在
func TestOperation_AuthCallback_ActionNotFound(t *testing.T) {
	payload := map[string]interface{}{
		"status":   "success",
		"progress": 100,
		"result":   map[string]interface{}{"volume_name": "test_vol"},
	}
	resp, err := c.Patch("/api/v1/actions/nonexistent-action", payload)
	if err != nil {
		t.Fatalf("Callback failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

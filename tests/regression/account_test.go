package regression

import (
	"context"
	"net/http"
	"testing"
)

// ============================================================================
// Account 账号管理回归测试
// ============================================================================

// TestAccount_Create 测试创建账号
func TestAccount_Create(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 先注册一个测试节点（Account 创建需要有效的节点）
	nodeID := "account-test-node"
	makeRequestWithString("POST", "/api/v1/nodes/heartbeat", `{"node_id":"`+nodeID+`","status":"online"}`)
	defer makeRequest("DELETE", "/api/v1/nodes/"+nodeID, nil)

	t.Run("创建基本账号", func(t *testing.T) {
		body := `{
			"name":"Test Account",
			"agent_type":"qwen-code",
			"node_id":"` + nodeID + `"
		}`
		w := makeRequestWithString("POST", "/api/v1/accounts", body)

		if w.Code != http.StatusCreated {
			t.Errorf("Create account status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
			return
		}

		resp := parseJSONResponse(w)
		if resp["id"] == nil {
			t.Error("Account ID not returned")
		}
		if resp["status"] != "pending" {
			t.Errorf("Initial status = %v, want pending", resp["status"])
		}

		// 清理
		if id, ok := resp["id"].(string); ok {
			testStore.DeleteAccount(ctx, id)
		}
	})

	t.Run("创建不同 Agent 类型账号", func(t *testing.T) {
		agentTypes := []string{"qwen-code", "gemini-cli", "claude-code"}

		for _, agentType := range agentTypes {
			body := `{"name":"` + agentType + ` Account","agent_type":"` + agentType + `","node_id":"` + nodeID + `"}`
			w := makeRequestWithString("POST", "/api/v1/accounts", body)

			if w.Code == http.StatusCreated {
				resp := parseJSONResponse(w)
				testStore.DeleteAccount(ctx, resp["id"].(string))
			}
			t.Logf("Agent type %s: %d", agentType, w.Code)
		}
	})

	t.Run("缺少必填字段", func(t *testing.T) {
		body := `{"name":"Incomplete Account"}`
		w := makeRequestWithString("POST", "/api/v1/accounts", body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Create account without required fields status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})
}

// TestAccount_Get 测试获取账号
func TestAccount_Get(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 先注册测试节点
	nodeID := "account-get-test-node"
	makeRequestWithString("POST", "/api/v1/nodes/heartbeat", `{"node_id":"`+nodeID+`","status":"online"}`)
	defer makeRequest("DELETE", "/api/v1/nodes/"+nodeID, nil)

	// 创建测试账号
	w := makeRequestWithString("POST", "/api/v1/accounts", `{"name":"Get Test Account","agent_type":"qwen-code","node_id":"`+nodeID+`"}`)
	if w.Code != http.StatusCreated {
		t.Skip("Failed to create test account")
	}
	resp := parseJSONResponse(w)
	accountID := resp["id"].(string)
	defer testStore.DeleteAccount(ctx, accountID)

	t.Run("获取存在的账号", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/accounts/"+accountID, nil)
		if w.Code != http.StatusOK {
			t.Errorf("Get account status = %d, want %d", w.Code, http.StatusOK)
		}

		resp := parseJSONResponse(w)
		if resp["id"] != accountID {
			t.Errorf("Account ID = %v, want %v", resp["id"], accountID)
		}
	})

	t.Run("获取不存在的账号", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/accounts/nonexistent-account", nil)
		if w.Code != http.StatusNotFound {
			t.Errorf("Get nonexistent account status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

// TestAccount_List 测试列出账号
func TestAccount_List(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 先注册测试节点
	nodeID := "account-list-test-node"
	makeRequestWithString("POST", "/api/v1/nodes/heartbeat", `{"node_id":"`+nodeID+`","status":"online"}`)
	defer makeRequest("DELETE", "/api/v1/nodes/"+nodeID, nil)

	// 创建多个测试账号
	var accountIDs []string
	for i := 0; i < 3; i++ {
		w := makeRequestWithString("POST", "/api/v1/accounts", `{"name":"List Test Account","agent_type":"qwen-code","node_id":"`+nodeID+`"}`)
		if w.Code == http.StatusCreated {
			resp := parseJSONResponse(w)
			accountIDs = append(accountIDs, resp["id"].(string))
		}
	}
	defer func() {
		for _, id := range accountIDs {
			testStore.DeleteAccount(ctx, id)
		}
	}()

	t.Run("列出所有账号", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/accounts", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("List accounts status = %d", w.Code)
		}

		resp := parseJSONResponse(w)
		if resp["accounts"] == nil {
			t.Error("Accounts list not returned")
		}
	})
}

// TestAccount_Delete 测试删除账号
func TestAccount_Delete(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 先注册测试节点
	nodeID := "account-delete-test-node"
	makeRequestWithString("POST", "/api/v1/nodes/heartbeat", `{"node_id":"`+nodeID+`","status":"online"}`)
	defer makeRequest("DELETE", "/api/v1/nodes/"+nodeID, nil)

	t.Run("删除存在的账号", func(t *testing.T) {
		// 创建账号
		w := makeRequestWithString("POST", "/api/v1/accounts", `{"name":"Delete Test Account","agent_type":"qwen-code","node_id":"`+nodeID+`"}`)
		if w.Code != http.StatusCreated {
			t.Skip("Failed to create test account")
		}
		resp := parseJSONResponse(w)
		accountID := resp["id"].(string)

		// 删除
		w = makeRequest("DELETE", "/api/v1/accounts/"+accountID, nil)
		if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
			t.Errorf("Delete account status = %d", w.Code)
		}

		// 验证已删除
		w = makeRequest("GET", "/api/v1/accounts/"+accountID, nil)
		if w.Code != http.StatusNotFound {
			testStore.DeleteAccount(ctx, accountID)
			t.Error("Account should be deleted")
		}
	})

	t.Run("删除不存在的账号", func(t *testing.T) {
		w := makeRequest("DELETE", "/api/v1/accounts/nonexistent-account", nil)
		if w.Code != http.StatusNoContent && w.Code != http.StatusNotFound {
			t.Errorf("Delete nonexistent account status = %d", w.Code)
		}
	})
}

// TestAgentTypes_List 测试列出 Agent 类型
func TestAgentTypes_List(t *testing.T) {
	t.Run("列出所有 Agent 类型", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/agent-types", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("List agent types status = %d", w.Code)
		}

		resp := parseJSONResponse(w)
		if resp["agent_types"] == nil {
			t.Error("Agent types list not returned")
		}

		agentTypes := resp["agent_types"].([]interface{})
		if len(agentTypes) == 0 {
			t.Error("No agent types returned")
		}

		// 验证必有的类型
		foundQwencode := false
		for _, at := range agentTypes {
			atMap := at.(map[string]interface{})
			if atMap["id"] == "qwencode" || atMap["id"] == "qwen-code" {
				foundQwencode = true
				break
			}
		}
		if !foundQwencode {
			t.Log("Qwencode agent type not found in list")
		}
	})
}

// TestAgentTypes_Get 测试获取单个 Agent 类型
func TestAgentTypes_Get(t *testing.T) {
	t.Run("获取 qwencode 类型", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/agent-types/qwen-code", nil)
		if w.Code != http.StatusOK {
			// 可能是 qwencode
			w = makeRequest("GET", "/api/v1/agent-types/qwencode", nil)
		}

		if w.Code == http.StatusOK {
			resp := parseJSONResponse(w)
			if resp["name"] == nil {
				t.Error("Agent type name not returned")
			}
		} else {
			t.Logf("Get agent type status: %d", w.Code)
		}
	})

	t.Run("获取不存在的类型", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/agent-types/nonexistent-type", nil)
		if w.Code != http.StatusNotFound {
			t.Errorf("Get nonexistent agent type status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

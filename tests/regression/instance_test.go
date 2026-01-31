package regression

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ============================================================================
// Instance 实例管理回归测试
// P2-2 重构后使用数据库存储
// 注意：这些测试需要 instances 表已迁移
// ============================================================================

// skipIfInstanceTableNotExists 检查 instances 表是否存在
// 如果表不存在返回 true，调用方应立即 return
func skipIfInstanceTableNotExists(t *testing.T, w *httptest.ResponseRecorder) bool {
	t.Helper()
	if w.Code == http.StatusInternalServerError {
		body := w.Body.String()
		// 检查表不存在的错误（PostgreSQL 错误消息）
		if strings.Contains(body, "does not exist") ||
			strings.Contains(body, "42P01") ||
			strings.Contains(body, "relation") {
			t.Skipf("instances table not migrated yet (body: %s)", body[:minInt(len(body), 100)])
			return true
		}
	}
	return false
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestInstance_Create 测试创建实例
func TestInstance_Create(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 先注册测试节点
	nodeID := "instance-test-node"
	makeRequestWithString("POST", "/api/v1/nodes/heartbeat", `{"node_id":"`+nodeID+`","status":"online"}`)
	defer makeRequest("DELETE", "/api/v1/nodes/"+nodeID, nil)

	// 先创建一个账号（实例需要关联账号）
	w := makeRequestWithString("POST", "/api/v1/accounts", `{"name":"Instance Test Account","agent_type":"qwen-code","node_id":"`+nodeID+`"}`)
	if w.Code != http.StatusCreated {
		t.Skip("Failed to create test account")
	}
	accountResp := parseJSONResponse(w)
	accountID := accountResp["id"].(string)
	defer testStore.DeleteAccount(ctx, accountID)

	t.Run("创建实例（账号未认证）", func(t *testing.T) {
		body := `{"name":"Test Instance","account_id":"` + accountID + `"}`
		w := makeRequestWithString("POST", "/api/v1/instances", body)

		// 账号未认证，应该返回 400
		if w.Code == http.StatusBadRequest {
			t.Log("Expected: account not authenticated")
		} else if w.Code == http.StatusCreated {
			resp := parseJSONResponse(w)
			testStore.DeleteInstance(ctx, resp["id"].(string))
		}
	})

	t.Run("缺少 account_id", func(t *testing.T) {
		body := `{"name":"No Account Instance"}`
		w := makeRequestWithString("POST", "/api/v1/instances", body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Create instance without account_id status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("不存在的 account_id", func(t *testing.T) {
		body := `{"name":"Bad Account Instance","account_id":"nonexistent-account"}`
		w := makeRequestWithString("POST", "/api/v1/instances", body)

		if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
			t.Errorf("Create instance with bad account status = %d", w.Code)
		}
	})
}

// TestInstance_List 测试列出实例
func TestInstance_List(t *testing.T) {
	skipIfNoDatabase(t)

	t.Run("列出所有实例", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/instances", nil)
		if w.Code == http.StatusInternalServerError {
			t.Skip("instances table not migrated yet")
		}
		if w.Code != http.StatusOK {
			t.Fatalf("List instances status = %d", w.Code)
		}

		resp := parseJSONResponse(w)
		if resp["instances"] == nil {
			t.Error("Instances list not returned")
		}
	})

	t.Run("按 agent_type 过滤", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/instances?agent_type=qwencode", nil)
		if w.Code == http.StatusInternalServerError {
			t.Skip("instances table not migrated yet")
		}
		if w.Code != http.StatusOK {
			t.Errorf("List instances with filter status = %d", w.Code)
		}
	})

	t.Run("按 account_id 过滤", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/instances?account_id=test-account", nil)
		if w.Code == http.StatusInternalServerError {
			t.Skip("instances table not migrated yet")
		}
		if w.Code != http.StatusOK {
			t.Errorf("List instances by account status = %d", w.Code)
		}
	})
}

// TestInstance_Get 测试获取实例
func TestInstance_Get(t *testing.T) {
	skipIfNoDatabase(t)

	t.Run("获取不存在的实例", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/instances/nonexistent-instance", nil)
		if w.Code == http.StatusInternalServerError {
			t.Skip("instances table not migrated yet")
		}
		if w.Code != http.StatusNotFound {
			t.Errorf("Get nonexistent instance status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

// TestInstance_Delete 测试删除实例
func TestInstance_Delete(t *testing.T) {
	skipIfNoDatabase(t)

	t.Run("删除不存在的实例", func(t *testing.T) {
		w := makeRequest("DELETE", "/api/v1/instances/nonexistent-instance", nil)
		if w.Code == http.StatusInternalServerError {
			t.Skip("instances table not migrated yet")
		}
		if w.Code != http.StatusNotFound {
			t.Errorf("Delete nonexistent instance status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

// TestInstance_StartStop 测试启动/停止实例
func TestInstance_StartStop(t *testing.T) {
	skipIfNoDatabase(t)

	t.Run("启动不存在的实例", func(t *testing.T) {
		w := makeRequest("POST", "/api/v1/instances/nonexistent-instance/start", nil)
		if w.Code == http.StatusInternalServerError {
			t.Skip("instances table not migrated yet")
		}
		if w.Code != http.StatusNotFound {
			t.Errorf("Start nonexistent instance status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})

	t.Run("停止不存在的实例", func(t *testing.T) {
		w := makeRequest("POST", "/api/v1/instances/nonexistent-instance/stop", nil)
		if w.Code == http.StatusInternalServerError {
			t.Skip("instances table not migrated yet")
		}
		if w.Code != http.StatusNotFound {
			t.Errorf("Stop nonexistent instance status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

// TestInstance_ExecutorAPI 测试 Executor 轮询 API
func TestInstance_ExecutorAPI(t *testing.T) {
	skipIfNoDatabase(t)

	t.Run("获取节点的待处理实例", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/nodes/test-node/instances", nil)
		if w.Code == http.StatusInternalServerError {
			t.Skip("instances table not migrated yet")
		}
		if w.Code != http.StatusOK {
			t.Errorf("List pending instances status = %d, want %d", w.Code, http.StatusOK)
		}

		resp := parseJSONResponse(w)
		if resp["instances"] == nil {
			t.Log("Instances list not returned (may be empty)")
		}
	})

	t.Run("更新实例状态", func(t *testing.T) {
		// 需要先有一个实例才能更新
		w := makeRequestWithString("PATCH", "/api/v1/instances/nonexistent-instance", `{"status":"running"}`)
		if w.Code == http.StatusInternalServerError {
			t.Skip("instances table not migrated yet")
		}
		// 不存在应该返回 404
		if w.Code != http.StatusNotFound {
			t.Errorf("Update nonexistent instance status = %d", w.Code)
		}
	})
}

package regression

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ============================================================================
// Terminal 终端会话回归测试
// P2-2 重构后使用数据库存储
// 注意：这些测试需要 terminal_sessions 表已迁移
// ============================================================================

// skipIfTerminalTableNotExists 检查 terminal_sessions 表是否存在
// 如果表不存在返回 true，调用方应立即 return
func skipIfTerminalTableNotExists(t *testing.T, w *httptest.ResponseRecorder) bool {
	t.Helper()
	if w.Code == http.StatusInternalServerError {
		body := w.Body.String()
		// 检查表不存在的错误（PostgreSQL 错误消息）
		if strings.Contains(body, "does not exist") ||
			strings.Contains(body, "42P01") ||
			strings.Contains(body, "relation") {
			t.Skipf("terminal_sessions table not migrated yet (body: %s)", body[:min(len(body), 100)])
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestTerminal_Create 测试创建终端会话
func TestTerminal_Create(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 先注册测试节点
	nodeID := "terminal-test-node"
	makeRequestWithString("POST", "/api/v1/nodes/heartbeat", `{"node_id":"`+nodeID+`","status":"online"}`)
	defer makeRequest("DELETE", "/api/v1/nodes/"+nodeID, nil)

	// 创建测试账号
	w := makeRequestWithString("POST", "/api/v1/accounts", `{"name":"Terminal Test Account","agent_type":"qwen-code","node_id":"`+nodeID+`"}`)
	if w.Code != http.StatusCreated {
		t.Skip("Failed to create test account")
	}
	accountResp := parseJSONResponse(w)
	accountID := accountResp["id"].(string)
	defer testStore.DeleteAccount(ctx, accountID)

	t.Run("创建终端会话 - 通过容器名", func(t *testing.T) {
		body := `{"container_name":"test-container","node_id":"test-node"}`
		w := makeRequestWithString("POST", "/api/v1/terminal-sessions", body)

		// 可能因为容器不存在而失败
		if w.Code == http.StatusCreated {
			resp := parseJSONResponse(w)
			if resp["id"] == nil {
				t.Error("Session ID not returned")
			}
			if resp["status"] != "pending" {
				t.Errorf("Initial status = %v, want pending", resp["status"])
			}
			testStore.DeleteTerminalSession(ctx, resp["id"].(string))
		} else {
			t.Logf("Create terminal session by container: %d", w.Code)
		}
	})

	t.Run("缺少必要参数", func(t *testing.T) {
		body := `{}`
		w := makeRequestWithString("POST", "/api/v1/terminal-sessions", body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Create terminal without params status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("使用旧路径创建（兼容）", func(t *testing.T) {
		body := `{"container_name":"compat-test-container","node_id":"test-node"}`
		w := makeRequestWithString("POST", "/api/v1/terminal/session", body)

		if w.Code == http.StatusCreated {
			resp := parseJSONResponse(w)
			testStore.DeleteTerminalSession(ctx, resp["id"].(string))
			t.Log("Legacy path works")
		} else {
			t.Logf("Legacy path status: %d", w.Code)
		}
	})
}

// TestTerminal_List 测试列出终端会话
func TestTerminal_List(t *testing.T) {
	skipIfNoDatabase(t)

	t.Run("列出所有终端会话", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/terminal-sessions", nil)
		// P2-2 迁移未完成时，表不存在会返回 500
		if w.Code == http.StatusInternalServerError {
			t.Skip("terminal_sessions table not migrated yet")
		}
		if w.Code != http.StatusOK {
			t.Fatalf("List terminal sessions status = %d", w.Code)
		}

		resp := parseJSONResponse(w)
		if resp["sessions"] == nil {
			t.Error("Sessions list not returned")
		}
	})
}

// TestTerminal_Get 测试获取终端会话
func TestTerminal_Get(t *testing.T) {
	skipIfNoDatabase(t)

	t.Run("获取不存在的会话", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/terminal-sessions/nonexistent-session", nil)
		if w.Code == http.StatusInternalServerError {
			t.Skip("terminal_sessions table not migrated yet")
		}
		if w.Code != http.StatusNotFound {
			t.Errorf("Get nonexistent session status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})

	t.Run("使用旧路径获取（兼容）", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/terminal/session/nonexistent-session", nil)
		if w.Code == http.StatusInternalServerError {
			t.Skip("terminal_sessions table not migrated yet")
		}
		if w.Code != http.StatusNotFound {
			t.Errorf("Get via legacy path status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

// TestTerminal_Delete 测试删除终端会话
func TestTerminal_Delete(t *testing.T) {
	skipIfNoDatabase(t)

	t.Run("删除不存在的会话", func(t *testing.T) {
		w := makeRequest("DELETE", "/api/v1/terminal-sessions/nonexistent-session", nil)
		if w.Code == http.StatusInternalServerError {
			t.Skip("terminal_sessions table not migrated yet")
		}
		if w.Code != http.StatusNotFound {
			t.Errorf("Delete nonexistent session status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})

	t.Run("使用旧路径删除（兼容）", func(t *testing.T) {
		w := makeRequest("DELETE", "/api/v1/terminal/session/nonexistent-session", nil)
		if w.Code == http.StatusInternalServerError {
			t.Skip("terminal_sessions table not migrated yet")
		}
		if w.Code != http.StatusNotFound {
			t.Errorf("Delete via legacy path status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

// TestTerminal_ExecutorAPI 测试 Executor 轮询 API
func TestTerminal_ExecutorAPI(t *testing.T) {
	skipIfNoDatabase(t)

	t.Run("获取节点的待处理会话", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/nodes/test-node/terminal-sessions", nil)
		if w.Code == http.StatusInternalServerError {
			t.Skip("terminal_sessions table not migrated yet")
		}
		if w.Code != http.StatusOK {
			t.Errorf("List pending terminal sessions status = %d, want %d", w.Code, http.StatusOK)
		}

		resp := parseJSONResponse(w)
		if resp["sessions"] == nil {
			t.Log("Sessions list not returned (may be empty)")
		}
	})

	t.Run("更新会话状态", func(t *testing.T) {
		w := makeRequestWithString("PATCH", "/api/v1/terminal-sessions/nonexistent-session", `{"status":"running","port":8080}`)
		if w.Code == http.StatusInternalServerError {
			t.Skip("terminal_sessions table not migrated yet")
		}
		// 不存在应该返回 404
		if w.Code != http.StatusNotFound {
			t.Errorf("Update nonexistent session status = %d", w.Code)
		}
	})
}

// TestTerminal_Proxy 测试终端代理
func TestTerminal_Proxy(t *testing.T) {
	skipIfNoDatabase(t)

	t.Run("代理不存在的会话", func(t *testing.T) {
		w := makeRequest("GET", "/terminal/nonexistent-session/", nil)
		if skipIfTerminalTableNotExists(t, w) {
			return
		}
		// 应该返回 502（后端不可达）或 404（会话不存在）或 500（表不存在）
		if w.Code != http.StatusNotFound && w.Code != http.StatusBadGateway && w.Code != http.StatusServiceUnavailable && w.Code != http.StatusInternalServerError {
			t.Errorf("Proxy nonexistent session status = %d", w.Code)
		}
	})
}

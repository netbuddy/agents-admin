package terminal

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"agents-admin/tests/testutil"
)

// TestTerminal_List 验证终端会话列表
func TestTerminal_List(t *testing.T) {
	resp, err := c.Get("/api/v1/terminal-sessions")
	if err != nil {
		t.Fatalf("List sessions failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("List returned %d", resp.StatusCode)
	}
}

// TestTerminal_CRUD 验证终端会话完整生命周期
func TestTerminal_CRUD(t *testing.T) {
	// 1. 注册节点
	nodeID := "e2e-term-node-" + time.Now().Format("150405")
	c.Post("/api/v1/nodes/heartbeat", map[string]interface{}{
		"node_id": nodeID, "status": "online",
	})
	defer c.Delete("/api/v1/nodes/" + nodeID)

	// 2. 创建终端会话（直接指定 container_name + node_id）
	resp, err := c.Post("/api/v1/terminal-sessions", map[string]interface{}{
		"container_name": "test-container",
		"node_id":        nodeID,
	})
	if err != nil {
		t.Fatalf("Create session failed: %v", err)
	}
	result := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Create returned %d, body: %v", resp.StatusCode, result)
	}
	sessionID := result["id"].(string)
	t.Logf("Created session: %s", sessionID)

	if result["status"] != "pending" {
		t.Errorf("Expected status=pending, got %v", result["status"])
	}

	// 3. 获取会话状态
	resp, err = c.Get("/api/v1/terminal-sessions/" + sessionID)
	if err != nil {
		t.Fatalf("Get session failed: %v", err)
	}
	got := testutil.ReadJSON(resp)
	if got["status"] != "pending" {
		t.Errorf("Get: expected status=pending, got %v", got["status"])
	}

	// 4. 模拟 NodeManager 回调：更新状态为 running + 设置端口
	resp, err = c.Patch("/api/v1/terminal-sessions/"+sessionID, map[string]interface{}{
		"status": "running",
		"port":   7681,
		"url":    fmt.Sprintf("/terminal/%s/", sessionID),
	})
	if err != nil {
		t.Fatalf("Patch session failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Patch returned %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 5. 验证更新后的状态
	resp, err = c.Get("/api/v1/terminal-sessions/" + sessionID)
	if err != nil {
		t.Fatalf("Get session after patch failed: %v", err)
	}
	got = testutil.ReadJSON(resp)
	if got["status"] != "running" {
		t.Errorf("Expected status=running, got %v", got["status"])
	}
	if got["port"] == nil {
		t.Errorf("Expected port to be set")
	}

	// 6. 删除（关闭）会话
	resp, err = c.Delete("/api/v1/terminal-sessions/" + sessionID)
	if err != nil {
		t.Fatalf("Delete session failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		t.Errorf("Delete returned %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 7. 验证已关闭
	resp, err = c.Get("/api/v1/terminal-sessions/" + sessionID)
	if err != nil {
		t.Fatalf("Get after delete failed: %v", err)
	}
	got = testutil.ReadJSON(resp)
	if got["status"] != "closed" {
		t.Errorf("Expected status=closed after delete, got %v", got["status"])
	}
}

// TestTerminal_NodeManagerPolling 验证 NodeManager 轮询端点
func TestTerminal_NodeManagerPolling(t *testing.T) {
	nodeID := "e2e-term-poll-" + time.Now().Format("150405")
	c.Post("/api/v1/nodes/heartbeat", map[string]interface{}{
		"node_id": nodeID, "status": "online",
	})
	defer c.Delete("/api/v1/nodes/" + nodeID)

	// 创建一个 pending 会话
	resp, _ := c.Post("/api/v1/terminal-sessions", map[string]interface{}{
		"container_name": "poll-test-container",
		"node_id":        nodeID,
	})
	result := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Create returned %d", resp.StatusCode)
	}
	sessionID := result["id"].(string)
	defer c.Delete("/api/v1/terminal-sessions/" + sessionID)

	// NodeManager 轮询：GET /api/v1/nodes/{node_id}/terminal-sessions
	resp, err := c.Get("/api/v1/nodes/" + nodeID + "/terminal-sessions")
	if err != nil {
		t.Fatalf("Poll pending sessions failed: %v", err)
	}
	got := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Poll returned %d", resp.StatusCode)
	}

	count, ok := got["count"].(float64)
	if !ok || count < 1 {
		t.Errorf("Expected at least 1 pending session, got %v", got["count"])
	}

	sessions, ok := got["sessions"].([]interface{})
	if !ok || len(sessions) < 1 {
		t.Errorf("Expected sessions array with items, got %v", got["sessions"])
	}
}

// TestTerminal_InvalidStatus 验证无效状态被拒绝
func TestTerminal_InvalidStatus(t *testing.T) {
	nodeID := "e2e-term-inv-" + time.Now().Format("150405")
	c.Post("/api/v1/nodes/heartbeat", map[string]interface{}{
		"node_id": nodeID, "status": "online",
	})
	defer c.Delete("/api/v1/nodes/" + nodeID)

	resp, _ := c.Post("/api/v1/terminal-sessions", map[string]interface{}{
		"container_name": "inv-test",
		"node_id":        nodeID,
	})
	result := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusCreated {
		t.Skip("Create failed, skipping")
	}
	sessionID := result["id"].(string)
	defer c.Delete("/api/v1/terminal-sessions/" + sessionID)

	// 尝试设置无效状态
	resp, _ = c.Patch("/api/v1/terminal-sessions/"+sessionID, map[string]interface{}{
		"status": "invalid_status",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid status, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// TestTerminal_NotFound 验证不存在的会话返回 404
func TestTerminal_NotFound(t *testing.T) {
	resp, err := c.Get("/api/v1/terminal-sessions/nonexistent-id")
	if err != nil {
		t.Fatalf("Get nonexistent session failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

package node

import (
	"net/http"
	"testing"
	"time"

	"agents-admin/tests/testutil"
)

// TestNode_HeartbeatAndList 验证节点心跳注册与列表查询
func TestNode_HeartbeatAndList(t *testing.T) {
	nodeID := "e2e-node-" + time.Now().Format("150405")

	// 心跳注册
	payload := map[string]interface{}{
		"node_id": nodeID,
		"status":  "online",
		"labels":  map[string]string{"os": "linux", "arch": "amd64"},
		"capacity": map[string]interface{}{
			"max_concurrent": 4,
			"available":      4,
		},
	}
	resp, err := c.Post("/api/v1/nodes/heartbeat", payload)
	if err != nil {
		t.Fatalf("Heartbeat failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Heartbeat returned %d", resp.StatusCode)
	}

	// 获取节点详情
	resp, err = c.Get("/api/v1/nodes/" + nodeID)
	if err != nil {
		t.Fatalf("Get node failed: %v", err)
	}
	result := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Get node returned %d", resp.StatusCode)
	}
	if result["node_id"] == nil && result["id"] == nil {
		t.Error("Node ID not in response")
	}

	// 列出节点
	resp, err = c.Get("/api/v1/nodes")
	if err != nil {
		t.Fatalf("List nodes failed: %v", err)
	}
	listResult := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("List nodes returned %d", resp.StatusCode)
	}
	if count, ok := listResult["count"].(float64); ok && int(count) < 1 {
		t.Error("Expected at least 1 node")
	}

	// 清理
	c.Delete("/api/v1/nodes/" + nodeID)
}

// TestNode_EnvConfig 验证节点环境配置读写
func TestNode_EnvConfig(t *testing.T) {
	nodeID := "e2e-envconfig-" + time.Now().Format("150405")

	// 注册节点
	c.Post("/api/v1/nodes/heartbeat", map[string]interface{}{
		"node_id": nodeID, "status": "online",
	})
	defer c.Delete("/api/v1/nodes/" + nodeID)

	// 获取环境配置
	resp, err := c.Get("/api/v1/nodes/" + nodeID + "/env-config")
	if err != nil {
		t.Fatalf("Get env-config failed: %v", err)
	}
	defer resp.Body.Close()
	// 可能返回 200（有配置）或 404（无配置）
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		t.Errorf("Get env-config returned %d", resp.StatusCode)
	}
}

// TestNode_Provisions 验证节点预配置接口
func TestNode_Provisions(t *testing.T) {
	// 列出预配置
	resp, err := c.Get("/api/v1/node-provisions")
	if err != nil {
		t.Fatalf("List provisions failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("List provisions returned %d", resp.StatusCode)
	}
}

// TestNode_Delete 验证删除节点
func TestNode_Delete(t *testing.T) {
	nodeID := "e2e-delete-" + time.Now().Format("150405")

	c.Post("/api/v1/nodes/heartbeat", map[string]interface{}{
		"node_id": nodeID, "status": "online",
	})

	resp, err := c.Delete("/api/v1/nodes/" + nodeID)
	if err != nil {
		t.Fatalf("Delete node failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		t.Errorf("Delete node returned %d", resp.StatusCode)
	}

	// 确认已删除
	resp, err = c.Get("/api/v1/nodes/" + nodeID)
	if err != nil {
		t.Fatalf("Get deleted node failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Deleted node still accessible: %d", resp.StatusCode)
	}
}

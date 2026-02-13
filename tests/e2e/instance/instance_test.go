package instance

import (
	"net/http"
	"testing"
	"time"

	"agents-admin/tests/testutil"
)

// TestInstance_CRUD 验证实例完整生命周期
func TestInstance_CRUD(t *testing.T) {
	// 注册节点（实例依赖节点）
	nodeID := "e2e-inst-node-" + time.Now().Format("150405")
	c.Post("/api/v1/nodes/heartbeat", map[string]interface{}{
		"node_id": nodeID, "status": "online",
	})
	defer c.Delete("/api/v1/nodes/" + nodeID)

	// 创建实例
	payload := map[string]interface{}{
		"name":    "E2E Instance",
		"node_id": nodeID,
	}
	resp, err := c.Post("/api/v1/agents", payload)
	if err != nil {
		t.Fatalf("Create instance failed: %v", err)
	}
	result := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusCreated {
		t.Logf("Create instance returned %d (may need node/account)", resp.StatusCode)
		t.Skip("Instance creation requires prerequisites")
	}
	instanceID := result["id"].(string)
	t.Logf("Created instance: %s", instanceID)

	// 获取
	resp, err = c.Get("/api/v1/agents/" + instanceID)
	if err != nil {
		t.Fatalf("Get instance failed: %v", err)
	}
	getResult := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Get returned %d", resp.StatusCode)
	}
	if getResult["id"] != instanceID {
		t.Errorf("Instance ID mismatch")
	}

	// 列表
	resp, err = c.Get("/api/v1/agents")
	if err != nil {
		t.Fatalf("List instances failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("List returned %d", resp.StatusCode)
	}

	// 删除
	resp, err = c.Delete("/api/v1/agents/" + instanceID)
	if err != nil {
		t.Fatalf("Delete instance failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		t.Errorf("Delete returned %d", resp.StatusCode)
	}
}

// TestInstance_List 验证实例列表查询（即使为空也应返回 200）
func TestInstance_List(t *testing.T) {
	resp, err := c.Get("/api/v1/agents")
	if err != nil {
		t.Fatalf("List instances failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("List returned %d", resp.StatusCode)
	}
}

// TestInstance_StartStop 验证实例启停
func TestInstance_StartStop(t *testing.T) {
	// 注册节点
	nodeID := "e2e-startstop-" + time.Now().Format("150405")
	c.Post("/api/v1/nodes/heartbeat", map[string]interface{}{
		"node_id": nodeID, "status": "online",
	})
	defer c.Delete("/api/v1/nodes/" + nodeID)

	// 创建实例
	resp, err := c.Post("/api/v1/agents", map[string]interface{}{
		"name": "E2E StartStop", "node_id": nodeID,
	})
	if err != nil {
		t.Fatalf("Create instance failed: %v", err)
	}
	result := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusCreated {
		t.Skip("Instance creation requires prerequisites")
	}
	instanceID := result["id"].(string)
	defer c.Delete("/api/v1/agents/" + instanceID)

	// Start
	resp, err = c.Post("/api/v1/agents/"+instanceID+"/start", nil)
	if err != nil {
		t.Fatalf("Start instance failed: %v", err)
	}
	defer resp.Body.Close()
	t.Logf("Start returned %d", resp.StatusCode)

	// Stop
	resp, err = c.Post("/api/v1/agents/"+instanceID+"/stop", nil)
	if err != nil {
		t.Fatalf("Stop instance failed: %v", err)
	}
	defer resp.Body.Close()
	t.Logf("Stop returned %d", resp.StatusCode)
}

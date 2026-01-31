package regression

import (
	"net/http"
	"testing"
	"time"
)

// ============================================================================
// Node 节点管理回归测试
// ============================================================================

// TestNode_Heartbeat 测试节点心跳
func TestNode_Heartbeat(t *testing.T) {
	skipIfNoDatabase(t)

	nodeID := "test-node-" + time.Now().Format("150405")

	t.Run("基本心跳", func(t *testing.T) {
		body := `{
			"node_id":"` + nodeID + `",
			"status":"online",
			"labels":{"os":"linux","arch":"amd64"},
			"capacity":{"max_concurrent":4,"available":4}
		}`
		w := makeRequestWithString("POST", "/api/v1/nodes/heartbeat", body)

		if w.Code != http.StatusOK {
			t.Errorf("Heartbeat status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		// 清理
		makeRequest("DELETE", "/api/v1/nodes/"+nodeID, nil)
	})

	t.Run("缺少 node_id", func(t *testing.T) {
		body := `{"status":"online"}`
		w := makeRequestWithString("POST", "/api/v1/nodes/heartbeat", body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Heartbeat without node_id status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("带标签心跳", func(t *testing.T) {
		body := `{
			"node_id":"` + nodeID + `-labels",
			"status":"online",
			"labels":{"os":"linux","gpu":"true","region":"us-west","env":"production"}
		}`
		w := makeRequestWithString("POST", "/api/v1/nodes/heartbeat", body)

		if w.Code != http.StatusOK {
			t.Errorf("Heartbeat with labels status = %d", w.Code)
		}

		// 清理
		makeRequest("DELETE", "/api/v1/nodes/"+nodeID+"-labels", nil)
	})

	t.Run("多次心跳更新", func(t *testing.T) {
		body := `{"node_id":"` + nodeID + `-multi","status":"online","capacity":{"available":4}}`
		w := makeRequestWithString("POST", "/api/v1/nodes/heartbeat", body)
		if w.Code != http.StatusOK {
			t.Fatalf("First heartbeat failed: %d", w.Code)
		}

		// 第二次心跳
		body = `{"node_id":"` + nodeID + `-multi","status":"online","capacity":{"available":3}}`
		w = makeRequestWithString("POST", "/api/v1/nodes/heartbeat", body)
		if w.Code != http.StatusOK {
			t.Errorf("Second heartbeat failed: %d", w.Code)
		}

		// 清理
		makeRequest("DELETE", "/api/v1/nodes/"+nodeID+"-multi", nil)
	})
}

// TestNode_List 测试列出节点
func TestNode_List(t *testing.T) {
	skipIfNoDatabase(t)

	// 注册多个节点
	nodeIDs := []string{
		"list-test-node-1-" + time.Now().Format("150405"),
		"list-test-node-2-" + time.Now().Format("150405"),
		"list-test-node-3-" + time.Now().Format("150405"),
	}

	for _, nodeID := range nodeIDs {
		body := `{"node_id":"` + nodeID + `","status":"online"}`
		makeRequestWithString("POST", "/api/v1/nodes/heartbeat", body)
	}

	defer func() {
		for _, nodeID := range nodeIDs {
			makeRequest("DELETE", "/api/v1/nodes/"+nodeID, nil)
		}
	}()

	t.Run("列出所有节点", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/nodes", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("List nodes status = %d", w.Code)
		}

		resp := parseJSONResponse(w)
		if resp["nodes"] == nil {
			t.Error("Nodes list not returned")
		}
	})
}

// TestNode_Get 测试获取节点详情
func TestNode_Get(t *testing.T) {
	skipIfNoDatabase(t)

	nodeID := "get-test-node-" + time.Now().Format("150405")

	// 注册节点
	body := `{"node_id":"` + nodeID + `","status":"online","labels":{"test":"true"}}`
	makeRequestWithString("POST", "/api/v1/nodes/heartbeat", body)
	defer makeRequest("DELETE", "/api/v1/nodes/"+nodeID, nil)

	t.Run("获取存在的节点", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/nodes/"+nodeID, nil)
		if w.Code != http.StatusOK {
			t.Errorf("Get node status = %d, want %d", w.Code, http.StatusOK)
		}

		resp := parseJSONResponse(w)
		if resp["id"] != nodeID {
			t.Errorf("Node ID = %v, want %v", resp["id"], nodeID)
		}
	})

	t.Run("获取不存在的节点", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/nodes/nonexistent-node", nil)
		if w.Code != http.StatusNotFound {
			t.Errorf("Get nonexistent node status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

// TestNode_Update 测试更新节点
func TestNode_Update(t *testing.T) {
	skipIfNoDatabase(t)

	nodeID := "update-test-node-" + time.Now().Format("150405")

	// 注册节点
	body := `{"node_id":"` + nodeID + `","status":"online"}`
	makeRequestWithString("POST", "/api/v1/nodes/heartbeat", body)
	defer makeRequest("DELETE", "/api/v1/nodes/"+nodeID, nil)

	t.Run("更新状态为 draining", func(t *testing.T) {
		w := makeRequestWithString("PATCH", "/api/v1/nodes/"+nodeID, `{"status":"draining"}`)
		if w.Code != http.StatusOK {
			t.Errorf("Update node status = %d, want %d", w.Code, http.StatusOK)
		}

		// 验证更新
		w = makeRequest("GET", "/api/v1/nodes/"+nodeID, nil)
		resp := parseJSONResponse(w)
		if resp["status"] != "draining" {
			t.Errorf("Node status = %v, want draining", resp["status"])
		}
	})

	t.Run("更新状态为 maintenance", func(t *testing.T) {
		w := makeRequestWithString("PATCH", "/api/v1/nodes/"+nodeID, `{"status":"maintenance"}`)
		if w.Code != http.StatusOK {
			t.Errorf("Update to maintenance status = %d", w.Code)
		}
	})

	t.Run("更新标签", func(t *testing.T) {
		w := makeRequestWithString("PATCH", "/api/v1/nodes/"+nodeID, `{"labels":{"gpu":"false","updated":"true"}}`)
		if w.Code != http.StatusOK {
			t.Errorf("Update labels status = %d", w.Code)
		}
	})

	t.Run("更新不存在的节点", func(t *testing.T) {
		w := makeRequestWithString("PATCH", "/api/v1/nodes/nonexistent-node", `{"status":"offline"}`)
		if w.Code != http.StatusNotFound {
			t.Errorf("Update nonexistent node status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

// TestNode_Delete 测试删除节点
func TestNode_Delete(t *testing.T) {
	skipIfNoDatabase(t)

	t.Run("删除存在的节点", func(t *testing.T) {
		nodeID := "delete-test-node-" + time.Now().Format("150405")

		// 注册节点
		body := `{"node_id":"` + nodeID + `","status":"online"}`
		makeRequestWithString("POST", "/api/v1/nodes/heartbeat", body)

		// 删除
		w := makeRequest("DELETE", "/api/v1/nodes/"+nodeID, nil)
		if w.Code != http.StatusNoContent {
			t.Errorf("Delete node status = %d, want %d", w.Code, http.StatusNoContent)
		}

		// 验证已删除
		w = makeRequest("GET", "/api/v1/nodes/"+nodeID, nil)
		if w.Code != http.StatusNotFound {
			t.Error("Node should be deleted")
		}
	})

	t.Run("删除不存在的节点", func(t *testing.T) {
		w := makeRequest("DELETE", "/api/v1/nodes/nonexistent-node", nil)
		// 删除不存在的资源通常返回 204 或 404
		if w.Code != http.StatusNoContent && w.Code != http.StatusNotFound {
			t.Errorf("Delete nonexistent node status = %d", w.Code)
		}
	})
}

// TestNode_Runs 测试获取节点的 Runs
func TestNode_Runs(t *testing.T) {
	skipIfNoDatabase(t)

	nodeID := "runs-test-node-" + time.Now().Format("150405")

	// 注册节点
	body := `{"node_id":"` + nodeID + `","status":"online"}`
	makeRequestWithString("POST", "/api/v1/nodes/heartbeat", body)
	defer makeRequest("DELETE", "/api/v1/nodes/"+nodeID, nil)

	t.Run("获取节点的 Runs", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/nodes/"+nodeID+"/runs", nil)
		if w.Code != http.StatusOK {
			t.Errorf("Get node runs status = %d, want %d", w.Code, http.StatusOK)
			return
		}

		resp := parseJSONResponse(w)
		// API 可能返回 "runs" 或空对象，两种都可接受
		t.Logf("Node runs response: %v", resp)
	})
}

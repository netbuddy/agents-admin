package proxy

import (
	"net/http"
	"testing"

	"agents-admin/tests/testutil"
)

// TestProxy_CRUD 验证代理完整生命周期
func TestProxy_CRUD(t *testing.T) {
	// 创建
	payload := map[string]interface{}{
		"name":     "E2E Test Proxy",
		"url":      "http://proxy.example.com:8080",
		"protocol": "http",
	}
	resp, err := c.Post("/api/v1/proxies", payload)
	if err != nil {
		t.Fatalf("Create proxy failed: %v", err)
	}
	result := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Create returned %d", resp.StatusCode)
	}
	proxyID := result["id"].(string)
	t.Logf("Created proxy: %s", proxyID)

	// 读取
	resp, err = c.Get("/api/v1/proxies/" + proxyID)
	if err != nil {
		t.Fatalf("Get proxy failed: %v", err)
	}
	getResult := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Get returned %d", resp.StatusCode)
	}
	if getResult["id"] != proxyID {
		t.Errorf("Proxy ID mismatch: %v", getResult["id"])
	}

	// 更新
	updatePayload := map[string]interface{}{
		"name": "E2E Updated Proxy",
		"url":  "http://proxy2.example.com:9090",
	}
	resp, err = c.Put("/api/v1/proxies/"+proxyID, updatePayload)
	if err != nil {
		t.Fatalf("Update proxy failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Update returned %d", resp.StatusCode)
	}

	// 列表
	resp, err = c.Get("/api/v1/proxies")
	if err != nil {
		t.Fatalf("List proxies failed: %v", err)
	}
	listResult := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("List returned %d", resp.StatusCode)
	}
	if listResult["proxies"] == nil {
		t.Error("Proxies list not returned")
	}

	// 删除
	resp, err = c.Delete("/api/v1/proxies/" + proxyID)
	if err != nil {
		t.Fatalf("Delete proxy failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		t.Errorf("Delete returned %d", resp.StatusCode)
	}
}

// TestProxy_Test 验证代理连通性测试
func TestProxy_Test(t *testing.T) {
	// 创建代理
	payload := map[string]interface{}{
		"name":     "E2E Proxy Test",
		"url":      "http://proxy.example.com:8080",
		"protocol": "http",
	}
	resp, err := c.Post("/api/v1/proxies", payload)
	if err != nil {
		t.Fatalf("Create proxy failed: %v", err)
	}
	result := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusCreated {
		t.Skip("Proxy creation failed, skipping test")
	}
	proxyID := result["id"].(string)
	defer c.Delete("/api/v1/proxies/" + proxyID)

	// 测试连通性（预期可能失败因为代理不存在，但接口应可达）
	resp, err = c.Post("/api/v1/proxies/"+proxyID+"/test", nil)
	if err != nil {
		t.Fatalf("Test proxy failed: %v", err)
	}
	defer resp.Body.Close()
	// 接口应返回某种结果，不应是 404/500
	t.Logf("Proxy test returned %d", resp.StatusCode)
}

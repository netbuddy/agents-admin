package regression

import (
	"context"
	"net/http"
	"testing"
)

// ============================================================================
// Proxy 代理管理回归测试
// ============================================================================

// TestProxy_Create 测试创建代理
func TestProxy_Create(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	t.Run("创建 HTTP 代理", func(t *testing.T) {
		body := `{
			"name":"Test HTTP Proxy",
			"type":"http",
			"host":"proxy.example.com",
			"port":8080
		}`
		w := makeRequestWithString("POST", "/api/v1/proxies", body)

		if w.Code != http.StatusCreated {
			t.Errorf("Create HTTP proxy status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
			return
		}

		resp := parseJSONResponse(w)
		if resp["id"] == nil {
			t.Error("Proxy ID not returned")
		}

		// 清理
		if id, ok := resp["id"].(string); ok {
			testStore.DeleteProxy(ctx, id)
		}
	})

	t.Run("创建 SOCKS5 代理", func(t *testing.T) {
		body := `{
			"name":"Test SOCKS5 Proxy",
			"type":"socks5",
			"host":"socks.example.com",
			"port":1080
		}`
		w := makeRequestWithString("POST", "/api/v1/proxies", body)

		if w.Code == http.StatusCreated {
			resp := parseJSONResponse(w)
			testStore.DeleteProxy(ctx, resp["id"].(string))
		}
	})

	t.Run("创建带认证的代理", func(t *testing.T) {
		body := `{
			"name":"Auth Proxy",
			"type":"http",
			"host":"proxy.example.com",
			"port":8080,
			"username":"user",
			"password":"pass"
		}`
		w := makeRequestWithString("POST", "/api/v1/proxies", body)

		if w.Code == http.StatusCreated {
			resp := parseJSONResponse(w)
			testStore.DeleteProxy(ctx, resp["id"].(string))
		}
	})

	t.Run("缺少必填字段", func(t *testing.T) {
		body := `{"name":"Incomplete Proxy"}`
		w := makeRequestWithString("POST", "/api/v1/proxies", body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Create proxy without required fields status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("无效的端口", func(t *testing.T) {
		body := `{"name":"Bad Port Proxy","type":"http","host":"proxy.example.com","port":-1}`
		w := makeRequestWithString("POST", "/api/v1/proxies", body)

		if w.Code == http.StatusCreated {
			resp := parseJSONResponse(w)
			testStore.DeleteProxy(ctx, resp["id"].(string))
		}
		t.Logf("Invalid port status: %d", w.Code)
	})
}

// TestProxy_Get 测试获取代理
func TestProxy_Get(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 创建测试代理
	w := makeRequestWithString("POST", "/api/v1/proxies", `{"name":"Get Test Proxy","type":"http","host":"test.com","port":8080}`)
	if w.Code != http.StatusCreated {
		t.Skip("Failed to create test proxy")
	}
	resp := parseJSONResponse(w)
	proxyID := resp["id"].(string)
	defer testStore.DeleteProxy(ctx, proxyID)

	t.Run("获取存在的代理", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/proxies/"+proxyID, nil)
		if w.Code != http.StatusOK {
			t.Errorf("Get proxy status = %d, want %d", w.Code, http.StatusOK)
		}

		resp := parseJSONResponse(w)
		if resp["id"] != proxyID {
			t.Errorf("Proxy ID = %v, want %v", resp["id"], proxyID)
		}
	})

	t.Run("获取不存在的代理", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/proxies/nonexistent-proxy", nil)
		if w.Code != http.StatusNotFound {
			t.Errorf("Get nonexistent proxy status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

// TestProxy_List 测试列出代理
func TestProxy_List(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 创建多个测试代理
	var proxyIDs []string
	for i := 0; i < 3; i++ {
		w := makeRequestWithString("POST", "/api/v1/proxies", `{"name":"List Test Proxy","type":"http","host":"test.com","port":8080}`)
		if w.Code == http.StatusCreated {
			resp := parseJSONResponse(w)
			proxyIDs = append(proxyIDs, resp["id"].(string))
		}
	}
	defer func() {
		for _, id := range proxyIDs {
			testStore.DeleteProxy(ctx, id)
		}
	}()

	t.Run("列出所有代理", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/proxies", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("List proxies status = %d", w.Code)
		}

		resp := parseJSONResponse(w)
		if resp["proxies"] == nil {
			t.Error("Proxies list not returned")
		}
	})
}

// TestProxy_Update 测试更新代理
func TestProxy_Update(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 创建测试代理
	w := makeRequestWithString("POST", "/api/v1/proxies", `{"name":"Update Test Proxy","type":"http","host":"test.com","port":8080}`)
	if w.Code != http.StatusCreated {
		t.Skip("Failed to create test proxy")
	}
	resp := parseJSONResponse(w)
	proxyID := resp["id"].(string)
	defer testStore.DeleteProxy(ctx, proxyID)

	t.Run("更新代理信息", func(t *testing.T) {
		body := `{
			"name":"Updated Proxy Name",
			"type":"http",
			"host":"new-host.com",
			"port":9090
		}`
		w := makeRequestWithString("PUT", "/api/v1/proxies/"+proxyID, body)
		if w.Code != http.StatusOK {
			t.Errorf("Update proxy status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		// 验证更新
		w = makeRequest("GET", "/api/v1/proxies/"+proxyID, nil)
		resp := parseJSONResponse(w)
		if resp["name"] != "Updated Proxy Name" {
			t.Errorf("Proxy name = %v, want Updated Proxy Name", resp["name"])
		}
	})

	t.Run("更新不存在的代理", func(t *testing.T) {
		body := `{"name":"Updated","type":"http","host":"test.com","port":8080}`
		w := makeRequestWithString("PUT", "/api/v1/proxies/nonexistent-proxy", body)
		if w.Code != http.StatusNotFound {
			t.Errorf("Update nonexistent proxy status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

// TestProxy_Delete 测试删除代理
func TestProxy_Delete(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	t.Run("删除存在的代理", func(t *testing.T) {
		// 创建代理
		w := makeRequestWithString("POST", "/api/v1/proxies", `{"name":"Delete Test Proxy","type":"http","host":"test.com","port":8080}`)
		if w.Code != http.StatusCreated {
			t.Skip("Failed to create test proxy")
		}
		resp := parseJSONResponse(w)
		proxyID := resp["id"].(string)

		// 删除
		w = makeRequest("DELETE", "/api/v1/proxies/"+proxyID, nil)
		if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
			t.Errorf("Delete proxy status = %d", w.Code)
		}

		// 验证已删除
		w = makeRequest("GET", "/api/v1/proxies/"+proxyID, nil)
		if w.Code != http.StatusNotFound {
			testStore.DeleteProxy(ctx, proxyID)
			t.Error("Proxy should be deleted")
		}
	})

	t.Run("删除不存在的代理", func(t *testing.T) {
		w := makeRequest("DELETE", "/api/v1/proxies/nonexistent-proxy", nil)
		if w.Code != http.StatusNoContent && w.Code != http.StatusNotFound {
			t.Errorf("Delete nonexistent proxy status = %d", w.Code)
		}
	})
}

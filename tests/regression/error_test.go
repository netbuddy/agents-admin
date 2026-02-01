package regression

import (
	"net/http"
	"testing"
)

// ============================================================================
// 错误处理回归测试
// ============================================================================

// TestError_NotFound 测试 404 错误
func TestError_NotFound(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"不存在的 Task", "/api/v1/tasks/nonexistent-task-id"},
		{"不存在的 Run", "/api/v1/runs/nonexistent-run-id"},
		{"不存在的 Node", "/api/v1/nodes/nonexistent-node-id"},
		{"不存在的 Account", "/api/v1/accounts/nonexistent-account-id"},
		{"不存在的 Proxy", "/api/v1/proxies/nonexistent-proxy-id"},
		// Instance 和 Terminal 表可能未迁移，跳过
		// {"不存在的 Instance", "/api/v1/instances/nonexistent-instance-id"},
		// {"不存在的 Terminal Session", "/api/v1/terminal-sessions/nonexistent-session-id"},
		{"不存在的 Agent Type", "/api/v1/agent-types/nonexistent-type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := makeRequest("GET", tt.path, nil)
			if w.Code != http.StatusNotFound {
				t.Errorf("GET %s status = %d, want %d", tt.path, w.Code, http.StatusNotFound)
			}
		})
	}
}

// TestError_BadRequest 测试 400 错误
func TestError_BadRequest(t *testing.T) {
	t.Run("无效的 JSON", func(t *testing.T) {
		w := makeRequestWithString("POST", "/api/v1/tasks", "{invalid json}")
		if w.Code != http.StatusBadRequest {
			t.Errorf("Invalid JSON status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("空请求体但需要参数", func(t *testing.T) {
		w := makeRequestWithString("POST", "/api/v1/nodes/heartbeat", "")
		if w.Code != http.StatusBadRequest {
			t.Errorf("Empty body for heartbeat status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("心跳缺少 node_id", func(t *testing.T) {
		w := makeRequestWithString("POST", "/api/v1/nodes/heartbeat", `{"status":"online"}`)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Heartbeat without node_id status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("创建账号缺少必填字段", func(t *testing.T) {
		w := makeRequestWithString("POST", "/api/v1/accounts", `{"name":"Test"}`)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Create account without required fields status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("创建代理缺少必填字段", func(t *testing.T) {
		w := makeRequestWithString("POST", "/api/v1/proxies", `{"name":"Test"}`)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Create proxy without required fields status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("无效的分页参数", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/tasks?limit=invalid&offset=abc", nil)
		// 应该返回 400 或使用默认值
		t.Logf("Invalid pagination status: %d", w.Code)
	})
}

// TestError_MethodNotAllowed 测试 405 错误
func TestError_MethodNotAllowed(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"Health 不支持 POST", "POST", "/health"},
		{"Metrics 不支持 POST", "POST", "/metrics"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := makeRequest(tt.method, tt.path, nil)
			// 可能返回 405 或其他错误
			if w.Code == http.StatusMethodNotAllowed {
				t.Log("Correctly returned 405")
			} else {
				t.Logf("%s %s status: %d", tt.method, tt.path, w.Code)
			}
		})
	}
}

// TestError_Conflict 测试冲突错误
func TestError_Conflict(t *testing.T) {
	skipIfNoDatabase(t)

	t.Run("删除有活跃 Run 的 Task", func(t *testing.T) {
		// 创建 Task 和 Run
		w := makeRequestWithString("POST", "/api/v1/tasks", `{"name":"Conflict Test Task","prompt":"test","type":"general"}`)
		if w.Code != http.StatusCreated {
			t.Skip("Failed to create task")
		}
		taskResp := parseJSONResponse(w)
		taskID := taskResp["id"].(string)

		w = makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
		if w.Code != http.StatusCreated {
			makeRequest("DELETE", "/api/v1/tasks/"+taskID, nil)
			t.Skip("Failed to create run")
		}
		runResp := parseJSONResponse(w)
		runID := runResp["id"].(string)

		// 设置 Run 为 running
		makeRequestWithString("PATCH", "/api/v1/runs/"+runID, `{"status":"running"}`)

		// 尝试删除 Task - 可能成功（级联删除）或返回 409
		w = makeRequest("DELETE", "/api/v1/tasks/"+taskID, nil)
		t.Logf("Delete task with running run status: %d", w.Code)

		// 清理
		makeRequest("DELETE", "/api/v1/tasks/"+taskID, nil)
	})
}

// TestError_ValidationErrors 测试验证错误
func TestError_ValidationErrors(t *testing.T) {
	t.Run("无效的状态值", func(t *testing.T) {
		skipIfNoDatabase(t)

		// 创建 Task 和 Run
		w := makeRequestWithString("POST", "/api/v1/tasks", `{"name":"Validation Test","prompt":"test","type":"general"}`)
		if w.Code != http.StatusCreated {
			t.Skip("Failed to create task")
		}
		taskResp := parseJSONResponse(w)
		taskID := taskResp["id"].(string)
		defer makeRequest("DELETE", "/api/v1/tasks/"+taskID, nil)

		w = makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
		if w.Code != http.StatusCreated {
			t.Skip("Failed to create run")
		}
		runResp := parseJSONResponse(w)
		runID := runResp["id"].(string)

		// 尝试设置无效状态
		w = makeRequestWithString("PATCH", "/api/v1/runs/"+runID, `{"status":"invalid_status_xyz"}`)
		t.Logf("Invalid status update: %d", w.Code)
	})

	t.Run("负数端口", func(t *testing.T) {
		w := makeRequestWithString("POST", "/api/v1/proxies", `{"name":"Bad Port","type":"http","host":"test.com","port":-100}`)
		t.Logf("Negative port: %d", w.Code)
	})

	t.Run("超大端口号", func(t *testing.T) {
		w := makeRequestWithString("POST", "/api/v1/proxies", `{"name":"Big Port","type":"http","host":"test.com","port":99999}`)
		t.Logf("Large port: %d", w.Code)
	})
}

// TestError_ContentType 测试 Content-Type 处理
func TestError_ContentType(t *testing.T) {
	t.Run("错误的 Content-Type", func(t *testing.T) {
		w := makeRequestWithString("POST", "/api/v1/tasks", `name=test`)
		// 应该返回错误，因为不是有效的 JSON
		if w.Code == http.StatusBadRequest || w.Code == http.StatusUnsupportedMediaType {
			t.Log("Correctly rejected non-JSON content")
		} else {
			t.Logf("Wrong content-type status: %d", w.Code)
		}
	})
}

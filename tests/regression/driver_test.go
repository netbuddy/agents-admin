package regression

import (
	"context"
	"net/http"
	"testing"
)

// ============================================================================
// Driver 功能回归测试
// ============================================================================

// TestDriver_TaskCreation 测试使用不同 Driver 创建任务
func TestDriver_TaskCreation(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	drivers := []struct {
		name      string
		agentType string
	}{
		{"QwenCode", "qwencode"},
		{"Gemini", "gemini"},
		{"Claude", "claude"},
	}

	for _, d := range drivers {
		t.Run(d.name+" Task 创建", func(t *testing.T) {
			body := `{
				"name":"` + d.name + ` Driver Test",
				"spec":{
					"prompt":"Test prompt for ` + d.name + `",
					"agent":{"type":"` + d.agentType + `"}
				}
			}`
			w := makeRequestWithString("POST", "/api/v1/tasks", body)

			if w.Code != http.StatusCreated {
				t.Errorf("Create %s task status = %d, want %d", d.name, w.Code, http.StatusCreated)
				return
			}

			resp := parseJSONResponse(w)
			taskID := resp["id"].(string)
			defer testStore.DeleteTask(ctx, taskID)

			// 验证 Task 创建成功
			if resp["spec"] != nil {
				spec := resp["spec"].(map[string]interface{})
				if agent, ok := spec["agent"].(map[string]interface{}); ok {
					if agent["type"] != d.agentType {
						t.Errorf("Agent type = %v, want %v", agent["type"], d.agentType)
					}
				}
			}
		})
	}
}

// TestDriver_TaskWithModel 测试指定模型的任务
func TestDriver_TaskWithModel(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	models := []struct {
		agentType string
		model     string
	}{
		{"qwencode", "qwen-coder-32b"},
		{"gemini", "gemini-pro"},
		{"claude", "claude-3-opus"},
	}

	for _, m := range models {
		t.Run(m.agentType+"/"+m.model, func(t *testing.T) {
			body := `{
				"name":"Model Test Task",
				"spec":{
					"prompt":"Test prompt",
					"agent":{
						"type":"` + m.agentType + `",
						"model":"` + m.model + `"
					}
				}
			}`
			w := makeRequestWithString("POST", "/api/v1/tasks", body)

			if w.Code == http.StatusCreated {
				resp := parseJSONResponse(w)
				testStore.DeleteTask(ctx, resp["id"].(string))
			}
			t.Logf("%s/%s: status %d", m.agentType, m.model, w.Code)
		})
	}
}

// TestDriver_TaskWithParameters 测试带参数的任务
func TestDriver_TaskWithParameters(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	t.Run("带温度参数", func(t *testing.T) {
		body := `{
			"name":"Parameters Test Task",
			"spec":{
				"prompt":"Test prompt",
				"agent":{
					"type":"gemini",
					"parameters":{
						"temperature":0.7,
						"max_tokens":2048
					}
				}
			}
		}`
		w := makeRequestWithString("POST", "/api/v1/tasks", body)

		if w.Code == http.StatusCreated {
			resp := parseJSONResponse(w)
			testStore.DeleteTask(ctx, resp["id"].(string))
		}
		t.Logf("Task with parameters: %d", w.Code)
	})

	t.Run("带工作区配置", func(t *testing.T) {
		body := `{
			"name":"Workspace Test Task",
			"spec":{
				"prompt":"Test prompt",
				"agent":{"type":"qwencode"},
				"workspace":{
					"path":"/workspace/project",
					"git_url":"https://github.com/example/repo.git"
				}
			}
		}`
		w := makeRequestWithString("POST", "/api/v1/tasks", body)

		if w.Code == http.StatusCreated {
			resp := parseJSONResponse(w)
			testStore.DeleteTask(ctx, resp["id"].(string))
		}
		t.Logf("Task with workspace: %d", w.Code)
	})
}

// TestDriver_RunWithDriver 测试运行使用不同 Driver 的任务
func TestDriver_RunWithDriver(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	drivers := []string{"qwencode", "gemini", "claude"}

	for _, d := range drivers {
		t.Run("Run "+d+" Task", func(t *testing.T) {
			// 创建任务
			body := `{
				"name":"Run ` + d + ` Test",
				"spec":{"prompt":"test","agent":{"type":"` + d + `"}}
			}`
			w := makeRequestWithString("POST", "/api/v1/tasks", body)
			if w.Code != http.StatusCreated {
				t.Skip("Failed to create task")
			}
			taskResp := parseJSONResponse(w)
			taskID := taskResp["id"].(string)
			defer testStore.DeleteTask(ctx, taskID)

			// 创建 Run
			w = makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
			if w.Code != http.StatusCreated {
				t.Errorf("Create run for %s task failed: %d", d, w.Code)
				return
			}

			runResp := parseJSONResponse(w)
			if runResp["id"] == nil {
				t.Error("Run ID not returned")
			}
			if runResp["status"] != "queued" {
				t.Errorf("Run status = %v, want queued", runResp["status"])
			}
		})
	}
}

// TestDriver_InvalidAgentType 测试无效的 Agent 类型
func TestDriver_InvalidAgentType(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	t.Run("不存在的 Agent 类型", func(t *testing.T) {
		body := `{
			"name":"Invalid Driver Test",
			"spec":{"prompt":"test","agent":{"type":"nonexistent_driver"}}
		}`
		w := makeRequestWithString("POST", "/api/v1/tasks", body)

		// 可能成功创建（验证在执行时进行）或返回错误
		if w.Code == http.StatusCreated {
			resp := parseJSONResponse(w)
			testStore.DeleteTask(ctx, resp["id"].(string))
		}
		t.Logf("Invalid agent type: %d", w.Code)
	})

	t.Run("空 Agent 类型", func(t *testing.T) {
		body := `{
			"name":"Empty Driver Test",
			"spec":{"prompt":"test","agent":{"type":""}}
		}`
		w := makeRequestWithString("POST", "/api/v1/tasks", body)

		if w.Code == http.StatusCreated {
			resp := parseJSONResponse(w)
			testStore.DeleteTask(ctx, resp["id"].(string))
		}
		t.Logf("Empty agent type: %d", w.Code)
	})
}

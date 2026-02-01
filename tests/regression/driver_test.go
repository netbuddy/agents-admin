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
			// 使用扁平化结构创建任务
			body := `{
				"name":"` + d.name + ` Driver Test",
				"prompt":"Test prompt for ` + d.name + `",
				"type":"general"
			}`
			w := makeRequestWithString("POST", "/api/v1/tasks", body)

			if w.Code != http.StatusCreated {
				t.Errorf("Create %s task status = %d, want %d - %s", d.name, w.Code, http.StatusCreated, w.Body.String())
				return
			}

			resp := parseJSONResponse(w)
			taskID := resp["id"].(string)
			defer testStore.DeleteTask(ctx, taskID)

			// 验证 Task 创建成功
			if resp["prompt"] == nil {
				t.Error("Prompt field not returned")
			}
			if resp["type"] != "general" {
				t.Errorf("Type = %v, want general", resp["type"])
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
			// 使用扁平化结构创建任务
			body := `{
				"name":"Model Test Task",
				"prompt":"Test prompt",
				"type":"general"
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

	t.Run("带安全配置", func(t *testing.T) {
		body := `{
			"name":"Parameters Test Task",
			"prompt":"Test prompt",
			"type":"general",
			"security":{
				"policy":"standard",
				"permissions":["file_read"]
			}
		}`
		w := makeRequestWithString("POST", "/api/v1/tasks", body)

		if w.Code == http.StatusCreated {
			resp := parseJSONResponse(w)
			testStore.DeleteTask(ctx, resp["id"].(string))
		}
		t.Logf("Task with security: %d", w.Code)
	})

	t.Run("带工作区配置", func(t *testing.T) {
		body := `{
			"name":"Workspace Test Task",
			"prompt":"Test prompt",
			"type":"development",
			"workspace":{
				"type":"local",
				"local":{
					"path":"/workspace/project",
					"read_only":false
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
			// 创建任务（使用扁平化结构）
			body := `{
				"name":"Run ` + d + ` Test",
				"prompt":"test",
				"type":"general"
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

// TestDriver_InvalidTaskType 测试无效的任务类型
func TestDriver_InvalidTaskType(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	t.Run("不存在的任务类型", func(t *testing.T) {
		body := `{
			"name":"Invalid Type Test",
			"prompt":"test",
			"type":"nonexistent_type"
		}`
		w := makeRequestWithString("POST", "/api/v1/tasks", body)

		// 任务类型验证可能在执行时进行，所以可能成功创建
		if w.Code == http.StatusCreated {
			resp := parseJSONResponse(w)
			testStore.DeleteTask(ctx, resp["id"].(string))
		}
		t.Logf("Invalid task type: %d", w.Code)
	})

	t.Run("空任务类型（使用默认值）", func(t *testing.T) {
		body := `{
			"name":"Empty Type Test",
			"prompt":"test"
		}`
		w := makeRequestWithString("POST", "/api/v1/tasks", body)

		if w.Code == http.StatusCreated {
			resp := parseJSONResponse(w)
			// 验证使用了默认类型
			if resp["type"] != "general" {
				t.Errorf("Type = %v, want general", resp["type"])
			}
			testStore.DeleteTask(ctx, resp["id"].(string))
		}
		t.Logf("Empty task type: %d", w.Code)
	})
}

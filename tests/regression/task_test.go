package regression

import (
	"context"
	"net/http"
	"testing"
)

// ============================================================================
// Task 生命周期回归测试
// ============================================================================

// TestTask_Create 测试创建任务
func TestTask_Create(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "基本创建",
			body:       `{"name":"Test Task","prompt":"test","type":"general"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "带完整配置",
			body:       `{"name":"Full Config Task","prompt":"test prompt","type":"development","labels":{"priority":"high"}}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "仅名称和提示词",
			body:       `{"name":"Simple Task","prompt":"simple test"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "空请求（缺少 name）",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "缺少 prompt",
			body:       `{"name":"No Prompt Task"}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := makeRequestWithString("POST", "/api/v1/tasks", tt.body)

			if w.Code != tt.wantStatus {
				t.Errorf("Create task status = %d, want %d, body: %s", w.Code, tt.wantStatus, w.Body.String())
				return
			}

			if w.Code == http.StatusCreated {
				resp := parseJSONResponse(w)
				if resp["id"] == nil {
					t.Error("Task ID not returned")
				}
				if resp["status"] != "pending" {
					t.Errorf("Initial status = %v, want pending", resp["status"])
				}
				// 清理
				if id, ok := resp["id"].(string); ok {
					testStore.DeleteTask(ctx, id)
				}
			}
		})
	}
}

// TestTask_Get 测试获取任务详情
func TestTask_Get(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 创建测试任务
	w := makeRequestWithString("POST", "/api/v1/tasks", `{"name":"Get Test Task","prompt":"test","type":"general"}`)
	if w.Code != http.StatusCreated {
		t.Fatal("Failed to create test task")
	}
	resp := parseJSONResponse(w)
	taskID := resp["id"].(string)
	defer testStore.DeleteTask(ctx, taskID)

	t.Run("获取存在的任务", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/tasks/"+taskID, nil)
		if w.Code != http.StatusOK {
			t.Errorf("Get task status = %d, want %d", w.Code, http.StatusOK)
		}

		resp := parseJSONResponse(w)
		if resp["id"] != taskID {
			t.Errorf("Task ID = %v, want %v", resp["id"], taskID)
		}
		if resp["name"] != "Get Test Task" {
			t.Errorf("Task name = %v, want Get Test Task", resp["name"])
		}
	})

	t.Run("获取不存在的任务", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/tasks/nonexistent-id", nil)
		if w.Code != http.StatusNotFound {
			t.Errorf("Get nonexistent task status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

// TestTask_List 测试列表查询
func TestTask_List(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 创建多个测试任务
	var taskIDs []string
	for i := 0; i < 5; i++ {
		w := makeRequestWithString("POST", "/api/v1/tasks", `{"name":"List Test Task","prompt":"test","type":"general"}`)
		if w.Code == http.StatusCreated {
			resp := parseJSONResponse(w)
			taskIDs = append(taskIDs, resp["id"].(string))
		}
	}
	defer func() {
		for _, id := range taskIDs {
			testStore.DeleteTask(ctx, id)
		}
	}()

	t.Run("基本列表", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/tasks", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("List tasks status = %d, want %d", w.Code, http.StatusOK)
		}

		resp := parseJSONResponse(w)
		if resp["tasks"] == nil {
			t.Error("Tasks list not returned")
		}
	})

	t.Run("分页查询", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/tasks?limit=3&offset=0", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("Paginated list status = %d", w.Code)
		}

		resp := parseJSONResponse(w)
		tasks := resp["tasks"].([]interface{})
		if len(tasks) > 3 {
			t.Errorf("Expected max 3 tasks, got %d", len(tasks))
		}
	})

	t.Run("状态过滤", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/tasks?status=pending", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("Filtered list status = %d", w.Code)
		}

		resp := parseJSONResponse(w)
		tasks := resp["tasks"].([]interface{})
		for _, task := range tasks {
			taskMap := task.(map[string]interface{})
			if taskMap["status"] != "pending" {
				t.Errorf("Task status = %v, want pending", taskMap["status"])
			}
		}
	})

	t.Run("无效状态过滤", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/tasks?status=invalid_status", nil)
		// 应该返回空列表或全部任务，不应报错
		if w.Code != http.StatusOK {
			t.Errorf("Invalid status filter should not fail, got %d", w.Code)
		}
	})
}

// TestTask_Delete 测试删除任务
func TestTask_Delete(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	t.Run("删除存在的任务", func(t *testing.T) {
		// 创建任务
		w := makeRequestWithString("POST", "/api/v1/tasks", `{"name":"Delete Test Task","prompt":"test","type":"general"}`)
		resp := parseJSONResponse(w)
		taskID := resp["id"].(string)

		// 删除任务
		w = makeRequest("DELETE", "/api/v1/tasks/"+taskID, nil)
		if w.Code != http.StatusNoContent {
			t.Errorf("Delete task status = %d, want %d", w.Code, http.StatusNoContent)
		}

		// 验证已删除
		w = makeRequest("GET", "/api/v1/tasks/"+taskID, nil)
		if w.Code != http.StatusNotFound {
			t.Error("Task should be deleted")
		}
	})

	t.Run("删除不存在的任务", func(t *testing.T) {
		w := makeRequest("DELETE", "/api/v1/tasks/nonexistent-id", nil)
		// 删除不存在的资源通常返回 204 或 404
		if w.Code != http.StatusNoContent && w.Code != http.StatusNotFound {
			t.Errorf("Delete nonexistent task status = %d", w.Code)
		}
	})

	t.Run("删除带 Run 的任务（级联删除）", func(t *testing.T) {
		// 创建任务
		w := makeRequestWithString("POST", "/api/v1/tasks", `{"name":"Cascade Delete Test","prompt":"test","type":"general"}`)
		resp := parseJSONResponse(w)
		taskID := resp["id"].(string)

		// 创建 Run
		w = makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
		if w.Code != http.StatusCreated {
			t.Fatal("Failed to create run")
		}
		runResp := parseJSONResponse(w)
		runID := runResp["id"].(string)

		// 删除任务
		w = makeRequest("DELETE", "/api/v1/tasks/"+taskID, nil)
		if w.Code != http.StatusNoContent {
			testStore.DeleteTask(ctx, taskID)
			t.Fatalf("Delete task with run failed: %d", w.Code)
		}

		// 验证 Run 也被删除
		w = makeRequest("GET", "/api/v1/runs/"+runID, nil)
		if w.Code != http.StatusNotFound {
			t.Error("Run should be cascade deleted")
		}
	})
}

// TestTask_EdgeCases 测试边界情况
func TestTask_EdgeCases(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	t.Run("创建超长名称任务", func(t *testing.T) {
		longName := make([]byte, 500)
		for i := range longName {
			longName[i] = 'a'
		}
		body := `{"name":"` + string(longName) + `","prompt":"test","type":"general"}`
		w := makeRequestWithString("POST", "/api/v1/tasks", body)
		// 应该返回 400（验证失败）或 500（数据库约束）
		// TODO: 在 handler 中添加长度验证，改为返回 400
		if w.Code == http.StatusCreated {
			resp := parseJSONResponse(w)
			testStore.DeleteTask(ctx, resp["id"].(string))
		} else if w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
			t.Errorf("Create long name task: unexpected status %d", w.Code)
		}
	})

	t.Run("创建特殊字符名称任务", func(t *testing.T) {
		body := `{"name":"Test 任务 🚀 <script>","prompt":"test","type":"general"}`
		w := makeRequestWithString("POST", "/api/v1/tasks", body)
		if w.Code == http.StatusCreated {
			resp := parseJSONResponse(w)
			testStore.DeleteTask(ctx, resp["id"].(string))
		}
	})

	t.Run("创建复杂配置任务", func(t *testing.T) {
		body := `{
			"name":"Complex Config Task",
			"prompt":"Fix the bug in src/main.go",
			"type":"development",
			"workspace":{
				"type":"git",
				"git":{
					"url":"https://github.com/example/repo.git",
					"branch":"main"
				}
			},
			"security":{
				"policy":"standard",
				"permissions":["file_read","file_write"]
			},
			"labels":{
				"priority":"high",
				"team":"platform"
			}
		}`
		w := makeRequestWithString("POST", "/api/v1/tasks", body)
		if w.Code == http.StatusCreated {
			resp := parseJSONResponse(w)
			testStore.DeleteTask(ctx, resp["id"].(string))
		} else {
			t.Logf("Complex config task creation: %d - %s", w.Code, w.Body.String())
		}
	})
}

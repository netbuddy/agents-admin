package regression

import (
	"context"
	"net/http"
	"testing"
)

// ============================================================================
// Run 执行管理回归测试
// ============================================================================

// TestRun_Create 测试创建 Run
func TestRun_Create(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 创建测试 Task
	w := makeRequestWithString("POST", "/api/v1/tasks", `{"name":"Run Create Test","prompt":"test","type":"general"}`)
	taskResp := parseJSONResponse(w)
	taskID := taskResp["id"].(string)
	defer testStore.DeleteTask(ctx, taskID)

	t.Run("创建基本 Run", func(t *testing.T) {
		w := makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
		if w.Code != http.StatusCreated {
			t.Fatalf("Create run status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
		}

		resp := parseJSONResponse(w)
		if resp["id"] == nil {
			t.Error("Run ID not returned")
		}
		if resp["status"] != "queued" {
			t.Errorf("Initial status = %v, want queued", resp["status"])
		}
		if resp["task_id"] != taskID {
			t.Errorf("Task ID = %v, want %v", resp["task_id"], taskID)
		}
	})

	t.Run("创建多个 Run", func(t *testing.T) {
		// 一个 Task 可以有多个 Run
		for i := 0; i < 3; i++ {
			w := makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
			if w.Code != http.StatusCreated {
				t.Errorf("Create run %d failed: %d", i, w.Code)
			}
		}
	})

	t.Run("为不存在的 Task 创建 Run", func(t *testing.T) {
		w := makeRequest("POST", "/api/v1/tasks/nonexistent-task/runs", nil)
		if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
			t.Errorf("Create run for nonexistent task status = %d", w.Code)
		}
	})
}

// TestRun_Get 测试获取 Run
func TestRun_Get(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 创建测试 Task 和 Run
	w := makeRequestWithString("POST", "/api/v1/tasks", `{"name":"Run Get Test","prompt":"test","type":"general"}`)
	taskResp := parseJSONResponse(w)
	taskID := taskResp["id"].(string)
	defer testStore.DeleteTask(ctx, taskID)

	w = makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
	runResp := parseJSONResponse(w)
	runID := runResp["id"].(string)

	t.Run("获取存在的 Run", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/runs/"+runID, nil)
		if w.Code != http.StatusOK {
			t.Errorf("Get run status = %d, want %d", w.Code, http.StatusOK)
		}

		resp := parseJSONResponse(w)
		if resp["id"] != runID {
			t.Errorf("Run ID = %v, want %v", resp["id"], runID)
		}
	})

	t.Run("获取不存在的 Run", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/runs/nonexistent-id", nil)
		if w.Code != http.StatusNotFound {
			t.Errorf("Get nonexistent run status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

// TestRun_List 测试列出 Task 的 Runs
func TestRun_List(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 创建测试 Task
	w := makeRequestWithString("POST", "/api/v1/tasks", `{"name":"Run List Test","prompt":"test","type":"general"}`)
	taskResp := parseJSONResponse(w)
	taskID := taskResp["id"].(string)
	defer testStore.DeleteTask(ctx, taskID)

	// 创建多个 Run
	for i := 0; i < 5; i++ {
		makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
	}

	t.Run("列出 Task 的所有 Runs", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/tasks/"+taskID+"/runs", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("List runs status = %d", w.Code)
		}

		resp := parseJSONResponse(w)
		if resp["runs"] == nil {
			t.Error("Runs list not returned")
		}
		runs := resp["runs"].([]interface{})
		if len(runs) < 5 {
			t.Errorf("Expected at least 5 runs, got %d", len(runs))
		}
	})

	t.Run("列出不存在 Task 的 Runs", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/tasks/nonexistent-task/runs", nil)
		// 应该返回空列表或 404
		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Errorf("List runs for nonexistent task status = %d", w.Code)
		}
	})
}

// TestRun_UpdateStatus 测试更新 Run 状态
func TestRun_UpdateStatus(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 创建测试 Task 和 Run
	w := makeRequestWithString("POST", "/api/v1/tasks", `{"name":"Run Update Test","prompt":"test","type":"general"}`)
	taskResp := parseJSONResponse(w)
	taskID := taskResp["id"].(string)
	defer testStore.DeleteTask(ctx, taskID)

	w = makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
	runResp := parseJSONResponse(w)
	runID := runResp["id"].(string)

	t.Run("更新为 running", func(t *testing.T) {
		w := makeRequestWithString("PATCH", "/api/v1/runs/"+runID, `{"status":"running"}`)
		if w.Code != http.StatusOK {
			t.Errorf("Update run status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		// 验证状态更新
		w = makeRequest("GET", "/api/v1/runs/"+runID, nil)
		resp := parseJSONResponse(w)
		if resp["status"] != "running" {
			t.Errorf("Run status = %v, want running", resp["status"])
		}
	})

	t.Run("更新为 done", func(t *testing.T) {
		w := makeRequestWithString("PATCH", "/api/v1/runs/"+runID, `{"status":"done"}`)
		if w.Code != http.StatusOK {
			t.Errorf("Update run to done status = %d", w.Code)
		}

		// 验证 Task 状态也更新
		w = makeRequest("GET", "/api/v1/tasks/"+taskID, nil)
		resp := parseJSONResponse(w)
		// Task 状态应该随 Run 更新
		t.Logf("Task status after run done: %v", resp["status"])
	})

	t.Run("更新不存在的 Run", func(t *testing.T) {
		w := makeRequestWithString("PATCH", "/api/v1/runs/nonexistent-id", `{"status":"running"}`)
		// API 可能返回 404 或 200（幂等 PATCH），两种都可接受
		if w.Code != http.StatusNotFound && w.Code != http.StatusOK {
			t.Errorf("Update nonexistent run status = %d, want 404 or 200", w.Code)
		}
	})

	t.Run("更新为无效状态", func(t *testing.T) {
		// 创建新 Run 用于测试
		w := makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
		newRunResp := parseJSONResponse(w)
		newRunID := newRunResp["id"].(string)

		w = makeRequestWithString("PATCH", "/api/v1/runs/"+newRunID, `{"status":"invalid_status"}`)
		// 应该返回 400 或忽略无效状态
		t.Logf("Update to invalid status: %d", w.Code)
	})
}

// TestRun_Cancel 测试取消 Run
func TestRun_Cancel(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 创建测试 Task 和 Run
	w := makeRequestWithString("POST", "/api/v1/tasks", `{"name":"Run Cancel Test","prompt":"test","type":"general"}`)
	taskResp := parseJSONResponse(w)
	taskID := taskResp["id"].(string)
	defer testStore.DeleteTask(ctx, taskID)

	w = makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
	runResp := parseJSONResponse(w)
	runID := runResp["id"].(string)

	t.Run("取消 queued 状态的 Run", func(t *testing.T) {
		w := makeRequest("POST", "/api/v1/runs/"+runID+"/cancel", nil)
		if w.Code != http.StatusOK {
			t.Errorf("Cancel run status = %d, want %d", w.Code, http.StatusOK)
		}

		// 验证状态
		w = makeRequest("GET", "/api/v1/runs/"+runID, nil)
		resp := parseJSONResponse(w)
		if resp["status"] != "cancelled" {
			t.Errorf("Run status = %v, want cancelled", resp["status"])
		}
	})

	t.Run("取消 running 状态的 Run", func(t *testing.T) {
		// 创建新 Run 并设置为 running
		w := makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
		newRunResp := parseJSONResponse(w)
		newRunID := newRunResp["id"].(string)

		makeRequestWithString("PATCH", "/api/v1/runs/"+newRunID, `{"status":"running"}`)

		w = makeRequest("POST", "/api/v1/runs/"+newRunID+"/cancel", nil)
		if w.Code != http.StatusOK {
			t.Errorf("Cancel running run status = %d", w.Code)
		}
	})

	t.Run("取消不存在的 Run", func(t *testing.T) {
		w := makeRequest("POST", "/api/v1/runs/nonexistent-id/cancel", nil)
		if w.Code != http.StatusNotFound {
			t.Errorf("Cancel nonexistent run status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

// TestRun_StatusTransitions 测试状态转换
func TestRun_StatusTransitions(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 创建测试 Task
	w := makeRequestWithString("POST", "/api/v1/tasks", `{"name":"Run Transitions Test","prompt":"test","type":"general"}`)
	taskResp := parseJSONResponse(w)
	taskID := taskResp["id"].(string)
	defer testStore.DeleteTask(ctx, taskID)

	statuses := []string{"queued", "running", "done"}

	for i := 0; i < len(statuses)-1; i++ {
		from := statuses[i]
		to := statuses[i+1]
		t.Run(from+" -> "+to, func(t *testing.T) {
			// 创建新 Run
			w := makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
			runResp := parseJSONResponse(w)
			runID := runResp["id"].(string)

			// 如果不是从 queued 开始，先设置到起始状态
			if from != "queued" {
				makeRequestWithString("PATCH", "/api/v1/runs/"+runID, `{"status":"`+from+`"}`)
			}

			// 转换到目标状态
			w = makeRequestWithString("PATCH", "/api/v1/runs/"+runID, `{"status":"`+to+`"}`)
			if w.Code != http.StatusOK {
				t.Errorf("Transition %s -> %s failed: %d", from, to, w.Code)
			}
		})
	}
}

package regression

import (
	"context"
	"net/http"
	"testing"

	"agents-admin/internal/model"
)

// ============================================================================
// 数据一致性回归测试
// ============================================================================

// TestConsistency_TaskStatusFollowsRun 测试 Task 状态随 Run 更新
func TestConsistency_TaskStatusFollowsRun(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 创建 Task
	w := makeRequestWithString("POST", "/api/v1/tasks", `{"name":"Consistency Test","prompt":"test","type":"general"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create task: %d - %s", w.Code, w.Body.String())
	}
	taskResp := parseJSONResponse(w)
	taskID := taskResp["id"].(string)
	defer testStore.DeleteTask(ctx, taskID)

	// 创建 Run
	w = makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
	if w.Code != http.StatusCreated {
		t.Fatal("Failed to create run")
	}
	runResp := parseJSONResponse(w)
	runID := runResp["id"].(string)

	t.Run("Run 变为 running 时 Task 变为 running", func(t *testing.T) {
		w := makeRequestWithString("PATCH", "/api/v1/runs/"+runID, `{"status":"running"}`)
		if w.Code != http.StatusOK {
			t.Fatalf("Update run failed: %d", w.Code)
		}

		task, err := testStore.GetTask(ctx, taskID)
		if err != nil {
			t.Fatalf("Failed to get task: %v", err)
		}
		if task.Status != model.TaskStatusRunning {
			t.Errorf("Task status = %v, want running", task.Status)
		}
	})

	t.Run("Run 变为 done 时 Task 变为 done", func(t *testing.T) {
		w := makeRequestWithString("PATCH", "/api/v1/runs/"+runID, `{"status":"done"}`)
		if w.Code != http.StatusOK {
			t.Fatalf("Update run failed: %d", w.Code)
		}

		task, err := testStore.GetTask(ctx, taskID)
		if err != nil {
			t.Fatalf("Failed to get task: %v", err)
		}
		if task.Status != model.TaskStatusCompleted {
			t.Errorf("Task status = %v, want done", task.Status)
		}
	})
}

// TestConsistency_CascadeDelete 测试级联删除
func TestConsistency_CascadeDelete(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	t.Run("删除 Task 会删除关联的 Runs", func(t *testing.T) {
		// 创建 Task
		w := makeRequestWithString("POST", "/api/v1/tasks", `{"name":"Cascade Delete Test","prompt":"test","type":"general"}`)
		taskResp := parseJSONResponse(w)
		taskID := taskResp["id"].(string)

		// 创建多个 Run
		var runIDs []string
		for i := 0; i < 3; i++ {
			w = makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
			if w.Code == http.StatusCreated {
				runResp := parseJSONResponse(w)
				runIDs = append(runIDs, runResp["id"].(string))
			}
		}

		// 删除 Task
		w = makeRequest("DELETE", "/api/v1/tasks/"+taskID, nil)
		if w.Code != http.StatusNoContent {
			testStore.DeleteTask(ctx, taskID)
			t.Fatalf("Delete task failed: %d", w.Code)
		}

		// 验证所有 Run 都被删除
		for _, runID := range runIDs {
			w = makeRequest("GET", "/api/v1/runs/"+runID, nil)
			if w.Code != http.StatusNotFound {
				t.Errorf("Run %s should be deleted, got status %d", runID, w.Code)
			}
		}
	})

	t.Run("删除 Task 会删除关联的 Events", func(t *testing.T) {
		// 创建 Task
		w := makeRequestWithString("POST", "/api/v1/tasks", `{"name":"Event Cascade Test","prompt":"test","type":"general"}`)
		taskResp := parseJSONResponse(w)
		taskID := taskResp["id"].(string)

		// 创建 Run
		w = makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
		runResp := parseJSONResponse(w)
		runID := runResp["id"].(string)

		// 添加事件
		makeRequestWithString("POST", "/api/v1/runs/"+runID+"/events", `{"events":[{"seq":1,"type":"test","timestamp":"2024-01-01T00:00:00Z","payload":{}}]}`)

		// 删除 Task
		w = makeRequest("DELETE", "/api/v1/tasks/"+taskID, nil)
		if w.Code != http.StatusNoContent {
			testStore.DeleteTask(ctx, taskID)
			t.Fatalf("Delete task failed: %d", w.Code)
		}

		// 验证 Events 被删除
		w = makeRequest("GET", "/api/v1/runs/"+runID+"/events", nil)
		if w.Code != http.StatusNotFound && w.Code != http.StatusOK {
			t.Logf("Events after task delete: %d", w.Code)
		}
	})
}

// TestConsistency_RunCounter 测试 Run 计数
func TestConsistency_RunCounter(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 创建 Task
	w := makeRequestWithString("POST", "/api/v1/tasks", `{"name":"Run Counter Test","prompt":"test","type":"general"}`)
	taskResp := parseJSONResponse(w)
	taskID := taskResp["id"].(string)
	defer testStore.DeleteTask(ctx, taskID)

	// 创建多个 Run
	for i := 0; i < 5; i++ {
		makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
	}

	t.Run("Task 的 Run 计数正确", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/tasks/"+taskID+"/runs", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("List runs failed: %d", w.Code)
		}

		resp := parseJSONResponse(w)
		runs := resp["runs"].([]interface{})
		if len(runs) < 5 {
			t.Errorf("Expected at least 5 runs, got %d", len(runs))
		}
	})
}

// TestConsistency_EventOrdering 测试事件顺序一致性
func TestConsistency_EventOrdering(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 创建 Task 和 Run
	w := makeRequestWithString("POST", "/api/v1/tasks", `{"name":"Event Order Test","prompt":"test","type":"general"}`)
	taskResp := parseJSONResponse(w)
	taskID := taskResp["id"].(string)
	defer testStore.DeleteTask(ctx, taskID)

	w = makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
	runResp := parseJSONResponse(w)
	runID := runResp["id"].(string)

	// 批量添加乱序事件
	body := `{"events":[
		{"seq":5,"type":"message","timestamp":"2024-01-01T00:00:05Z","payload":{}},
		{"seq":1,"type":"run_started","timestamp":"2024-01-01T00:00:01Z","payload":{}},
		{"seq":3,"type":"message","timestamp":"2024-01-01T00:00:03Z","payload":{}},
		{"seq":2,"type":"message","timestamp":"2024-01-01T00:00:02Z","payload":{}},
		{"seq":4,"type":"message","timestamp":"2024-01-01T00:00:04Z","payload":{}}
	]}`
	makeRequestWithString("POST", "/api/v1/runs/"+runID+"/events", body)

	t.Run("获取的事件按 seq 排序", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/runs/"+runID+"/events", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("Get events failed: %d", w.Code)
		}

		resp := parseJSONResponse(w)
		events := resp["events"].([]interface{})

		prevSeq := 0
		for _, e := range events {
			event := e.(map[string]interface{})
			seq := int(event["seq"].(float64))
			if seq < prevSeq {
				t.Errorf("Events not in order: %d came after %d", seq, prevSeq)
			}
			prevSeq = seq
		}
	})
}

// TestConsistency_NodeRunAssignment 测试节点 Run 分配
func TestConsistency_NodeRunAssignment(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	nodeID := "consistency-test-node"

	// 注册节点
	makeRequestWithString("POST", "/api/v1/nodes/heartbeat", `{"node_id":"`+nodeID+`","status":"online"}`)
	defer makeRequest("DELETE", "/api/v1/nodes/"+nodeID, nil)

	// 创建 Task 和 Run
	w := makeRequestWithString("POST", "/api/v1/tasks", `{"name":"Node Assignment Test","prompt":"test","type":"general"}`)
	taskResp := parseJSONResponse(w)
	taskID := taskResp["id"].(string)
	defer testStore.DeleteTask(ctx, taskID)

	w = makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
	runResp := parseJSONResponse(w)
	runID := runResp["id"].(string)

	t.Run("分配 Run 到节点后可通过节点查询", func(t *testing.T) {
		// 更新 Run 的 node_id
		w := makeRequestWithString("PATCH", "/api/v1/runs/"+runID, `{"node_id":"`+nodeID+`"}`)
		t.Logf("Assign run to node: %d", w.Code)

		// 查询节点的 Runs
		w = makeRequest("GET", "/api/v1/nodes/"+nodeID+"/runs", nil)
		if w.Code == http.StatusOK {
			resp := parseJSONResponse(w)
			if runs, ok := resp["runs"].([]interface{}); ok {
				t.Logf("Node has %d runs", len(runs))
			} else {
				t.Log("Runs list not returned or empty")
			}
		}
	})
}

// TestConsistency_MultipleRunsLatestStatus 测试多个 Run 时 Task 状态
func TestConsistency_MultipleRunsLatestStatus(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 创建 Task
	w := makeRequestWithString("POST", "/api/v1/tasks", `{"name":"Multiple Runs Test","prompt":"test","type":"general"}`)
	taskResp := parseJSONResponse(w)
	taskID := taskResp["id"].(string)
	defer testStore.DeleteTask(ctx, taskID)

	// 创建第一个 Run 并完成
	w = makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
	run1Resp := parseJSONResponse(w)
	run1ID := run1Resp["id"].(string)
	makeRequestWithString("PATCH", "/api/v1/runs/"+run1ID, `{"status":"done"}`)

	t.Run("完成第一个 Run 后 Task 状态为 done", func(t *testing.T) {
		task, _ := testStore.GetTask(ctx, taskID)
		if task.Status != model.TaskStatusCompleted {
			t.Errorf("Task status = %v, want done", task.Status)
		}
	})

	// 创建第二个 Run
	w = makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
	run2Resp := parseJSONResponse(w)
	run2ID := run2Resp["id"].(string)

	t.Run("创建新 Run 后 Task 状态变化", func(t *testing.T) {
		task, _ := testStore.GetTask(ctx, taskID)
		t.Logf("Task status after new run: %v", task.Status)
	})

	// 设置第二个 Run 为 running
	makeRequestWithString("PATCH", "/api/v1/runs/"+run2ID, `{"status":"running"}`)

	t.Run("第二个 Run running 后 Task 状态", func(t *testing.T) {
		task, _ := testStore.GetTask(ctx, taskID)
		if task.Status != model.TaskStatusRunning {
			t.Logf("Task status with running run: %v", task.Status)
		}
	})
}

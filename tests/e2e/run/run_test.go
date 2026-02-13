package run

import (
	"net/http"
	"testing"
	"time"

	"agents-admin/tests/testutil"
)

// TestRun_Lifecycle 验证 Run 完整生命周期：创建→状态查询→事件上报→取消
func TestRun_Lifecycle(t *testing.T) {
	// 创建 Task
	taskPayload := map[string]interface{}{
		"name": "E2E Run Lifecycle - " + time.Now().Format(time.RFC3339),
		"spec": map[string]interface{}{
			"prompt": "run lifecycle test",
			"agent":  map[string]interface{}{"type": "gemini"},
		},
	}
	resp, err := c.Post("/api/v1/tasks", taskPayload)
	if err != nil {
		t.Fatalf("Create task failed: %v", err)
	}
	taskResult := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Create task returned %d", resp.StatusCode)
	}
	taskID := taskResult["id"].(string)
	defer c.Delete("/api/v1/tasks/" + taskID)

	// 创建 Run
	resp, err = c.Post("/api/v1/tasks/"+taskID+"/runs", nil)
	if err != nil {
		t.Fatalf("Create run failed: %v", err)
	}
	runResult := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Create run returned %d", resp.StatusCode)
	}
	runID := runResult["id"].(string)
	t.Logf("Created run: %s", runID)

	// 获取 Run 详情
	resp, err = c.Get("/api/v1/runs/" + runID)
	if err != nil {
		t.Fatalf("Get run failed: %v", err)
	}
	getResult := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Get run returned %d", resp.StatusCode)
	}
	status := getResult["status"].(string)
	if status != "queued" && status != "running" {
		t.Errorf("Run status = %v, want queued or running", status)
	}

	// 取消 Run
	resp, err = c.Post("/api/v1/runs/"+runID+"/cancel", nil)
	if err != nil {
		t.Fatalf("Cancel run failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Cancel run returned %d", resp.StatusCode)
	}
}

// TestRun_Events 验证事件上报与获取
func TestRun_Events(t *testing.T) {
	// 创建 Task + Run
	taskPayload := map[string]interface{}{
		"name": "E2E Run Events - " + time.Now().Format(time.RFC3339),
		"spec": map[string]interface{}{
			"prompt": "events test",
			"agent":  map[string]interface{}{"type": "gemini"},
		},
	}
	resp, err := c.Post("/api/v1/tasks", taskPayload)
	if err != nil {
		t.Fatalf("Create task failed: %v", err)
	}
	taskResult := testutil.ReadJSON(resp)
	taskID := taskResult["id"].(string)
	defer c.Delete("/api/v1/tasks/" + taskID)

	resp, err = c.Post("/api/v1/tasks/"+taskID+"/runs", nil)
	if err != nil {
		t.Fatalf("Create run failed: %v", err)
	}
	runResult := testutil.ReadJSON(resp)
	runID := runResult["id"].(string)

	// 上报事件
	now := time.Now().Format(time.RFC3339Nano)
	events := map[string]interface{}{
		"events": []map[string]interface{}{
			{"seq": 1, "type": "run_started", "timestamp": now, "payload": map[string]interface{}{"node_id": "e2e-node"}},
			{"seq": 2, "type": "message", "timestamp": now, "payload": map[string]interface{}{"role": "assistant", "content": "Starting..."}},
			{"seq": 3, "type": "tool_use_start", "timestamp": now, "payload": map[string]interface{}{"tool_id": "t1", "tool_name": "Read"}},
		},
	}
	resp, err = c.Post("/api/v1/runs/"+runID+"/events", events)
	if err != nil {
		t.Fatalf("Post events failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Post events returned %d", resp.StatusCode)
	}

	// 获取事件
	resp, err = c.Get("/api/v1/runs/" + runID + "/events")
	if err != nil {
		t.Fatalf("Get events failed: %v", err)
	}
	eventsResult := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Get events returned %d", resp.StatusCode)
	}
	if count, ok := eventsResult["count"].(float64); ok {
		if int(count) < 3 {
			t.Errorf("Expected >= 3 events, got %d", int(count))
		}
	}
}

// TestRun_ListByTask 验证按任务列出 Runs
func TestRun_ListByTask(t *testing.T) {
	taskPayload := map[string]interface{}{
		"name": "E2E Run List - " + time.Now().Format(time.RFC3339),
		"spec": map[string]interface{}{
			"prompt": "list test",
			"agent":  map[string]interface{}{"type": "gemini"},
		},
	}
	resp, err := c.Post("/api/v1/tasks", taskPayload)
	if err != nil {
		t.Fatalf("Create task failed: %v", err)
	}
	taskResult := testutil.ReadJSON(resp)
	taskID := taskResult["id"].(string)
	defer c.Delete("/api/v1/tasks/" + taskID)

	// 创建 3 个 Run
	for i := 0; i < 3; i++ {
		c.Post("/api/v1/tasks/"+taskID+"/runs", nil)
	}

	// 列出
	resp, err = c.Get("/api/v1/tasks/" + taskID + "/runs")
	if err != nil {
		t.Fatalf("List runs failed: %v", err)
	}
	result := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("List runs returned %d", resp.StatusCode)
	}
	if result["runs"] == nil {
		t.Error("Runs list not returned")
	}
}

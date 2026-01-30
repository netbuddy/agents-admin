// Package e2e 端到端测试
// 测试完整的用户流程：创建任务 → 启动 Run → 事件流 → 完成
package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

var apiBaseURL string

func TestMain(m *testing.M) {
	apiBaseURL = os.Getenv("API_BASE_URL")
	if apiBaseURL == "" {
		apiBaseURL = "http://localhost:8080"
	}

	// 等待 API Server 就绪
	ready := waitForAPI(apiBaseURL, 10*time.Second)
	if !ready {
		fmt.Println("API Server not ready, skipping E2E tests")
		os.Exit(0)
	}

	os.Exit(m.Run())
}

func waitForAPI(baseURL string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

func TestE2E_FullTaskLifecycle(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}

	// Step 1: 创建任务
	t.Log("Step 1: Creating task...")
	taskPayload := map[string]interface{}{
		"name": "E2E Test Task - " + time.Now().Format(time.RFC3339),
		"spec": map[string]interface{}{
			"prompt": "This is an E2E test prompt",
			"agent": map[string]interface{}{
				"type": "gemini",
			},
		},
	}
	taskBody, _ := json.Marshal(taskPayload)

	resp, err := client.Post(apiBaseURL+"/api/v1/tasks", "application/json", bytes.NewReader(taskBody))
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Create task returned %d", resp.StatusCode)
	}

	var taskResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&taskResp)
	taskID := taskResp["id"].(string)
	t.Logf("Created task: %s", taskID)

	// Step 2: 验证任务创建成功
	t.Log("Step 2: Verifying task...")
	resp, err = client.Get(apiBaseURL + "/api/v1/tasks/" + taskID)
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Get task returned %d", resp.StatusCode)
	}

	var getTaskResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&getTaskResp)
	if getTaskResp["status"] != "pending" {
		t.Errorf("Initial status should be 'pending', got %v", getTaskResp["status"])
	}

	// Step 3: 创建 Run
	t.Log("Step 3: Creating run...")
	resp, err = client.Post(apiBaseURL+"/api/v1/tasks/"+taskID+"/runs", "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to create run: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Create run returned %d", resp.StatusCode)
	}

	var runResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&runResp)
	runID := runResp["id"].(string)
	t.Logf("Created run: %s", runID)

	// Step 4: 验证 Run 状态
	t.Log("Step 4: Verifying run status...")
	resp, err = client.Get(apiBaseURL + "/api/v1/runs/" + runID)
	if err != nil {
		t.Fatalf("Failed to get run: %v", err)
	}
	defer resp.Body.Close()

	var getRunResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&getRunResp)
	status := getRunResp["status"].(string)
	if status != "queued" && status != "running" {
		t.Errorf("Run status should be 'queued' or 'running', got %v", status)
	}

	// Step 5: 模拟事件上报
	t.Log("Step 5: Posting events...")
	now := time.Now().Format(time.RFC3339Nano)
	eventsPayload := map[string]interface{}{
		"events": []map[string]interface{}{
			{
				"seq":       1,
				"type":      "run_started",
				"timestamp": now,
				"payload":   map[string]interface{}{"node_id": "e2e-test-node"},
			},
			{
				"seq":       2,
				"type":      "message",
				"timestamp": now,
				"payload":   map[string]interface{}{"role": "assistant", "content": "Starting task..."},
			},
			{
				"seq":       3,
				"type":      "tool_use_start",
				"timestamp": now,
				"payload":   map[string]interface{}{"tool_id": "tool_001", "tool_name": "Read"},
			},
		},
	}
	eventsBody, _ := json.Marshal(eventsPayload)

	resp, err = client.Post(apiBaseURL+"/api/v1/runs/"+runID+"/events", "application/json", bytes.NewReader(eventsBody))
	if err != nil {
		t.Fatalf("Failed to post events: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Post events returned %d", resp.StatusCode)
	}

	// Step 6: 获取事件
	t.Log("Step 6: Getting events...")
	resp, err = client.Get(apiBaseURL + "/api/v1/runs/" + runID + "/events")
	if err != nil {
		t.Fatalf("Failed to get events: %v", err)
	}
	defer resp.Body.Close()

	var eventsResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&eventsResp)
	eventCount := int(eventsResp["count"].(float64))
	if eventCount < 3 {
		t.Errorf("Expected at least 3 events, got %d", eventCount)
	}
	t.Logf("Retrieved %d events", eventCount)

	// Step 7: 取消 Run
	t.Log("Step 7: Cancelling run...")
	resp, err = client.Post(apiBaseURL+"/api/v1/runs/"+runID+"/cancel", "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to cancel run: %v", err)
	}
	defer resp.Body.Close()

	// Step 8: 清理 - 删除任务
	t.Log("Step 8: Cleaning up...")
	req, _ := http.NewRequest("DELETE", apiBaseURL+"/api/v1/tasks/"+taskID, nil)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Delete task returned %d", resp.StatusCode)
	}

	t.Log("E2E test completed successfully!")
}

func TestE2E_NodeHeartbeatAndScheduling(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}

	// Step 1: 注册节点
	t.Log("Step 1: Registering node...")
	heartbeatPayload := map[string]interface{}{
		"node_id": "e2e-node-" + time.Now().Format("150405"),
		"status":  "online",
		"labels": map[string]string{
			"os":   "linux",
			"arch": "amd64",
		},
		"capacity": map[string]interface{}{
			"max_concurrent": 4,
			"available":      4,
		},
	}
	heartbeatBody, _ := json.Marshal(heartbeatPayload)

	resp, err := client.Post(apiBaseURL+"/api/v1/nodes/heartbeat", "application/json", bytes.NewReader(heartbeatBody))
	if err != nil {
		t.Fatalf("Failed to send heartbeat: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Heartbeat returned %d", resp.StatusCode)
	}

	// Step 2: 验证节点列表
	t.Log("Step 2: Listing nodes...")
	resp, err = client.Get(apiBaseURL + "/api/v1/nodes")
	if err != nil {
		t.Fatalf("Failed to list nodes: %v", err)
	}
	defer resp.Body.Close()

	var nodesResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&nodesResp)
	nodeCount := int(nodesResp["count"].(float64))
	t.Logf("Found %d online nodes", nodeCount)

	if nodeCount < 1 {
		t.Error("Expected at least 1 node")
	}

	t.Log("Node heartbeat test completed!")
}

func TestE2E_APIHealthCheck(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(apiBaseURL + "/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Health check returned %d", resp.StatusCode)
	}

	var healthResp map[string]string
	json.NewDecoder(resp.Body).Decode(&healthResp)

	if healthResp["status"] != "ok" {
		t.Errorf("Health status = %v, want 'ok'", healthResp["status"])
	}
}

func TestE2E_TaskListPagination(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}

	// 创建多个任务
	taskIDs := []string{}
	for i := 0; i < 5; i++ {
		payload := map[string]interface{}{
			"name": fmt.Sprintf("Pagination Test %d", i),
			"spec": map[string]interface{}{
				"prompt": "test",
				"agent":  map[string]interface{}{"type": "gemini"},
			},
		}
		body, _ := json.Marshal(payload)
		resp, err := client.Post(apiBaseURL+"/api/v1/tasks", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
		var taskResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&taskResp)
		resp.Body.Close()
		taskIDs = append(taskIDs, taskResp["id"].(string))
	}

	// 分页获取
	resp, err := client.Get(apiBaseURL + "/api/v1/tasks?limit=3&offset=0")
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}
	var listResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&listResp)
	resp.Body.Close()

	count := int(listResp["count"].(float64))
	if count > 3 {
		t.Errorf("Expected max 3 tasks with limit=3, got %d", count)
	}

	// 清理
	for _, id := range taskIDs {
		req, _ := http.NewRequest("DELETE", apiBaseURL+"/api/v1/tasks/"+id, nil)
		client.Do(req)
	}
}

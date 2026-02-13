package hitl

import (
	"net/http"
	"testing"
	"time"

	"agents-admin/tests/testutil"
)

// createTaskAndRun 创建测试用的 Task + Run，返回 taskID, runID
func createTaskAndRun(t *testing.T) (string, string) {
	t.Helper()
	taskPayload := map[string]interface{}{
		"name": "E2E HITL - " + time.Now().Format(time.RFC3339),
		"spec": map[string]interface{}{
			"prompt": "hitl test",
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

	resp, err = c.Post("/api/v1/tasks/"+taskID+"/runs", nil)
	if err != nil {
		t.Fatalf("Create run failed: %v", err)
	}
	runResult := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Create run returned %d", resp.StatusCode)
	}
	runID := runResult["id"].(string)
	return taskID, runID
}

// TestHITL_Approvals 验证审批请求列表
func TestHITL_Approvals(t *testing.T) {
	taskID, runID := createTaskAndRun(t)
	defer c.Delete("/api/v1/tasks/" + taskID)

	resp, err := c.Get("/api/v1/runs/" + runID + "/approvals")
	if err != nil {
		t.Fatalf("List approvals failed: %v", err)
	}
	defer resp.Body.Close()
	// 应返回空列表或 200
	if resp.StatusCode != http.StatusOK {
		t.Errorf("List approvals returned %d", resp.StatusCode)
	}
}

// TestHITL_Feedbacks 验证人工反馈接口
func TestHITL_Feedbacks(t *testing.T) {
	taskID, runID := createTaskAndRun(t)
	defer c.Delete("/api/v1/tasks/" + taskID)

	// 列出反馈
	resp, err := c.Get("/api/v1/runs/" + runID + "/feedbacks")
	if err != nil {
		t.Fatalf("List feedbacks failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("List feedbacks returned %d", resp.StatusCode)
	}

	// 创建反馈
	payload := map[string]interface{}{
		"content": "E2E test feedback",
		"type":    "instruction",
	}
	resp, err = c.Post("/api/v1/runs/"+runID+"/feedbacks", payload)
	if err != nil {
		t.Fatalf("Create feedback failed: %v", err)
	}
	defer resp.Body.Close()
	t.Logf("Create feedback returned %d", resp.StatusCode)
}

// TestHITL_Interventions 验证干预操作接口
func TestHITL_Interventions(t *testing.T) {
	taskID, runID := createTaskAndRun(t)
	defer c.Delete("/api/v1/tasks/" + taskID)

	resp, err := c.Get("/api/v1/runs/" + runID + "/interventions")
	if err != nil {
		t.Fatalf("List interventions failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("List interventions returned %d", resp.StatusCode)
	}
}

// TestHITL_Confirmations 验证确认请求接口
func TestHITL_Confirmations(t *testing.T) {
	taskID, runID := createTaskAndRun(t)
	defer c.Delete("/api/v1/tasks/" + taskID)

	resp, err := c.Get("/api/v1/runs/" + runID + "/confirmations")
	if err != nil {
		t.Fatalf("List confirmations failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("List confirmations returned %d", resp.StatusCode)
	}
}

// TestHITL_PendingItems 验证 HITL 待处理汇总
func TestHITL_PendingItems(t *testing.T) {
	taskID, runID := createTaskAndRun(t)
	defer c.Delete("/api/v1/tasks/" + taskID)

	resp, err := c.Get("/api/v1/runs/" + runID + "/hitl/pending")
	if err != nil {
		t.Fatalf("Get pending HITL items failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Get pending returned %d", resp.StatusCode)
	}
}

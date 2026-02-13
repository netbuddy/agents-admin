package task

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"agents-admin/tests/testutil"
)

// TestTask_CRUD 验证任务完整生命周期：创建→读取→列表→删除
func TestTask_CRUD(t *testing.T) {
	// 创建
	payload := map[string]interface{}{
		"name": "E2E Task CRUD - " + time.Now().Format(time.RFC3339),
		"spec": map[string]interface{}{
			"prompt": "Test CRUD lifecycle",
			"agent":  map[string]interface{}{"type": "gemini"},
		},
	}
	resp, err := c.Post("/api/v1/tasks", payload)
	if err != nil {
		t.Fatalf("Create task failed: %v", err)
	}
	result := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Create returned %d", resp.StatusCode)
	}
	taskID := result["id"].(string)
	t.Logf("Created task: %s", taskID)

	// 读取
	resp, err = c.Get("/api/v1/tasks/" + taskID)
	if err != nil {
		t.Fatalf("Get task failed: %v", err)
	}
	result = testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Get returned %d", resp.StatusCode)
	}
	if result["id"] != taskID {
		t.Errorf("Task ID mismatch: %v", result["id"])
	}

	// 列表（应包含刚创建的任务）
	resp, err = c.Get("/api/v1/tasks")
	if err != nil {
		t.Fatalf("List tasks failed: %v", err)
	}
	listResult := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("List returned %d", resp.StatusCode)
	}
	if listResult["tasks"] == nil {
		t.Error("Tasks list not returned")
	}

	// 删除
	resp, err = c.Delete("/api/v1/tasks/" + taskID)
	if err != nil {
		t.Fatalf("Delete task failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Delete returned %d", resp.StatusCode)
	}

	// 确认已删除
	resp, err = c.Get("/api/v1/tasks/" + taskID)
	if err != nil {
		t.Fatalf("Get deleted task failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Deleted task still accessible: %d", resp.StatusCode)
	}
}

// TestTask_Pagination 验证分页查询
func TestTask_Pagination(t *testing.T) {
	// 创建 5 个任务
	var taskIDs []string
	for i := 0; i < 5; i++ {
		payload := map[string]interface{}{
			"name": fmt.Sprintf("E2E Pagination %d", i),
			"spec": map[string]interface{}{
				"prompt": "pagination test",
				"agent":  map[string]interface{}{"type": "gemini"},
			},
		}
		resp, err := c.Post("/api/v1/tasks", payload)
		if err != nil {
			t.Fatalf("Create task %d failed: %v", i, err)
		}
		result := testutil.ReadJSON(resp)
		if id, ok := result["id"].(string); ok {
			taskIDs = append(taskIDs, id)
		}
	}
	defer func() {
		for _, id := range taskIDs {
			c.Delete("/api/v1/tasks/" + id)
		}
	}()

	// 分页获取（limit=3）
	resp, err := c.Get("/api/v1/tasks?limit=3&offset=0")
	if err != nil {
		t.Fatalf("Paginated list failed: %v", err)
	}
	result := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Paginated list returned %d", resp.StatusCode)
	}
	tasks := result["tasks"].([]interface{})
	if len(tasks) > 3 {
		t.Errorf("Expected max 3 tasks, got %d", len(tasks))
	}
}

// TestTask_SubTasks 验证子任务查询
func TestTask_SubTasks(t *testing.T) {
	// 创建父任务
	payload := map[string]interface{}{
		"name": "E2E Parent Task",
		"spec": map[string]interface{}{
			"prompt": "parent task",
			"agent":  map[string]interface{}{"type": "gemini"},
		},
	}
	resp, err := c.Post("/api/v1/tasks", payload)
	if err != nil {
		t.Fatalf("Create parent task failed: %v", err)
	}
	result := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Create returned %d", resp.StatusCode)
	}
	parentID := result["id"].(string)
	defer c.Delete("/api/v1/tasks/" + parentID)

	// 获取子任务列表（可能为空）
	resp, err = c.Get("/api/v1/tasks/" + parentID + "/subtasks")
	if err != nil {
		t.Fatalf("Get subtasks failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Get subtasks returned %d", resp.StatusCode)
	}

	// 获取任务树
	resp, err = c.Get("/api/v1/tasks/" + parentID + "/tree")
	if err != nil {
		t.Fatalf("Get tree failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Get tree returned %d", resp.StatusCode)
	}
}

// TestTask_NotFound 验证不存在任务返回 404
func TestTask_NotFound(t *testing.T) {
	resp, err := c.Get("/api/v1/tasks/nonexistent-task-id")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Nonexistent task returned %d, want 404", resp.StatusCode)
	}
}

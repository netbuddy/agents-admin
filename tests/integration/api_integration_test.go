// Package integration 集成测试
// 需要运行中的 PostgreSQL 实例
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"agents-admin/internal/api"
	"agents-admin/internal/model"
	"agents-admin/internal/storage"
)

var testStore *storage.PostgresStore
var testHandler *api.Handler

func TestMain(m *testing.M) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://agents:agents_dev_password@localhost:5432/agents_admin?sslmode=disable"
	}

	var err error
	testStore, err = storage.NewPostgresStore(dbURL)
	if err != nil {
		// 如果无法连接数据库，跳过集成测试
		os.Exit(0)
	}
	defer testStore.Close()

	testHandler = api.NewHandler(testStore, nil, nil) // redis 和 etcd 在测试中可选

	os.Exit(m.Run())
}

func TestTaskCRUD_Integration(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	router := testHandler.Router()

	// 1. Create Task
	createBody := `{"name":"Integration Test Task","spec":{"prompt":"test prompt","agent":{"type":"gemini"}}}`
	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Create task failed: %d - %s", w.Code, w.Body.String())
	}

	var createResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&createResp)
	taskID := createResp["id"].(string)
	t.Logf("Created task: %s", taskID)

	// 2. Get Task
	req = httptest.NewRequest("GET", "/api/v1/tasks/"+taskID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Get task failed: %d", w.Code)
	}

	var getResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&getResp)
	if getResp["name"] != "Integration Test Task" {
		t.Errorf("Name = %v, want 'Integration Test Task'", getResp["name"])
	}

	// 3. List Tasks
	req = httptest.NewRequest("GET", "/api/v1/tasks", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("List tasks failed: %d", w.Code)
	}

	// 4. Delete Task
	req = httptest.NewRequest("DELETE", "/api/v1/tasks/"+taskID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("Delete task failed: %d", w.Code)
	}

	// 5. Verify Deleted
	req = httptest.NewRequest("GET", "/api/v1/tasks/"+taskID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 after delete, got %d", w.Code)
	}
}

func TestRunLifecycle_Integration(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	router := testHandler.Router()
	ctx := context.Background()

	// 1. Create Task
	createBody := `{"name":"Run Lifecycle Test","spec":{"prompt":"lifecycle test","agent":{"type":"gemini"}}}`
	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var taskResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&taskResp)
	taskID := taskResp["id"].(string)
	defer testStore.DeleteTask(ctx, taskID)

	// 2. Create Run
	req = httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Create run failed: %d - %s", w.Code, w.Body.String())
	}

	var runResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&runResp)
	runID := runResp["id"].(string)
	t.Logf("Created run: %s", runID)

	// 3. Get Run
	req = httptest.NewRequest("GET", "/api/v1/runs/"+runID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Get run failed: %d", w.Code)
	}

	var getRunResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&getRunResp)
	if getRunResp["status"] != "queued" {
		t.Errorf("Status = %v, want 'queued'", getRunResp["status"])
	}

	// 4. List Runs for Task
	req = httptest.NewRequest("GET", "/api/v1/tasks/"+taskID+"/runs", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("List runs failed: %d", w.Code)
	}

	// 5. Cancel Run
	req = httptest.NewRequest("POST", "/api/v1/runs/"+runID+"/cancel", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Cancel run failed: %d", w.Code)
	}
}

func TestEventReporting_Integration(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	router := testHandler.Router()
	ctx := context.Background()

	// Setup: Create task and run
	createBody := `{"name":"Event Test","spec":{"prompt":"event test","agent":{"type":"gemini"}}}`
	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var taskResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&taskResp)
	taskID := taskResp["id"].(string)
	defer testStore.DeleteTask(ctx, taskID)

	req = httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var runResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&runResp)
	runID := runResp["id"].(string)

	// Post Events
	now := time.Now().Format(time.RFC3339Nano)
	eventsBody := `{"events":[
		{"seq":1,"type":"run_started","timestamp":"` + now + `","payload":{"node":"node-001"}},
		{"seq":2,"type":"message","timestamp":"` + now + `","payload":{"content":"hello"}}
	]}`
	req = httptest.NewRequest("POST", "/api/v1/runs/"+runID+"/events", bytes.NewBufferString(eventsBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Post events failed: %d - %s", w.Code, w.Body.String())
	}

	// Get Events
	req = httptest.NewRequest("GET", "/api/v1/runs/"+runID+"/events", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Get events failed: %d", w.Code)
	}

	var eventsResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&eventsResp)

	count := int(eventsResp["count"].(float64))
	if count != 2 {
		t.Errorf("Expected 2 events, got %d", count)
	}
}

func TestNodeHeartbeat_Integration(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	router := testHandler.Router()

	// Send Heartbeat
	heartbeatBody := `{
		"node_id": "test-node-001",
		"status": "online",
		"labels": {"os": "linux", "gpu": "true"},
		"capacity": {"max_concurrent": 4, "available": 4}
	}`
	req := httptest.NewRequest("POST", "/api/v1/nodes/heartbeat", bytes.NewBufferString(heartbeatBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Heartbeat failed: %d - %s", w.Code, w.Body.String())
	}

	// List Nodes
	req = httptest.NewRequest("GET", "/api/v1/nodes", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("List nodes failed: %d", w.Code)
	}

	var nodesResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&nodesResp)

	nodes := nodesResp["nodes"].([]interface{})
	found := false
	for _, n := range nodes {
		node := n.(map[string]interface{})
		if node["id"] == "test-node-001" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find test-node-001 in nodes list")
	}
}

// TestUpdateRun_Integration 测试更新 Run 状态
func TestUpdateRun_Integration(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	router := testHandler.Router()
	ctx := context.Background()

	// 1. 创建 Task
	createBody := `{"name":"UpdateRun Test","spec":{"prompt":"test","agent":{"type":"gemini"}}}`
	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var taskResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&taskResp)
	taskID := taskResp["id"].(string)
	defer testStore.DeleteTask(ctx, taskID)

	// 2. 创建 Run
	req = httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Create run failed: %d - %s", w.Code, w.Body.String())
	}

	var runResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&runResp)
	runID := runResp["id"].(string)

	// 3. 更新 Run 状态为 done
	updateBody := `{"status":"done"}`
	req = httptest.NewRequest("PATCH", "/api/v1/runs/"+runID, bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Update run failed: %d - %s", w.Code, w.Body.String())
	}

	// 4. 验证状态已更新
	req = httptest.NewRequest("GET", "/api/v1/runs/"+runID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var getRunResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&getRunResp)
	if getRunResp["status"] != "done" {
		t.Errorf("Status = %v, want 'done'", getRunResp["status"])
	}
}

// TestGetNodeRuns_Integration 测试获取节点分配的 Runs
func TestGetNodeRuns_Integration(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	router := testHandler.Router()
	ctx := context.Background()

	// 1. 注册节点
	nodeID := "test-node-runs-001"
	heartbeatBody := `{
		"node_id": "` + nodeID + `",
		"status": "online",
		"labels": {"os": "linux"},
		"capacity": {"max_concurrent": 2}
	}`
	req := httptest.NewRequest("POST", "/api/v1/nodes/heartbeat", bytes.NewBufferString(heartbeatBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Heartbeat failed: %d", w.Code)
	}

	// 2. 创建 Task 和 Run
	createBody := `{"name":"NodeRuns Test","spec":{"prompt":"test"}}`
	req = httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var taskResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&taskResp)
	taskID := taskResp["id"].(string)
	defer testStore.DeleteTask(ctx, taskID)

	req = httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 3. 获取节点的 Runs（此时可能为空，因为调度器可能还没运行）
	req = httptest.NewRequest("GET", "/api/v1/nodes/"+nodeID+"/runs", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Get node runs failed: %d - %s", w.Code, w.Body.String())
	}

	var runsResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&runsResp)

	// 验证响应格式正确
	if _, ok := runsResp["runs"]; !ok {
		t.Error("Expected 'runs' field in response")
	}
	if _, ok := runsResp["count"]; !ok {
		t.Error("Expected 'count' field in response")
	}
}

// TestGetNode_Integration 测试获取单个节点
func TestGetNode_Integration(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	router := testHandler.Router()

	// 1. 注册节点
	nodeID := "test-get-node-001"
	heartbeatBody := `{
		"node_id": "` + nodeID + `",
		"status": "online",
		"labels": {"os": "linux", "env": "test"},
		"capacity": {"max_concurrent": 2}
	}`
	req := httptest.NewRequest("POST", "/api/v1/nodes/heartbeat", bytes.NewBufferString(heartbeatBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Heartbeat failed: %d", w.Code)
	}

	// 2. 获取节点详情
	req = httptest.NewRequest("GET", "/api/v1/nodes/"+nodeID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Get node failed: %d - %s", w.Code, w.Body.String())
	}

	var nodeResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&nodeResp)

	if nodeResp["id"] != nodeID {
		t.Errorf("Node ID = %v, want %v", nodeResp["id"], nodeID)
	}
	if nodeResp["status"] != "online" {
		t.Errorf("Node status = %v, want 'online'", nodeResp["status"])
	}

	// 3. 测试获取不存在的节点
	req = httptest.NewRequest("GET", "/api/v1/nodes/non-existent-node", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for non-existent node, got %d", w.Code)
	}
}

// TestUpdateNode_Integration 测试更新节点
func TestUpdateNode_Integration(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	router := testHandler.Router()

	// 1. 注册节点
	nodeID := "test-update-node-001"
	heartbeatBody := `{
		"node_id": "` + nodeID + `",
		"status": "online",
		"labels": {"os": "linux"},
		"capacity": {"max_concurrent": 2}
	}`
	req := httptest.NewRequest("POST", "/api/v1/nodes/heartbeat", bytes.NewBufferString(heartbeatBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Heartbeat failed: %d", w.Code)
	}

	// 2. 更新节点状态为 draining
	updateBody := `{"status": "draining"}`
	req = httptest.NewRequest("PATCH", "/api/v1/nodes/"+nodeID, bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Update node failed: %d - %s", w.Code, w.Body.String())
	}

	var updateResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&updateResp)

	if updateResp["status"] != "draining" {
		t.Errorf("Status = %v, want 'draining'", updateResp["status"])
	}

	// 3. 测试更新不存在的节点
	req = httptest.NewRequest("PATCH", "/api/v1/nodes/non-existent-node", bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for non-existent node, got %d", w.Code)
	}
}

// TestDeleteNode_Integration 测试删除节点
func TestDeleteNode_Integration(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	router := testHandler.Router()

	// 1. 注册节点
	nodeID := "test-delete-node-001"
	heartbeatBody := `{
		"node_id": "` + nodeID + `",
		"status": "online",
		"labels": {"os": "linux"},
		"capacity": {"max_concurrent": 2}
	}`
	req := httptest.NewRequest("POST", "/api/v1/nodes/heartbeat", bytes.NewBufferString(heartbeatBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Heartbeat failed: %d", w.Code)
	}

	// 2. 将节点状态改为 offline（避免 "has running tasks" 错误）
	updateBody := `{"status": "offline"}`
	req = httptest.NewRequest("PATCH", "/api/v1/nodes/"+nodeID, bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 3. 删除节点
	req = httptest.NewRequest("DELETE", "/api/v1/nodes/"+nodeID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("Delete node failed: %d - %s", w.Code, w.Body.String())
	}

	// 4. 验证节点已删除
	req = httptest.NewRequest("GET", "/api/v1/nodes/"+nodeID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 after delete, got %d", w.Code)
	}

	// 5. 测试删除不存在的节点
	req = httptest.NewRequest("DELETE", "/api/v1/nodes/non-existent-node", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for non-existent node, got %d", w.Code)
	}
}

// TestHealthCheck_Integration 测试健康检查接口
func TestHealthCheck_Integration(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	router := testHandler.Router()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Health check failed: %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["status"] != "ok" {
		t.Errorf("Status = %v, want 'ok'", resp["status"])
	}
}

// TestTaskWithFilters_Integration 测试带过滤条件的任务列表
func TestTaskWithFilters_Integration(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	router := testHandler.Router()
	ctx := context.Background()

	// 1. 创建测试任务
	createBody := `{"name":"Filter Test Task","spec":{"prompt":"test"}}`
	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var taskResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&taskResp)
	taskID := taskResp["id"].(string)
	defer testStore.DeleteTask(ctx, taskID)

	// 2. 测试按状态过滤
	req = httptest.NewRequest("GET", "/api/v1/tasks?status=pending", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("List tasks with filter failed: %d", w.Code)
	}

	// 3. 测试分页
	req = httptest.NewRequest("GET", "/api/v1/tasks?limit=5&offset=0", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("List tasks with pagination failed: %d", w.Code)
	}
}

// TestEventsWithPagination_Integration 测试事件分页
func TestEventsWithPagination_Integration(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	router := testHandler.Router()
	ctx := context.Background()

	// 1. 创建 Task 和 Run
	createBody := `{"name":"Events Pagination Test","spec":{"prompt":"test"}}`
	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var taskResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&taskResp)
	taskID := taskResp["id"].(string)
	defer testStore.DeleteTask(ctx, taskID)

	req = httptest.NewRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var runResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&runResp)
	runID := runResp["id"].(string)

	// 2. 上报多个事件
	now := time.Now().Format(time.RFC3339Nano)
	eventsBody := `{"events":[
		{"seq":1,"type":"run_started","timestamp":"` + now + `","payload":{}},
		{"seq":2,"type":"message","timestamp":"` + now + `","payload":{"content":"step 1"}},
		{"seq":3,"type":"message","timestamp":"` + now + `","payload":{"content":"step 2"}},
		{"seq":4,"type":"message","timestamp":"` + now + `","payload":{"content":"step 3"}},
		{"seq":5,"type":"run_completed","timestamp":"` + now + `","payload":{}}
	]}`
	req = httptest.NewRequest("POST", "/api/v1/runs/"+runID+"/events", bytes.NewBufferString(eventsBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Post events failed: %d", w.Code)
	}

	// 3. 测试 from_seq 参数
	req = httptest.NewRequest("GET", "/api/v1/runs/"+runID+"/events?from_seq=2", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Get events with from_seq failed: %d", w.Code)
	}

	var eventsResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&eventsResp)

	count := int(eventsResp["count"].(float64))
	if count != 3 { // seq 3, 4, 5
		t.Errorf("Expected 3 events (from_seq=2), got %d", count)
	}

	// 4. 测试 limit 参数
	req = httptest.NewRequest("GET", "/api/v1/runs/"+runID+"/events?limit=2", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Get events with limit failed: %d", w.Code)
	}

	json.NewDecoder(w.Body).Decode(&eventsResp)
	count = int(eventsResp["count"].(float64))
	if count != 2 {
		t.Errorf("Expected 2 events (limit=2), got %d", count)
	}
}

// TestHeartbeatValidation_Integration 测试心跳请求验证
func TestHeartbeatValidation_Integration(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	router := testHandler.Router()

	// 测试缺少 node_id 的请求
	heartbeatBody := `{
		"status": "online",
		"labels": {"os": "linux"}
	}`
	req := httptest.NewRequest("POST", "/api/v1/nodes/heartbeat", bytes.NewBufferString(heartbeatBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing node_id, got %d", w.Code)
	}

	// 测试无效 JSON
	req = httptest.NewRequest("POST", "/api/v1/nodes/heartbeat", bytes.NewBufferString(`{invalid}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
	}
}

// TestTaskContext_Integration 测试任务上下文功能
func TestTaskContext_Integration(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	router := testHandler.Router()
	ctx := context.Background()

	// 1. 创建父任务（带初始上下文）
	createBody := `{
		"name": "Parent Task",
		"spec": {"prompt": "parent task prompt"},
		"context": {
			"produced_context": [
				{"type": "file", "name": "config.yaml", "content": "key: value", "source": "parent"}
			]
		}
	}`
	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Create parent task failed: %d - %s", w.Code, w.Body.String())
	}

	var parentResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&parentResp)
	parentID := parentResp["id"].(string)
	defer testStore.DeleteTask(ctx, parentID)

	t.Logf("Created parent task: %s", parentID)

	// 2. 创建子任务（继承父任务上下文）
	createChildBody := `{
		"name": "Child Task",
		"spec": {"prompt": "child task prompt"},
		"parent_id": "` + parentID + `"
	}`
	req = httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBufferString(createChildBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Create child task failed: %d - %s", w.Code, w.Body.String())
	}

	var childResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&childResp)
	childID := childResp["id"].(string)

	t.Logf("Created child task: %s", childID)

	// 验证子任务的 parent_id
	if childResp["parent_id"] != parentID {
		t.Errorf("Child parent_id = %v, want %v", childResp["parent_id"], parentID)
	}

	// 3. 获取子任务列表
	req = httptest.NewRequest("GET", "/api/v1/tasks/"+parentID+"/subtasks", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("List subtasks failed: %d - %s", w.Code, w.Body.String())
	}

	var subtasksResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&subtasksResp)
	count := int(subtasksResp["count"].(float64))
	if count != 1 {
		t.Errorf("Expected 1 subtask, got %d", count)
	}

	// 4. 获取任务树
	req = httptest.NewRequest("GET", "/api/v1/tasks/"+parentID+"/tree", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Get task tree failed: %d - %s", w.Code, w.Body.String())
	}

	var treeResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&treeResp)
	treeCount := int(treeResp["count"].(float64))
	if treeCount != 2 { // parent + child
		t.Errorf("Expected 2 tasks in tree, got %d", treeCount)
	}

	// 5. 更新任务上下文
	updateContextBody := `{
		"produced_context": [
			{"type": "summary", "name": "result", "content": "task completed successfully", "source": "child"}
		],
		"conversation_history": [
			{"role": "user", "content": "do something", "timestamp": "2026-01-30T10:00:00Z"},
			{"role": "assistant", "content": "done", "timestamp": "2026-01-30T10:00:01Z"}
		]
	}`
	req = httptest.NewRequest("PUT", "/api/v1/tasks/"+childID+"/context", bytes.NewBufferString(updateContextBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Update task context failed: %d - %s", w.Code, w.Body.String())
	}

	// 6. 验证上下文已更新
	req = httptest.NewRequest("GET", "/api/v1/tasks/"+childID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Get child task failed: %d", w.Code)
	}

	var getChildResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&getChildResp)

	// 验证 context 字段存在
	if getChildResp["context"] == nil {
		t.Error("Expected context field in response")
	}

	t.Log("TaskContext integration test passed")
}

// TestTaskWithInstanceID_Integration 测试任务与实例关联
func TestTaskWithInstanceID_Integration(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	router := testHandler.Router()
	ctx := context.Background()

	// 先创建对应实例记录（兼容 tasks.instance_id 可能存在的外键约束）
	instanceID := fmt.Sprintf("inst-test-%d", time.Now().UnixNano())
	_ = testStore.DeleteInstance(ctx, instanceID) // 幂等清理
	now := time.Now()
	if err := testStore.CreateInstance(ctx, &model.Instance{
		ID:          instanceID,
		Name:        "Test Instance",
		AccountID:   "acc-test-001",
		AgentTypeID: "qwen-code",
		Status:      model.InstanceStatusRunning,
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("Create instance failed: %v", err)
	}
	defer testStore.DeleteInstance(ctx, instanceID)

	// 创建带 instance_id 的任务
	createBody := fmt.Sprintf(`{
		"name": "Task with Instance",
		"spec": {"prompt": "test"},
		"instance_id": %q
	}`, instanceID)
	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Create task failed: %d - %s", w.Code, w.Body.String())
	}

	var taskResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&taskResp)
	taskID := taskResp["id"].(string)
	defer testStore.DeleteTask(ctx, taskID)

	// 验证 instance_id
	if taskResp["instance_id"] != instanceID {
		t.Errorf("instance_id = %v, want %q", taskResp["instance_id"], instanceID)
	}
}

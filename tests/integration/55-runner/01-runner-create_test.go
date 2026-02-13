// Package runner Runner (Instance) 管理集成测试
//
// 测试用例来源：docs/business-flow/55-Agent-Runner/01-Runner创建.md
//
// 测试架构：
//
//	测试代码 ──HTTP请求──→ httptest.Server ──→ Handler ──→ PostgreSQL
//
// 注意：本测试仅覆盖 API Server 层（Phase 1），不涉及节点管理器的容器创建（Phase 2）。
// Phase 2 的端到端测试需要 Docker 环境，属于 e2e 测试范畴。
package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"agents-admin/internal/apiserver/server"
	"agents-admin/internal/shared/model"
	"agents-admin/internal/shared/storage"
)

var testStore *storage.PostgresStore
var testHandler *server.Handler
var testServer *httptest.Server

func TestMain(m *testing.M) {
	envPaths := []string{".env", "../../../.env", "../../../../.env"}
	for _, p := range envPaths {
		if err := godotenv.Load(p); err == nil {
			break
		}
	}

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		dbURL = "postgres://agents:agents_dev_password@localhost:5432/agents_admin?sslmode=disable"
	}

	var err error
	testStore, err = storage.NewPostgresStore(dbURL)
	if err != nil {
		os.Exit(0)
	}

	testHandler = server.NewHandler(testStore, storage.NewNoOpCacheStore())
	testServer = httptest.NewServer(testHandler.Router())

	code := m.Run()

	testServer.Close()
	testStore.Close()

	os.Exit(code)
}

// createTestAccount 创建测试账号，返回 account ID
func createTestAccount(t *testing.T, suffix string, status model.AccountStatus, hasVolume bool) string {
	t.Helper()
	ctx := context.Background()

	accountID := "test-account-" + suffix + "-" + time.Now().Format("150405.000")
	var volumeName *string
	if hasVolume {
		v := "vol_" + accountID
		volumeName = &v
	}

	account := &model.Account{
		ID:          accountID,
		AgentTypeID: "qwen-code",
		Status:      status,
		VolumeName:  volumeName,
		NodeID:      "node-test-001",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := testStore.CreateAccount(ctx, account); err != nil {
		t.Fatalf("创建测试账号失败: %v", err)
	}
	return accountID
}

// cleanupAccount 清理测试账号
func cleanupAccount(accountID string) {
	ctx := context.Background()
	testStore.DeleteAccount(ctx, accountID)
}

// cleanupInstance 清理测试实例
func cleanupInstance(instanceID string) {
	ctx := context.Background()
	testStore.DeleteAgentInstance(ctx, instanceID)
}

// ============================================================================
// TC-RUNNER-CREATE-001: 基本创建流程
// ============================================================================

func TestRunnerCreate_Basic(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()

	// 前置：创建已认证且有 volume 的账号
	accountID := createTestAccount(t, "basic", model.AccountStatusAuthenticated, true)
	defer cleanupAccount(accountID)

	// 步骤：POST /api/v1/agents
	createBody := `{"account_id": "` + accountID + `", "name": "test-runner"}`
	resp, err := http.Post(
		testServer.URL+"/api/v1/agents",
		"application/json",
		bytes.NewBufferString(createBody),
	)
	if err != nil {
		t.Fatalf("TC-RUNNER-CREATE-001: HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证 HTTP 状态码 = 201
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("TC-RUNNER-CREATE-001: HTTP 状态码 = %d, 期望 201, 响应: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("TC-RUNNER-CREATE-001: 响应解析失败: %v", err)
	}

	// 验证响应 id 非空，格式 inst-xxx
	instanceID, ok := result["id"].(string)
	if !ok || instanceID == "" {
		t.Fatalf("TC-RUNNER-CREATE-001: 响应 id 为空")
	}
	defer cleanupInstance(instanceID)

	if !strings.HasPrefix(instanceID, "inst-") {
		t.Errorf("TC-RUNNER-CREATE-001: 响应 id 格式错误: %s, 期望 inst-xxx", instanceID)
	}

	// 验证响应 status = pending
	if result["status"] != "pending" {
		t.Errorf("TC-RUNNER-CREATE-001: 响应 status = %v, 期望 pending", result["status"])
	}

	// 验证响应 account_id
	if result["account_id"] != accountID {
		t.Errorf("TC-RUNNER-CREATE-001: 响应 account_id = %v, 期望 %s", result["account_id"], accountID)
	}

	// 验证响应 agent_type_id
	if result["agent_type_id"] != "qwen-code" {
		t.Errorf("TC-RUNNER-CREATE-001: 响应 agent_type_id = %v, 期望 qwen-code", result["agent_type_id"])
	}

	// 验证响应 name
	if result["name"] != "test-runner" {
		t.Errorf("TC-RUNNER-CREATE-001: 响应 name = %v, 期望 test-runner", result["name"])
	}

	// 验证 DB: instances 表存在记录
	instance, err := testStore.GetAgentInstance(ctx, instanceID)
	if err != nil {
		t.Fatalf("TC-RUNNER-CREATE-001: 查询数据库失败: %v", err)
	}
	if instance == nil {
		t.Fatalf("TC-RUNNER-CREATE-001: DB 中不存在 id=%s 的记录", instanceID)
	}

	// 验证 DB: status = pending
	if instance.Status != model.InstanceStatusPending {
		t.Errorf("TC-RUNNER-CREATE-001: DB status = %s, 期望 pending", instance.Status)
	}

	// 验证 DB: container_name = NULL
	if instance.ContainerName != nil {
		t.Errorf("TC-RUNNER-CREATE-001: DB container_name = %v, 期望 NULL", instance.ContainerName)
	}

	// 验证 DB: account_id
	if instance.AccountID != accountID {
		t.Errorf("TC-RUNNER-CREATE-001: DB account_id = %s, 期望 %s", instance.AccountID, accountID)
	}

	t.Logf("TC-RUNNER-CREATE-001: 测试通过, Instance ID: %s", instanceID)
}

// ============================================================================
// TC-RUNNER-CREATE-002: 账号不存在
// ============================================================================

func TestRunnerCreate_AccountNotFound(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	createBody := `{"account_id": "not-exist-account-xxx"}`
	resp, err := http.Post(
		testServer.URL+"/api/v1/agents",
		"application/json",
		bytes.NewBufferString(createBody),
	)
	if err != nil {
		t.Fatalf("TC-RUNNER-CREATE-002: HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证 HTTP 状态码 = 400
	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("TC-RUNNER-CREATE-002: HTTP 状态码 = %d, 期望 400, 响应: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	errMsg, _ := result["error"].(string)
	if !strings.Contains(errMsg, "not found") {
		t.Errorf("TC-RUNNER-CREATE-002: 错误信息 = %s, 期望包含 'not found'", errMsg)
	}
}

// ============================================================================
// TC-RUNNER-CREATE-003: 账号未认证
// ============================================================================

func TestRunnerCreate_AccountNotAuthenticated(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	// 前置：创建 pending 状态的账号
	accountID := createTestAccount(t, "notauth", model.AccountStatusPending, true)
	defer cleanupAccount(accountID)

	createBody := `{"account_id": "` + accountID + `"}`
	resp, err := http.Post(
		testServer.URL+"/api/v1/agents",
		"application/json",
		bytes.NewBufferString(createBody),
	)
	if err != nil {
		t.Fatalf("TC-RUNNER-CREATE-003: HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("TC-RUNNER-CREATE-003: HTTP 状态码 = %d, 期望 400, 响应: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	errMsg, _ := result["error"].(string)
	if !strings.Contains(errMsg, "not authenticated") {
		t.Errorf("TC-RUNNER-CREATE-003: 错误信息 = %s, 期望包含 'not authenticated'", errMsg)
	}
}

// ============================================================================
// TC-RUNNER-CREATE-004: 账号无 Volume
// ============================================================================

func TestRunnerCreate_AccountNoVolume(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	// 前置：创建 authenticated 但无 volume 的账号
	accountID := createTestAccount(t, "novol", model.AccountStatusAuthenticated, false)
	defer cleanupAccount(accountID)

	createBody := `{"account_id": "` + accountID + `"}`
	resp, err := http.Post(
		testServer.URL+"/api/v1/agents",
		"application/json",
		bytes.NewBufferString(createBody),
	)
	if err != nil {
		t.Fatalf("TC-RUNNER-CREATE-004: HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("TC-RUNNER-CREATE-004: HTTP 状态码 = %d, 期望 400, 响应: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	errMsg, _ := result["error"].(string)
	if !strings.Contains(errMsg, "no volume") {
		t.Errorf("TC-RUNNER-CREATE-004: 错误信息 = %s, 期望包含 'no volume'", errMsg)
	}
}

// ============================================================================
// TC-RUNNER-CREATE-005: 缺少 account_id
// ============================================================================

func TestRunnerCreate_MissingAccountID(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	createBody := `{}`
	resp, err := http.Post(
		testServer.URL+"/api/v1/agents",
		"application/json",
		bytes.NewBufferString(createBody),
	)
	if err != nil {
		t.Fatalf("TC-RUNNER-CREATE-005: HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("TC-RUNNER-CREATE-005: HTTP 状态码 = %d, 期望 400, 响应: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	errMsg, _ := result["error"].(string)
	if !strings.Contains(errMsg, "account_id") {
		t.Errorf("TC-RUNNER-CREATE-005: 错误信息 = %s, 期望包含 'account_id'", errMsg)
	}
}

// ============================================================================
// TC-RUNNER-CREATE-006: 默认使用账号的 node_id
// ============================================================================

func TestRunnerCreate_DefaultNodeID(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()

	accountID := createTestAccount(t, "defnode", model.AccountStatusAuthenticated, true)
	defer cleanupAccount(accountID)

	createBody := `{"account_id": "` + accountID + `"}`
	resp, err := http.Post(
		testServer.URL+"/api/v1/agents",
		"application/json",
		bytes.NewBufferString(createBody),
	)
	if err != nil {
		t.Fatalf("TC-RUNNER-CREATE-006: HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("TC-RUNNER-CREATE-006: HTTP 状态码 = %d, 期望 201, 响应: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	instanceID := result["id"].(string)
	defer cleanupInstance(instanceID)

	// 验证 DB: node_id = 账号的 node_id (node-test-001)
	instance, _ := testStore.GetAgentInstance(ctx, instanceID)
	if instance == nil {
		t.Fatalf("TC-RUNNER-CREATE-006: DB 中不存在记录")
	}
	if instance.NodeID == nil || *instance.NodeID != "node-test-001" {
		nodeID := "<nil>"
		if instance.NodeID != nil {
			nodeID = *instance.NodeID
		}
		t.Errorf("TC-RUNNER-CREATE-006: DB node_id = %s, 期望 node-test-001", nodeID)
	}
}

// ============================================================================
// TC-RUNNER-CREATE-007: 获取 Instance 详情
// ============================================================================

func TestRunnerGet(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	accountID := createTestAccount(t, "get", model.AccountStatusAuthenticated, true)
	defer cleanupAccount(accountID)

	// 先创建
	createBody := `{"account_id": "` + accountID + `"}`
	createResp, _ := http.Post(testServer.URL+"/api/v1/agents", "application/json", bytes.NewBufferString(createBody))
	var created map[string]interface{}
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	instanceID := created["id"].(string)
	defer cleanupInstance(instanceID)

	// GET
	resp, err := http.Get(testServer.URL + "/api/v1/agents/" + instanceID)
	if err != nil {
		t.Fatalf("TC-RUNNER-GET: HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("TC-RUNNER-GET: HTTP 状态码 = %d, 期望 200, 响应: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["id"] != instanceID {
		t.Errorf("TC-RUNNER-GET: 响应 id = %v, 期望 %s", result["id"], instanceID)
	}
}

// ============================================================================
// TC-RUNNER-CREATE-008: 获取不存在的 Instance
// ============================================================================

func TestRunnerGet_NotFound(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	resp, err := http.Get(testServer.URL + "/api/v1/agents/not-exist-inst")
	if err != nil {
		t.Fatalf("TC-RUNNER-GET-NOTFOUND: HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("TC-RUNNER-GET-NOTFOUND: HTTP 状态码 = %d, 期望 404", resp.StatusCode)
	}
}

// ============================================================================
// TC-RUNNER-DELETE-001: 删除 Instance
// ============================================================================

func TestRunnerDelete(t *testing.T) {
	if testStore == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()

	accountID := createTestAccount(t, "del", model.AccountStatusAuthenticated, true)
	defer cleanupAccount(accountID)

	// 先创建
	createBody := `{"account_id": "` + accountID + `"}`
	createResp, _ := http.Post(testServer.URL+"/api/v1/agents", "application/json", bytes.NewBufferString(createBody))
	var created map[string]interface{}
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	instanceID := created["id"].(string)

	// DELETE
	req, _ := http.NewRequest("DELETE", testServer.URL+"/api/v1/agents/"+instanceID, nil)
	delResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("TC-RUNNER-DELETE-001: DELETE 请求失败: %v", err)
	}
	defer delResp.Body.Close()

	if delResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(delResp.Body)
		t.Errorf("TC-RUNNER-DELETE-001: DELETE 状态码 = %d, 期望 200, 响应: %s", delResp.StatusCode, string(body))
	}

	// 验证 DB 已删除
	instance, _ := testStore.GetAgentInstance(ctx, instanceID)
	if instance != nil {
		t.Errorf("TC-RUNNER-DELETE-001: DB 中仍存在已删除的记录")
		cleanupInstance(instanceID)
	}

	// 再次 GET 应返回 404
	getResp, err := http.Get(testServer.URL + "/api/v1/agents/" + instanceID)
	if err != nil {
		t.Fatalf("TC-RUNNER-DELETE-001: GET 请求失败: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusNotFound {
		t.Errorf("TC-RUNNER-DELETE-001: 删除后 GET 状态码 = %d, 期望 404", getResp.StatusCode)
	}
}

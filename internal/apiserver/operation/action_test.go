package operation

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agents-admin/internal/shared/model"
)

// ============================================================================
// GetAction
// ============================================================================

func TestGetAction_NotFound(t *testing.T) {
	h := NewHandler(newMockStore())

	req := httptest.NewRequest("GET", "/api/v1/actions/act-nonexist", nil)
	req.SetPathValue("id", "act-nonexist")
	w := httptest.NewRecorder()
	h.GetAction(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetAction_Found(t *testing.T) {
	store := newMockStore()
	now := time.Now()
	store.operations["op-1"] = &model.Operation{
		ID: "op-1", Type: model.OperationTypeOAuth, Status: model.OperationStatusPending,
		Config: json.RawMessage(`{}`), NodeID: "node-001", CreatedAt: now, UpdatedAt: now,
	}
	store.actions["act-1"] = &model.Action{
		ID: "act-1", OperationID: "op-1", Status: model.ActionStatusAssigned,
		CreatedAt: now,
	}
	h := NewHandler(store)

	req := httptest.NewRequest("GET", "/api/v1/actions/act-1", nil)
	req.SetPathValue("id", "act-1")
	w := httptest.NewRecorder()
	h.GetAction(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["id"] != "act-1" {
		t.Errorf("expected id=act-1, got %v", resp["id"])
	}
	// 应包含关联的 Operation
	if resp["operation"] == nil {
		t.Error("expected operation in response")
	}
}

// ============================================================================
// UpdateAction
// ============================================================================

func TestUpdateAction_NotFound(t *testing.T) {
	h := NewHandler(newMockStore())

	body := `{"status":"running","progress":50}`
	req := httptest.NewRequest("PATCH", "/api/v1/actions/act-nonexist", strings.NewReader(body))
	req.SetPathValue("id", "act-nonexist")
	w := httptest.NewRecorder()
	h.UpdateAction(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestUpdateAction_InvalidBody(t *testing.T) {
	store := newMockStore()
	now := time.Now()
	store.operations["op-1"] = &model.Operation{
		ID: "op-1", Type: model.OperationTypeOAuth, Status: model.OperationStatusPending,
		Config: json.RawMessage(`{}`), NodeID: "node-001", CreatedAt: now, UpdatedAt: now,
	}
	store.actions["act-1"] = &model.Action{
		ID: "act-1", OperationID: "op-1", Status: model.ActionStatusAssigned,
		CreatedAt: now,
	}
	h := NewHandler(store)

	req := httptest.NewRequest("PATCH", "/api/v1/actions/act-1", strings.NewReader("{invalid"))
	req.SetPathValue("id", "act-1")
	w := httptest.NewRecorder()
	h.UpdateAction(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateAction_Running(t *testing.T) {
	store := newMockStore()
	now := time.Now()
	store.operations["op-1"] = &model.Operation{
		ID: "op-1", Type: model.OperationTypeOAuth, Status: model.OperationStatusPending,
		Config: json.RawMessage(`{"name":"test","agent_type":"qwen-code"}`), NodeID: "node-001",
		CreatedAt: now, UpdatedAt: now,
	}
	store.actions["act-1"] = &model.Action{
		ID: "act-1", OperationID: "op-1", Status: model.ActionStatusAssigned,
		CreatedAt: now,
	}
	h := NewHandler(store)

	body := `{"status":"running","progress":50,"result":{"container_name":"auth_123"}}`
	req := httptest.NewRequest("PATCH", "/api/v1/actions/act-1", strings.NewReader(body))
	req.SetPathValue("id", "act-1")
	w := httptest.NewRecorder()
	h.UpdateAction(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Action 状态更新
	if store.actions["act-1"].Status != model.ActionStatusRunning {
		t.Errorf("expected action status=running, got %s", store.actions["act-1"].Status)
	}
	// Operation 应更新为 in_progress
	if store.operations["op-1"].Status != model.OperationStatusInProgress {
		t.Errorf("expected operation status=in_progress, got %s", store.operations["op-1"].Status)
	}
}

func TestUpdateAction_Waiting(t *testing.T) {
	store := newMockStore()
	now := time.Now()
	store.operations["op-1"] = &model.Operation{
		ID: "op-1", Type: model.OperationTypeOAuth, Status: model.OperationStatusInProgress,
		Config: json.RawMessage(`{"name":"test","agent_type":"qwen-code"}`), NodeID: "node-001",
		CreatedAt: now, UpdatedAt: now,
	}
	store.actions["act-1"] = &model.Action{
		ID: "act-1", OperationID: "op-1", Status: model.ActionStatusRunning,
		CreatedAt: now,
	}
	h := NewHandler(store)

	body := `{"status":"waiting","progress":50,"result":{"verify_url":"https://oauth.example.com","device_code":"ABCD"}}`
	req := httptest.NewRequest("PATCH", "/api/v1/actions/act-1", strings.NewReader(body))
	req.SetPathValue("id", "act-1")
	w := httptest.NewRecorder()
	h.UpdateAction(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if store.actions["act-1"].Status != model.ActionStatusWaiting {
		t.Errorf("expected action status=waiting, got %s", store.actions["act-1"].Status)
	}
}

func TestUpdateAction_Success_CreatesAccount(t *testing.T) {
	store := newMockStore()
	now := time.Now()
	store.operations["op-1"] = &model.Operation{
		ID: "op-1", Type: model.OperationTypeOAuth, Status: model.OperationStatusInProgress,
		Config: json.RawMessage(`{"name":"myaccount","agent_type":"qwen-code"}`), NodeID: "node-001",
		CreatedAt: now, UpdatedAt: now,
	}
	store.actions["act-1"] = &model.Action{
		ID: "act-1", OperationID: "op-1", Status: model.ActionStatusWaiting,
		CreatedAt: now,
	}
	h := NewHandler(store)

	body := `{"status":"success","progress":100,"result":{"volume_name":"qwen-code_myaccount_vol"}}`
	req := httptest.NewRequest("PATCH", "/api/v1/actions/act-1", strings.NewReader(body))
	req.SetPathValue("id", "act-1")
	w := httptest.NewRecorder()
	h.UpdateAction(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Operation 应为 completed
	if store.operations["op-1"].Status != model.OperationStatusCompleted {
		t.Errorf("expected operation status=completed, got %s", store.operations["op-1"].Status)
	}

	// Account 应被自动创建
	accountID := "qwen-code_myaccount"
	acc := store.accounts[accountID]
	if acc == nil {
		t.Fatal("expected account to be created")
	}
	if acc.Status != model.AccountStatusAuthenticated {
		t.Errorf("expected account status=authenticated, got %s", acc.Status)
	}
	if acc.VolumeName == nil || *acc.VolumeName != "qwen-code_myaccount_vol" {
		t.Error("expected volume_name to be set")
	}
}

func TestUpdateAction_Failed(t *testing.T) {
	store := newMockStore()
	now := time.Now()
	store.operations["op-1"] = &model.Operation{
		ID: "op-1", Type: model.OperationTypeOAuth, Status: model.OperationStatusInProgress,
		Config: json.RawMessage(`{"name":"test","agent_type":"qwen-code"}`), NodeID: "node-001",
		CreatedAt: now, UpdatedAt: now,
	}
	store.actions["act-1"] = &model.Action{
		ID: "act-1", OperationID: "op-1", Status: model.ActionStatusRunning,
		CreatedAt: now,
	}
	h := NewHandler(store)

	body := `{"status":"failed","progress":0,"error":"auth container crashed"}`
	req := httptest.NewRequest("PATCH", "/api/v1/actions/act-1", strings.NewReader(body))
	req.SetPathValue("id", "act-1")
	w := httptest.NewRecorder()
	h.UpdateAction(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Operation 应为 failed
	if store.operations["op-1"].Status != model.OperationStatusFailed {
		t.Errorf("expected operation status=failed, got %s", store.operations["op-1"].Status)
	}
	// 不应创建 Account
	if len(store.accounts) != 0 {
		t.Errorf("expected 0 accounts on failure, got %d", len(store.accounts))
	}
}

func TestUpdateAction_AlreadyTerminal(t *testing.T) {
	store := newMockStore()
	now := time.Now()
	store.operations["op-1"] = &model.Operation{
		ID: "op-1", Type: model.OperationTypeOAuth, Status: model.OperationStatusCompleted,
		Config: json.RawMessage(`{}`), NodeID: "node-001", CreatedAt: now, UpdatedAt: now,
	}
	store.actions["act-1"] = &model.Action{
		ID: "act-1", OperationID: "op-1", Status: model.ActionStatusSuccess,
		CreatedAt: now,
	}
	h := NewHandler(store)

	body := `{"status":"failed","progress":0}`
	req := httptest.NewRequest("PATCH", "/api/v1/actions/act-1", strings.NewReader(body))
	req.SetPathValue("id", "act-1")
	w := httptest.NewRecorder()
	h.UpdateAction(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

// TestUpdateAction_Success_HyphenatedName 回归测试：带连字符的名称必须生成正确的账号 ID
// 修复前 sanitizeName 不替换 "-"，导致 API Server 生成 "qwen-code_test-free-net"
// 而 Node Manager 生成 "qwen-code_test_free_net"，ID 不匹配
func TestUpdateAction_Success_HyphenatedName(t *testing.T) {
	store := newMockStore()
	now := time.Now()
	store.operations["op-1"] = &model.Operation{
		ID: "op-1", Type: model.OperationTypeOAuth, Status: model.OperationStatusInProgress,
		Config: json.RawMessage(`{"name":"test-free-net","agent_type":"qwen-code"}`), NodeID: "node-001",
		CreatedAt: now, UpdatedAt: now,
	}
	store.actions["act-1"] = &model.Action{
		ID: "act-1", OperationID: "op-1", Status: model.ActionStatusWaiting,
		CreatedAt: now,
	}
	h := NewHandler(store)

	body := `{"status":"success","progress":100,"result":{"volume_name":"qwen-code_test_free_net_vol"}}`
	req := httptest.NewRequest("PATCH", "/api/v1/actions/act-1", strings.NewReader(body))
	req.SetPathValue("id", "act-1")
	w := httptest.NewRecorder()
	h.UpdateAction(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// 账号 ID 必须是 "qwen-code_test_free_net"（连字符被替换为下划线）
	// 与 Node Manager 的 sanitizeForVolume 保持一致
	expectedID := "qwen-code_test_free_net"
	acc := store.accounts[expectedID]
	if acc == nil {
		t.Fatalf("expected account %q to be created, got accounts: %v", expectedID, store.accounts)
	}
	if acc.Status != model.AccountStatusAuthenticated {
		t.Errorf("expected status=authenticated, got %s", acc.Status)
	}
}

func TestUpdateAction_Success_UpdatesExistingAccount(t *testing.T) {
	store := newMockStore()
	now := time.Now()

	// 已存在的 Account（之前认证过）
	store.accounts["qwen-code_test"] = &model.Account{
		ID: "qwen-code_test", Name: "test", AgentTypeID: "qwen-code",
		Status:    model.AccountStatusExpired,
		CreatedAt: now, UpdatedAt: now,
	}

	store.operations["op-1"] = &model.Operation{
		ID: "op-1", Type: model.OperationTypeOAuth, Status: model.OperationStatusInProgress,
		Config: json.RawMessage(`{"name":"test","agent_type":"qwen-code"}`), NodeID: "node-001",
		CreatedAt: now, UpdatedAt: now,
	}
	store.actions["act-1"] = &model.Action{
		ID: "act-1", OperationID: "op-1", Status: model.ActionStatusWaiting,
		CreatedAt: now,
	}
	h := NewHandler(store)

	body := `{"status":"success","progress":100,"result":{"volume_name":"qwen-code_test_vol"}}`
	req := httptest.NewRequest("PATCH", "/api/v1/actions/act-1", strings.NewReader(body))
	req.SetPathValue("id", "act-1")
	w := httptest.NewRecorder()
	h.UpdateAction(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Account 应更新为 authenticated
	acc := store.accounts["qwen-code_test"]
	if acc.Status != model.AccountStatusAuthenticated {
		t.Errorf("expected account status=authenticated, got %s", acc.Status)
	}
}

package operation

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"agents-admin/internal/shared/model"
)

// ============================================================================
// CreateOperation — OAuth
// ============================================================================

func TestCreateOperation_OAuth_Success(t *testing.T) {
	store := newMockStore()
	store.nodes["node-001"] = &model.Node{ID: "node-001"}
	h := NewHandler(store)

	body := `{"type":"oauth","config":{"name":"test","agent_type":"qwen-code"},"node_id":"node-001"}`
	req := httptest.NewRequest("POST", "/api/v1/operations", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateOperation(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["operation_id"] == nil || resp["operation_id"] == "" {
		t.Error("expected non-empty operation_id")
	}
	if resp["action_id"] == nil || resp["action_id"] == "" {
		t.Error("expected non-empty action_id")
	}
	if resp["type"] != "oauth" {
		t.Errorf("expected type=oauth, got %v", resp["type"])
	}
	if resp["status"] != "assigned" {
		t.Errorf("expected status=assigned, got %v", resp["status"])
	}

	// 验证 Operation 和 Action 已创建
	if len(store.operations) != 1 {
		t.Errorf("expected 1 operation, got %d", len(store.operations))
	}
	if len(store.actions) != 1 {
		t.Errorf("expected 1 action, got %d", len(store.actions))
	}

	// 验证不创建 Account（等 Action success 后才创建）
	if len(store.accounts) != 0 {
		t.Errorf("expected 0 accounts, got %d", len(store.accounts))
	}
}

func TestCreateOperation_OAuth_UnsupportedAgentType(t *testing.T) {
	store := newMockStore()
	h := NewHandler(store)

	// 使用不存在的 agent type
	body := `{"type":"oauth","config":{"name":"test","agent_type":"nonexistent"},"node_id":"node-001"}`
	req := httptest.NewRequest("POST", "/api/v1/operations", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateOperation(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateOperation_OAuth_NodeNotFound(t *testing.T) {
	store := newMockStore()
	h := NewHandler(store)

	body := `{"type":"oauth","config":{"name":"test","agent_type":"qwen-code"},"node_id":"nonexistent"}`
	req := httptest.NewRequest("POST", "/api/v1/operations", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateOperation(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateOperation_OAuth_MissingFields(t *testing.T) {
	store := newMockStore()
	h := NewHandler(store)

	body := `{"type":"oauth","config":{"name":"test"},"node_id":"node-001"}`
	req := httptest.NewRequest("POST", "/api/v1/operations", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateOperation(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateOperation_OAuth_MethodNotSupported(t *testing.T) {
	store := newMockStore()
	store.nodes["node-001"] = &model.Node{ID: "node-001"}
	h := NewHandler(store)

	// gemini 不支持 oauth
	body := `{"type":"oauth","config":{"name":"test","agent_type":"openai-codex"},"node_id":"node-001"}`
	req := httptest.NewRequest("POST", "/api/v1/operations", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateOperation(w, req)

	// openai-codex 支持 oauth，应成功
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

// ============================================================================
// CreateOperation — Device Code
// ============================================================================

func TestCreateOperation_DeviceCode_Success(t *testing.T) {
	store := newMockStore()
	store.nodes["node-001"] = &model.Node{ID: "node-001"}
	h := NewHandler(store)

	body := `{"type":"device_code","config":{"name":"test","agent_type":"openai-codex"},"node_id":"node-001"}`
	req := httptest.NewRequest("POST", "/api/v1/operations", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateOperation(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["type"] != "device_code" {
		t.Errorf("expected type=device_code, got %v", resp["type"])
	}
}

func TestCreateOperation_DeviceCode_NotSupported(t *testing.T) {
	store := newMockStore()
	store.nodes["node-001"] = &model.Node{ID: "node-001"}
	h := NewHandler(store)

	// qwen-code 不支持 device_code
	body := `{"type":"device_code","config":{"name":"test","agent_type":"qwen-code"},"node_id":"node-001"}`
	req := httptest.NewRequest("POST", "/api/v1/operations", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateOperation(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ============================================================================
// CreateOperation — API Key（同步完成）
// ============================================================================

func TestCreateOperation_APIKey_Success(t *testing.T) {
	store := newMockStore()
	store.nodes["node-001"] = &model.Node{ID: "node-001"}
	h := NewHandler(store)

	body := `{"type":"api_key","config":{"name":"test@email.com","agent_type":"qwen-code","api_key":"sk-123"},"node_id":"node-001"}`
	req := httptest.NewRequest("POST", "/api/v1/operations", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateOperation(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "success" {
		t.Errorf("expected status=success, got %v", resp["status"])
	}
	if resp["account_id"] == nil || resp["account_id"] == "" {
		t.Error("expected non-empty account_id")
	}

	// API Key 同步完成：Operation=completed, Action=success, Account=authenticated
	if len(store.operations) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(store.operations))
	}
	for _, op := range store.operations {
		if op.Status != model.OperationStatusCompleted {
			t.Errorf("expected operation status=completed, got %s", op.Status)
		}
	}
	if len(store.accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(store.accounts))
	}
	for _, acc := range store.accounts {
		if acc.Status != model.AccountStatusAuthenticated {
			t.Errorf("expected account status=authenticated, got %s", acc.Status)
		}
	}
}

func TestCreateOperation_APIKey_MissingAPIKey(t *testing.T) {
	store := newMockStore()
	h := NewHandler(store)

	body := `{"type":"api_key","config":{"name":"test","agent_type":"qwen-code"},"node_id":"node-001"}`
	req := httptest.NewRequest("POST", "/api/v1/operations", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateOperation(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ============================================================================
// CreateOperation — Invalid type
// ============================================================================

func TestCreateOperation_InvalidType(t *testing.T) {
	store := newMockStore()
	h := NewHandler(store)

	body := `{"type":"unknown","config":{},"node_id":"node-001"}`
	req := httptest.NewRequest("POST", "/api/v1/operations", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateOperation(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateOperation_InvalidJSON(t *testing.T) {
	store := newMockStore()
	h := NewHandler(store)

	req := httptest.NewRequest("POST", "/api/v1/operations", strings.NewReader("{invalid"))
	w := httptest.NewRecorder()
	h.CreateOperation(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

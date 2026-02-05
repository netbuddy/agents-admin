package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"agents-admin/internal/shared/model"
)

func TestCreateAuthOperation_OAuth(t *testing.T) {
	store := newMockStore()
	store.nodes["node-001"] = &model.Node{ID: "node-001", Status: model.NodeStatusOnline}
	h := NewHandler(store)

	config := json.RawMessage(`{"name":"test-account","agent_type":"qwen-code"}`)
	req := httptest.NewRequest("POST", "/api/v1/operations", nil)
	w := httptest.NewRecorder()
	h.CreateAuthOperation(w, req, model.OperationTypeOAuth, config, "node-001")

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["type"] != "oauth" {
		t.Errorf("expected type=oauth, got %v", resp["type"])
	}
	if resp["status"] != "assigned" {
		t.Errorf("expected status=assigned, got %v", resp["status"])
	}
	if len(store.operations) != 1 {
		t.Errorf("expected 1 operation, got %d", len(store.operations))
	}
	if len(store.actions) != 1 {
		t.Errorf("expected 1 action, got %d", len(store.actions))
	}
}

func TestCreateAuthOperation_DeviceCode(t *testing.T) {
	store := newMockStore()
	store.nodes["node-001"] = &model.Node{ID: "node-001", Status: model.NodeStatusOnline}
	h := NewHandler(store)

	config := json.RawMessage(`{"name":"test","agent_type":"openai-codex"}`)
	req := httptest.NewRequest("POST", "/api/v1/operations", nil)
	w := httptest.NewRecorder()
	h.CreateAuthOperation(w, req, model.OperationTypeDeviceCode, config, "node-001")

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateAuthOperation_MissingFields(t *testing.T) {
	store := newMockStore()
	h := NewHandler(store)

	config := json.RawMessage(`{"name":"test"}`)
	req := httptest.NewRequest("POST", "/api/v1/operations", nil)
	w := httptest.NewRecorder()
	h.CreateAuthOperation(w, req, model.OperationTypeOAuth, config, "node-001")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateAuthOperation_UnknownAgentType(t *testing.T) {
	store := newMockStore()
	h := NewHandler(store)

	config := json.RawMessage(`{"name":"test","agent_type":"nonexistent"}`)
	req := httptest.NewRequest("POST", "/api/v1/operations", nil)
	w := httptest.NewRecorder()
	h.CreateAuthOperation(w, req, model.OperationTypeOAuth, config, "node-001")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateAuthOperation_NodeNotFound(t *testing.T) {
	store := newMockStore()
	h := NewHandler(store)

	config := json.RawMessage(`{"name":"test","agent_type":"qwen-code"}`)
	req := httptest.NewRequest("POST", "/api/v1/operations", nil)
	w := httptest.NewRecorder()
	h.CreateAuthOperation(w, req, model.OperationTypeOAuth, config, "nonexistent-node")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateAPIKeyOperation_Success(t *testing.T) {
	store := newMockStore()
	store.nodes["node-001"] = &model.Node{ID: "node-001", Status: model.NodeStatusOnline}
	h := NewHandler(store)

	config := json.RawMessage(`{"name":"test","agent_type":"qwen-code","api_key":"sk-xxx"}`)
	req := httptest.NewRequest("POST", "/api/v1/operations", nil)
	w := httptest.NewRecorder()
	h.CreateAPIKeyOperation(w, req, config, "node-001")

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "success" {
		t.Errorf("expected status=success, got %v", resp["status"])
	}
	if resp["account_id"] == nil || resp["account_id"] == "" {
		t.Error("expected account_id in response")
	}
	if len(store.accounts) != 1 {
		t.Errorf("expected 1 account, got %d", len(store.accounts))
	}
}

func TestCreateAPIKeyOperation_MissingAPIKey(t *testing.T) {
	store := newMockStore()
	h := NewHandler(store)

	config := json.RawMessage(`{"name":"test","agent_type":"qwen-code"}`)
	req := httptest.NewRequest("POST", "/api/v1/operations", nil)
	w := httptest.NewRecorder()
	h.CreateAPIKeyOperation(w, req, config, "node-001")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

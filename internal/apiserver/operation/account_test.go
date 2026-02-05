package operation

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agents-admin/internal/shared/model"
)

// ============================================================================
// Query Tests (ListOperations / GetOperation)
// ============================================================================

func TestListOperations_Empty(t *testing.T) {
	h := NewHandler(newMockStore())

	req := httptest.NewRequest("GET", "/api/v1/operations", nil)
	w := httptest.NewRecorder()
	h.ListOperations(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestListOperations_WithFilter(t *testing.T) {
	store := newMockStore()
	now := time.Now()
	store.operations["op-1"] = &model.Operation{
		ID: "op-1", Type: model.OperationTypeOAuth, Status: model.OperationStatusPending,
		Config: json.RawMessage(`{}`), NodeID: "node-001", CreatedAt: now, UpdatedAt: now,
	}
	store.operations["op-2"] = &model.Operation{
		ID: "op-2", Type: model.OperationTypeAPIKey, Status: model.OperationStatusCompleted,
		Config: json.RawMessage(`{}`), NodeID: "node-001", CreatedAt: now, UpdatedAt: now,
	}
	h := NewHandler(store)

	// 按 type 过滤
	req := httptest.NewRequest("GET", "/api/v1/operations?type=oauth", nil)
	w := httptest.NewRecorder()
	h.ListOperations(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	ops := resp["operations"].([]interface{})
	if len(ops) != 1 {
		t.Errorf("expected 1 oauth operation, got %d", len(ops))
	}
}

func TestGetOperation_NotFound(t *testing.T) {
	h := NewHandler(newMockStore())

	req := httptest.NewRequest("GET", "/api/v1/operations/nonexist", nil)
	req.SetPathValue("id", "nonexist")
	w := httptest.NewRecorder()
	h.GetOperation(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetOperation_WithActions(t *testing.T) {
	store := newMockStore()
	now := time.Now()
	store.operations["op-1"] = &model.Operation{
		ID: "op-1", Type: model.OperationTypeOAuth, Status: model.OperationStatusPending,
		Config: json.RawMessage(`{}`), NodeID: "node-001", CreatedAt: now, UpdatedAt: now,
	}
	store.actions["act-1"] = &model.Action{
		ID: "act-1", OperationID: "op-1", Status: model.ActionStatusAssigned, CreatedAt: now,
	}
	h := NewHandler(store)

	req := httptest.NewRequest("GET", "/api/v1/operations/op-1", nil)
	req.SetPathValue("id", "op-1")
	w := httptest.NewRecorder()
	h.GetOperation(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	actions := resp["actions"].([]interface{})
	if len(actions) != 1 {
		t.Errorf("expected 1 action, got %d", len(actions))
	}
}

// ============================================================================
// GetNodeActions Tests
// ============================================================================

func TestGetNodeActions(t *testing.T) {
	store := newMockStore()
	now := time.Now()
	store.operations["op-1"] = &model.Operation{
		ID: "op-1", Type: model.OperationTypeOAuth, Status: model.OperationStatusPending,
		Config: json.RawMessage(`{"name":"test","agent_type":"qwen-code"}`),
		NodeID: "node-001", CreatedAt: now, UpdatedAt: now,
	}
	store.actions["act-1"] = &model.Action{
		ID: "act-1", OperationID: "op-1", Status: model.ActionStatusAssigned, CreatedAt: now,
	}
	// 另一个节点的 Action
	store.operations["op-2"] = &model.Operation{
		ID: "op-2", Type: model.OperationTypeOAuth, Status: model.OperationStatusPending,
		Config: json.RawMessage(`{}`), NodeID: "node-002", CreatedAt: now, UpdatedAt: now,
	}
	store.actions["act-2"] = &model.Action{
		ID: "act-2", OperationID: "op-2", Status: model.ActionStatusAssigned, CreatedAt: now,
	}
	h := NewHandler(store)

	req := httptest.NewRequest("GET", "/api/v1/nodes/node-001/actions", nil)
	req.SetPathValue("id", "node-001")
	w := httptest.NewRecorder()
	h.GetNodeActions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	actions := resp["actions"].([]interface{})
	if len(actions) != 1 {
		t.Errorf("expected 1 action for node-001, got %d", len(actions))
	}
}

func TestGetNodeActions_DefaultStatus(t *testing.T) {
	store := newMockStore()
	now := time.Now()
	store.operations["op-1"] = &model.Operation{
		ID: "op-1", Type: model.OperationTypeOAuth, Status: model.OperationStatusInProgress,
		Config: json.RawMessage(`{}`), NodeID: "node-001", CreatedAt: now, UpdatedAt: now,
	}
	store.actions["act-1"] = &model.Action{
		ID: "act-1", OperationID: "op-1", Status: model.ActionStatusRunning, CreatedAt: now,
	}
	h := NewHandler(store)

	// 默认 status=assigned，running 的不应返回
	req := httptest.NewRequest("GET", "/api/v1/nodes/node-001/actions", nil)
	req.SetPathValue("id", "node-001")
	w := httptest.NewRecorder()
	h.GetNodeActions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	actions, _ := resp["actions"].([]interface{})
	if len(actions) != 0 {
		t.Errorf("expected 0 assigned actions, got %d", len(actions))
	}
}

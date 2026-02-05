package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"agents-admin/internal/shared/model"
)

func TestListAgentTypes(t *testing.T) {
	h := NewHandler(newMockStore())

	req := httptest.NewRequest("GET", "/api/v1/agent-types", nil)
	w := httptest.NewRecorder()
	h.ListAgentTypes(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	types := resp["agent_types"].([]interface{})
	if len(types) != len(model.PredefinedAgentTypes) {
		t.Errorf("expected %d agent types, got %d", len(model.PredefinedAgentTypes), len(types))
	}
}

func TestGetAgentType_Found(t *testing.T) {
	h := NewHandler(newMockStore())

	req := httptest.NewRequest("GET", "/api/v1/agent-types/qwen-code", nil)
	req.SetPathValue("id", "qwen-code")
	w := httptest.NewRecorder()
	h.GetAgentType(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["id"] != "qwen-code" {
		t.Errorf("expected id=qwen-code, got %v", resp["id"])
	}
}

func TestGetAgentType_NotFound(t *testing.T) {
	h := NewHandler(newMockStore())

	req := httptest.NewRequest("GET", "/api/v1/agent-types/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()
	h.GetAgentType(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestFindAgentType(t *testing.T) {
	at := findAgentType("qwen-code")
	if at == nil {
		t.Fatal("expected to find qwen-code")
	}
	if at.ID != "qwen-code" {
		t.Errorf("expected id=qwen-code, got %s", at.ID)
	}

	at = findAgentType("nonexistent")
	if at != nil {
		t.Error("expected nil for nonexistent agent type")
	}
}

func TestAgentTypeSupportsMethod(t *testing.T) {
	at := findAgentType("qwen-code")
	if !agentTypeSupportsMethod(at, "oauth") {
		t.Error("qwen-code should support oauth")
	}
	if !agentTypeSupportsMethod(at, "api_key") {
		t.Error("qwen-code should support api_key")
	}
	if agentTypeSupportsMethod(at, "device_code") {
		t.Error("qwen-code should not support device_code")
	}
}

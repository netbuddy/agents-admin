package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agents-admin/internal/shared/model"
)

func TestListAccounts_Empty(t *testing.T) {
	h := NewHandler(newMockStore())

	req := httptest.NewRequest("GET", "/api/v1/accounts", nil)
	w := httptest.NewRecorder()
	h.ListAccounts(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestListAccounts_WithFilter(t *testing.T) {
	store := newMockStore()
	now := time.Now()
	store.accounts["acc-1"] = &model.Account{
		ID: "acc-1", Name: "a", AgentTypeID: "qwen-code",
		Status: model.AccountStatusAuthenticated, CreatedAt: now, UpdatedAt: now,
	}
	store.accounts["acc-2"] = &model.Account{
		ID: "acc-2", Name: "b", AgentTypeID: "openai-codex",
		Status: model.AccountStatusAuthenticated, CreatedAt: now, UpdatedAt: now,
	}
	h := NewHandler(store)

	req := httptest.NewRequest("GET", "/api/v1/accounts?agent_type=qwen-code", nil)
	w := httptest.NewRecorder()
	h.ListAccounts(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	accounts := resp["accounts"].([]interface{})
	if len(accounts) != 1 {
		t.Errorf("expected 1 filtered account, got %d", len(accounts))
	}
}

func TestGetAccount_Found(t *testing.T) {
	store := newMockStore()
	now := time.Now()
	store.accounts["acc-1"] = &model.Account{
		ID: "acc-1", Name: "test", AgentTypeID: "qwen-code",
		Status: model.AccountStatusAuthenticated, CreatedAt: now, UpdatedAt: now,
	}
	h := NewHandler(store)

	req := httptest.NewRequest("GET", "/api/v1/accounts/acc-1", nil)
	req.SetPathValue("id", "acc-1")
	w := httptest.NewRecorder()
	h.GetAccount(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetAccount_NotFound(t *testing.T) {
	h := NewHandler(newMockStore())

	req := httptest.NewRequest("GET", "/api/v1/accounts/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()
	h.GetAccount(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestDeleteAccount_Success(t *testing.T) {
	store := newMockStore()
	now := time.Now()
	store.accounts["acc-1"] = &model.Account{
		ID: "acc-1", Name: "test", AgentTypeID: "qwen-code",
		Status: model.AccountStatusAuthenticated, CreatedAt: now, UpdatedAt: now,
	}
	h := NewHandler(store)

	req := httptest.NewRequest("DELETE", "/api/v1/accounts/acc-1", nil)
	req.SetPathValue("id", "acc-1")
	w := httptest.NewRecorder()
	h.DeleteAccount(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if len(store.accounts) != 0 {
		t.Errorf("expected account deleted, got %d accounts", len(store.accounts))
	}
}

func TestDeleteAccount_NotFound(t *testing.T) {
	h := NewHandler(newMockStore())

	req := httptest.NewRequest("DELETE", "/api/v1/accounts/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()
	h.DeleteAccount(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSanitizeName(t *testing.T) {
	cases := []struct{ input, want string }{
		{"test@email.com", "test_email_com"},
		{"simple", "simple"},
		{"with space", "with_space"},
	}
	for _, tc := range cases {
		got := sanitizeName(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

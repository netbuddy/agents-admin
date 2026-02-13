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
	// 注意：必须与 nodemanager/auth_controller.go 的 sanitizeForVolume 保持一致
	cases := []struct{ input, want string }{
		{"test@email.com", "test_email_com"},
		{"simple", "simple"},
		{"with space", "with_space"},
		{"test-name", "test_name"},         // 连字符必须替换（与 sanitizeForVolume 一致）
		{"test-free-net", "test_free_net"}, // 真实场景
		{"a-b.c@d e", "a_b_c_d_e"},         // 混合特殊字符
	}
	for _, tc := range cases {
		got := sanitizeName(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

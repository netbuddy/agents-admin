package auth

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"agents-admin/internal/shared/model"
)

func TestHandleAuthSuccess_CreateNewAccount(t *testing.T) {
	store := newMockStore()
	h := NewHandler(store)

	op := &model.Operation{
		ID:     "op-1",
		Type:   model.OperationTypeOAuth,
		Config: json.RawMessage(`{"name":"test","agent_type":"qwen-code"}`),
		NodeID: "node-001",
	}
	resultJSON := json.RawMessage(`{"volume_name":"qwen-code_test_vol"}`)

	h.HandleAuthSuccess(context.Background(), op, resultJSON)

	if len(store.accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(store.accounts))
	}

	acc := store.accounts["qwen-code_test"]
	if acc == nil {
		t.Fatal("expected account qwen-code_test")
	}
	if acc.Status != model.AccountStatusAuthenticated {
		t.Errorf("expected status authenticated, got %s", acc.Status)
	}
	if acc.VolumeName == nil || *acc.VolumeName != "qwen-code_test_vol" {
		t.Error("expected volume_name qwen-code_test_vol")
	}
}

func TestHandleAuthSuccess_UpdateExistingAccount(t *testing.T) {
	store := newMockStore()
	now := time.Now()
	store.accounts["qwen-code_test"] = &model.Account{
		ID: "qwen-code_test", Name: "test", AgentTypeID: "qwen-code",
		Status: model.AccountStatusExpired, CreatedAt: now, UpdatedAt: now,
	}
	h := NewHandler(store)

	op := &model.Operation{
		ID:     "op-1",
		Type:   model.OperationTypeOAuth,
		Config: json.RawMessage(`{"name":"test","agent_type":"qwen-code"}`),
		NodeID: "node-001",
	}
	resultJSON := json.RawMessage(`{"volume_name":"qwen-code_test_vol"}`)

	h.HandleAuthSuccess(context.Background(), op, resultJSON)

	// 应该更新而非新建
	if len(store.accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(store.accounts))
	}
	acc := store.accounts["qwen-code_test"]
	if acc.Status != model.AccountStatusAuthenticated {
		t.Errorf("expected status updated to authenticated, got %s", acc.Status)
	}
	if acc.VolumeName == nil || *acc.VolumeName != "qwen-code_test_vol" {
		t.Error("expected volume updated")
	}
}

func TestHandleAuthSuccess_NoResult(t *testing.T) {
	store := newMockStore()
	h := NewHandler(store)

	op := &model.Operation{
		ID:     "op-1",
		Type:   model.OperationTypeDeviceCode,
		Config: json.RawMessage(`{"name":"test","agent_type":"openai-codex"}`),
		NodeID: "node-001",
	}

	h.HandleAuthSuccess(context.Background(), op, nil)

	if len(store.accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(store.accounts))
	}
	acc := store.accounts["openai-codex_test"]
	if acc == nil {
		t.Fatal("expected account created")
	}
	if acc.VolumeName != nil {
		t.Errorf("expected nil volume, got %v", acc.VolumeName)
	}
}

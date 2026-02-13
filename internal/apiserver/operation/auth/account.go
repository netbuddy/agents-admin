package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"agents-admin/internal/shared/model"
)

// CreateAccount 创建账号
//
// POST /api/v1/accounts
func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Name      string `json:"name"`
		AgentType string `json:"agent_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.AgentType == "" {
		writeError(w, http.StatusBadRequest, "name and agent_type are required")
		return
	}

	now := time.Now()
	accountID := fmt.Sprintf("%s_%s", req.AgentType, sanitizeName(req.Name))
	volumeName := fmt.Sprintf("%s_%s_vol", req.AgentType, sanitizeName(req.Name))

	account := &model.Account{
		ID:          accountID,
		Name:        req.Name,
		AgentTypeID: req.AgentType,
		VolumeName:  &volumeName,
		Status:      model.AccountStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.store.CreateAccount(ctx, account); err != nil {
		log.Printf("[auth] CreateAccount error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create account")
		return
	}

	log.Printf("[auth] Account created: %s", accountID)
	writeJSON(w, http.StatusCreated, account)
}

// ListAccounts 列出所有账号
//
// GET /api/v1/accounts
// 可选查询参数: agent_type
func (h *Handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accounts, err := h.store.ListAccounts(ctx)
	if err != nil {
		log.Printf("[auth] ListAccounts error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list accounts")
		return
	}

	// 按 agent_type 过滤
	agentType := r.URL.Query().Get("agent_type")
	if agentType != "" {
		filtered := accounts[:0]
		for _, a := range accounts {
			if a.AgentTypeID == agentType {
				filtered = append(filtered, a)
			}
		}
		accounts = filtered
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"accounts": accounts})
}

// GetAccount 获取账号详情
//
// GET /api/v1/accounts/{id}
func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	account, err := h.store.GetAccount(ctx, id)
	if err != nil {
		log.Printf("[auth] GetAccount error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get account")
		return
	}
	if account == nil {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}

	writeJSON(w, http.StatusOK, account)
}

// DeleteAccount 删除账号
//
// DELETE /api/v1/accounts/{id}
func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	account, err := h.store.GetAccount(ctx, id)
	if err != nil {
		log.Printf("[auth] DeleteAccount error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get account")
		return
	}
	if account == nil {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}

	if err := h.store.DeleteAccount(ctx, id); err != nil {
		log.Printf("[auth] DeleteAccount error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to delete account")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

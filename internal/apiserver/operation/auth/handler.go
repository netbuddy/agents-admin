package auth

import (
	"net/http"

	"agents-admin/internal/shared/storage"
)

// Handler 认证操作领域 HTTP 处理器
type Handler struct {
	store storage.PersistentStore
}

// NewHandler 创建认证操作处理器
func NewHandler(store storage.PersistentStore) *Handler {
	return &Handler{store: store}
}

// RegisterRoutes 注册认证相关路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Agent 类型（只读）
	mux.HandleFunc("GET /api/v1/agent-types", h.ListAgentTypes)
	mux.HandleFunc("GET /api/v1/agent-types/{id}", h.GetAgentType)

	// 账号（只读资源，由 Operation 成功后创建）
	mux.HandleFunc("GET /api/v1/accounts", h.ListAccounts)
	mux.HandleFunc("GET /api/v1/accounts/{id}", h.GetAccount)
	mux.HandleFunc("DELETE /api/v1/accounts/{id}", h.DeleteAccount)
}

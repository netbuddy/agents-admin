package auth

import (
	"net/http"

	objstore "agents-admin/internal/shared/minio"
	"agents-admin/internal/shared/storage"
)

// Handler 认证操作领域 HTTP 处理器
type Handler struct {
	store storage.PersistentStore
	minio *objstore.Client
}

// NewHandler 创建认证操作处理器
func NewHandler(store storage.PersistentStore) *Handler {
	return &Handler{store: store}
}

// SetMinIOClient 设置 MinIO 客户端（可选，用于 volume archive）
func (h *Handler) SetMinIOClient(mc *objstore.Client) {
	h.minio = mc
}

// RegisterRoutes 注册认证相关路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Agent 类型（只读）
	mux.HandleFunc("GET /api/v1/agent-types", h.ListAgentTypes)
	mux.HandleFunc("GET /api/v1/agent-types/{id}", h.GetAgentType)

	// 账号
	mux.HandleFunc("POST /api/v1/accounts", h.CreateAccount)
	mux.HandleFunc("GET /api/v1/accounts", h.ListAccounts)
	mux.HandleFunc("GET /api/v1/accounts/{id}", h.GetAccount)
	mux.HandleFunc("DELETE /api/v1/accounts/{id}", h.DeleteAccount)

	// Volume 归档（MinIO 代理）
	mux.HandleFunc("PUT /api/v1/accounts/{id}/volume-archive", h.UploadVolumeArchive)
	mux.HandleFunc("GET /api/v1/accounts/{id}/volume-archive", h.DownloadVolumeArchive)
}

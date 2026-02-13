// Package operation 系统操作领域 - HTTP 处理
//
// 包结构：
//   - operation/: 共享层 — Operation/Action 生命周期管理
//   - operation/auth/: 认证操作 — AgentType、Account、认证创建与结果处理
//   - operation/runtime/: 运行时操作 — 创建/启动/停止/销毁（占位）
//
// 文件组织（本包）：
//   - handler.go: Handler 结构、路由注册、子 Handler 组合
//   - create.go: POST /operations — 创建操作（统一入口，按类型分发）
//   - query.go: GET /operations, GET /operations/{id} — 查询操作
//   - action.go: GET /actions/{id}, PATCH /actions/{id} — Action 查询与更新
//   - node_actions.go: GET /nodes/{id}/actions — 节点轮询
//   - result.go: Action 结果处理（分发到子 Handler）
//   - helpers.go: 辅助函数
package operation

import (
	"net/http"

	"agents-admin/internal/apiserver/operation/auth"
	objstore "agents-admin/internal/shared/minio"
	"agents-admin/internal/shared/storage"
)

// Handler 系统操作领域 HTTP 处理器
type Handler struct {
	store       storage.PersistentStore
	authHandler *auth.Handler
}

// NewHandler 创建系统操作处理器
func NewHandler(store storage.PersistentStore) *Handler {
	return &Handler{
		store:       store,
		authHandler: auth.NewHandler(store),
	}
}

// SetMinIOClient 设置 MinIO 客户端（传递给 auth 子处理器）
func (h *Handler) SetMinIOClient(mc *objstore.Client) {
	h.authHandler.SetMinIOClient(mc)
}

// RegisterRoutes 注册系统操作相关路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// 认证子领域路由（Agent 类型、账号管理）
	h.authHandler.RegisterRoutes(mux)

	// Operation（统一入口）
	mux.HandleFunc("POST /api/v1/operations", h.CreateOperation)
	mux.HandleFunc("GET /api/v1/operations", h.ListOperations)
	mux.HandleFunc("GET /api/v1/operations/{id}", h.GetOperation)

	// Action（执行实例）
	mux.HandleFunc("GET /api/v1/actions/{id}", h.GetAction)
	mux.HandleFunc("PATCH /api/v1/actions/{id}", h.UpdateAction)

	// 节点统一轮询（替代旧的 /nodes/{id}/auth-tasks）
	mux.HandleFunc("GET /api/v1/nodes/{id}/actions", h.GetNodeActions)
}

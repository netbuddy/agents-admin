// Package server 路由配置与核心基础设施
//
// 本文件定义 HTTP API 路由，将请求分发到各领域独立包。
// 仍保留在本包的模块：
//   - events.go: 事件接口（依赖 EventGateway）
//   - monitor.go / monitor_ws.go: 监控接口（依赖工作流缓存/事件总线）
//   - websocket.go: WebSocket 事件网关
//   - metrics.go: Prometheus 指标
//   - runs.go: StartScheduler（调度器入口）
package server

import (
	"net/http"

	"agents-admin/internal/apiserver/auth"
	"agents-admin/internal/apiserver/hitl"
	"agents-admin/internal/apiserver/instance"
	"agents-admin/internal/apiserver/node"
	"agents-admin/internal/apiserver/operation"
	"agents-admin/internal/apiserver/proxy"
	"agents-admin/internal/apiserver/run"
	"agents-admin/internal/apiserver/task"
	"agents-admin/internal/apiserver/template"
	"agents-admin/internal/apiserver/terminal"
)

// Router 返回配置好的 HTTP 路由
//
// 路由规则：
//
// 健康检查:
//   - GET /health - 服务健康检查
//
// 任务管理 (Task):
//   - GET    /api/v1/tasks           - 列出任务
//   - POST   /api/v1/tasks           - 创建任务
//   - GET    /api/v1/tasks/{id}      - 获取任务详情
//   - DELETE /api/v1/tasks/{id}      - 删除任务
//
// 执行管理 (Run):
//   - POST   /api/v1/tasks/{id}/runs - 创建执行
//   - GET    /api/v1/tasks/{id}/runs - 列出任务的执行记录
//   - GET    /api/v1/runs/{id}       - 获取执行详情
//   - PATCH  /api/v1/runs/{id}       - 更新执行状态
//   - POST   /api/v1/runs/{id}/cancel - 取消执行
//
// 事件管理 (Event):
//   - GET    /api/v1/runs/{id}/events - 获取事件列表
//   - POST   /api/v1/runs/{id}/events - 批量上报事件
//
// 节点管理 (Node):
//   - POST   /api/v1/nodes/heartbeat  - 节点心跳
//   - GET    /api/v1/nodes            - 列出在线节点
//   - GET    /api/v1/nodes/{id}       - 获取节点详情
//   - PATCH  /api/v1/nodes/{id}       - 更新节点
//   - DELETE /api/v1/nodes/{id}       - 删除节点
//   - GET    /api/v1/nodes/{id}/runs  - 获取节点的执行任务
//
// WebSocket:
//   - GET    /ws/runs/{id}/events     - 实时事件推送
func (h *Handler) Router() http.Handler {
	mux := http.NewServeMux()

	// 健康检查
	mux.HandleFunc("GET /health", h.Health)

	// Prometheus 指标端点
	mux.Handle("GET /metrics", MetricsHandler())

	// Task 接口（已迁移到 task 包）
	taskHandler := task.NewHandler(h.store)
	taskHandler.RegisterRoutes(mux)

	// Run 接口（已迁移到 run 包）
	// 传入调度队列支持事件驱动调度
	runHandler := run.NewHandler(h.store, h.schedulerQueue)
	runHandler.RegisterRoutes(mux)

	// Event 接口
	mux.HandleFunc("GET /api/v1/runs/{id}/events", h.GetEvents)
	mux.HandleFunc("POST /api/v1/runs/{id}/events", h.PostEvents)

	// Node 接口（已迁移到 node 包）
	nodeHandler := node.NewHandler(h.store, h.nodeCache)
	nodeHandler.RegisterRoutes(mux)

	// ========== 新架构 API ==========

	// 系统操作（Operation/Action 统一模型）
	opHandler := operation.NewHandler(h.store)
	opHandler.RegisterRoutes(mux)

	// 代理管理接口（已迁移到 proxy 包）
	proxyHandler := proxy.NewHandler(h.store)
	proxyHandler.RegisterRoutes(mux)

	// 实例管理接口（已迁移到 instance 包）
	instHandler := instance.NewHandler(h.store)
	instHandler.RegisterRoutes(mux)
	instHandler.RegisterNodeManagerRoutes(mux)

	// 终端会话接口（已迁移到 terminal 包）
	termHandler := terminal.NewHandler(h.store)
	termHandler.RegisterRoutes(mux)
	termHandler.RegisterNodeManagerRoutes(mux)

	// 模板 API（已迁移到 template 包）
	tmplHandler := template.NewHandler(h.store)
	tmplHandler.RegisterRoutes(mux)

	// HITL 接口（已迁移到 hitl 包）
	hitlHandler := hitl.NewHandler(h.store)
	hitlHandler.RegisterRoutes(mux)

	// ========== 监控 API ==========
	mux.HandleFunc("GET /api/v1/monitor/workflows", h.ListWorkflows)
	mux.HandleFunc("GET /api/v1/monitor/workflows/{type}/{id}", h.GetWorkflow)
	mux.HandleFunc("GET /api/v1/monitor/workflows/{type}/{id}/events", h.GetWorkflowEvents)
	mux.HandleFunc("GET /api/v1/monitor/stats", h.GetMonitorStats)

	// Auth 路由
	authCfg := auth.Config{
		JWTSecret:       h.authConfig.JWTSecret,
		AccessTokenTTL:  h.authConfig.AccessTokenTTL,
		RefreshTokenTTL: h.authConfig.RefreshTokenTTL,
	}
	authHandler := auth.NewHandler(h.store, authCfg)
	authHandler.RegisterRoutes(mux)

	// 应用指标中间件到 REST API
	apiHandler := h.metrics.MetricsMiddleware(mux)

	// 应用认证中间件
	authedHandler := auth.Middleware(authCfg)(apiHandler)

	// 应用 CORS 中间件
	corsHandler := corsMiddleware(authedHandler)

	// 创建顶层路由，WebSocket 绑过 metrics 中间件（避免 http.Hijacker 问题）
	topMux := http.NewServeMux()
	monitorWS := NewMonitorWSHandler(h)
	topMux.HandleFunc("GET /ws/monitor", monitorWS.HandleWebSocket)
	topMux.HandleFunc("/ws/runs/{id}/events", h.eventGateway.HandleWebSocket)
	topMux.Handle("/", corsHandler)

	return topMux
}

// corsMiddleware 添加 CORS 头支持跨域请求
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Package api 路由配置
//
// 本文件定义 HTTP API 路由，将请求分发到对应的处理函数。
// 具体的处理逻辑分布在以下文件中：
//   - common.go: 通用工具函数和 Handler 定义
//   - tasks.go: 任务相关接口
//   - runs.go: 执行相关接口
//   - events.go: 事件相关接口
//   - nodes.go: 节点相关接口
package api

import (
	"net/http"
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

	// Task 接口
	mux.HandleFunc("GET /api/v1/tasks", h.ListTasks)
	mux.HandleFunc("POST /api/v1/tasks", h.CreateTask)
	mux.HandleFunc("GET /api/v1/tasks/{id}", h.GetTask)
	mux.HandleFunc("DELETE /api/v1/tasks/{id}", h.DeleteTask)
	mux.HandleFunc("GET /api/v1/tasks/{id}/subtasks", h.ListSubTasks)
	mux.HandleFunc("GET /api/v1/tasks/{id}/tree", h.GetTaskTree)
	mux.HandleFunc("PUT /api/v1/tasks/{id}/context", h.UpdateTaskContext)

	// Run 接口
	mux.HandleFunc("POST /api/v1/tasks/{id}/runs", h.CreateRun)
	mux.HandleFunc("GET /api/v1/tasks/{id}/runs", h.ListRuns)
	mux.HandleFunc("GET /api/v1/runs/{id}", h.GetRun)
	mux.HandleFunc("PATCH /api/v1/runs/{id}", h.UpdateRun)
	mux.HandleFunc("POST /api/v1/runs/{id}/cancel", h.CancelRun)

	// Event 接口
	mux.HandleFunc("GET /api/v1/runs/{id}/events", h.GetEvents)
	mux.HandleFunc("POST /api/v1/runs/{id}/events", h.PostEvents)

	// Node 接口
	mux.HandleFunc("POST /api/v1/nodes/heartbeat", h.NodeHeartbeat)
	mux.HandleFunc("GET /api/v1/nodes", h.ListNodes)
	mux.HandleFunc("GET /api/v1/nodes/{id}", h.GetNode)
	mux.HandleFunc("PATCH /api/v1/nodes/{id}", h.UpdateNode)
	mux.HandleFunc("DELETE /api/v1/nodes/{id}", h.DeleteNode)
	mux.HandleFunc("GET /api/v1/nodes/{id}/runs", h.GetNodeRuns)

	// ========== 新架构 API ==========

	// Agent 类型接口
	mux.HandleFunc("GET /api/v1/agent-types", h.ListAgentTypes)
	mux.HandleFunc("GET /api/v1/agent-types/{id}", h.GetAgentType)

	// 账号管理接口
	mux.HandleFunc("GET /api/v1/accounts", h.ListAccounts)
	mux.HandleFunc("POST /api/v1/accounts", h.CreateAccount)
	mux.HandleFunc("GET /api/v1/accounts/{id}", h.GetAccount)
	mux.HandleFunc("DELETE /api/v1/accounts/{id}", h.DeleteAccount)
	mux.HandleFunc("POST /api/v1/accounts/{id}/auth", h.StartAccountAuth)
	mux.HandleFunc("GET /api/v1/accounts/{id}/auth/status", h.GetAccountAuthStatus)

	// 认证任务接口（Node Agent 调用）
	mux.HandleFunc("GET /api/v1/auth-tasks/{id}", h.GetAuthTask)
	mux.HandleFunc("PATCH /api/v1/auth-tasks/{id}", h.UpdateAuthTask)
	mux.HandleFunc("GET /api/v1/nodes/{id}/auth-tasks", h.GetNodeAuthTasks)

	// 代理管理接口
	mux.HandleFunc("GET /api/v1/proxies", h.ListProxies)
	mux.HandleFunc("POST /api/v1/proxies", h.CreateProxy)
	mux.HandleFunc("GET /api/v1/proxies/{id}", h.GetProxy)
	mux.HandleFunc("PUT /api/v1/proxies/{id}", h.UpdateProxy)
	mux.HandleFunc("DELETE /api/v1/proxies/{id}", h.DeleteProxy)
	mux.HandleFunc("POST /api/v1/proxies/{id}/test", h.TestProxy)
	mux.HandleFunc("POST /api/v1/proxies/{id}/set-default", h.SetDefaultProxy)
	mux.HandleFunc("POST /api/v1/proxies/clear-default", h.ClearDefaultProxy)

	// 节点环境配置接口
	mux.HandleFunc("GET /api/v1/nodes/{id}/env-config", h.GetNodeEnvConfig)
	mux.HandleFunc("PUT /api/v1/nodes/{id}/env-config", h.UpdateNodeEnvConfig)
	mux.HandleFunc("POST /api/v1/nodes/{id}/env-config/test-proxy", h.TestNodeProxy)

	// 实例管理接口
	mux.HandleFunc("GET /api/v1/instances", h.ListInstances)
	mux.HandleFunc("POST /api/v1/instances", h.CreateInstance)
	mux.HandleFunc("GET /api/v1/instances/{id}", h.GetInstance)
	mux.HandleFunc("PATCH /api/v1/instances/{id}", h.UpdateInstanceStatus)
	mux.HandleFunc("DELETE /api/v1/instances/{id}", h.DeleteInstance)
	mux.HandleFunc("POST /api/v1/instances/{id}/start", h.StartInstance)
	mux.HandleFunc("POST /api/v1/instances/{id}/stop", h.StopInstance)

	// Executor 实例轮询接口
	mux.HandleFunc("GET /api/v1/nodes/{node_id}/instances", h.ListPendingInstances)

	// 终端会话接口
	mux.HandleFunc("GET /api/v1/terminal-sessions", h.ListTerminalSessions)
	mux.HandleFunc("POST /api/v1/terminal-sessions", h.CreateTerminalSession)
	mux.HandleFunc("GET /api/v1/terminal-sessions/{id}", h.GetTerminalSession)
	mux.HandleFunc("PATCH /api/v1/terminal-sessions/{id}", h.UpdateTerminalSessionStatus)
	mux.HandleFunc("DELETE /api/v1/terminal-sessions/{id}", h.DeleteTerminalSession)
	mux.HandleFunc("/terminal/{id}/", h.ProxyTerminalSession)

	// Executor 终端会话轮询接口
	mux.HandleFunc("GET /api/v1/nodes/{node_id}/terminal-sessions", h.ListPendingTerminalSessions)

	// 兼容旧路径（将废弃）
	mux.HandleFunc("POST /api/v1/terminal/session", h.CreateTerminalSession)
	mux.HandleFunc("GET /api/v1/terminal/session/{id}", h.GetTerminalSession)
	mux.HandleFunc("DELETE /api/v1/terminal/session/{id}", h.DeleteTerminalSession)

	// ========== 旧 Runner API（兼容，将废弃）==========
	mux.HandleFunc("GET /api/v1/runners", h.ListRunners)
	mux.HandleFunc("POST /api/v1/runners", h.CreateRunner)
	mux.HandleFunc("POST /api/v1/runners/login", h.StartRunnerLogin)
	mux.HandleFunc("GET /api/v1/runners/login/status", h.GetRunnerLoginStatus)
	mux.HandleFunc("DELETE /api/v1/runners", h.DeleteRunner)

	// ========== 监控 API ==========
	mux.HandleFunc("GET /api/v1/monitor/workflows", h.ListWorkflows)
	mux.HandleFunc("GET /api/v1/monitor/workflows/{type}/{id}", h.GetWorkflow)
	mux.HandleFunc("GET /api/v1/monitor/workflows/{type}/{id}/events", h.GetWorkflowEvents)
	mux.HandleFunc("GET /api/v1/monitor/stats", h.GetMonitorStats)

	// 应用指标中间件到 REST API
	apiHandler := h.metrics.MetricsMiddleware(mux)

	// 应用 CORS 中间件
	corsHandler := corsMiddleware(apiHandler)

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

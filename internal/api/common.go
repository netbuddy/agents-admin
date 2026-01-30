// Package api 提供 HTTP API 处理器
//
// 本包实现了 Agent Kanban 系统的 RESTful API，包括：
//   - 任务管理（Task）接口
//   - 执行管理（Run）接口
//   - 事件管理（Event）接口
//   - 节点管理（Node）接口
//   - WebSocket 实时推送
//
// 文件组织：
//   - common.go: 通用工具函数和 Handler 定义
//   - tasks.go: 任务相关接口
//   - runs.go: 执行相关接口
//   - events.go: 事件相关接口
//   - nodes.go: 节点相关接口
//   - scheduler.go: 调度器实现
//   - websocket.go: WebSocket 事件网关
package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"agents-admin/internal/storage"
)

// Handler API 处理器
//
// Handler 是所有 HTTP API 的入口，负责：
//   - 路由请求到对应的处理函数
//   - 管理存储层连接
//   - 协调调度器和事件网关
type Handler struct {
	store        *storage.PostgresStore // PostgreSQL 存储层（持久化业务数据）
	redisStore   *storage.RedisStore    // Redis 存储层（运行时状态、事件流）
	etcdStore    *storage.EtcdStore     // etcd 存储层（已弃用，保留兼容）
	scheduler    *Scheduler             // 任务调度器
	eventGateway *EventGateway          // WebSocket 事件网关
	eventBus     *storage.EtcdEventBus  // 事件总线（已弃用，保留兼容）
	metrics      *Metrics               // Prometheus 指标
}

// NewHandler 创建 Handler 实例
//
// 参数：
//   - store: PostgreSQL 存储层实例
//   - redisStore: Redis 存储层实例（运行时状态、事件流）
//   - etcdStore: etcd 存储层实例（可选，已弃用）
//
// 返回：
//   - 初始化完成的 Handler 实例
func NewHandler(store *storage.PostgresStore, redisStore *storage.RedisStore, etcdStore *storage.EtcdStore) *Handler {
	h := &Handler{store: store, redisStore: redisStore, etcdStore: etcdStore}
	h.scheduler = NewScheduler(store, h)
	h.eventGateway = NewEventGateway(store, redisStore)
	h.metrics = NewMetrics("api")
	return h
}

// GetRedisStore 获取 Redis 存储层
func (h *Handler) GetRedisStore() *storage.RedisStore {
	return h.redisStore
}

// GetMetrics 返回指标实例
func (h *Handler) GetMetrics() *Metrics {
	return h.metrics
}

// SetEventBus 设置事件总线（用于事件驱动模式）
func (h *Handler) SetEventBus(eventBus *storage.EtcdEventBus) {
	h.eventBus = eventBus
	if h.eventGateway != nil {
		h.eventGateway.SetEventBus(eventBus)
	}
}

// GetEventBus 获取事件总线
func (h *Handler) GetEventBus() *storage.EtcdEventBus {
	return h.eventBus
}

// writeJSON 将数据以 JSON 格式写入 HTTP 响应
//
// 参数：
//   - w: HTTP 响应写入器
//   - status: HTTP 状态码
//   - data: 要序列化为 JSON 的数据
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError 将错误信息以 JSON 格式写入 HTTP 响应
//
// 参数：
//   - w: HTTP 响应写入器
//   - status: HTTP 状态码
//   - message: 错误信息
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// generateID 生成带前缀的唯一标识符
//
// 使用加密安全的随机数生成 6 字节（12 个十六进制字符）的 ID，
// 格式为：prefix-xxxxxxxxxxxx
//
// 参数：
//   - prefix: ID 前缀（如 "task"、"run"）
//
// 返回：
//   - 生成的唯一标识符
func generateID(prefix string) string {
	b := make([]byte, 6)
	rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}

// Health 健康检查接口
//
// 路由: GET /health
//
// 用于负载均衡器和监控系统检查服务状态。
// 返回 {"status": "ok"} 表示服务正常运行。
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

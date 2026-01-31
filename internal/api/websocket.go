// Package api WebSocket 事件网关
//
// 事件网关提供实时事件推送能力，支持前端实时监控 Agent 执行过程。
// 使用 WebSocket 协议，支持双向通信。
package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"agents-admin/internal/storage"
)

// upgrader WebSocket 升级器配置
//
// 配置说明：
//   - ReadBufferSize: 读缓冲区大小
//   - WriteBufferSize: 写缓冲区大小
//   - CheckOrigin: 跨域检查（当前允许所有来源，生产环境应限制）
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// EventGateway WebSocket 事件网关
//
// 事件网关负责：
//   - 管理 WebSocket 连接
//   - 通过 Redis Streams 接收实时事件（新架构）
//   - 降级使用轮询数据库获取新事件（兼容模式）
//   - 将事件推送给订阅的客户端
//   - 在 Run 结束时通知客户端
//
// 使用场景：
//   - 前端实时显示 Agent 执行日志
//   - 监控 Run 状态变化
type EventGateway struct {
	store      *storage.PostgresStore              // PostgreSQL 存储层
	redisStore *storage.RedisStore                 // Redis 存储层（事件流）
	eventBus   *storage.EtcdEventBus               // 事件总线（已弃用，保留兼容）
	clients    map[string]map[*websocket.Conn]bool // 按 RunID 索引的客户端连接
	mu         sync.RWMutex                        // 保护 clients 映射
}

// NewEventGateway 创建事件网关实例
//
// 参数：
//   - store: PostgreSQL 存储层
//   - redisStore: Redis 存储层（事件流）
//
// 返回：
//   - 初始化完成的事件网关实例
func NewEventGateway(store *storage.PostgresStore, redisStore *storage.RedisStore) *EventGateway {
	return &EventGateway{
		store:      store,
		redisStore: redisStore,
		clients:    make(map[string]map[*websocket.Conn]bool),
	}
}

// SetEventBus 设置事件总线（用于事件驱动模式）
func (g *EventGateway) SetEventBus(eventBus *storage.EtcdEventBus) {
	g.eventBus = eventBus
}

// HandleWebSocket 处理 WebSocket 连接请求
//
// 路由: GET /ws/runs/{id}/events
//
// 路径参数：
//   - id: Run ID
//
// 查询参数：
//   - from_seq: 起始事件序号（可选），用于断线重连恢复
//
// 推送消息格式：
//
//	事件消息：{"type": "event", "data": {...}}
//	状态消息：{"type": "status", "data": {"status": "done", "finished_at": "..."}}
//
// 客户端消息：
//
//	心跳：{"type": "ping"} -> 响应 {"type": "pong"}
//
// 事件驱动优先级（P2-3）：
//   1. Redis Streams（推荐，统一方案）
//   2. etcd EventBus（已弃用，保留兼容）
//   3. 轮询模式（降级方案）
func (g *EventGateway) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		http.Error(w, "run_id required", http.StatusBadRequest)
		return
	}

	fromSeq, _ := strconv.Atoi(r.URL.Query().Get("from_seq"))

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	g.addClient(runID, conn)
	defer g.removeClient(runID, conn)

	log.Printf("WebSocket client connected for run %s", runID)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go g.readPump(conn, cancel)

	// P2-3: 优先使用 Redis Streams 事件驱动
	if g.redisStore != nil {
		g.writePumpRedisStreams(ctx, conn, runID, fromSeq)
		return
	}

	// 降级：使用 etcd EventBus（已弃用）
	if g.eventBus != nil {
		g.writePumpEventDriven(ctx, conn, runID, fromSeq)
		return
	}

	// 最后降级：轮询模式
	g.writePump(ctx, conn, runID, fromSeq)
}

// addClient 添加客户端连接
//
// 将 WebSocket 连接添加到指定 Run 的客户端列表中。
//
// 参数：
//   - runID: Run ID
//   - conn: WebSocket 连接
func (g *EventGateway) addClient(runID string, conn *websocket.Conn) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.clients[runID] == nil {
		g.clients[runID] = make(map[*websocket.Conn]bool)
	}
	g.clients[runID][conn] = true
}

// removeClient 移除客户端连接
//
// 从指定 Run 的客户端列表中移除连接。
// 如果该 Run 没有其他连接，则清理整个条目。
//
// 参数：
//   - runID: Run ID
//   - conn: WebSocket 连接
func (g *EventGateway) removeClient(runID string, conn *websocket.Conn) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if clients, ok := g.clients[runID]; ok {
		delete(clients, conn)
		if len(clients) == 0 {
			delete(g.clients, runID)
		}
	}
}

// readPump 读取客户端消息
//
// 在独立 goroutine 中运行，处理客户端发送的消息：
//   - 心跳消息（ping）：响应 pong
//   - 连接关闭：取消上下文
//
// 参数：
//   - conn: WebSocket 连接
//   - cancel: 上下文取消函数
func (g *EventGateway) readPump(conn *websocket.Conn, cancel context.CancelFunc) {
	defer cancel()
	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			return
		}

		var req map[string]interface{}
		if json.Unmarshal(msg, &req) == nil {
			if req["type"] == "ping" {
				conn.WriteJSON(map[string]string{"type": "pong"})
			}
		}
	}
}

// writePump 向客户端推送事件
//
// 主循环处理事件推送：
//   - 每 500ms 检查新事件并推送
//   - 每 30s 发送 ping 保持连接
//   - Run 结束时发送状态通知并退出
//
// 参数：
//   - ctx: 上下文
//   - conn: WebSocket 连接
//   - runID: Run ID
//   - fromSeq: 起始事件序号
func (g *EventGateway) writePump(ctx context.Context, conn *websocket.Conn, runID string, fromSeq int) {
	ticker := time.NewTicker(500 * time.Millisecond)
	pingTicker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	defer pingTicker.Stop()

	lastSeq := fromSeq

	for {
		select {
		case <-ctx.Done():
			return
		case <-pingTicker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-ticker.C:
			events, err := g.store.GetEventsByRun(ctx, runID, lastSeq, 100)
			if err != nil {
				log.Printf("Failed to get events: %v", err)
				continue
			}

			for _, event := range events {
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				msg := map[string]interface{}{
					"type": "event",
					"data": event,
				}
				if err := conn.WriteJSON(msg); err != nil {
					log.Printf("WebSocket write error: %v", err)
					return
				}
				if event.Seq > lastSeq {
					lastSeq = event.Seq
				}
			}

			run, err := g.store.GetRun(ctx, runID)
			if err == nil && run != nil {
				if run.Status == "done" || run.Status == "failed" || run.Status == "cancelled" {
					conn.WriteJSON(map[string]interface{}{
						"type": "status",
						"data": map[string]interface{}{
							"status":      run.Status,
							"finished_at": run.FinishedAt,
						},
					})
					return
				}
			}
		}
	}
}

// writePumpRedisStreams Redis Streams 事件驱动模式
//
// P2-3 统一方案：使用 Redis Streams 接收实时事件
// 相比 etcd Watch，Redis Streams 更适合高吞吐事件流场景
//
// 参数：
//   - ctx: 上下文
//   - conn: WebSocket 连接
//   - runID: Run ID
//   - fromSeq: 起始事件序号
func (g *EventGateway) writePumpRedisStreams(ctx context.Context, conn *websocket.Conn, runID string, fromSeq int) {
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// 首先推送历史事件（如果需要恢复）
	if fromSeq > 0 {
		events, err := g.store.GetEventsByRun(ctx, runID, fromSeq, 100)
		if err == nil {
			for _, event := range events {
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				msg := map[string]interface{}{
					"type": "event",
					"data": event,
				}
				if err := conn.WriteJSON(msg); err != nil {
					log.Printf("WebSocket write error: %v", err)
					return
				}
			}
		}
	}

	// 订阅 Redis Streams 事件流
	eventCh, err := g.redisStore.SubscribeRunEvents(ctx, runID)
	if err != nil {
		log.Printf("Failed to subscribe to Redis Streams: %v", err)
		// 降级到轮询模式
		g.writePump(ctx, conn, runID, fromSeq)
		return
	}

	log.Printf("WebSocket using Redis Streams for run %s", runID)

	for {
		select {
		case <-ctx.Done():
			return
		case <-pingTicker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case event, ok := <-eventCh:
			if !ok {
				// 事件通道关闭，检查 Run 状态
				run, err := g.store.GetRun(ctx, runID)
				if err == nil && run != nil {
					if run.Status == "done" || run.Status == "failed" || run.Status == "cancelled" {
						conn.WriteJSON(map[string]interface{}{
							"type": "status",
							"data": map[string]interface{}{
								"status":      run.Status,
								"finished_at": run.FinishedAt,
							},
						})
					}
				}
				return
			}

			// 推送事件
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			msg := map[string]interface{}{
				"type": "event",
				"data": map[string]interface{}{
					"seq":       event.Seq,
					"type":      event.Type,
					"timestamp": event.Timestamp,
					"payload":   event.Payload,
				},
			}
			if err := conn.WriteJSON(msg); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

			// 检查是否是终止事件
			if event.Type == "run_completed" || event.Type == "run_failed" {
				conn.WriteJSON(map[string]interface{}{
					"type": "status",
					"data": map[string]interface{}{
						"status": event.Type,
					},
				})
				return
			}
		}
	}
}

// writePumpEventDriven etcd 事件驱动模式的写入循环
//
// Deprecated: P2-3 后使用 writePumpRedisStreams 替代
// 保留此方法用于兼容，使用 etcd Watch 接收实时事件
//
// 参数：
//   - ctx: 上下文
//   - conn: WebSocket 连接
//   - runID: Run ID
//   - fromSeq: 起始事件序号
func (g *EventGateway) writePumpEventDriven(ctx context.Context, conn *websocket.Conn, runID string, fromSeq int) {
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// 订阅 etcd 事件流
	// 注意：这里使用 "run" 作为 workflowType，与任务执行工作流对应
	eventCh, err := g.eventBus.Subscribe(ctx, "run", runID, int64(fromSeq))
	if err != nil {
		log.Printf("Failed to subscribe to events: %v", err)
		// 降级到轮询模式
		g.writePump(ctx, conn, runID, fromSeq)
		return
	}

	log.Printf("WebSocket using event-driven mode for run %s", runID)

	for {
		select {
		case <-ctx.Done():
			return
		case <-pingTicker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case event, ok := <-eventCh:
			if !ok {
				// 事件通道关闭，检查 Run 状态
				run, err := g.store.GetRun(ctx, runID)
				if err == nil && run != nil {
					if run.Status == "done" || run.Status == "failed" || run.Status == "cancelled" {
						conn.WriteJSON(map[string]interface{}{
							"type": "status",
							"data": map[string]interface{}{
								"status":      run.Status,
								"finished_at": run.FinishedAt,
							},
						})
					}
				}
				return
			}

			// 推送事件
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			msg := map[string]interface{}{
				"type": "event",
				"data": map[string]interface{}{
					"seq":       event.Seq,
					"type":      event.Type,
					"timestamp": event.Timestamp,
					"payload":   event.Data,
				},
			}
			if err := conn.WriteJSON(msg); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

			// 检查是否是终止事件
			if event.Type == "run_completed" || event.Type == "run_failed" {
				conn.WriteJSON(map[string]interface{}{
					"type": "status",
					"data": map[string]interface{}{
						"status": event.Type,
					},
				})
				return
			}
		}
	}
}

// Broadcast 广播事件到指定 Run 的所有客户端
//
// 用于主动推送事件，不依赖轮询。
// 可在事件写入数据库后立即调用，实现更低延迟的推送。
//
// 参数：
//   - runID: Run ID
//   - event: 要广播的事件数据
func (g *EventGateway) Broadcast(runID string, event interface{}) {
	g.mu.RLock()
	clients := g.clients[runID]
	g.mu.RUnlock()

	msg := map[string]interface{}{
		"type": "event",
		"data": event,
	}

	for conn := range clients {
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("Broadcast error: %v", err)
		}
	}
}

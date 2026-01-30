// Package api 工作流监控 WebSocket
//
// 本文件提供工作流监控的 WebSocket 实时推送功能。
package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var monitorUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域（开发环境）
	},
}

// MonitorMessage WebSocket 消息
type MonitorMessage struct {
	Type      string      `json:"type"`      // workflows, stats, event
	Data      interface{} `json:"data"`      // 消息数据
	Timestamp time.Time   `json:"timestamp"` // 时间戳
}

// MonitorWSHandler WebSocket 监控连接处理器
type MonitorWSHandler struct {
	handler *Handler
	clients map[*websocket.Conn]bool
	mu      sync.RWMutex
}

// NewMonitorWSHandler 创建监控 WebSocket 处理器
func NewMonitorWSHandler(h *Handler) *MonitorWSHandler {
	mws := &MonitorWSHandler{
		handler: h,
		clients: make(map[*websocket.Conn]bool),
	}
	// 启动广播协程
	go mws.broadcastLoop()
	return mws
}

// HandleWebSocket 处理 WebSocket 连接
//
// 路由: GET /ws/monitor
func (m *MonitorWSHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := monitorUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[MonitorWS] Upgrade error: %v", err)
		return
	}

	m.mu.Lock()
	m.clients[conn] = true
	m.mu.Unlock()

	log.Printf("[MonitorWS] Client connected, total: %d", len(m.clients))

	// 发送初始数据
	m.sendInitialData(conn)

	// 读取客户端消息（保持连接）
	go m.readPump(conn)
}

func (m *MonitorWSHandler) readPump(conn *websocket.Conn) {
	defer func() {
		m.mu.Lock()
		delete(m.clients, conn)
		m.mu.Unlock()
		conn.Close()
		log.Printf("[MonitorWS] Client disconnected, remaining: %d", len(m.clients))
	}()

	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[MonitorWS] Read error: %v", err)
			}
			break
		}
	}
}

func (m *MonitorWSHandler) sendInitialData(conn *websocket.Conn) {
	ctx := context.Background()

	// 发送工作流列表
	workflows := m.handler.getAuthWorkflows(ctx, "")
	workflows = append(workflows, m.handler.getRunWorkflows(ctx, "")...)

	m.sendToClient(conn, MonitorMessage{
		Type:      "workflows",
		Data:      workflows,
		Timestamp: time.Now(),
	})

	// 发送统计信息
	stats := m.handler.calculateStats(ctx)
	m.sendToClient(conn, MonitorMessage{
		Type:      "stats",
		Data:      stats,
		Timestamp: time.Now(),
	})
}

func (m *MonitorWSHandler) sendToClient(conn *websocket.Conn, msg MonitorMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[MonitorWS] Marshal error: %v", err)
		return
	}

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("[MonitorWS] Write error: %v", err)
	}
}

func (m *MonitorWSHandler) broadcast(msg MonitorMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[MonitorWS] Marshal error: %v", err)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for conn := range m.clients {
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("[MonitorWS] Broadcast error: %v", err)
		}
	}
}

func (m *MonitorWSHandler) broadcastLoop() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.RLock()
		clientCount := len(m.clients)
		m.mu.RUnlock()

		if clientCount == 0 {
			continue
		}

		ctx := context.Background()

		// 广播工作流更新
		workflows := m.handler.getAuthWorkflows(ctx, "")
		workflows = append(workflows, m.handler.getRunWorkflows(ctx, "")...)

		m.broadcast(MonitorMessage{
			Type:      "workflows",
			Data:      workflows,
			Timestamp: time.Now(),
		})

		// 广播统计更新
		stats := m.handler.calculateStats(ctx)
		m.broadcast(MonitorMessage{
			Type:      "stats",
			Data:      stats,
			Timestamp: time.Now(),
		})

		// 发送心跳
		m.mu.RLock()
		for conn := range m.clients {
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("[MonitorWS] Ping error: %v", err)
			}
		}
		m.mu.RUnlock()
	}
}

// Package server 工作流监控 WebSocket 单元测试
//
// 本文件测试 monitor_ws.go 中的 MonitorWSHandler 功能：
//
// # 测试分组
//
// ## 构造与连接管理
//   - TestNewMonitorWSHandler: 验证处理器创建和字段初始化
//   - TestMonitorWS_ClientConnect: 客户端连接后注册到 clients map
//   - TestMonitorWS_ClientDisconnect: 客户端断开后自动清理
//
// ## 初始数据推送
//   - TestMonitorWS_InitialData: 连接后立即收到 workflows 和 stats 消息
//
// ## 广播
//   - TestMonitorWS_BroadcastToMultiple: 多客户端同时收到广播
//
// # 运行方式
//
//	go test -v -run TestNewMonitorWSHandler ./internal/apiserver/server/
//	go test -v -run TestMonitorWS ./internal/apiserver/server/
package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"agents-admin/internal/shared/model"
)

// ============================================================================
// 构造与连接管理测试
// ============================================================================

// newMonitorTestHandler 创建用于 monitor_ws 测试的 Handler
//
// 复用 monitor_test.go 中的 testMetrics 避免 Prometheus 重复注册。
func newMonitorTestHandler() *Handler {
	store := &mockMonitorStore{
		Tasks:     []*model.Task{},
		Runs:      map[string][]*model.Run{},
		Events:    map[string][]*model.Event{},
		AuthTasks: []*model.AuthTask{},
	}
	cs := &mockCacheStore{}
	return &Handler{
		store:      store,
		redisStore: cs,
		metrics:    testMetrics,
	}
}

// TestNewMonitorWSHandler 验证处理器创建
//
// 检查点：
//   - handler 正确注入
//   - clients map 已初始化
//   - broadcastLoop goroutine 已启动（通过短暂等待验证不 panic）
func TestNewMonitorWSHandler(t *testing.T) {
	h := newMonitorTestHandler()
	mws := NewMonitorWSHandler(h)

	if mws == nil {
		t.Fatal("NewMonitorWSHandler returned nil")
	}
	if mws.handler != h {
		t.Error("handler not set correctly")
	}
	if mws.clients == nil {
		t.Error("clients map should be initialized")
	}

	// broadcastLoop 在后台运行，验证不 panic
	time.Sleep(50 * time.Millisecond)
}

// TestMonitorWS_ClientConnect 客户端连接后注册到 clients map
//
// 验证 HandleWebSocket 将连接添加到管理列表。
func TestMonitorWS_ClientConnect(t *testing.T) {
	h := newMonitorTestHandler()
	mws := &MonitorWSHandler{
		handler: h,
		clients: make(map[*websocket.Conn]bool),
	}

	server := httptest.NewServer(http.HandlerFunc(mws.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer client.Close()

	// 等待连接注册
	time.Sleep(50 * time.Millisecond)

	mws.mu.RLock()
	count := len(mws.clients)
	mws.mu.RUnlock()

	if count != 1 {
		t.Errorf("client count = %d, want 1", count)
	}
}

// TestMonitorWS_ClientDisconnect 客户端断开后自动清理
//
// 验证 readPump 在连接关闭后将客户端从 map 中移除。
func TestMonitorWS_ClientDisconnect(t *testing.T) {
	h := newMonitorTestHandler()
	mws := &MonitorWSHandler{
		handler: h,
		clients: make(map[*websocket.Conn]bool),
	}

	server := httptest.NewServer(http.HandlerFunc(mws.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}

	// 等待连接注册
	time.Sleep(50 * time.Millisecond)

	mws.mu.RLock()
	if len(mws.clients) != 1 {
		t.Fatalf("expected 1 client before disconnect, got %d", len(mws.clients))
	}
	mws.mu.RUnlock()

	// 关闭客户端连接
	client.Close()

	// 等待 readPump 检测到断开并清理
	time.Sleep(200 * time.Millisecond)

	mws.mu.RLock()
	count := len(mws.clients)
	mws.mu.RUnlock()

	if count != 0 {
		t.Errorf("client count after disconnect = %d, want 0", count)
	}
}

// ============================================================================
// 初始数据推送测试
// ============================================================================

// TestMonitorWS_InitialData 连接后立即收到 workflows 和 stats 消息
//
// 验证 sendInitialData 在连接建立后发送两条消息：
//   1. type="workflows" — 工作流列表
//   2. type="stats" — 统计信息
func TestMonitorWS_InitialData(t *testing.T) {
	h := newMonitorTestHandler()
	mws := &MonitorWSHandler{
		handler: h,
		clients: make(map[*websocket.Conn]bool),
	}

	server := httptest.NewServer(http.HandlerFunc(mws.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer client.Close()

	// 读取初始消息
	var messages []MonitorMessage
	for i := 0; i < 2; i++ {
		client.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := client.ReadMessage()
		if err != nil {
			t.Fatalf("read message %d error: %v", i, err)
		}
		var m MonitorMessage
		if err := json.Unmarshal(msg, &m); err != nil {
			t.Fatalf("unmarshal message %d error: %v", i, err)
		}
		messages = append(messages, m)
	}

	// 验证收到 workflows 和 stats
	types := map[string]bool{}
	for _, m := range messages {
		types[m.Type] = true
	}

	if !types["workflows"] {
		t.Error("missing 'workflows' initial message")
	}
	if !types["stats"] {
		t.Error("missing 'stats' initial message")
	}
}

// ============================================================================
// 广播测试
// ============================================================================

// TestMonitorWS_BroadcastToMultiple 多客户端同时收到广播
//
// 验证 broadcast 方法将消息发送给所有已连接的客户端。
func TestMonitorWS_BroadcastToMultiple(t *testing.T) {
	h := newMonitorTestHandler()
	mws := &MonitorWSHandler{
		handler: h,
		clients: make(map[*websocket.Conn]bool),
	}

	server := httptest.NewServer(http.HandlerFunc(mws.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// 连接两个客户端
	c1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial c1 error: %v", err)
	}
	defer c1.Close()

	c2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial c2 error: %v", err)
	}
	defer c2.Close()

	// 消费掉初始消息（每个客户端 2 条：workflows + stats）
	for _, c := range []*websocket.Conn{c1, c2} {
		for i := 0; i < 2; i++ {
			c.SetReadDeadline(time.Now().Add(2 * time.Second))
			_, _, err := c.ReadMessage()
			if err != nil {
				t.Fatalf("drain initial message error: %v", err)
			}
		}
	}

	// 手动广播
	testMsg := MonitorMessage{
		Type:      "test_broadcast",
		Data:      map[string]string{"key": "value"},
		Timestamp: time.Now(),
	}
	mws.broadcast(testMsg)

	// 两个客户端都应收到
	for i, c := range []*websocket.Conn{c1, c2} {
		c.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := c.ReadMessage()
		if err != nil {
			t.Fatalf("client %d read error: %v", i+1, err)
		}
		var received MonitorMessage
		json.Unmarshal(msg, &received)
		if received.Type != "test_broadcast" {
			t.Errorf("client %d: type = %q, want 'test_broadcast'", i+1, received.Type)
		}
	}
}

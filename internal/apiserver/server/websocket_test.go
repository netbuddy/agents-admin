// Package server WebSocket 事件网关单元测试
//
// 本文件测试 EventGateway 的核心功能：
//
// # 测试分组
//
// ## 构造与初始化
//   - TestNewEventGateway: 验证网关创建、字段初始化
//   - TestNewEventGateway_NilEventBus: 验证无事件总线时的降级行为
//
// ## 客户端连接管理
//   - TestAddRemoveClient: 添加/移除单个客户端
//   - TestAddRemoveClient_MultipleClients: 同一 RunID 多客户端管理
//   - TestAddRemoveClient_MultipleRuns: 多个 RunID 独立管理
//   - TestRemoveClient_CleanupEmptyRun: 最后一个客户端移除后清理 RunID 条目
//   - TestClientCount: 并发安全的客户端计数
//
// ## 广播
//   - TestBroadcast: 向指定 Run 的所有客户端广播消息
//   - TestBroadcast_NoClients: 无客户端时广播不 panic
//   - TestBroadcast_IsolatedByRunID: 不同 RunID 互不影响
//
// ## WebSocket 集成（使用 httptest + gorilla/websocket）
//   - TestHandleWebSocket_PollingMode: 无事件总线时轮询模式
//   - TestHandleWebSocket_EventBusMode: 有事件总线时的事件驱动模式
//   - TestHandleWebSocket_MissingRunID: 缺少 RunID 参数返回 400
//   - TestHandleWebSocket_PingPong: 心跳消息处理
//
// # 使用的 Mock
//   - mockEventStore: 实现 eventStore 接口（GetEventsByRun, GetRun）
//   - mockRunEventBus: 实现 eventbus.RunEventBus 接口
//
// # 运行方式
//
//	go test -v -run TestNewEventGateway ./internal/apiserver/server/
//	go test -v -run TestBroadcast ./internal/apiserver/server/
//	go test -v -run TestHandleWebSocket ./internal/apiserver/server/
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"agents-admin/internal/shared/eventbus"
	"agents-admin/internal/shared/model"
)

// ============================================================================
// Mock 实现
// ============================================================================

// mockEventStore 模拟 eventStore 接口
//
// 可通过设置字段控制返回值：
//   - Events: GetEventsByRun 返回的事件列表
//   - Run: GetRun 返回的 Run 对象
//   - Err: 所有方法返回的错误
type mockEventStore struct {
	Events []*model.Event
	Run    *model.Run
	Err    error

	// 调用记录
	GetEventsByRunCalls []getEventsByRunCall
	GetRunCalls         []string
	mu                  sync.Mutex
}

type getEventsByRunCall struct {
	RunID   string
	FromSeq int
	Limit   int
}

func (m *mockEventStore) GetEventsByRun(_ context.Context, runID string, fromSeq int, limit int) ([]*model.Event, error) {
	m.mu.Lock()
	m.GetEventsByRunCalls = append(m.GetEventsByRunCalls, getEventsByRunCall{runID, fromSeq, limit})
	m.mu.Unlock()
	return m.Events, m.Err
}

func (m *mockEventStore) GetRun(_ context.Context, id string) (*model.Run, error) {
	m.mu.Lock()
	m.GetRunCalls = append(m.GetRunCalls, id)
	m.mu.Unlock()
	return m.Run, m.Err
}

// mockRunEventBus 模拟 RunEventBus 接口
//
// 可通过 EventCh 字段控制 SubscribeRunEvents 返回的通道。
// 如果 SubscribeErr 非 nil，SubscribeRunEvents 返回错误。
type mockRunEventBus struct {
	EventCh      chan *eventbus.RunEvent
	SubscribeErr error
}

func (m *mockRunEventBus) PublishRunEvent(_ context.Context, _ string, _ *eventbus.RunEvent) error {
	return nil
}

func (m *mockRunEventBus) GetRunEvents(_ context.Context, _ string, _ int, _ int64) ([]*eventbus.RunEvent, error) {
	return nil, nil
}

func (m *mockRunEventBus) GetRunEventCount(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

func (m *mockRunEventBus) SubscribeRunEvents(_ context.Context, _ string) (<-chan *eventbus.RunEvent, error) {
	if m.SubscribeErr != nil {
		return nil, m.SubscribeErr
	}
	return m.EventCh, nil
}

func (m *mockRunEventBus) DeleteRunEvents(_ context.Context, _ string) error {
	return nil
}

// ============================================================================
// 构造与初始化测试
// ============================================================================

// TestNewEventGateway 验证网关正确初始化
//
// 检查点：
//   - store 和 runEventBus 正确注入
//   - clients map 已初始化（非 nil）
func TestNewEventGateway(t *testing.T) {
	store := &mockEventStore{}
	bus := &mockRunEventBus{}

	gw := NewEventGateway(store, bus)

	if gw == nil {
		t.Fatal("NewEventGateway returned nil")
	}
	if gw.store != store {
		t.Error("store not set correctly")
	}
	if gw.runEventBus != bus {
		t.Error("runEventBus not set correctly")
	}
	if gw.clients == nil {
		t.Error("clients map should be initialized")
	}
	if len(gw.clients) != 0 {
		t.Errorf("clients map should be empty, got %d", len(gw.clients))
	}
}

// TestNewEventGateway_NilEventBus 验证无事件总线时也能正常创建
//
// 当 runEventBus 为 nil 时，HandleWebSocket 会降级到轮询模式。
func TestNewEventGateway_NilEventBus(t *testing.T) {
	gw := NewEventGateway(&mockEventStore{}, nil)

	if gw == nil {
		t.Fatal("NewEventGateway returned nil with nil eventbus")
	}
	if gw.runEventBus != nil {
		t.Error("runEventBus should be nil")
	}
}

// ============================================================================
// 客户端连接管理测试
// ============================================================================

// TestAddRemoveClient 测试添加和移除单个客户端
//
// 验证：
//   - addClient 后客户端存在于 clients map 中
//   - removeClient 后客户端被移除
//   - 最后一个客户端移除后 RunID 条目被清理
func TestAddRemoveClient(t *testing.T) {
	gw := NewEventGateway(&mockEventStore{}, nil)
	conn := &websocket.Conn{} // 用作 map key，不需要真实连接

	// 添加
	gw.addClient("run-1", conn)

	gw.mu.RLock()
	if len(gw.clients["run-1"]) != 1 {
		t.Errorf("expected 1 client, got %d", len(gw.clients["run-1"]))
	}
	if !gw.clients["run-1"][conn] {
		t.Error("conn should be in clients map")
	}
	gw.mu.RUnlock()

	// 移除
	gw.removeClient("run-1", conn)

	gw.mu.RLock()
	if _, ok := gw.clients["run-1"]; ok {
		t.Error("run-1 entry should be cleaned up after last client removed")
	}
	gw.mu.RUnlock()
}

// TestAddRemoveClient_MultipleClients 同一 RunID 多客户端
//
// 验证：
//   - 同一 RunID 可以有多个客户端
//   - 移除一个不影响其他
func TestAddRemoveClient_MultipleClients(t *testing.T) {
	gw := NewEventGateway(&mockEventStore{}, nil)
	conn1 := &websocket.Conn{}
	conn2 := &websocket.Conn{}

	gw.addClient("run-1", conn1)
	gw.addClient("run-1", conn2)

	gw.mu.RLock()
	if len(gw.clients["run-1"]) != 2 {
		t.Errorf("expected 2 clients, got %d", len(gw.clients["run-1"]))
	}
	gw.mu.RUnlock()

	// 移除 conn1
	gw.removeClient("run-1", conn1)

	gw.mu.RLock()
	if len(gw.clients["run-1"]) != 1 {
		t.Errorf("expected 1 client after removal, got %d", len(gw.clients["run-1"]))
	}
	if !gw.clients["run-1"][conn2] {
		t.Error("conn2 should still exist")
	}
	gw.mu.RUnlock()
}

// TestAddRemoveClient_MultipleRuns 多个 RunID 独立管理
//
// 验证不同 RunID 的客户端互不影响。
func TestAddRemoveClient_MultipleRuns(t *testing.T) {
	gw := NewEventGateway(&mockEventStore{}, nil)
	conn1 := &websocket.Conn{}
	conn2 := &websocket.Conn{}

	gw.addClient("run-1", conn1)
	gw.addClient("run-2", conn2)

	gw.mu.RLock()
	if len(gw.clients) != 2 {
		t.Errorf("expected 2 run entries, got %d", len(gw.clients))
	}
	gw.mu.RUnlock()

	// 移除 run-1 不影响 run-2
	gw.removeClient("run-1", conn1)

	gw.mu.RLock()
	if _, ok := gw.clients["run-1"]; ok {
		t.Error("run-1 should be cleaned up")
	}
	if len(gw.clients["run-2"]) != 1 {
		t.Error("run-2 should still have 1 client")
	}
	gw.mu.RUnlock()
}

// TestRemoveClient_NonExistentRun 移除不存在的 RunID 不 panic
func TestRemoveClient_NonExistentRun(t *testing.T) {
	gw := NewEventGateway(&mockEventStore{}, nil)
	conn := &websocket.Conn{}

	// 不应 panic
	gw.removeClient("non-existent", conn)
}

// TestClientCount 验证并发安全的客户端操作
//
// 在多个 goroutine 中同时添加/移除客户端，验证最终状态一致。
func TestClientCount(t *testing.T) {
	gw := NewEventGateway(&mockEventStore{}, nil)

	var wg sync.WaitGroup
	conns := make([]*websocket.Conn, 100)
	for i := range conns {
		conns[i] = &websocket.Conn{}
	}

	// 并发添加
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			gw.addClient("run-concurrent", conns[idx])
		}(i)
	}
	wg.Wait()

	gw.mu.RLock()
	if len(gw.clients["run-concurrent"]) != 100 {
		t.Errorf("expected 100 clients, got %d", len(gw.clients["run-concurrent"]))
	}
	gw.mu.RUnlock()

	// 并发移除
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			gw.removeClient("run-concurrent", conns[idx])
		}(i)
	}
	wg.Wait()

	gw.mu.RLock()
	if _, ok := gw.clients["run-concurrent"]; ok {
		t.Error("run-concurrent entry should be cleaned up")
	}
	gw.mu.RUnlock()
}

// ============================================================================
// 广播测试
// ============================================================================

// TestBroadcast 向指定 Run 的所有客户端广播消息
//
// 使用 httptest.Server + WebSocket 客户端验证实际消息传递。
func TestBroadcast(t *testing.T) {
	store := &mockEventStore{
		// Run 处于 running 状态，不会触发 status 消息退出
		Run: &model.Run{ID: "run-1", Status: model.RunStatusRunning},
	}
	gw := NewEventGateway(store, nil)

	// 启动 WebSocket 测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}
		gw.addClient("run-1", conn)
		defer gw.removeClient("run-1", conn)

		// 保持连接直到服务器关闭
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer server.Close()

	// 连接 WebSocket 客户端
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer client.Close()

	// 等待连接建立
	time.Sleep(50 * time.Millisecond)

	// 广播消息
	testEvent := map[string]interface{}{
		"seq":  float64(1),
		"type": "test_event",
	}
	gw.Broadcast("run-1", testEvent)

	// 读取消息
	client.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var received map[string]interface{}
	if err := json.Unmarshal(msg, &received); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if received["type"] != "event" {
		t.Errorf("message type = %v, want 'event'", received["type"])
	}
	data, ok := received["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data should be a map")
	}
	if data["type"] != "test_event" {
		t.Errorf("event type = %v, want 'test_event'", data["type"])
	}
}

// TestBroadcast_NoClients 无客户端时广播不 panic
func TestBroadcast_NoClients(t *testing.T) {
	gw := NewEventGateway(&mockEventStore{}, nil)

	// 不应 panic
	gw.Broadcast("non-existent-run", map[string]string{"type": "test"})
}

// TestBroadcast_IsolatedByRunID 不同 RunID 的广播互不影响
//
// 验证：向 run-1 广播时，run-2 的客户端不会收到消息。
func TestBroadcast_IsolatedByRunID(t *testing.T) {
	gw := NewEventGateway(&mockEventStore{}, nil)

	// 服务端：根据查询参数将连接注册到不同 RunID
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		runID := r.URL.Query().Get("run_id")
		gw.addClient(runID, conn)
		defer gw.removeClient(runID, conn)

		// 保持连接打开
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// 连接 run-1 和 run-2
	c1, _, err := websocket.DefaultDialer.Dial(wsURL+"?run_id=run-1", nil)
	if err != nil {
		t.Fatalf("dial run-1 error: %v", err)
	}
	defer c1.Close()

	c2, _, err := websocket.DefaultDialer.Dial(wsURL+"?run_id=run-2", nil)
	if err != nil {
		t.Fatalf("dial run-2 error: %v", err)
	}
	defer c2.Close()

	// 等待连接注册完成
	time.Sleep(50 * time.Millisecond)

	// 只广播到 run-1
	gw.Broadcast("run-1", map[string]string{"type": "test"})

	// run-1 客户端应收到消息
	c1.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := c1.ReadMessage()
	if err != nil {
		t.Fatalf("run-1 read error: %v", err)
	}
	var received map[string]interface{}
	json.Unmarshal(msg, &received)
	if received["type"] != "event" {
		t.Errorf("run-1 message type = %v, want 'event'", received["type"])
	}

	// run-2 客户端不应收到消息（短超时验证）
	c2.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, err = c2.ReadMessage()
	if err == nil {
		t.Error("run-2 should NOT receive run-1's broadcast")
	}
	// 超时错误是预期的
}

// ============================================================================
// WebSocket 集成测试
// ============================================================================

// TestHandleWebSocket_MissingRunID 缺少 RunID 返回 400
func TestHandleWebSocket_MissingRunID(t *testing.T) {
	gw := NewEventGateway(&mockEventStore{}, nil)

	req := httptest.NewRequest("GET", "/ws/runs//events", nil)
	w := httptest.NewRecorder()

	gw.HandleWebSocket(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestHandleWebSocket_PollingMode 无事件总线时使用轮询模式
//
// 验证：
//   - Run 完成后发送 status 消息并关闭连接
//   - 客户端收到完整的事件流 + 状态通知
func TestHandleWebSocket_PollingMode(t *testing.T) {
	now := time.Now()
	finishedAt := now.Add(1 * time.Minute)

	store := &mockEventStore{
		Events: []*model.Event{
			{ID: 1, RunID: "run-1", Seq: 1, Type: "run_started", Timestamp: now},
		},
		Run: &model.Run{
			ID:         "run-1",
			Status:     model.RunStatusDone,
			FinishedAt: &finishedAt,
		},
	}

	// 无事件总线 → 轮询模式
	gw := NewEventGateway(store, nil)

	// 使用 Go 1.22 路由模式设置 PathValue
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/runs/{id}/events", gw.HandleWebSocket)
	server := httptest.NewServer(mux)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/runs/run-1/events"
	client, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer client.Close()

	// 读取消息（轮询间隔 500ms，应在 2s 内收到事件+状态）
	var messages []map[string]interface{}
	client.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		_, msg, err := client.ReadMessage()
		if err != nil {
			break
		}
		var m map[string]interface{}
		json.Unmarshal(msg, &m)
		messages = append(messages, m)

		// 收到 status 消息后退出
		if m["type"] == "status" {
			break
		}
	}

	if len(messages) < 2 {
		t.Fatalf("expected at least 2 messages (event + status), got %d", len(messages))
	}

	// 验证第一条消息是事件
	if messages[0]["type"] != "event" {
		t.Errorf("first message type = %v, want 'event'", messages[0]["type"])
	}

	// 验证最后一条消息是状态
	lastMsg := messages[len(messages)-1]
	if lastMsg["type"] != "status" {
		t.Errorf("last message type = %v, want 'status'", lastMsg["type"])
	}
	statusData, _ := lastMsg["data"].(map[string]interface{})
	if statusData["status"] != "done" {
		t.Errorf("status = %v, want 'done'", statusData["status"])
	}
}

// TestHandleWebSocket_EventBusMode 有事件总线时使用事件驱动模式
//
// 验证：
//   - 事件通过 RunEventBus 通道推送
//   - 终止事件触发 status 消息
func TestHandleWebSocket_EventBusMode(t *testing.T) {
	now := time.Now()
	store := &mockEventStore{
		Run: &model.Run{ID: "run-1", Status: model.RunStatusRunning},
	}

	eventCh := make(chan *eventbus.RunEvent, 10)
	bus := &mockRunEventBus{EventCh: eventCh}

	gw := NewEventGateway(store, bus)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws/runs/{id}/events", gw.HandleWebSocket)
	server := httptest.NewServer(mux)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/runs/run-1/events"
	client, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer client.Close()

	// 等待连接建立和订阅完成
	time.Sleep(100 * time.Millisecond)

	// 通过事件总线推送事件
	eventCh <- &eventbus.RunEvent{
		Seq:       1,
		Type:      "message",
		Timestamp: now,
		Payload:   map[string]interface{}{"content": "hello"},
	}

	// 读取消息
	client.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var received map[string]interface{}
	json.Unmarshal(msg, &received)

	if received["type"] != "event" {
		t.Errorf("message type = %v, want 'event'", received["type"])
	}

	// 推送终止事件
	eventCh <- &eventbus.RunEvent{
		Seq:  2,
		Type: "run_completed",
	}

	// 应收到事件 + 状态消息
	var gotStatus bool
	for i := 0; i < 3; i++ {
		client.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err = client.ReadMessage()
		if err != nil {
			break
		}
		var m map[string]interface{}
		json.Unmarshal(msg, &m)
		if m["type"] == "status" {
			gotStatus = true
			break
		}
	}

	if !gotStatus {
		t.Error("expected status message after run_completed event")
	}
}

// TestHandleWebSocket_EventBusSubscribeFail 事件总线订阅失败时降级到轮询
func TestHandleWebSocket_EventBusSubscribeFail(t *testing.T) {
	now := time.Now()
	finishedAt := now.Add(1 * time.Minute)

	store := &mockEventStore{
		Events: []*model.Event{
			{ID: 1, RunID: "run-1", Seq: 1, Type: "run_started", Timestamp: now},
		},
		Run: &model.Run{
			ID:         "run-1",
			Status:     model.RunStatusDone,
			FinishedAt: &finishedAt,
		},
	}

	bus := &mockRunEventBus{
		SubscribeErr: context.DeadlineExceeded,
	}
	gw := NewEventGateway(store, bus)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws/runs/{id}/events", gw.HandleWebSocket)
	server := httptest.NewServer(mux)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/runs/run-1/events"
	client, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer client.Close()

	// 应降级到轮询模式并最终收到 status done
	var gotStatus bool
	client.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		_, msg, err := client.ReadMessage()
		if err != nil {
			break
		}
		var m map[string]interface{}
		json.Unmarshal(msg, &m)
		if m["type"] == "status" {
			gotStatus = true
			break
		}
	}

	if !gotStatus {
		t.Error("expected status message from polling fallback")
	}
}

// TestHandleWebSocket_FromSeq 断线重连恢复参数
func TestHandleWebSocket_FromSeq(t *testing.T) {
	store := &mockEventStore{
		Events: []*model.Event{},
		Run:    &model.Run{ID: "run-1", Status: model.RunStatusDone},
	}

	gw := NewEventGateway(store, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws/runs/{id}/events", gw.HandleWebSocket)
	server := httptest.NewServer(mux)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/runs/run-1/events?from_seq=5"
	client, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer client.Close()

	// 等待轮询执行至少一次
	time.Sleep(600 * time.Millisecond)

	// 验证 store 被调用时传入了 fromSeq=5
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.GetEventsByRunCalls) == 0 {
		t.Fatal("GetEventsByRun should have been called")
	}
	// 第一次调用的 fromSeq 应该是 5
	if store.GetEventsByRunCalls[0].FromSeq != 5 {
		t.Errorf("fromSeq = %d, want 5", store.GetEventsByRunCalls[0].FromSeq)
	}
}

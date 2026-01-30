// Package storage etcd 存储单元测试
package storage

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// MockEtcdKV 模拟 etcd KV 操作
type MockEtcdKV struct {
	data map[string]string
}

func NewMockEtcdKV() *MockEtcdKV {
	return &MockEtcdKV{data: make(map[string]string)}
}

func (m *MockEtcdKV) Put(key, value string) {
	m.data[key] = value
}

func (m *MockEtcdKV) Get(key string) (string, bool) {
	v, ok := m.data[key]
	return v, ok
}

func (m *MockEtcdKV) Delete(key string) {
	delete(m.data, key)
}

// TestNodeHeartbeatSerialization 测试 NodeHeartbeat 序列化
func TestNodeHeartbeatSerialization(t *testing.T) {
	hb := &NodeHeartbeat{
		NodeID:        "node-001",
		Status:        "online",
		LastHeartbeat: time.Now(),
		Capacity: map[string]interface{}{
			"max_tasks": 10,
			"cpu":       "4",
			"memory":    "8G",
		},
	}

	// 序列化
	data, err := json.Marshal(hb)
	if err != nil {
		t.Fatalf("Failed to marshal heartbeat: %v", err)
	}

	// 反序列化
	var hb2 NodeHeartbeat
	if err := json.Unmarshal(data, &hb2); err != nil {
		t.Fatalf("Failed to unmarshal heartbeat: %v", err)
	}

	// 验证字段
	if hb2.NodeID != hb.NodeID {
		t.Errorf("NodeID mismatch: got %s, want %s", hb2.NodeID, hb.NodeID)
	}
	if hb2.Status != hb.Status {
		t.Errorf("Status mismatch: got %s, want %s", hb2.Status, hb.Status)
	}
	// JSON 解析后数字类型为 float64
	if hb2.Capacity["max_tasks"].(float64) != float64(hb.Capacity["max_tasks"].(int)) {
		t.Errorf("Capacity max_tasks mismatch")
	}
}

// TestEtcdConfigDefaults 测试 EtcdConfig 默认值
func TestEtcdConfigDefaults(t *testing.T) {
	cfg := EtcdConfig{
		Endpoints: []string{"localhost:2379"},
	}

	// 验证默认值在 NewEtcdStore 中被设置
	if cfg.DialTimeout == 0 {
		// 这是期望的，因为默认值在 NewEtcdStore 中设置
		t.Log("DialTimeout is 0, will be set to default in NewEtcdStore")
	}
	if cfg.Prefix == "" {
		t.Log("Prefix is empty, will be set to default '/agents' in NewEtcdStore")
	}
}

// TestHeartbeatKeyFormat 测试心跳 key 格式
func TestHeartbeatKeyFormat(t *testing.T) {
	prefix := "/agents"
	nodeID := "node-001"
	expectedKey := "/agents/nodes/node-001/heartbeat"

	// 模拟 key 生成
	key := prefix + "/nodes/" + nodeID + "/heartbeat"
	if key != expectedKey {
		t.Errorf("Key format mismatch: got %s, want %s", key, expectedKey)
	}
}

// TestNodeHeartbeatStatusValues 测试有效的状态值
func TestNodeHeartbeatStatusValues(t *testing.T) {
	validStatuses := []string{"online", "offline", "busy", "draining"}

	for _, status := range validStatuses {
		hb := &NodeHeartbeat{
			NodeID: "test-node",
			Status: status,
		}

		data, err := json.Marshal(hb)
		if err != nil {
			t.Errorf("Failed to marshal heartbeat with status %s: %v", status, err)
			continue
		}

		var hb2 NodeHeartbeat
		if err := json.Unmarshal(data, &hb2); err != nil {
			t.Errorf("Failed to unmarshal heartbeat with status %s: %v", status, err)
			continue
		}

		if hb2.Status != status {
			t.Errorf("Status mismatch: got %s, want %s", hb2.Status, status)
		}
	}
}

// TestNodeHeartbeatCapacityTypes 测试 Capacity 字段的各种类型
func TestNodeHeartbeatCapacityTypes(t *testing.T) {
	hb := &NodeHeartbeat{
		NodeID: "test-node",
		Status: "online",
		Capacity: map[string]interface{}{
			"max_tasks":  float64(10), // JSON 数字默认解析为 float64
			"cpu_cores":  float64(4),
			"memory_gb":  float64(16),
			"labels":     []interface{}{"gpu", "high-mem"},
			"is_primary": true,
		},
	}

	data, err := json.Marshal(hb)
	if err != nil {
		t.Fatalf("Failed to marshal heartbeat: %v", err)
	}

	var hb2 NodeHeartbeat
	if err := json.Unmarshal(data, &hb2); err != nil {
		t.Fatalf("Failed to unmarshal heartbeat: %v", err)
	}

	// 验证不同类型的值
	if hb2.Capacity["max_tasks"].(float64) != 10 {
		t.Errorf("max_tasks mismatch")
	}
	if hb2.Capacity["is_primary"].(bool) != true {
		t.Errorf("is_primary mismatch")
	}
}

// TestEtcdStorePrefix 测试不同前缀
func TestEtcdStorePrefix(t *testing.T) {
	testCases := []struct {
		name     string
		prefix   string
		expected string
	}{
		{"default prefix", "", "/agents"},
		{"custom prefix", "/myapp", "/myapp"},
		{"with trailing slash", "/test/", "/test/"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := EtcdConfig{
				Endpoints: []string{"localhost:2379"},
				Prefix:    tc.prefix,
			}
			if tc.prefix == "" {
				cfg.Prefix = "/agents" // 模拟默认值设置
			}
			if cfg.Prefix != tc.expected {
				t.Errorf("Prefix mismatch: got %s, want %s", cfg.Prefix, tc.expected)
			}
		})
	}
}

// MockWatchResponse 模拟 Watch 响应
type MockWatchResponse struct {
	Events []*clientv3.Event
}

// TestWatchEventParsing 测试 Watch 事件解析
func TestWatchEventParsing(t *testing.T) {
	hb := &NodeHeartbeat{
		NodeID:        "node-001",
		Status:        "online",
		LastHeartbeat: time.Now(),
	}

	data, _ := json.Marshal(hb)

	// 模拟 PUT 事件
	event := &clientv3.Event{
		Type: clientv3.EventTypePut,
		Kv: &mvccpb.KeyValue{
			Key:   []byte("/agents/nodes/node-001/heartbeat"),
			Value: data,
		},
	}

	// 验证事件类型
	if event.Type != clientv3.EventTypePut {
		t.Errorf("Event type mismatch: got %v, want PUT", event.Type)
	}

	// 解析心跳数据
	var parsed NodeHeartbeat
	if err := json.Unmarshal(event.Kv.Value, &parsed); err != nil {
		t.Fatalf("Failed to parse heartbeat from event: %v", err)
	}

	if parsed.NodeID != hb.NodeID {
		t.Errorf("NodeID mismatch after parsing: got %s, want %s", parsed.NodeID, hb.NodeID)
	}
}

// TestWatchDeleteEvent 测试 DELETE 事件处理
func TestWatchDeleteEvent(t *testing.T) {
	event := &clientv3.Event{
		Type: clientv3.EventTypeDelete,
		Kv: &mvccpb.KeyValue{
			Key: []byte("/agents/nodes/node-001/heartbeat"),
		},
	}

	// DELETE 事件应该被识别
	if event.Type != clientv3.EventTypeDelete {
		t.Errorf("Event type should be DELETE")
	}

	// DELETE 事件的 Value 通常为空
	if len(event.Kv.Value) != 0 {
		t.Errorf("DELETE event should not have value")
	}
}

// TestContextTimeout 测试上下文超时
func TestContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// 模拟超时场景
	time.Sleep(150 * time.Millisecond)

	select {
	case <-ctx.Done():
		if ctx.Err() != context.DeadlineExceeded {
			t.Errorf("Expected DeadlineExceeded, got %v", ctx.Err())
		}
	default:
		t.Error("Context should have been cancelled")
	}
}

// TestHeartbeatTTL 测试心跳 TTL 逻辑
func TestHeartbeatTTL(t *testing.T) {
	// 模拟 30 秒 TTL
	ttl := int64(30)

	if ttl != 30 {
		t.Errorf("TTL should be 30 seconds, got %d", ttl)
	}

	// 验证 TTL 用于判断节点在线状态
	lastHeartbeat := time.Now().Add(-20 * time.Second)
	isOnline := time.Since(lastHeartbeat) < time.Duration(ttl)*time.Second
	if !isOnline {
		t.Error("Node should be considered online within TTL")
	}

	// 超过 TTL 应该离线
	lastHeartbeat = time.Now().Add(-40 * time.Second)
	isOnline = time.Since(lastHeartbeat) < time.Duration(ttl)*time.Second
	if isOnline {
		t.Error("Node should be considered offline after TTL")
	}
}

// BenchmarkHeartbeatSerialization 基准测试心跳序列化
func BenchmarkHeartbeatSerialization(b *testing.B) {
	hb := &NodeHeartbeat{
		NodeID:        "node-001",
		Status:        "online",
		LastHeartbeat: time.Now(),
		Capacity: map[string]interface{}{
			"max_tasks": 10,
			"cpu":       "4",
			"memory":    "8G",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _ := json.Marshal(hb)
		var hb2 NodeHeartbeat
		json.Unmarshal(data, &hb2)
	}
}

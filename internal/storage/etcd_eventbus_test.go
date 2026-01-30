// Package storage etcd 事件总线单元测试
package storage

import (
	"encoding/json"
	"testing"
	"time"
)

// TestWorkflowStateValues 测试工作流状态值
func TestWorkflowStateValues(t *testing.T) {
	tests := []struct {
		state    WorkflowState
		expected string
	}{
		{WorkflowStatePending, "pending"},
		{WorkflowStateRunning, "running"},
		{WorkflowStateWaiting, "waiting"},
		{WorkflowStateCompleted, "completed"},
		{WorkflowStateFailed, "failed"},
		{WorkflowStateCancelled, "cancelled"},
	}

	for _, tt := range tests {
		if string(tt.state) != tt.expected {
			t.Errorf("WorkflowState mismatch: got %s, want %s", tt.state, tt.expected)
		}
	}
}

// TestWorkflowEventSerialization 测试 WorkflowEvent 序列化
func TestWorkflowEventSerialization(t *testing.T) {
	event := &WorkflowEvent{
		ID:         "evt-001",
		WorkflowID: "wf-001",
		Type:       "auth.started",
		Seq:        1,
		Data: map[string]interface{}{
			"account_id": "acc-001",
			"method":     "oauth",
		},
		ProducerID: "node-001",
		Timestamp:  time.Now(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	var event2 WorkflowEvent
	if err := json.Unmarshal(data, &event2); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if event2.ID != event.ID {
		t.Errorf("ID mismatch: got %s, want %s", event2.ID, event.ID)
	}
	if event2.WorkflowID != event.WorkflowID {
		t.Errorf("WorkflowID mismatch: got %s, want %s", event2.WorkflowID, event.WorkflowID)
	}
	if event2.Type != event.Type {
		t.Errorf("Type mismatch: got %s, want %s", event2.Type, event.Type)
	}
	if event2.Seq != event.Seq {
		t.Errorf("Seq mismatch: got %d, want %d", event2.Seq, event.Seq)
	}
	if event2.ProducerID != event.ProducerID {
		t.Errorf("ProducerID mismatch: got %s, want %s", event2.ProducerID, event.ProducerID)
	}
}

// TestWorkflowStateDataSerialization 测试 WorkflowStateData 序列化
func TestWorkflowStateDataSerialization(t *testing.T) {
	state := &WorkflowStateData{
		ID:    "wf-001",
		Type:  "auth",
		State: WorkflowStateRunning,
		Data: map[string]interface{}{
			"step":    "oauth_pending",
			"message": "Waiting for user authorization",
		},
		UpdatedAt: time.Now(),
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Failed to marshal state: %v", err)
	}

	var state2 WorkflowStateData
	if err := json.Unmarshal(data, &state2); err != nil {
		t.Fatalf("Failed to unmarshal state: %v", err)
	}

	if state2.ID != state.ID {
		t.Errorf("ID mismatch: got %s, want %s", state2.ID, state.ID)
	}
	if state2.Type != state.Type {
		t.Errorf("Type mismatch: got %s, want %s", state2.Type, state.Type)
	}
	if state2.State != state.State {
		t.Errorf("State mismatch: got %s, want %s", state2.State, state.State)
	}
}

// TestEventKeyGeneration 测试事件 key 生成
func TestEventKeyGeneration(t *testing.T) {
	bus := &EtcdEventBus{prefix: "/agents"}

	tests := []struct {
		workflowType string
		workflowID   string
		seq          int64
		expected     string
	}{
		{"auth", "task-001", 1, "/agents/events/auth/task-001/000001"},
		{"auth", "task-001", 10, "/agents/events/auth/task-001/000010"},
		{"auth", "task-001", 100, "/agents/events/auth/task-001/000100"},
		{"run", "run-abc", 999999, "/agents/events/run/run-abc/999999"},
	}

	for _, tt := range tests {
		key := bus.eventKey(tt.workflowType, tt.workflowID, tt.seq)
		if key != tt.expected {
			t.Errorf("eventKey mismatch: got %s, want %s", key, tt.expected)
		}
	}
}

// TestEventPrefixGeneration 测试事件前缀生成
func TestEventPrefixGeneration(t *testing.T) {
	bus := &EtcdEventBus{prefix: "/agents"}

	tests := []struct {
		workflowType string
		workflowID   string
		expected     string
	}{
		{"auth", "task-001", "/agents/events/auth/task-001/"},
		{"run", "run-abc", "/agents/events/run/run-abc/"},
	}

	for _, tt := range tests {
		prefix := bus.eventPrefix(tt.workflowType, tt.workflowID)
		if prefix != tt.expected {
			t.Errorf("eventPrefix mismatch: got %s, want %s", prefix, tt.expected)
		}
	}
}

// TestStateKeyGeneration 测试状态 key 生成
func TestStateKeyGeneration(t *testing.T) {
	bus := &EtcdEventBus{prefix: "/agents"}

	tests := []struct {
		workflowType string
		workflowID   string
		expected     string
	}{
		{"auth", "task-001", "/agents/state/auth/task-001"},
		{"run", "run-abc", "/agents/state/run/run-abc"},
	}

	for _, tt := range tests {
		key := bus.stateKey(tt.workflowType, tt.workflowID)
		if key != tt.expected {
			t.Errorf("stateKey mismatch: got %s, want %s", key, tt.expected)
		}
	}
}

// TestStatePrefixGeneration 测试状态前缀生成
func TestStatePrefixGeneration(t *testing.T) {
	bus := &EtcdEventBus{prefix: "/agents"}

	tests := []struct {
		workflowType string
		expected     string
	}{
		{"auth", "/agents/state/auth/"},
		{"run", "/agents/state/run/"},
	}

	for _, tt := range tests {
		prefix := bus.statePrefix(tt.workflowType)
		if prefix != tt.expected {
			t.Errorf("statePrefix mismatch: got %s, want %s", prefix, tt.expected)
		}
	}
}

// TestNewEtcdEventBus 测试创建事件总线
func TestNewEtcdEventBus(t *testing.T) {
	// 测试默认前缀
	bus := NewEtcdEventBus(nil, "")
	if bus.prefix != "/agents" {
		t.Errorf("Default prefix should be /agents, got %s", bus.prefix)
	}

	// 测试自定义前缀
	bus = NewEtcdEventBus(nil, "/custom")
	if bus.prefix != "/custom" {
		t.Errorf("Custom prefix should be /custom, got %s", bus.prefix)
	}
}

// TestWorkflowEventTimestamp 测试事件时间戳处理
func TestWorkflowEventTimestamp(t *testing.T) {
	event := &WorkflowEvent{
		ID:         "evt-001",
		WorkflowID: "wf-001",
		Type:       "test",
	}

	// 空时间戳应该被处理
	if !event.Timestamp.IsZero() {
		t.Error("Initial timestamp should be zero")
	}

	// 设置时间戳
	now := time.Now()
	event.Timestamp = now

	data, _ := json.Marshal(event)
	var event2 WorkflowEvent
	json.Unmarshal(data, &event2)

	// 验证时间戳被正确序列化
	if event2.Timestamp.Unix() != now.Unix() {
		t.Errorf("Timestamp not preserved: got %v, want %v", event2.Timestamp, now)
	}
}

// TestWorkflowEventSequence 测试事件序列号
func TestWorkflowEventSequence(t *testing.T) {
	// 测试序列号排序
	events := []*WorkflowEvent{
		{ID: "evt-3", Seq: 3},
		{ID: "evt-1", Seq: 1},
		{ID: "evt-2", Seq: 2},
	}

	// 按序列号排序
	for i := 0; i < len(events)-1; i++ {
		for j := i + 1; j < len(events); j++ {
			if events[i].Seq > events[j].Seq {
				events[i], events[j] = events[j], events[i]
			}
		}
	}

	// 验证排序结果
	for i, event := range events {
		expectedSeq := int64(i + 1)
		if event.Seq != expectedSeq {
			t.Errorf("Event at index %d has seq %d, want %d", i, event.Seq, expectedSeq)
		}
	}
}

// TestWorkflowStateTransitions 测试状态转换
func TestWorkflowStateTransitions(t *testing.T) {
	validTransitions := map[WorkflowState][]WorkflowState{
		WorkflowStatePending:   {WorkflowStateRunning, WorkflowStateCancelled},
		WorkflowStateRunning:   {WorkflowStateWaiting, WorkflowStateCompleted, WorkflowStateFailed},
		WorkflowStateWaiting:   {WorkflowStateRunning, WorkflowStateCompleted, WorkflowStateFailed, WorkflowStateCancelled},
		WorkflowStateCompleted: {}, // 终态
		WorkflowStateFailed:    {}, // 终态
		WorkflowStateCancelled: {}, // 终态
	}

	for from, validTos := range validTransitions {
		t.Logf("State %s can transition to: %v", from, validTos)
	}
}

// TestEventDataTypes 测试事件数据类型
func TestEventDataTypes(t *testing.T) {
	event := &WorkflowEvent{
		ID:   "evt-001",
		Type: "auth.oauth_url",
		Data: map[string]interface{}{
			"oauth_url": "https://example.com/auth",
			"user_code": "ABC-123",
			"expires":   float64(300),
			"verified":  false,
		},
	}

	data, _ := json.Marshal(event)
	var event2 WorkflowEvent
	json.Unmarshal(data, &event2)

	// 验证不同类型的数据
	if event2.Data["oauth_url"].(string) != "https://example.com/auth" {
		t.Error("oauth_url mismatch")
	}
	if event2.Data["user_code"].(string) != "ABC-123" {
		t.Error("user_code mismatch")
	}
	if event2.Data["expires"].(float64) != 300 {
		t.Error("expires mismatch")
	}
	if event2.Data["verified"].(bool) != false {
		t.Error("verified mismatch")
	}
}

// TestEventBusInterfaceCompliance 测试 EventBus 接口实现
func TestEventBusInterfaceCompliance(t *testing.T) {
	// 验证 EtcdEventBus 实现了 EventBus 接口
	var _ EventBus = (*EtcdEventBus)(nil)
	t.Log("EtcdEventBus implements EventBus interface")
}

// TestSeqKeyParsing 测试从 key 解析序列号
func TestSeqKeyParsing(t *testing.T) {
	tests := []struct {
		key         string
		expectedSeq int64
	}{
		{"/agents/events/auth/task-001/000001", 1},
		{"/agents/events/auth/task-001/000010", 10},
		{"/agents/events/auth/task-001/000100", 100},
		{"/agents/events/auth/task-001/999999", 999999},
	}

	for _, tt := range tests {
		// 解析 key 中的序列号
		parts := []byte(tt.key)
		seqStr := string(parts[len(parts)-6:])

		var seq int64
		for _, c := range seqStr {
			seq = seq*10 + int64(c-'0')
		}

		if seq != tt.expectedSeq {
			t.Errorf("Seq parsing failed for %s: got %d, want %d", tt.key, seq, tt.expectedSeq)
		}
	}
}

// TestAuthEventTypes 测试认证相关事件类型
func TestAuthEventTypes(t *testing.T) {
	authEventTypes := []string{
		"auth.started",
		"auth.running",
		"auth.waiting_oauth",
		"auth.oauth_url",
		"auth.success",
		"auth.failed",
		"auth.timeout",
		"auth.cancelled",
	}

	for _, eventType := range authEventTypes {
		event := &WorkflowEvent{
			ID:         "evt-001",
			WorkflowID: "task-001",
			Type:       eventType,
			Timestamp:  time.Now(),
		}

		data, err := json.Marshal(event)
		if err != nil {
			t.Errorf("Failed to marshal event with type %s: %v", eventType, err)
			continue
		}

		var event2 WorkflowEvent
		if err := json.Unmarshal(data, &event2); err != nil {
			t.Errorf("Failed to unmarshal event with type %s: %v", eventType, err)
			continue
		}

		if event2.Type != eventType {
			t.Errorf("Event type mismatch: got %s, want %s", event2.Type, eventType)
		}
	}
}

// BenchmarkEventSerialization 基准测试事件序列化
func BenchmarkEventSerialization(b *testing.B) {
	event := &WorkflowEvent{
		ID:         "evt-001",
		WorkflowID: "wf-001",
		Type:       "auth.oauth_url",
		Seq:        1,
		Data: map[string]interface{}{
			"oauth_url": "https://example.com/auth",
			"user_code": "ABC-123",
		},
		ProducerID: "node-001",
		Timestamp:  time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _ := json.Marshal(event)
		var event2 WorkflowEvent
		json.Unmarshal(data, &event2)
	}
}

// BenchmarkStateDataSerialization 基准测试状态数据序列化
func BenchmarkStateDataSerialization(b *testing.B) {
	state := &WorkflowStateData{
		ID:    "wf-001",
		Type:  "auth",
		State: WorkflowStateRunning,
		Data: map[string]interface{}{
			"step":    "oauth_pending",
			"message": "Waiting for user authorization",
		},
		UpdatedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _ := json.Marshal(state)
		var state2 WorkflowStateData
		json.Unmarshal(data, &state2)
	}
}

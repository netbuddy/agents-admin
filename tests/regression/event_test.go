package regression

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// ============================================================================
// Event 事件流回归测试
// ============================================================================

// TestEvent_Post 测试事件上报
func TestEvent_Post(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 创建测试 Task 和 Run
	w := makeRequestWithString("POST", "/api/v1/tasks", `{"name":"Event Post Test","prompt":"test","type":"general"}`)
	taskResp := parseJSONResponse(w)
	taskID := taskResp["id"].(string)
	defer testStore.DeleteTask(ctx, taskID)

	w = makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
	runResp := parseJSONResponse(w)
	runID := runResp["id"].(string)

	t.Run("上报单个事件", func(t *testing.T) {
		now := time.Now().Format(time.RFC3339Nano)
		body := `{"events":[{"seq":1,"type":"run_started","timestamp":"` + now + `","payload":{"node":"test-node"}}]}`
		w := makeRequestWithString("POST", "/api/v1/runs/"+runID+"/events", body)

		if w.Code != http.StatusCreated {
			t.Errorf("Post event status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
		}
	})

	t.Run("上报多个事件", func(t *testing.T) {
		now := time.Now().Format(time.RFC3339Nano)
		body := `{"events":[
			{"seq":2,"type":"message","timestamp":"` + now + `","payload":{"role":"assistant","content":"Starting..."}},
			{"seq":3,"type":"tool_use_start","timestamp":"` + now + `","payload":{"tool_id":"t1","tool_name":"Read"}},
			{"seq":4,"type":"tool_result","timestamp":"` + now + `","payload":{"tool_id":"t1","result":"file content"}}
		]}`
		w := makeRequestWithString("POST", "/api/v1/runs/"+runID+"/events", body)

		if w.Code != http.StatusCreated {
			t.Errorf("Post multiple events status = %d", w.Code)
		}

		resp := parseJSONResponse(w)
		if created, ok := resp["created"].(float64); ok {
			if int(created) != 3 {
				t.Errorf("Created = %d, want 3", int(created))
			}
		}
	})

	t.Run("上报不同类型事件", func(t *testing.T) {
		eventTypes := []string{
			"run_started", "message", "tool_use_start", "tool_result",
			"file_read", "file_write", "command", "command_output",
			"thinking", "progress", "error", "run_completed",
		}

		now := time.Now().Format(time.RFC3339Nano)
		for i, eventType := range eventTypes {
			body := `{"events":[{"seq":` + string(rune('0'+i+10)) + `,"type":"` + eventType + `","timestamp":"` + now + `","payload":{}}]}`
			w := makeRequestWithString("POST", "/api/v1/runs/"+runID+"/events", body)
			if w.Code != http.StatusCreated {
				t.Logf("Event type %s: status %d", eventType, w.Code)
			}
		}
	})

	t.Run("上报空事件列表", func(t *testing.T) {
		body := `{"events":[]}`
		w := makeRequestWithString("POST", "/api/v1/runs/"+runID+"/events", body)
		// 空列表应该成功
		if w.Code != http.StatusCreated && w.Code != http.StatusOK {
			t.Errorf("Post empty events status = %d", w.Code)
		}
	})

	t.Run("上报到不存在的 Run", func(t *testing.T) {
		now := time.Now().Format(time.RFC3339Nano)
		body := `{"events":[{"seq":1,"type":"run_started","timestamp":"` + now + `","payload":{}}]}`
		w := makeRequestWithString("POST", "/api/v1/runs/nonexistent-run/events", body)
		// 可能返回 404（not found）、400（bad request）或 500（internal error）
		if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
			t.Errorf("Post to nonexistent run status = %d", w.Code)
		}
	})
}

// TestEvent_Get 测试获取事件
func TestEvent_Get(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 创建测试 Task 和 Run
	w := makeRequestWithString("POST", "/api/v1/tasks", `{"name":"Event Get Test","prompt":"test","type":"general"}`)
	taskResp := parseJSONResponse(w)
	taskID := taskResp["id"].(string)
	defer testStore.DeleteTask(ctx, taskID)

	w = makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
	runResp := parseJSONResponse(w)
	runID := runResp["id"].(string)

	// 上报一些事件
	now := time.Now().Format(time.RFC3339Nano)
	body := `{"events":[
		{"seq":1,"type":"run_started","timestamp":"` + now + `","payload":{"node":"test"}},
		{"seq":2,"type":"message","timestamp":"` + now + `","payload":{"content":"hello"}},
		{"seq":3,"type":"message","timestamp":"` + now + `","payload":{"content":"world"}},
		{"seq":4,"type":"tool_use_start","timestamp":"` + now + `","payload":{"tool":"read"}},
		{"seq":5,"type":"run_completed","timestamp":"` + now + `","payload":{"status":"done"}}
	]}`
	makeRequestWithString("POST", "/api/v1/runs/"+runID+"/events", body)

	t.Run("获取所有事件", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/runs/"+runID+"/events", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("Get events status = %d", w.Code)
		}

		resp := parseJSONResponse(w)
		if resp["events"] == nil {
			t.Error("Events list not returned")
		}
		if count, ok := resp["count"].(float64); ok {
			if int(count) < 5 {
				t.Errorf("Event count = %d, want >= 5", int(count))
			}
		}
	})

	t.Run("增量获取事件 from_seq=1", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/runs/"+runID+"/events?from_seq=1", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("Get events from_seq status = %d", w.Code)
		}

		resp := parseJSONResponse(w)
		events := resp["events"].([]interface{})
		for _, e := range events {
			event := e.(map[string]interface{})
			seq := int(event["seq"].(float64))
			if seq <= 1 {
				t.Errorf("Got event with seq %d, should be > 1", seq)
			}
		}
	})

	t.Run("增量获取事件 from_seq=3", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/runs/"+runID+"/events?from_seq=3", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("Get events from_seq=3 status = %d", w.Code)
		}

		resp := parseJSONResponse(w)
		events := resp["events"].([]interface{})
		if len(events) < 2 {
			t.Errorf("Expected at least 2 events after seq 3, got %d", len(events))
		}
	})

	t.Run("获取不存在 Run 的事件", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/runs/nonexistent-run/events", nil)
		// 应该返回空列表或 404
		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Errorf("Get events for nonexistent run status = %d", w.Code)
		}
	})
}

// TestEvent_Ordering 测试事件顺序
func TestEvent_Ordering(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 创建测试 Task 和 Run
	w := makeRequestWithString("POST", "/api/v1/tasks", `{"name":"Event Ordering Test","prompt":"test","type":"general"}`)
	taskResp := parseJSONResponse(w)
	taskID := taskResp["id"].(string)
	defer testStore.DeleteTask(ctx, taskID)

	w = makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
	runResp := parseJSONResponse(w)
	runID := runResp["id"].(string)

	// 上报乱序事件
	now := time.Now().Format(time.RFC3339Nano)
	body := `{"events":[
		{"seq":5,"type":"message","timestamp":"` + now + `","payload":{"content":"five"}},
		{"seq":1,"type":"run_started","timestamp":"` + now + `","payload":{}},
		{"seq":3,"type":"message","timestamp":"` + now + `","payload":{"content":"three"}},
		{"seq":2,"type":"message","timestamp":"` + now + `","payload":{"content":"two"}},
		{"seq":4,"type":"message","timestamp":"` + now + `","payload":{"content":"four"}}
	]}`
	makeRequestWithString("POST", "/api/v1/runs/"+runID+"/events", body)

	t.Run("获取的事件按 seq 排序", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/runs/"+runID+"/events", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("Get events status = %d", w.Code)
		}

		resp := parseJSONResponse(w)
		events := resp["events"].([]interface{})

		prevSeq := 0
		for _, e := range events {
			event := e.(map[string]interface{})
			seq := int(event["seq"].(float64))
			if seq < prevSeq {
				t.Errorf("Events not in order: %d came after %d", seq, prevSeq)
			}
			prevSeq = seq
		}
	})
}

// TestEvent_LargePayload 测试大事件载荷
func TestEvent_LargePayload(t *testing.T) {
	skipIfNoDatabase(t)
	ctx := context.Background()

	// 创建测试 Task 和 Run
	w := makeRequestWithString("POST", "/api/v1/tasks", `{"name":"Large Payload Test","prompt":"test","type":"general"}`)
	taskResp := parseJSONResponse(w)
	taskID := taskResp["id"].(string)
	defer testStore.DeleteTask(ctx, taskID)

	w = makeRequest("POST", "/api/v1/tasks/"+taskID+"/runs", nil)
	runResp := parseJSONResponse(w)
	runID := runResp["id"].(string)

	t.Run("大文本内容", func(t *testing.T) {
		// 生成 10KB 的内容
		largeContent := make([]byte, 10*1024)
		for i := range largeContent {
			largeContent[i] = 'a'
		}

		now := time.Now().Format(time.RFC3339Nano)
		body := `{"events":[{"seq":1,"type":"message","timestamp":"` + now + `","payload":{"content":"` + string(largeContent) + `"}}]}`
		w := makeRequestWithString("POST", "/api/v1/runs/"+runID+"/events", body)

		// 应该成功处理
		t.Logf("Large payload status: %d", w.Code)
	})
}

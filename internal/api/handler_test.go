// Package api 单元测试
//
// 本文件包含 API 处理器的单元测试，主要测试：
//   - 通用工具函数（writeJSON、writeError、generateID）
//   - 请求体解析和验证
//   - HTTP 响应格式
//
// 注意：涉及数据库操作的测试在集成测试中进行。
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ============================================================================
// 通用函数测试
// ============================================================================

// TestHealthEndpoint 测试健康检查接口
func TestHealthEndpoint(t *testing.T) {
	handler := &Handler{}

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("status = %v, want ok", resp["status"])
	}
}

// TestWriteJSON 测试 JSON 响应写入
func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		data           interface{}
		wantStatusCode int
		wantKey        string
		wantValue      string
	}{
		{
			name:           "正常响应",
			status:         http.StatusOK,
			data:           map[string]string{"message": "hello"},
			wantStatusCode: http.StatusOK,
			wantKey:        "message",
			wantValue:      "hello",
		},
		{
			name:           "创建成功响应",
			status:         http.StatusCreated,
			data:           map[string]string{"id": "task-123"},
			wantStatusCode: http.StatusCreated,
			wantKey:        "id",
			wantValue:      "task-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeJSON(w, tt.status, tt.data)

			if w.Code != tt.wantStatusCode {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatusCode)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %v, want application/json", contentType)
			}

			var resp map[string]string
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if resp[tt.wantKey] != tt.wantValue {
				t.Errorf("%s = %v, want %v", tt.wantKey, resp[tt.wantKey], tt.wantValue)
			}
		})
	}
}

// TestWriteError 测试错误响应写入
func TestWriteError(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		message    string
		wantStatus int
	}{
		{
			name:       "400 Bad Request",
			status:     http.StatusBadRequest,
			message:    "invalid input",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "404 Not Found",
			status:     http.StatusNotFound,
			message:    "resource not found",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "500 Internal Server Error",
			status:     http.StatusInternalServerError,
			message:    "database error",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeError(w, tt.status, tt.message)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			var resp map[string]string
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if resp["error"] != tt.message {
				t.Errorf("error = %v, want %v", resp["error"], tt.message)
			}
		})
	}
}

// TestGenerateID 测试 ID 生成
func TestGenerateID(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
	}{
		{"task prefix", "task"},
		{"run prefix", "run"},
		{"node prefix", "node"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id1 := generateID(tt.prefix)
			id2 := generateID(tt.prefix)

			// 唯一性
			if id1 == id2 {
				t.Error("Expected unique IDs")
			}

			// 前缀检查
			expectedPrefix := tt.prefix + "-"
			if !strings.HasPrefix(id1, expectedPrefix) {
				t.Errorf("ID should start with '%s': got %v", expectedPrefix, id1)
			}

			// 长度检查（prefix + "-" + 12位hex = len(prefix) + 1 + 12）
			expectedLen := len(tt.prefix) + 1 + 12
			if len(id1) != expectedLen {
				t.Errorf("ID length = %d, want %d", len(id1), expectedLen)
			}
		})
	}
}

// ============================================================================
// 请求结构体解析测试
// ============================================================================

// TestCreateTaskRequest_Parsing 测试创建任务请求解析
func TestCreateTaskRequest_Parsing(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantName  string
		wantSpec  bool
		wantError bool
	}{
		{
			name:     "完整请求",
			body:     `{"name":"Test Task","spec":{"prompt":"hello","agent":{"type":"gemini"}}}`,
			wantName: "Test Task",
			wantSpec: true,
		},
		{
			name:     "仅名称",
			body:     `{"name":"Simple Task"}`,
			wantName: "Simple Task",
			wantSpec: false,
		},
		{
			name:      "无效 JSON",
			body:      `{invalid}`,
			wantError: true,
		},
		{
			name:     "空名称",
			body:     `{"name":"","spec":{}}`,
			wantName: "",
			wantSpec: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req CreateTaskRequest
			err := json.NewDecoder(bytes.NewBufferString(tt.body)).Decode(&req)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if req.Name != tt.wantName {
				t.Errorf("Name = %v, want %v", req.Name, tt.wantName)
			}

			hasSpec := req.Spec != nil
			if hasSpec != tt.wantSpec {
				t.Errorf("HasSpec = %v, want %v", hasSpec, tt.wantSpec)
			}
		})
	}
}

// TestUpdateRunRequest_Parsing 测试更新 Run 请求解析
func TestUpdateRunRequest_Parsing(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus string
		wantError  bool
	}{
		{
			name:       "更新为 done",
			body:       `{"status":"done"}`,
			wantStatus: "done",
		},
		{
			name:       "更新为 failed",
			body:       `{"status":"failed"}`,
			wantStatus: "failed",
		},
		{
			name:       "更新为 running",
			body:       `{"status":"running"}`,
			wantStatus: "running",
		},
		{
			name:      "无效 JSON",
			body:      `{invalid}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req UpdateRunRequest
			err := json.NewDecoder(bytes.NewBufferString(tt.body)).Decode(&req)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if req.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", req.Status, tt.wantStatus)
			}
		})
	}
}

// TestPostEventsRequest_Parsing 测试事件上报请求解析
func TestPostEventsRequest_Parsing(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantCount int
		wantType  string
		wantSeq   int
		wantError bool
	}{
		{
			name: "单个事件",
			body: `{
				"events": [
					{"seq": 1, "type": "message", "timestamp": "2024-01-01T00:00:00Z", "payload": {"content": "hello"}}
				]
			}`,
			wantCount: 1,
			wantType:  "message",
			wantSeq:   1,
		},
		{
			name: "多个事件",
			body: `{
				"events": [
					{"seq": 1, "type": "run_started", "timestamp": "2024-01-01T00:00:00Z", "payload": {}},
					{"seq": 2, "type": "message", "timestamp": "2024-01-01T00:00:01Z", "payload": {"content": "test"}},
					{"seq": 3, "type": "run_completed", "timestamp": "2024-01-01T00:00:02Z", "payload": {}}
				]
			}`,
			wantCount: 3,
			wantType:  "run_started",
			wantSeq:   1,
		},
		{
			name:      "空事件列表",
			body:      `{"events": []}`,
			wantCount: 0,
		},
		{
			name:      "无效 JSON",
			body:      `{invalid}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req PostEventsRequest
			err := json.NewDecoder(bytes.NewBufferString(tt.body)).Decode(&req)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(req.Events) != tt.wantCount {
				t.Errorf("Event count = %d, want %d", len(req.Events), tt.wantCount)
			}

			if tt.wantCount > 0 {
				if req.Events[0].Type != tt.wantType {
					t.Errorf("Type = %v, want %v", req.Events[0].Type, tt.wantType)
				}
				if req.Events[0].Seq != tt.wantSeq {
					t.Errorf("Seq = %v, want %v", req.Events[0].Seq, tt.wantSeq)
				}
			}
		})
	}
}

// TestHeartbeatRequest_Parsing 测试节点心跳请求解析
func TestHeartbeatRequest_Parsing(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantNodeID string
		wantStatus string
		wantLabels map[string]string
		wantError  bool
	}{
		{
			name: "完整心跳",
			body: `{
				"node_id": "node-001",
				"status": "online",
				"labels": {"os": "linux", "gpu": "true"},
				"capacity": {"max_concurrent": 4}
			}`,
			wantNodeID: "node-001",
			wantStatus: "online",
			wantLabels: map[string]string{"os": "linux", "gpu": "true"},
		},
		{
			name:       "最小心跳",
			body:       `{"node_id": "node-002", "status": "online"}`,
			wantNodeID: "node-002",
			wantStatus: "online",
		},
		{
			name:       "排空状态",
			body:       `{"node_id": "node-003", "status": "draining"}`,
			wantNodeID: "node-003",
			wantStatus: "draining",
		},
		{
			name:      "无效 JSON",
			body:      `{invalid}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req HeartbeatRequest
			err := json.NewDecoder(bytes.NewBufferString(tt.body)).Decode(&req)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if req.NodeID != tt.wantNodeID {
				t.Errorf("NodeID = %v, want %v", req.NodeID, tt.wantNodeID)
			}

			if req.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", req.Status, tt.wantStatus)
			}

			for k, v := range tt.wantLabels {
				if req.Labels[k] != v {
					t.Errorf("Labels[%s] = %v, want %v", k, req.Labels[k], v)
				}
			}
		})
	}
}

// TestUpdateNodeRequest_Parsing 测试更新节点请求解析
func TestUpdateNodeRequest_Parsing(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus *string
		wantLabels bool
		wantError  bool
	}{
		{
			name:       "仅更新状态",
			body:       `{"status": "draining"}`,
			wantStatus: strPtr("draining"),
		},
		{
			name:       "仅更新标签",
			body:       `{"labels": {"gpu": "false"}}`,
			wantLabels: true,
		},
		{
			name:       "同时更新",
			body:       `{"status": "maintenance", "labels": {"gpu": "false"}}`,
			wantStatus: strPtr("maintenance"),
			wantLabels: true,
		},
		{
			name:      "无效 JSON",
			body:      `{invalid}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req UpdateNodeRequest
			err := json.NewDecoder(bytes.NewBufferString(tt.body)).Decode(&req)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.wantStatus != nil {
				if req.Status == nil || *req.Status != *tt.wantStatus {
					t.Errorf("Status = %v, want %v", req.Status, *tt.wantStatus)
				}
			}

			hasLabels := req.Labels != nil
			if hasLabels != tt.wantLabels {
				t.Errorf("HasLabels = %v, want %v", hasLabels, tt.wantLabels)
			}
		})
	}
}

// ============================================================================
// 辅助函数
// ============================================================================

// strPtr 返回字符串指针
func strPtr(s string) *string {
	return &s
}

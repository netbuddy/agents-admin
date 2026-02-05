package task

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// TestCreateRequest_Parsing 测试创建任务请求解析
func TestCreateRequest_Parsing(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantName   string
		wantPrompt string
		wantError  bool
	}{
		{
			name: "完整请求",
			body: `{
				"name": "测试任务",
				"prompt": "执行测试"
			}`,
			wantName:   "测试任务",
			wantPrompt: "执行测试",
			wantError:  false,
		},
		{
			name: "带提示词",
			body: `{
				"name": "代码审查",
				"prompt": "请审查代码",
				"prompt_description": "代码审查任务"
			}`,
			wantName:   "代码审查",
			wantPrompt: "请审查代码",
			wantError:  false,
		},
		{
			name:      "无效 JSON",
			body:      `{invalid json}`,
			wantError: true,
		},
		{
			name: "空名称",
			body: `{
				"name": "",
				"prompt": "执行测试"
			}`,
			wantName:   "",
			wantPrompt: "执行测试",
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req CreateRequest
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
				t.Errorf("Name = %q, want %q", req.Name, tt.wantName)
			}
			if req.Prompt != tt.wantPrompt {
				t.Errorf("Prompt = %q, want %q", req.Prompt, tt.wantPrompt)
			}
		})
	}
}

// TestGenerateID 测试 ID 生成
func TestGenerateID(t *testing.T) {
	tests := []struct {
		prefix string
	}{
		{"task"},
		{"run"},
		{"node"},
	}

	for _, tt := range tests {
		t.Run(tt.prefix+"_prefix", func(t *testing.T) {
			id := generateID(tt.prefix)

			if !strings.HasPrefix(id, tt.prefix+"-") {
				t.Errorf("ID %q should start with %q-", id, tt.prefix)
			}

			// 格式：prefix-xxxxxxxxxxxx（prefix + 1 + 12 = 13+ 字符）
			expectedLen := len(tt.prefix) + 1 + 12
			if len(id) != expectedLen {
				t.Errorf("ID length = %d, want %d", len(id), expectedLen)
			}
		})
	}

	// 测试唯一性
	t.Run("uniqueness", func(t *testing.T) {
		ids := make(map[string]bool)
		for i := 0; i < 100; i++ {
			id := generateID("test")
			if ids[id] {
				t.Errorf("Duplicate ID generated: %s", id)
			}
			ids[id] = true
		}
	})
}

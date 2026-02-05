package claude

import (
	"context"
	"testing"

	"agents-admin/internal/nodemanager/adapter"
)

func TestClaudeAdapterName(t *testing.T) {
	a := New()
	if a.Name() != "claude-v1" {
		t.Errorf("Name() = %v, want claude-v1", a.Name())
	}
}

func TestClaudeAdapterValidate(t *testing.T) {
	a := New()

	tests := []struct {
		name    string
		agent   *adapter.AgentConfig
		wantErr bool
	}{
		{
			name:    "valid agent",
			agent:   &adapter.AgentConfig{Type: "claude"},
			wantErr: false,
		},
		{
			name:    "wrong agent type",
			agent:   &adapter.AgentConfig{Type: "gemini"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := a.Validate(tt.agent)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClaudeAdapterBuildCommand(t *testing.T) {
	a := New()
	spec := &adapter.TaskSpec{
		ID:     "task-123",
		Prompt: "Fix the bug",
		Security: adapter.SecurityConfig{
			Policy: adapter.SecurityPolicyStrict,
		},
	}
	agent := &adapter.AgentConfig{
		Type:  "claude",
		Model: "claude-sonnet-4-20250514",
		Parameters: map[string]interface{}{
			"allowed_tools": []interface{}{"Read", "Write", "Bash"},
			"sandbox":       true,
		},
	}

	cfg, err := a.BuildCommand(context.Background(), spec, agent)
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	if len(cfg.Command) == 0 {
		t.Error("Expected non-empty command")
	}

	if cfg.Command[0] != "claude" {
		t.Errorf("Command[0] = %v, want claude", cfg.Command[0])
	}

	// 验证基本参数存在
	foundOutputFormat := false
	for _, arg := range cfg.Args {
		if arg == "--output-format" {
			foundOutputFormat = true
			break
		}
	}
	if !foundOutputFormat {
		t.Error("Expected --output-format in args")
	}
}

func TestClaudeAdapterParseEvent(t *testing.T) {
	a := New()

	tests := []struct {
		name     string
		line     string
		wantType adapter.EventType
		wantErr  bool
		wantNil  bool
	}{
		{
			name:     "assistant message",
			line:     `{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}]}}`,
			wantType: adapter.EventMessage,
			wantErr:  false,
		},
		{
			name:     "tool_use event",
			line:     `{"type":"tool_use","name":"Read","input":{"path":"test.go"}}`,
			wantType: adapter.EventToolUseStart,
			wantErr:  false,
		},
		{
			name:     "result event",
			line:     `{"type":"result","result":"success"}`,
			wantType: adapter.EventRunCompleted,
			wantErr:  false,
		},
		{
			name:    "invalid json",
			line:    `{invalid}`,
			wantNil: true, // ParseEvent 返回 (nil, nil) 而非错误
		},
		{
			name:    "empty line",
			line:    "",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := a.ParseEvent(tt.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantNil && event != nil {
				t.Error("Expected nil event")
				return
			}
			if !tt.wantErr && !tt.wantNil && event.Type != tt.wantType {
				t.Errorf("ParseEvent() type = %v, want %v", event.Type, tt.wantType)
			}
		})
	}
}

func TestClaudeAdapterCollectArtifacts(t *testing.T) {
	a := New()

	artifacts, err := a.CollectArtifacts(context.Background(), "/tmp/nonexistent")
	if err != nil {
		t.Fatalf("CollectArtifacts() error = %v", err)
	}

	if artifacts == nil {
		t.Error("Expected non-nil artifacts")
	}
}

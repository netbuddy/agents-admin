package gemini

import (
	"context"
	"encoding/json"
	"testing"

	"agents-admin/internal/nodemanager/adapter"
)

func TestGeminiAdapterName(t *testing.T) {
	a := New()
	if a.Name() != "gemini-v1" {
		t.Errorf("Name() = %v, want gemini-v1", a.Name())
	}
}

func TestGeminiAdapterValidate(t *testing.T) {
	a := New()

	tests := []struct {
		name    string
		agent   *adapter.AgentConfig
		wantErr bool
	}{
		{
			name:    "valid agent",
			agent:   &adapter.AgentConfig{Type: "gemini"},
			wantErr: false,
		},
		{
			name:    "wrong agent type",
			agent:   &adapter.AgentConfig{Type: "claude"},
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

func TestGeminiAdapterBuildCommand(t *testing.T) {
	a := New()
	spec := &adapter.TaskSpec{
		ID:     "task-123",
		Prompt: "Fix the bug",
	}
	agent := &adapter.AgentConfig{
		Type:  "gemini",
		Model: "gemini-2.5-pro",
		Parameters: map[string]interface{}{
			"sandbox": true,
		},
	}

	cfg, err := a.BuildCommand(context.Background(), spec, agent)
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	if len(cfg.Command) == 0 {
		t.Error("Expected non-empty command")
	}

	if cfg.Command[0] != "gemini" {
		t.Errorf("Command[0] = %v, want gemini", cfg.Command[0])
	}

	foundPrompt := false
	for i, arg := range cfg.Args {
		if arg == "-p" && i+1 < len(cfg.Args) && cfg.Args[i+1] == "Fix the bug" {
			foundPrompt = true
			break
		}
	}
	if !foundPrompt {
		t.Error("Expected prompt in args")
	}
}

func TestGeminiAdapterParseEvent(t *testing.T) {
	a := New()

	tests := []struct {
		name     string
		line     string
		wantType adapter.EventType
		wantErr  bool
		wantNil  bool
	}{
		{
			name:     "message event",
			line:     `{"type":"message","content":"hello"}`,
			wantType: adapter.EventMessage,
			wantErr:  false,
		},
		{
			name:     "tool_call event",
			line:     `{"type":"tool_call","tool":"read_file"}`,
			wantType: adapter.EventToolUseStart,
			wantErr:  false,
		},
		{
			name:     "tool_result event",
			line:     `{"type":"tool_result","result":"ok"}`,
			wantType: adapter.EventToolResult,
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

func TestGeminiAdapterCollectArtifacts(t *testing.T) {
	a := New()

	artifacts, err := a.CollectArtifacts(context.Background(), "/tmp/nonexistent")
	if err != nil {
		t.Fatalf("CollectArtifacts() error = %v", err)
	}

	if artifacts == nil {
		t.Error("Expected non-nil artifacts")
	}
}

func TestGeminiEventMapping(t *testing.T) {
	a := New()

	eventJSON := `{"type":"message","role":"assistant","content":"I will help you"}`
	event, err := a.ParseEvent(eventJSON)
	if err != nil {
		t.Fatalf("ParseEvent() error = %v", err)
	}

	if event.Type != adapter.EventMessage {
		t.Errorf("Type = %v, want message", event.Type)
	}

	payloadBytes, _ := json.Marshal(event.Payload)
	var payload map[string]interface{}
	json.Unmarshal(payloadBytes, &payload)

	if payload["content"] != "I will help you" {
		t.Errorf("content = %v, want 'I will help you'", payload["content"])
	}
}

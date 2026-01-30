package gemini

import (
	"context"
	"encoding/json"
	"testing"

	"agents-admin/pkg/driver"
)

func TestGeminiDriverName(t *testing.T) {
	d := New()
	if d.Name() != "gemini-v1" {
		t.Errorf("Name() = %v, want gemini-v1", d.Name())
	}
}

func TestGeminiDriverValidate(t *testing.T) {
	d := New()

	tests := []struct {
		name    string
		agent   *driver.AgentConfig
		wantErr bool
	}{
		{
			name:    "valid agent",
			agent:   &driver.AgentConfig{Type: "gemini"},
			wantErr: false,
		},
		{
			name:    "wrong agent type",
			agent:   &driver.AgentConfig{Type: "claude"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := d.Validate(tt.agent)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGeminiDriverBuildCommand(t *testing.T) {
	d := New()
	spec := &driver.TaskSpec{
		ID:     "task-123",
		Prompt: "Fix the bug",
	}
	agent := &driver.AgentConfig{
		Type:  "gemini",
		Model: "gemini-2.5-pro",
		Parameters: map[string]interface{}{
			"sandbox": true,
		},
	}

	cfg, err := d.BuildCommand(context.Background(), spec, agent)
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

func TestGeminiDriverParseEvent(t *testing.T) {
	d := New()

	tests := []struct {
		name     string
		line     string
		wantType driver.EventType
		wantErr  bool
		wantNil  bool
	}{
		{
			name:     "message event",
			line:     `{"type":"message","content":"hello"}`,
			wantType: driver.EventMessage,
			wantErr:  false,
		},
		{
			name:     "tool_call event",
			line:     `{"type":"tool_call","tool":"read_file"}`,
			wantType: driver.EventToolUseStart,
			wantErr:  false,
		},
		{
			name:     "tool_result event",
			line:     `{"type":"tool_result","result":"ok"}`,
			wantType: driver.EventToolResult,
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
			event, err := d.ParseEvent(tt.line)
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

func TestGeminiDriverCollectArtifacts(t *testing.T) {
	d := New()

	artifacts, err := d.CollectArtifacts(context.Background(), "/tmp/nonexistent")
	if err != nil {
		t.Fatalf("CollectArtifacts() error = %v", err)
	}

	if artifacts == nil {
		t.Error("Expected non-nil artifacts")
	}
}

func TestGeminiEventMapping(t *testing.T) {
	d := New()

	eventJSON := `{"type":"message","role":"assistant","content":"I will help you"}`
	event, err := d.ParseEvent(eventJSON)
	if err != nil {
		t.Fatalf("ParseEvent() error = %v", err)
	}

	if event.Type != driver.EventMessage {
		t.Errorf("Type = %v, want message", event.Type)
	}

	payloadBytes, _ := json.Marshal(event.Payload)
	var payload map[string]interface{}
	json.Unmarshal(payloadBytes, &payload)

	if payload["content"] != "I will help you" {
		t.Errorf("content = %v, want 'I will help you'", payload["content"])
	}
}

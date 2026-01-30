package qwencode

import (
	"context"
	"testing"

	"agents-admin/pkg/driver"
)

func TestDriver_Name(t *testing.T) {
	d := New()
	if d.Name() != "qwencode-v1" {
		t.Errorf("Name() = %v, want qwencode-v1", d.Name())
	}
}

func TestDriver_Validate(t *testing.T) {
	d := New()

	tests := []struct {
		name    string
		agent   *driver.AgentConfig
		wantErr bool
	}{
		{
			name:    "valid qwencode type",
			agent:   &driver.AgentConfig{Type: "qwencode"},
			wantErr: false,
		},
		{
			name:    "valid qwen-code type",
			agent:   &driver.AgentConfig{Type: "qwen-code"},
			wantErr: false,
		},
		{
			name:    "valid qwen type",
			agent:   &driver.AgentConfig{Type: "qwen"},
			wantErr: false,
		},
		{
			name:    "invalid type",
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

func TestDriver_BuildCommand(t *testing.T) {
	d := New()
	ctx := context.Background()

	spec := &driver.TaskSpec{
		Prompt: "Write a hello world program",
	}

	tests := []struct {
		name       string
		agent      *driver.AgentConfig
		wantYolo   bool
		wantModel  string
		wantAPIKey bool
	}{
		{
			name: "basic config",
			agent: &driver.AgentConfig{
				Type:       "qwencode",
				Parameters: map[string]interface{}{},
			},
			wantYolo:  false,
			wantModel: "qwen3-coder",
		},
		{
			name: "with yolo mode",
			agent: &driver.AgentConfig{
				Type: "qwencode",
				Parameters: map[string]interface{}{
					"yolo": true,
				},
			},
			wantYolo: true,
		},
		{
			name: "with custom model",
			agent: &driver.AgentConfig{
				Type: "qwencode",
				Parameters: map[string]interface{}{
					"model": "gpt-4o",
				},
			},
			wantModel: "gpt-4o",
		},
		{
			name: "with api key",
			agent: &driver.AgentConfig{
				Type: "qwencode",
				Parameters: map[string]interface{}{
					"api_key":  "sk-test",
					"base_url": "https://api.openai.com/v1",
				},
			},
			wantAPIKey: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := d.BuildCommand(ctx, spec, tt.agent)
			if err != nil {
				t.Fatalf("BuildCommand() error = %v", err)
			}

			if cfg.Image != "runners/qwencode:latest" {
				t.Errorf("Image = %v, want runners/qwencode:latest", cfg.Image)
			}

			if cfg.Command[0] != "qwen" {
				t.Errorf("Command = %v, want [qwen]", cfg.Command)
			}

			// 检查 -p 参数
			hasPrompt := false
			hasYolo := false
			for i, arg := range cfg.Args {
				if arg == "-p" && i+1 < len(cfg.Args) && cfg.Args[i+1] == spec.Prompt {
					hasPrompt = true
				}
				if arg == "--yolo" {
					hasYolo = true
				}
			}

			if !hasPrompt {
				t.Error("BuildCommand() missing -p argument")
			}

			if tt.wantYolo != hasYolo {
				t.Errorf("BuildCommand() yolo = %v, want %v", hasYolo, tt.wantYolo)
			}

			if tt.wantAPIKey {
				if cfg.Env["OPENAI_API_KEY"] == "" {
					t.Error("BuildCommand() missing OPENAI_API_KEY env")
				}
			}
		})
	}
}

func TestDriver_ParseEvent(t *testing.T) {
	d := New()

	tests := []struct {
		name     string
		line     string
		wantType driver.EventType
		wantNil  bool
	}{
		{
			name:     "message event",
			line:     `{"type":"message","content":"Hello"}`,
			wantType: driver.EventMessage,
		},
		{
			name:     "tool call event",
			line:     `{"type":"tool_call","tool":"read_file"}`,
			wantType: driver.EventToolUseStart,
		},
		{
			name:     "file write event",
			line:     `{"type":"file_write","path":"test.go"}`,
			wantType: driver.EventFileWrite,
		},
		{
			name:     "done event",
			line:     `{"type":"done"}`,
			wantType: driver.EventRunCompleted,
		},
		{
			name:     "thinking event",
			line:     `{"type":"thinking","content":"Let me think..."}`,
			wantType: driver.EventMessage,
		},
		{
			name:    "non-json line",
			line:    "This is not JSON",
			wantNil: true,
		},
		{
			name:    "json without type",
			line:    `{"content":"Hello"}`,
			wantNil: true,
		},
		{
			name:    "unknown event type",
			line:    `{"type":"unknown_event"}`,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := d.ParseEvent(tt.line)
			if err != nil {
				t.Fatalf("ParseEvent() error = %v", err)
			}

			if tt.wantNil {
				if event != nil {
					t.Errorf("ParseEvent() = %v, want nil", event)
				}
				return
			}

			if event == nil {
				t.Fatal("ParseEvent() = nil, want non-nil")
			}

			if event.Type != tt.wantType {
				t.Errorf("ParseEvent().Type = %v, want %v", event.Type, tt.wantType)
			}
		})
	}
}

func TestDriver_CollectArtifacts(t *testing.T) {
	d := New()
	ctx := context.Background()

	artifacts, err := d.CollectArtifacts(ctx, "/workspace")
	if err != nil {
		t.Fatalf("CollectArtifacts() error = %v", err)
	}

	expected := "/workspace/.qwen/events.jsonl"
	if artifacts.EventsFile != expected {
		t.Errorf("EventsFile = %v, want %v", artifacts.EventsFile, expected)
	}
}

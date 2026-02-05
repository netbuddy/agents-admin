// Package gemini 实现 Gemini CLI Adapter
package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"

	"agents-admin/internal/nodemanager/adapter"
)

// Adapter Gemini CLI 适配器
type Adapter struct{}

// New 创建 Gemini Adapter
func New() *Adapter {
	return &Adapter{}
}

// Name 返回适配器名称
func (a *Adapter) Name() string {
	return "gemini-v1"
}

// Validate 验证 AgentConfig
func (a *Adapter) Validate(agent *adapter.AgentConfig) error {
	if agent.Type != "gemini" {
		return fmt.Errorf("agent type mismatch: expected gemini, got %s", agent.Type)
	}
	return nil
}

// BuildCommand 构建运行命令
// ctx 用于超时控制（当前实现未使用，预留接口）
func (a *Adapter) BuildCommand(ctx context.Context, spec *adapter.TaskSpec, agent *adapter.AgentConfig) (*adapter.RunConfig, error) {
	args := []string{
		"-p", spec.Prompt,
		"--output-format", "json",
	}

	// 沙箱模式
	if sandbox, ok := agent.Parameters["sandbox"].(bool); ok && sandbox {
		args = append(args, "--sandbox")
	}

	// 最大轮次
	if maxTurns, ok := agent.Parameters["max_turns"].(float64); ok {
		args = append(args, "--max-turns", strconv.Itoa(int(maxTurns)))
	}

	return &adapter.RunConfig{
		Image:      "runners/gemini:latest",
		Command:    []string{"gemini"},
		Args:       args,
		Env:        map[string]string{},
		WorkingDir: "/workspace",
	}, nil
}

// ParseEvent 解析事件
func (a *Adapter) ParseEvent(line string) (*adapter.CanonicalEvent, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return nil, nil // 非 JSON 行，忽略
	}

	eventType, _ := raw["type"].(string)
	if eventType == "" {
		return nil, nil
	}

	// 映射 Gemini 事件类型到平台事件类型
	canonicalType := mapEventType(eventType)
	if canonicalType == "" {
		return nil, nil
	}

	return &adapter.CanonicalEvent{
		Type:    canonicalType,
		Payload: raw,
	}, nil
}

func mapEventType(geminiType string) adapter.EventType {
	mapping := map[string]adapter.EventType{
		"message":      adapter.EventMessage,
		"tool_call":    adapter.EventToolUseStart,
		"tool_result":  adapter.EventToolResult,
		"command":      adapter.EventCommand,
		"command_done": adapter.EventCommandOutput,
		"file_read":    adapter.EventFileRead,
		"file_write":   adapter.EventFileWrite,
		"error":        adapter.EventError,
		"done":         adapter.EventRunCompleted,
	}
	return mapping[geminiType]
}

// CollectArtifacts 收集产物
func (a *Adapter) CollectArtifacts(ctx context.Context, workspaceDir string) (*adapter.Artifacts, error) {
	return &adapter.Artifacts{
		EventsFile: filepath.Join(workspaceDir, ".agent", "events.jsonl"),
	}, nil
}

// Package gemini 实现 Gemini CLI Driver
package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"

	"agents-admin/pkg/driver"
)

// Driver Gemini CLI 驱动
type Driver struct{}

// New 创建 Gemini Driver
func New() *Driver {
	return &Driver{}
}

// Name 返回驱动名称
func (d *Driver) Name() string {
	return "gemini-v1"
}

// Validate 验证 AgentConfig
func (d *Driver) Validate(agent *driver.AgentConfig) error {
	if agent.Type != "gemini" {
		return fmt.Errorf("agent type mismatch: expected gemini, got %s", agent.Type)
	}
	return nil
}

// BuildCommand 构建运行命令
// ctx 用于超时控制（当前实现未使用，预留接口）
func (d *Driver) BuildCommand(ctx context.Context, spec *driver.TaskSpec, agent *driver.AgentConfig) (*driver.RunConfig, error) {
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

	return &driver.RunConfig{
		Image:      "runners/gemini:latest",
		Command:    []string{"gemini"},
		Args:       args,
		Env:        map[string]string{},
		WorkingDir: "/workspace",
	}, nil
}

// ParseEvent 解析事件
func (d *Driver) ParseEvent(line string) (*driver.CanonicalEvent, error) {
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

	return &driver.CanonicalEvent{
		Type:    canonicalType,
		Payload: raw,
	}, nil
}

func mapEventType(geminiType string) driver.EventType {
	mapping := map[string]driver.EventType{
		"message":      driver.EventMessage,
		"tool_call":    driver.EventToolUseStart,
		"tool_result":  driver.EventToolResult,
		"command":      driver.EventCommand,
		"command_done": driver.EventCommandOutput,
		"file_read":    driver.EventFileRead,
		"file_write":   driver.EventFileWrite,
		"error":        driver.EventError,
		"done":         driver.EventRunCompleted,
	}
	return mapping[geminiType]
}

// CollectArtifacts 收集产物
func (d *Driver) CollectArtifacts(ctx context.Context, workspaceDir string) (*driver.Artifacts, error) {
	return &driver.Artifacts{
		EventsFile: filepath.Join(workspaceDir, ".agent", "events.jsonl"),
	}, nil
}

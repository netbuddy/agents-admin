// Package claude 实现 Claude Code CLI Driver
package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"agents-admin/pkg/driver"
)

// Driver Claude Code CLI 驱动
type Driver struct{}

// New 创建 Claude Driver
func New() *Driver {
	return &Driver{}
}

// Name 返回驱动名称
func (d *Driver) Name() string {
	return "claude-v1"
}

// Validate 验证 AgentConfig
func (d *Driver) Validate(agent *driver.AgentConfig) error {
	if agent.Type != "claude" {
		return fmt.Errorf("agent type mismatch: expected claude, got %s", agent.Type)
	}
	return nil
}

// BuildCommand 构建运行命令
// ctx 用于超时控制（当前实现未使用，预留接口）
func (d *Driver) BuildCommand(ctx context.Context, spec *driver.TaskSpec, agent *driver.AgentConfig) (*driver.RunConfig, error) {
	args := []string{
		"-p", spec.Prompt,
		"--output-format", "stream-json",
	}

	// 最大轮次
	if maxTurns, ok := agent.Parameters["max_turns"].(float64); ok {
		args = append(args, "--max-turns", strconv.Itoa(int(maxTurns)))
	}

	// 允许的工具
	if allowedTools, ok := agent.Parameters["allowed_tools"].([]interface{}); ok && len(allowedTools) > 0 {
		tools := make([]string, 0, len(allowedTools))
		for _, t := range allowedTools {
			if s, ok := t.(string); ok {
				tools = append(tools, s)
			}
		}
		args = append(args, "--allowed-tools", strings.Join(tools, ","))
	}

	// 禁止的工具
	if disallowedTools, ok := agent.Parameters["disallowed_tools"].([]interface{}); ok && len(disallowedTools) > 0 {
		tools := make([]string, 0, len(disallowedTools))
		for _, t := range disallowedTools {
			if s, ok := t.(string); ok {
				tools = append(tools, s)
			}
		}
		args = append(args, "--disallowed-tools", strings.Join(tools, ","))
	}

	// 沙箱模式
	if sandbox, ok := agent.Parameters["sandbox"].(bool); ok && sandbox {
		args = append(args, "--no-permissions")
	}

	return &driver.RunConfig{
		Image:      "runners/claude:latest",
		Command:    []string{"claude"},
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

	canonicalType := mapEventType(eventType)
	if canonicalType == "" {
		return nil, nil
	}

	return &driver.CanonicalEvent{
		Type:    canonicalType,
		Payload: raw,
	}, nil
}

func mapEventType(claudeType string) driver.EventType {
	mapping := map[string]driver.EventType{
		"assistant":   driver.EventMessage,
		"user":        driver.EventMessage,
		"tool_use":    driver.EventToolUseStart,
		"tool_result": driver.EventToolResult,
		"error":       driver.EventError,
		"result":      driver.EventRunCompleted,
	}
	return mapping[claudeType]
}

// CollectArtifacts 收集产物
func (d *Driver) CollectArtifacts(ctx context.Context, workspaceDir string) (*driver.Artifacts, error) {
	return &driver.Artifacts{
		EventsFile: filepath.Join(workspaceDir, ".agent", "events.jsonl"),
	}, nil
}

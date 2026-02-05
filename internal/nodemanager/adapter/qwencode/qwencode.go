// Package qwencode 实现 Qwen-Code CLI Adapter
//
// Qwen-Code 是基于 Google Gemini CLI 的开源 AI Agent，
// 支持 Qwen OAuth 免费认证（2000 请求/天）。
//
// 官方文档: https://github.com/QwenLM/qwen-code
//
// 使用方式:
//   - 交互模式: qwen
//   - Headless 模式: qwen -p "your question"
//
// 认证方式:
//   - Qwen OAuth: 免费，2000 请求/天
//   - OpenAI 兼容 API: 设置 OPENAI_API_KEY 环境变量
package qwencode

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"agents-admin/internal/nodemanager/adapter"
)

// Adapter Qwen-Code CLI 适配器
type Adapter struct{}

// New 创建 Qwen-Code Adapter
func New() *Adapter {
	return &Adapter{}
}

// Name 返回适配器名称
func (a *Adapter) Name() string {
	return "qwencode-v1"
}

// Validate 验证 AgentConfig
func (a *Adapter) Validate(agent *adapter.AgentConfig) error {
	if agent.Type != "qwencode" && agent.Type != "qwen-code" && agent.Type != "qwen" {
		return fmt.Errorf("agent type mismatch: expected qwencode/qwen-code/qwen, got %s", agent.Type)
	}
	return nil
}

// BuildCommand 构建运行命令
// ctx 用于超时控制（当前实现未使用，预留接口）
//
// Qwen Code Headless 模式参数:
//   - -p, --prompt: 提示词（必需）
//   - --output-format: 输出格式（stream-json 用于流式解析）
//   - --yolo, -y: 自动批准所有操作（CI/自动化必需）
//   - --max-turns: 最大交互轮次
//
// 参考: https://qwenlm.github.io/qwen-code-docs/en/users/features/headless/
func (a *Adapter) BuildCommand(ctx context.Context, spec *adapter.TaskSpec, agent *adapter.AgentConfig) (*adapter.RunConfig, error) {
	args := []string{
		"-p", spec.Prompt,
		"--output-format", "stream-json", // 使用流式 JSON 输出以便实时解析
	}

	// yolo 模式（可选，仅在明确指定时启用）
	// 自动批准所有操作，适用于 CI/自动化场景
	if yolo, ok := agent.Parameters["yolo"].(bool); ok && yolo {
		args = append(args, "--yolo")
	}

	// 最大轮次（可选）
	if maxTurns, ok := agent.Parameters["max_turns"].(float64); ok {
		args = append(args, "--max-turns", strconv.Itoa(int(maxTurns)))
	}

	// 自定义模型（可选）
	model := ""
	if m, ok := agent.Parameters["model"].(string); ok && m != "" {
		model = m
	}

	// 构建环境变量
	env := map[string]string{}

	// 支持 OpenAI 兼容 API（可选，默认使用 Qwen OAuth）
	if apiKey, ok := agent.Parameters["api_key"].(string); ok && apiKey != "" {
		env["OPENAI_API_KEY"] = apiKey
	}
	if baseURL, ok := agent.Parameters["base_url"].(string); ok && baseURL != "" {
		env["OPENAI_BASE_URL"] = baseURL
	}
	if model != "" {
		env["OPENAI_MODEL"] = model
	}

	return &adapter.RunConfig{
		Image:      "runners/qwencode:latest",
		Command:    []string{"qwen"},
		Args:       args,
		Env:        env,
		WorkingDir: "/workspace",
	}, nil
}

// ParseEvent 解析事件
//
// Qwen Code stream-json 格式输出每行一个 JSON 对象，格式如：
//   {"type": "system", "subtype": "session_start", ...}
//   {"type": "assistant", "message": {...}, ...}
//   {"type": "result", "subtype": "success", ...}
//
// 参考: https://qwenlm.github.io/qwen-code-docs/en/users/features/headless/
func (a *Adapter) ParseEvent(line string) (*adapter.CanonicalEvent, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		// 非 JSON 行，忽略（符合 Driver 契约）
		return nil, nil
	}

	eventType, _ := raw["type"].(string)
	if eventType == "" {
		return nil, nil
	}

	// 映射 Qwen-Code stream-json 事件类型到平台事件类型
	canonicalType := mapEventType(eventType, raw)
	if canonicalType == "" {
		return nil, nil
	}

	// 提取有用的内容
	payload := extractPayload(eventType, raw)

	return &adapter.CanonicalEvent{
		Type:    canonicalType,
		Payload: payload,
	}, nil
}

func mapEventType(eventType string, raw map[string]interface{}) adapter.EventType {
	switch eventType {
	case "system":
		// 系统消息（init 等），作为系统信息处理，不作为 run_started
		// run_started 由 executor 单独上报
		return adapter.EventSystemInfo
	case "assistant", "message":
		// 助手消息
		return adapter.EventMessage
	case "result", "done":
		// 结果消息，作为结果信息处理，不作为 run_completed
		// run_completed 由 executor 单独上报
		return adapter.EventRunCompleted
	case "tool_use", "tool_call":
		return adapter.EventToolUseStart
	case "tool_result":
		return adapter.EventToolResult
	case "file_read":
		return adapter.EventFileRead
	case "file_write":
		return adapter.EventFileWrite
	case "shell", "command":
		return adapter.EventCommand
	case "shell_output", "command_output", "command_done":
		return adapter.EventCommandOutput
	case "error":
		return adapter.EventError
	case "thinking", "plan":
		// thinking 事件作为消息处理
		return adapter.EventMessage
	default:
		// 未知类型，忽略（符合 Adapter 契约）
		return ""
	}
}

func extractPayload(eventType string, raw map[string]interface{}) map[string]interface{} {
	payload := make(map[string]interface{})

	switch eventType {
	case "assistant":
		// 从 message.content 提取文本
		if msg, ok := raw["message"].(map[string]interface{}); ok {
			if content, ok := msg["content"].([]interface{}); ok && len(content) > 0 {
				for _, c := range content {
					if block, ok := c.(map[string]interface{}); ok {
						if text, ok := block["text"].(string); ok {
							payload["content"] = text
							break
						}
					}
				}
			}
		}
		payload["type"] = "message"
	case "result":
		payload["result"] = raw["result"]
		payload["subtype"] = raw["subtype"]
		if usage, ok := raw["usage"].(map[string]interface{}); ok {
			payload["usage"] = usage
		}
	case "tool_use", "tool_call":
		payload["tool"] = raw["name"]
		payload["input"] = raw["input"]
	case "tool_result":
		payload["output"] = raw["output"]
		payload["success"] = raw["is_error"] != true
	default:
		// 复制所有字段
		for k, v := range raw {
			payload[k] = v
		}
	}

	return payload
}

// CollectArtifacts 收集产物
func (a *Adapter) CollectArtifacts(ctx context.Context, workspaceDir string) (*adapter.Artifacts, error) {
	return &adapter.Artifacts{
		EventsFile: filepath.Join(workspaceDir, ".qwen", "events.jsonl"),
	}, nil
}

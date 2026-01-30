package driver

// ============================================================================
// AgentConfig - Agent 配置
// ============================================================================

// AgentConfig 定义执行任务的 Agent 配置
//
// AgentConfig 回答"谁来做"的问题：
//   - Type：使用哪个 Agent（claude/gemini/codex）
//   - Model：使用哪个模型版本
//   - Capabilities：Agent 具备哪些能力
//   - Parameters：Agent 特定的参数
//
// 设计说明：
//   - AgentConfig 已从 TaskSpec 分离，现在属于 Run 级别
//   - 这允许同一任务使用不同 Agent 执行（A/B 测试、失败重试）
//   - AgentConfig.Type 决定使用哪个 Driver
//   - Driver 根据 AgentConfig 生成 RunConfig
type AgentConfig struct {
	// Type Agent 类型，对应 Driver 名称
	// 例如：claude、gemini、codex、custom
	Type string `json:"type"`

	// Model 模型版本（可选）
	// 例如：claude-sonnet-4-20250514、gemini-2.0-flash
	Model string `json:"model,omitempty"`

	// Capabilities 能力声明列表
	// 例如：["tool_use", "code_execution", "file_access"]
	// 用于验证任务需求与 Agent 能力是否匹配
	Capabilities []string `json:"capabilities,omitempty"`

	// Parameters Agent 特定参数
	// 不同 Agent 有不同的参数，通过 map 传递
	// 例如：{"temperature": 0.7, "max_tokens": 4096}
	Parameters map[string]interface{} `json:"parameters,omitempty"`

	// MCPServers MCP Server 配置列表（可选）
	// 用于扩展 Agent 能力
	MCPServers []MCPServerConfig `json:"mcp_servers,omitempty"`
}

// MCPServerConfig MCP Server 配置
type MCPServerConfig struct {
	// Name 服务名称
	Name string `json:"name"`

	// Command 启动命令
	Command string `json:"command"`

	// Args 启动参数
	Args []string `json:"args,omitempty"`

	// Env 环境变量
	Env map[string]string `json:"env,omitempty"`
}

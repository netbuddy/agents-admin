// Package model 定义核心数据模型
//
// mcp.go 包含 Model Context Protocol（MCP）相关的数据模型定义：
//   - MCPServer：MCP 服务端配置
//   - MCPSource：MCP Server 来源枚举
//   - MCPTransport：MCP 传输协议枚举
//   - MCPCapabilities：MCP Server 能力声明
//   - MCPResource：MCP 资源定义
//   - MCPTool：MCP 工具定义
//   - MCPPrompt：MCP 提示模板定义
//   - MCPRegistry：MCP 市场/注册中心
//   - AgentMCPServer：Agent 与 MCPServer 的关联
//
// 设计理念：
//   - MCP 是开源标准，用于连接 AI 应用与外部系统
//   - MCPServer 定义可连接的外部服务（工具、数据源、工作流）
//   - MCPRegistry 管理 MCPServer 集合
//   - Agent 可以连接多个 MCPServer 获取能力
//
// 参考：https://modelcontextprotocol.io
package model

import (
	"encoding/json"
	"time"
)

// ============================================================================
// MCPSource - MCP Server 来源枚举
// ============================================================================

// MCPSource MCP Server 来源
type MCPSource string

const (
	// MCPSourceBuiltin 内置（文件系统、终端等基础服务）
	MCPSourceBuiltin MCPSource = "builtin"

	// MCPSourceOfficial 官方（GitHub、Slack 等官方集成）
	MCPSourceOfficial MCPSource = "official"

	// MCPSourceCommunity 社区（社区贡献的服务）
	MCPSourceCommunity MCPSource = "community"

	// MCPSourceCustom 用户自定义
	MCPSourceCustom MCPSource = "custom"
)

// ============================================================================
// MCPTransport - MCP 传输协议枚举
// ============================================================================

// MCPTransport MCP 传输协议
type MCPTransport string

const (
	// MCPTransportStdio 标准输入输出（本地进程通信）
	MCPTransportStdio MCPTransport = "stdio"

	// MCPTransportSSE Server-Sent Events（实时推送）
	MCPTransportSSE MCPTransport = "sse"

	// MCPTransportHTTP HTTP（请求-响应模式）
	MCPTransportHTTP MCPTransport = "http"
)

// ============================================================================
// MCPResource - MCP 资源定义
// ============================================================================

// MCPResource MCP 资源定义
//
// 资源是 MCP Server 提供的数据源，如文件、数据库记录等。
type MCPResource struct {
	// URI 资源标识符（如 file:///path/to/file）
	URI string `json:"uri"`

	// Name 资源名称
	Name string `json:"name"`

	// Description 资源描述
	Description string `json:"description,omitempty"`

	// MimeType MIME 类型
	MimeType string `json:"mime_type,omitempty"`
}

// ============================================================================
// MCPTool - MCP 工具定义
// ============================================================================

// MCPTool MCP 工具定义
//
// 工具是 MCP Server 提供的可调用功能。
type MCPTool struct {
	// Name 工具名称
	Name string `json:"name"`

	// Description 工具描述
	Description string `json:"description,omitempty"`

	// InputSchema 输入参数定义（JSON Schema 格式）
	InputSchema json.RawMessage `json:"input_schema,omitempty"`
}

// ============================================================================
// MCPPromptArg - MCP 提示模板参数
// ============================================================================

// MCPPromptArg MCP 提示模板参数
type MCPPromptArg struct {
	// Name 参数名称
	Name string `json:"name"`

	// Description 参数描述
	Description string `json:"description,omitempty"`

	// Required 是否必填
	Required bool `json:"required"`
}

// ============================================================================
// MCPPrompt - MCP 提示模板定义
// ============================================================================

// MCPPrompt MCP 提示模板定义
//
// 提示模板是 MCP Server 提供的预定义提示词。
type MCPPrompt struct {
	// Name 模板名称
	Name string `json:"name"`

	// Description 模板描述
	Description string `json:"description,omitempty"`

	// Arguments 模板参数
	Arguments []MCPPromptArg `json:"arguments,omitempty"`
}

// ============================================================================
// MCPCapabilities - MCP Server 能力声明
// ============================================================================

// MCPCapabilities MCP Server 能力声明
//
// 描述 MCP Server 提供的资源、工具和提示模板。
type MCPCapabilities struct {
	// Resources 数据资源列表
	Resources []MCPResource `json:"resources,omitempty"`

	// Tools 可调用工具列表
	Tools []MCPTool `json:"tools,omitempty"`

	// Prompts 提示模板列表
	Prompts []MCPPrompt `json:"prompts,omitempty"`
}

// ============================================================================
// MCPServer - MCP 服务端配置
// ============================================================================

// MCPServer MCP 服务端配置
//
// MCPServer 定义一个 MCP 服务的配置：
//   - 连接方式（stdio/sse/http）
//   - 能力声明（提供的工具、资源、提示模板）
//   - 元数据（版本、作者、仓库）
//
// 使用场景：
//   - 文件系统访问（builtin）
//   - GitHub 集成（official）
//   - 自定义工具服务（custom）
type MCPServer struct {
	// === 基础字段 ===

	// ID 唯一标识
	ID string `json:"id" db:"id"`

	// Name 服务名称
	Name string `json:"name" db:"name"`

	// Description 服务描述
	Description string `json:"description" db:"description"`

	// === 来源 ===

	// Source 服务来源
	Source MCPSource `json:"source" db:"source"`

	// === 连接配置 ===

	// Transport 传输协议
	Transport MCPTransport `json:"transport" db:"transport"`

	// Command stdio 模式的启动命令
	Command string `json:"command,omitempty" db:"command"`

	// Args 命令参数
	Args []string `json:"args,omitempty" db:"args"`

	// URL sse/http 模式的 URL
	URL string `json:"url,omitempty" db:"url"`

	// Headers HTTP 头
	Headers map[string]string `json:"headers,omitempty" db:"headers"`

	// === 能力声明 ===

	// Capabilities 能力声明
	Capabilities MCPCapabilities `json:"capabilities" db:"capabilities"`

	// === 元数据 ===

	// Version 版本号
	Version string `json:"version" db:"version"`

	// Author 作者
	Author string `json:"author,omitempty" db:"author"`

	// Repository 仓库地址
	Repository string `json:"repository,omitempty" db:"repository"`

	// IsBuiltin 是否内置
	IsBuiltin bool `json:"is_builtin" db:"is_builtin"`

	// Tags 标签
	Tags []string `json:"tags,omitempty" db:"tags"`

	// === 时间戳 ===

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ============================================================================
// MCPServer 辅助方法
// ============================================================================

// IsStdio 判断是否使用 stdio 传输
func (s *MCPServer) IsStdio() bool {
	return s.Transport == MCPTransportStdio
}

// IsHTTP 判断是否使用 HTTP 传输
func (s *MCPServer) IsHTTP() bool {
	return s.Transport == MCPTransportHTTP || s.Transport == MCPTransportSSE
}

// GetToolNames 获取工具名称列表
func (s *MCPServer) GetToolNames() []string {
	names := make([]string, len(s.Capabilities.Tools))
	for i, t := range s.Capabilities.Tools {
		names[i] = t.Name
	}
	return names
}

// GetResourceURIs 获取资源 URI 列表
func (s *MCPServer) GetResourceURIs() []string {
	uris := make([]string, len(s.Capabilities.Resources))
	for i, r := range s.Capabilities.Resources {
		uris[i] = r.URI
	}
	return uris
}

// ============================================================================
// MCPRegistry - MCP 市场/注册中心
// ============================================================================

// MCPRegistry MCP 市场/注册中心
//
// 管理一组 MCP Server 的集合。
type MCPRegistry struct {
	// ID 唯一标识
	ID string `json:"id" db:"id"`

	// Name 名称
	Name string `json:"name" db:"name"`

	// Description 描述
	Description string `json:"description" db:"description"`

	// Type 类型
	Type MCPSource `json:"type" db:"type"`

	// OwnerID 所有者 ID
	OwnerID *string `json:"owner_id,omitempty" db:"owner_id"`

	// IsPublic 是否公开
	IsPublic bool `json:"is_public" db:"is_public"`

	// ServerCount 服务数量
	ServerCount int `json:"server_count" db:"server_count"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ============================================================================
// AgentMCPServer - Agent 与 MCPServer 关联
// ============================================================================

// AgentMCPServer Agent 与 MCPServer 的关联（N:M）
//
// 记录 Agent 连接的 MCP Server 及其配置。
type AgentMCPServer struct {
	// AgentID Agent ID
	AgentID string `json:"agent_id" db:"agent_id"`

	// MCPServerID MCP Server ID
	MCPServerID string `json:"mcp_server_id" db:"mcp_server_id"`

	// Enabled 是否启用
	Enabled bool `json:"enabled" db:"enabled"`

	// Config 自定义配置（覆盖默认配置）
	Config map[string]string `json:"config,omitempty" db:"config"`

	// Priority 优先级（数值越小优先级越高）
	Priority int `json:"priority" db:"priority"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ============================================================================
// MCPClientConfig - Agent MCP 客户端配置
// ============================================================================

// MCPServerConnection MCP Server 连接配置
type MCPServerConnection struct {
	// ServerID MCP Server ID
	ServerID string `json:"server_id"`

	// Enabled 是否启用
	Enabled bool `json:"enabled"`

	// Config 连接时的自定义配置
	Config map[string]string `json:"config,omitempty"`
}

// MCPClientConfig Agent 内置的 MCP 客户端配置
type MCPClientConfig struct {
	// Servers 已连接的 MCP Server 列表
	Servers []MCPServerConnection `json:"servers"`
}

// ============================================================================
// 内置 MCP Server
// ============================================================================

// BuiltinMCPServers 内置 MCP Server 列表
var BuiltinMCPServers = []MCPServer{
	{
		ID:          "builtin-filesystem",
		Name:        "文件系统",
		Description: "提供文件读写能力",
		Source:      MCPSourceBuiltin,
		Transport:   MCPTransportStdio,
		Command:     "npx",
		Args:        []string{"-y", "@anthropic/mcp-server-filesystem"},
		Capabilities: MCPCapabilities{
			Tools: []MCPTool{
				{Name: "file_read", Description: "读取文件内容"},
				{Name: "file_write", Description: "写入文件内容"},
				{Name: "file_list", Description: "列出目录内容"},
				{Name: "file_delete", Description: "删除文件"},
			},
		},
		Version:   "1.0.0",
		IsBuiltin: true,
		Tags:      []string{"filesystem", "file", "io"},
	},
	{
		ID:          "builtin-terminal",
		Name:        "终端",
		Description: "提供命令执行能力",
		Source:      MCPSourceBuiltin,
		Transport:   MCPTransportStdio,
		Command:     "npx",
		Args:        []string{"-y", "@anthropic/mcp-server-shell"},
		Capabilities: MCPCapabilities{
			Tools: []MCPTool{
				{Name: "shell_execute", Description: "执行 shell 命令"},
				{Name: "shell_interactive", Description: "交互式 shell"},
			},
		},
		Version:   "1.0.0",
		IsBuiltin: true,
		Tags:      []string{"terminal", "shell", "command"},
	},
	{
		ID:          "builtin-browser",
		Name:        "浏览器",
		Description: "提供网页浏览和抓取能力",
		Source:      MCPSourceBuiltin,
		Transport:   MCPTransportStdio,
		Command:     "npx",
		Args:        []string{"-y", "@anthropic/mcp-server-puppeteer"},
		Capabilities: MCPCapabilities{
			Tools: []MCPTool{
				{Name: "browser_navigate", Description: "导航到 URL"},
				{Name: "browser_screenshot", Description: "截取页面截图"},
				{Name: "browser_click", Description: "点击页面元素"},
				{Name: "browser_type", Description: "输入文本"},
			},
		},
		Version:   "1.0.0",
		IsBuiltin: true,
		Tags:      []string{"browser", "web", "puppeteer"},
	},
}

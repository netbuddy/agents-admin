// Package model 定义核心数据模型的测试
package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// 阶段8：MCP 模型测试
// ============================================================================

// TestMCPSource_Values 验证 MCPSource 枚举值
func TestMCPSource_Values(t *testing.T) {
	sources := []MCPSource{
		MCPSourceBuiltin,
		MCPSourceOfficial,
		MCPSourceCommunity,
		MCPSourceCustom,
	}

	for _, s := range sources {
		assert.NotEmpty(t, string(s))
	}

	assert.Equal(t, MCPSource("builtin"), MCPSourceBuiltin)
	assert.Equal(t, MCPSource("official"), MCPSourceOfficial)
	assert.Equal(t, MCPSource("community"), MCPSourceCommunity)
	assert.Equal(t, MCPSource("custom"), MCPSourceCustom)
}

// TestMCPTransport_Values 验证 MCPTransport 枚举值
func TestMCPTransport_Values(t *testing.T) {
	transports := []MCPTransport{
		MCPTransportStdio,
		MCPTransportSSE,
		MCPTransportHTTP,
	}

	for _, tr := range transports {
		assert.NotEmpty(t, string(tr))
	}

	assert.Equal(t, MCPTransport("stdio"), MCPTransportStdio)
	assert.Equal(t, MCPTransport("sse"), MCPTransportSSE)
	assert.Equal(t, MCPTransport("http"), MCPTransportHTTP)
}

// TestMCPResource_BasicFields 验证 MCPResource 基础字段
func TestMCPResource_BasicFields(t *testing.T) {
	resource := MCPResource{
		URI:         "file:///workspace/README.md",
		Name:        "README",
		Description: "项目说明文档",
		MimeType:    "text/markdown",
	}

	assert.Equal(t, "file:///workspace/README.md", resource.URI)
	assert.Equal(t, "README", resource.Name)
	assert.Equal(t, "text/markdown", resource.MimeType)
}

// TestMCPTool_BasicFields 验证 MCPTool 基础字段
func TestMCPTool_BasicFields(t *testing.T) {
	tool := MCPTool{
		Name:        "file_read",
		Description: "读取文件内容",
		InputSchema: json.RawMessage(`{"type": "object", "properties": {"path": {"type": "string"}}}`),
	}

	assert.Equal(t, "file_read", tool.Name)
	assert.Equal(t, "读取文件内容", tool.Description)
	assert.NotNil(t, tool.InputSchema)
}

// TestMCPPrompt_BasicFields 验证 MCPPrompt 基础字段
func TestMCPPrompt_BasicFields(t *testing.T) {
	prompt := MCPPrompt{
		Name:        "code_review",
		Description: "代码审查提示模板",
		Arguments: []MCPPromptArg{
			{Name: "code", Description: "要审查的代码", Required: true},
			{Name: "language", Description: "编程语言", Required: false},
		},
	}

	assert.Equal(t, "code_review", prompt.Name)
	assert.Len(t, prompt.Arguments, 2)
	assert.True(t, prompt.Arguments[0].Required)
	assert.False(t, prompt.Arguments[1].Required)
}

// TestMCPCapabilities_BasicFields 验证 MCPCapabilities 基础字段
func TestMCPCapabilities_BasicFields(t *testing.T) {
	caps := MCPCapabilities{
		Resources: []MCPResource{
			{URI: "file:///test", Name: "test"},
		},
		Tools: []MCPTool{
			{Name: "file_read", Description: "读取文件"},
			{Name: "file_write", Description: "写入文件"},
		},
		Prompts: []MCPPrompt{
			{Name: "greeting", Description: "问候语"},
		},
	}

	assert.Len(t, caps.Resources, 1)
	assert.Len(t, caps.Tools, 2)
	assert.Len(t, caps.Prompts, 1)
}

// TestMCPServer_BasicFields 验证 MCPServer 基础字段
func TestMCPServer_BasicFields(t *testing.T) {
	now := time.Now()

	server := MCPServer{
		ID:          "mcp-001",
		Name:        "文件系统服务",
		Description: "提供文件读写能力",
		Source:      MCPSourceBuiltin,
		Transport:   MCPTransportStdio,
		Command:     "npx",
		Args:        []string{"-y", "@anthropic/mcp-server-filesystem"},
		Capabilities: MCPCapabilities{
			Tools: []MCPTool{
				{Name: "file_read"},
				{Name: "file_write"},
			},
		},
		Version:   "1.0.0",
		Author:    "Anthropic",
		IsBuiltin: true,
		Tags:      []string{"filesystem", "io"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, "mcp-001", server.ID)
	assert.Equal(t, "文件系统服务", server.Name)
	assert.Equal(t, MCPSourceBuiltin, server.Source)
	assert.Equal(t, MCPTransportStdio, server.Transport)
	assert.Equal(t, "npx", server.Command)
	assert.Len(t, server.Args, 2)
	assert.True(t, server.IsBuiltin)
}

// TestMCPServer_IsStdio 验证 stdio 传输判断
func TestMCPServer_IsStdio(t *testing.T) {
	stdio := MCPServer{Transport: MCPTransportStdio}
	assert.True(t, stdio.IsStdio())

	http := MCPServer{Transport: MCPTransportHTTP}
	assert.False(t, http.IsStdio())
}

// TestMCPServer_IsHTTP 验证 HTTP 传输判断
func TestMCPServer_IsHTTP(t *testing.T) {
	http := MCPServer{Transport: MCPTransportHTTP}
	assert.True(t, http.IsHTTP())

	sse := MCPServer{Transport: MCPTransportSSE}
	assert.True(t, sse.IsHTTP())

	stdio := MCPServer{Transport: MCPTransportStdio}
	assert.False(t, stdio.IsHTTP())
}

// TestMCPServer_GetToolNames 验证获取工具名称
func TestMCPServer_GetToolNames(t *testing.T) {
	server := MCPServer{
		Capabilities: MCPCapabilities{
			Tools: []MCPTool{
				{Name: "file_read"},
				{Name: "file_write"},
				{Name: "file_delete"},
			},
		},
	}

	names := server.GetToolNames()
	assert.Len(t, names, 3)
	assert.Contains(t, names, "file_read")
	assert.Contains(t, names, "file_write")
	assert.Contains(t, names, "file_delete")
}

// TestMCPServer_GetResourceURIs 验证获取资源 URI
func TestMCPServer_GetResourceURIs(t *testing.T) {
	server := MCPServer{
		Capabilities: MCPCapabilities{
			Resources: []MCPResource{
				{URI: "file:///workspace"},
				{URI: "db://localhost/users"},
			},
		},
	}

	uris := server.GetResourceURIs()
	assert.Len(t, uris, 2)
	assert.Contains(t, uris, "file:///workspace")
	assert.Contains(t, uris, "db://localhost/users")
}

// TestMCPServer_JSONSerialization 验证 MCPServer JSON 序列化
func TestMCPServer_JSONSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	server := MCPServer{
		ID:          "mcp-json-001",
		Name:        "GitHub 集成",
		Description: "GitHub API 集成服务",
		Source:      MCPSourceOfficial,
		Transport:   MCPTransportHTTP,
		URL:         "https://api.github.com/mcp",
		Headers:     map[string]string{"Authorization": "Bearer xxx"},
		Capabilities: MCPCapabilities{
			Tools: []MCPTool{
				{Name: "repo_list", Description: "列出仓库"},
				{Name: "issue_create", Description: "创建 Issue"},
			},
		},
		Version:    "2.0.0",
		Author:     "GitHub",
		Repository: "https://github.com/anthropics/mcp-servers",
		IsBuiltin:  false,
		Tags:       []string{"github", "vcs"},
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// 序列化
	data, err := json.Marshal(server)
	require.NoError(t, err)

	// 反序列化
	var decoded MCPServer
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// 验证
	assert.Equal(t, server.ID, decoded.ID)
	assert.Equal(t, server.Name, decoded.Name)
	assert.Equal(t, server.Source, decoded.Source)
	assert.Equal(t, server.Transport, decoded.Transport)
	assert.Equal(t, server.URL, decoded.URL)
	assert.Len(t, decoded.Headers, 1)
	assert.Len(t, decoded.Capabilities.Tools, 2)
}

// TestMCPRegistry_BasicFields 验证 MCPRegistry 基础字段
func TestMCPRegistry_BasicFields(t *testing.T) {
	now := time.Now()
	ownerID := "org-001"

	registry := MCPRegistry{
		ID:          "registry-001",
		Name:        "官方 MCP 市场",
		Description: "Anthropic 官方 MCP Server 集合",
		Type:        MCPSourceOfficial,
		OwnerID:     &ownerID,
		IsPublic:    true,
		ServerCount: 20,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, "registry-001", registry.ID)
	assert.Equal(t, "官方 MCP 市场", registry.Name)
	assert.Equal(t, MCPSourceOfficial, registry.Type)
	assert.True(t, registry.IsPublic)
	assert.Equal(t, 20, registry.ServerCount)
}

// TestAgentMCPServer_BasicFields 验证 AgentMCPServer 关联
func TestAgentMCPServer_BasicFields(t *testing.T) {
	now := time.Now()

	agentMCP := AgentMCPServer{
		AgentID:     "agent-001",
		MCPServerID: "mcp-001",
		Enabled:     true,
		Config:      map[string]string{"workspace": "/home/user/project"},
		Priority:    1,
		CreatedAt:   now,
	}

	assert.Equal(t, "agent-001", agentMCP.AgentID)
	assert.Equal(t, "mcp-001", agentMCP.MCPServerID)
	assert.True(t, agentMCP.Enabled)
	assert.Equal(t, 1, agentMCP.Priority)
	assert.Equal(t, "/home/user/project", agentMCP.Config["workspace"])
}

// TestMCPClientConfig_BasicFields 验证 MCPClientConfig 基础字段
func TestMCPClientConfig_BasicFields(t *testing.T) {
	config := MCPClientConfig{
		Servers: []MCPServerConnection{
			{ServerID: "mcp-001", Enabled: true},
			{ServerID: "mcp-002", Enabled: false, Config: map[string]string{"timeout": "30s"}},
		},
	}

	assert.Len(t, config.Servers, 2)
	assert.True(t, config.Servers[0].Enabled)
	assert.False(t, config.Servers[1].Enabled)
	assert.Equal(t, "30s", config.Servers[1].Config["timeout"])
}

// TestBuiltinMCPServers 验证内置 MCP Server
func TestBuiltinMCPServers(t *testing.T) {
	// 验证内置服务数量
	assert.GreaterOrEqual(t, len(BuiltinMCPServers), 3)

	// 验证每个内置服务
	for _, server := range BuiltinMCPServers {
		assert.NotEmpty(t, server.ID, "ID should not be empty")
		assert.NotEmpty(t, server.Name, "Name should not be empty")
		assert.True(t, server.IsBuiltin, "IsBuiltin should be true")
		assert.Equal(t, MCPSourceBuiltin, server.Source, "Source should be builtin")
	}

	// 验证文件系统服务
	var fsServer *MCPServer
	for i := range BuiltinMCPServers {
		if BuiltinMCPServers[i].ID == "builtin-filesystem" {
			fsServer = &BuiltinMCPServers[i]
			break
		}
	}
	require.NotNil(t, fsServer)
	assert.Equal(t, "文件系统", fsServer.Name)
	assert.Equal(t, MCPTransportStdio, fsServer.Transport)
	assert.True(t, fsServer.IsStdio())
	assert.GreaterOrEqual(t, len(fsServer.Capabilities.Tools), 4)
}

// TestMCPServer_StdioVsHTTP 验证 stdio 和 HTTP 服务配置差异
func TestMCPServer_StdioVsHTTP(t *testing.T) {
	// stdio 服务
	stdioServer := MCPServer{
		ID:        "stdio-server",
		Transport: MCPTransportStdio,
		Command:   "npx",
		Args:      []string{"-y", "some-package"},
	}
	assert.True(t, stdioServer.IsStdio())
	assert.False(t, stdioServer.IsHTTP())
	assert.NotEmpty(t, stdioServer.Command)

	// HTTP 服务
	httpServer := MCPServer{
		ID:        "http-server",
		Transport: MCPTransportHTTP,
		URL:       "https://api.example.com/mcp",
		Headers:   map[string]string{"Authorization": "Bearer token"},
	}
	assert.False(t, httpServer.IsStdio())
	assert.True(t, httpServer.IsHTTP())
	assert.NotEmpty(t, httpServer.URL)
}

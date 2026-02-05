-- 011_mcp.sql
-- MCP（Model Context Protocol）表结构
-- 阶段8：MCP Server 和 Registry

-- ============================================================================
-- MCPServers 表 - MCP 服务端配置
-- ============================================================================

CREATE TABLE IF NOT EXISTS mcp_servers (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- 来源
    source VARCHAR(32) NOT NULL,          -- builtin, official, community, custom
    
    -- 连接配置
    transport VARCHAR(32) NOT NULL,       -- stdio, sse, http
    command VARCHAR(512),                 -- stdio 模式启动命令
    args JSONB,                           -- 命令参数
    url VARCHAR(512),                     -- sse/http 模式 URL
    headers JSONB,                        -- HTTP 头
    
    -- 能力声明
    capabilities JSONB,                   -- 工具、资源、提示模板
    
    -- 元数据
    version VARCHAR(32) NOT NULL DEFAULT '1.0.0',
    author VARCHAR(128),
    repository VARCHAR(512),
    is_builtin BOOLEAN NOT NULL DEFAULT FALSE,
    tags JSONB,
    
    -- 时间戳
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_mcp_servers_source ON mcp_servers(source);
CREATE INDEX IF NOT EXISTS idx_mcp_servers_transport ON mcp_servers(transport);
CREATE INDEX IF NOT EXISTS idx_mcp_servers_is_builtin ON mcp_servers(is_builtin);

-- ============================================================================
-- MCPRegistries 表 - MCP 市场
-- ============================================================================

CREATE TABLE IF NOT EXISTS mcp_registries (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(32) NOT NULL,            -- builtin, official, community, custom
    owner_id VARCHAR(64),
    is_public BOOLEAN NOT NULL DEFAULT FALSE,
    server_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_mcp_registries_type ON mcp_registries(type);
CREATE INDEX IF NOT EXISTS idx_mcp_registries_is_public ON mcp_registries(is_public);

-- ============================================================================
-- AgentMCPServers 表 - Agent 与 MCPServer 关联（N:M）
-- ============================================================================

CREATE TABLE IF NOT EXISTS agent_mcp_servers (
    agent_id VARCHAR(64) NOT NULL,
    mcp_server_id VARCHAR(64) NOT NULL REFERENCES mcp_servers(id),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    config JSONB,
    priority INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (agent_id, mcp_server_id)
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_agent_mcp_servers_agent_id ON agent_mcp_servers(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_mcp_servers_mcp_server_id ON agent_mcp_servers(mcp_server_id);

-- ============================================================================
-- 插入内置 MCP Server
-- ============================================================================

INSERT INTO mcp_servers (id, name, description, source, transport, command, args, capabilities, version, is_builtin, tags, created_at, updated_at)
VALUES 
    (
        'builtin-filesystem', 
        '文件系统', 
        '提供文件读写能力',
        'builtin',
        'stdio',
        'npx',
        '["-y", "@anthropic/mcp-server-filesystem"]',
        '{"tools": [{"name": "file_read", "description": "读取文件内容"}, {"name": "file_write", "description": "写入文件内容"}, {"name": "file_list", "description": "列出目录内容"}, {"name": "file_delete", "description": "删除文件"}]}',
        '1.0.0',
        TRUE,
        '["filesystem", "file", "io"]',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        'builtin-terminal', 
        '终端', 
        '提供命令执行能力',
        'builtin',
        'stdio',
        'npx',
        '["-y", "@anthropic/mcp-server-shell"]',
        '{"tools": [{"name": "shell_execute", "description": "执行 shell 命令"}, {"name": "shell_interactive", "description": "交互式 shell"}]}',
        '1.0.0',
        TRUE,
        '["terminal", "shell", "command"]',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        'builtin-browser', 
        '浏览器', 
        '提供网页浏览和抓取能力',
        'builtin',
        'stdio',
        'npx',
        '["-y", "@anthropic/mcp-server-puppeteer"]',
        '{"tools": [{"name": "browser_navigate", "description": "导航到 URL"}, {"name": "browser_screenshot", "description": "截取页面截图"}, {"name": "browser_click", "description": "点击页面元素"}, {"name": "browser_type", "description": "输入文本"}]}',
        '1.0.0',
        TRUE,
        '["browser", "web", "puppeteer"]',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    )
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    capabilities = EXCLUDED.capabilities,
    updated_at = CURRENT_TIMESTAMP;

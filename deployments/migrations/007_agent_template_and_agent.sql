-- 007_agent_template_and_agent.sql
-- AgentTemplate + Agent + Runtime 表结构
-- 阶段4：智能体模板与实例

-- ============================================================================
-- AgentTemplate 表 - 智能体模板
-- ============================================================================

CREATE TABLE IF NOT EXISTS agent_templates (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(32) NOT NULL DEFAULT 'custom',
    role VARCHAR(255),
    description TEXT,
    
    -- 身份与性格
    personality JSONB,
    system_prompt TEXT,
    
    -- 能力配置
    skills JSONB,
    tools JSONB,
    mcp_servers JSONB,
    documents JSONB,
    
    -- 自动化配置
    gambits JSONB,
    hooks JSONB,
    
    -- 运行参数
    model VARCHAR(64),
    temperature DECIMAL(3,2) DEFAULT 0.7,
    max_context INT DEFAULT 128000,
    
    -- 安全配置
    default_security_policy_id VARCHAR(64),
    
    -- 元数据
    is_builtin BOOLEAN NOT NULL DEFAULT FALSE,
    category VARCHAR(64),
    tags JSONB,
    
    -- 时间戳
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_agent_templates_type ON agent_templates(type);
CREATE INDEX IF NOT EXISTS idx_agent_templates_is_builtin ON agent_templates(is_builtin);
CREATE INDEX IF NOT EXISTS idx_agent_templates_category ON agent_templates(category);

-- ============================================================================
-- Agent 表 - 智能体实例（扩展现有 instances 表或新建）
-- ============================================================================

-- 注意：如果 instances 表已存在，需要先进行数据迁移
-- 这里假设创建新的 agents 表

CREATE TABLE IF NOT EXISTS agents (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    template_id VARCHAR(64) REFERENCES agent_templates(id),
    type VARCHAR(32) NOT NULL DEFAULT 'custom',
    
    -- 绑定关系
    account_id VARCHAR(64) NOT NULL,
    runtime_id VARCHAR(64),
    node_id VARCHAR(64),
    
    -- 状态
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    current_task_id VARCHAR(64),
    
    -- 配置覆盖
    config_overrides JSONB,
    security_policy_id VARCHAR(64),
    
    -- 记忆与能力
    memory_enabled BOOLEAN DEFAULT FALSE,
    
    -- 时间戳
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_active_at TIMESTAMP
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_agents_template_id ON agents(template_id);
CREATE INDEX IF NOT EXISTS idx_agents_account_id ON agents(account_id);
CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);
CREATE INDEX IF NOT EXISTS idx_agents_node_id ON agents(node_id);

-- ============================================================================
-- Runtime 表 - 运行时环境
-- ============================================================================

CREATE TABLE IF NOT EXISTS runtimes (
    id VARCHAR(64) PRIMARY KEY,
    type VARCHAR(32) NOT NULL DEFAULT 'container',
    status VARCHAR(32) NOT NULL DEFAULT 'creating',
    
    -- 关联
    agent_id VARCHAR(64) NOT NULL,
    node_id VARCHAR(64) NOT NULL,
    
    -- 容器特定字段
    container_id VARCHAR(255),
    container_name VARCHAR(255),
    image VARCHAR(255),
    
    -- 工作空间
    workspace_path TEXT,
    
    -- 网络
    ip_address VARCHAR(45),
    ports JSONB,
    
    -- 资源
    resources JSONB,
    
    -- 时间戳
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    stopped_at TIMESTAMP
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_runtimes_agent_id ON runtimes(agent_id);
CREATE INDEX IF NOT EXISTS idx_runtimes_node_id ON runtimes(node_id);
CREATE INDEX IF NOT EXISTS idx_runtimes_status ON runtimes(status);

-- ============================================================================
-- 添加外键约束（如果 agents 表创建成功）
-- ============================================================================

-- 注意：根据实际情况决定是否添加外键
-- ALTER TABLE runtimes ADD CONSTRAINT fk_runtimes_agent 
--     FOREIGN KEY (agent_id) REFERENCES agents(id);
-- ALTER TABLE agents ADD CONSTRAINT fk_agents_runtime 
--     FOREIGN KEY (runtime_id) REFERENCES runtimes(id);

-- ============================================================================
-- 插入内置 AgentTemplate
-- ============================================================================

INSERT INTO agent_templates (id, name, type, role, description, personality, model, temperature, max_context, is_builtin, category, created_at, updated_at)
VALUES 
    ('builtin-claude-dev', 'Claude 开发助手', 'claude', '代码开发助手', '基于 Claude 的代码开发智能体，擅长代码编写、审查和重构', '["专业", "严谨", "乐于助人"]', 'claude-3-opus', 0.7, 128000, TRUE, 'development', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('builtin-gemini-research', 'Gemini 研究助手', 'gemini', '技术研究助手', '基于 Gemini 的研究智能体，擅长技术调研和方案设计', '["博学", "分析性强", "客观"]', 'gemini-pro', 0.5, 100000, TRUE, 'research', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('builtin-qwen-code', 'Qwen 编程助手', 'qwen', '编程助手', '基于 Qwen 的编程智能体，支持多语言代码开发', '["高效", "精确", "实用"]', 'qwen-coder', 0.6, 64000, TRUE, 'development', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    updated_at = CURRENT_TIMESTAMP;

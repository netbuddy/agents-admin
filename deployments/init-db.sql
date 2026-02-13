-- ==========================================================================
-- Agent Admin 数据库初始化脚本（全量）
-- Schema Version: 1.0.0 (migration_id = 019)
-- 合并范围: migrations 002-019
-- 生成日期: 2025-07-14
--
-- 使用方法（新安装）:
--   psql "postgres://agents:agents_dev_password@localhost:5432/agents_admin" \
--     -f deployments/init-db.sql
--
-- 升级方法（已有数据）:
--   请按顺序执行 deployments/migrations/ 中尚未应用的增量脚本。
--   当前版本可通过 SELECT * FROM schema_version; 查询。
--   详见 deployments/UPGRADE.md
--
-- 说明: 本脚本为幂等设计，可重复执行。所有 CREATE 使用 IF NOT EXISTS。
-- ==========================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ==========================================================================
-- 更新时间触发器函数（全局共用）
-- ==========================================================================
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ==========================================================================
-- 1. tasks — 任务定义
-- ==========================================================================
CREATE TABLE IF NOT EXISTS tasks (
    id VARCHAR(20) PRIMARY KEY,
    parent_id VARCHAR(20) REFERENCES tasks(id) ON DELETE SET NULL,
    name VARCHAR(200) NOT NULL,
    type VARCHAR(32) NOT NULL DEFAULT 'general',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    description TEXT,
    -- 结构化提示词（JSONB）
    prompt JSONB,
    -- 旧字段保留兼容
    spec JSONB,
    context JSONB DEFAULT '{}',
    -- 扩展字段（005 扁平化）
    workspace JSONB,
    security JSONB,
    labels JSONB,
    template_id VARCHAR(64),
    agent_id VARCHAR(64),
    instance_id VARCHAR(64),
    -- 执行模式（015）
    mode VARCHAR(32) NOT NULL DEFAULT 'simple',
    decomposition JSONB,
    parent_close_policy VARCHAR(32) NOT NULL DEFAULT 'terminate',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_type ON tasks(type);
CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tasks_parent_id ON tasks(parent_id);
CREATE INDEX IF NOT EXISTS idx_tasks_instance_id ON tasks(instance_id);
CREATE INDEX IF NOT EXISTS idx_tasks_context ON tasks USING GIN (context);
CREATE INDEX IF NOT EXISTS idx_tasks_template_id ON tasks(template_id);
CREATE INDEX IF NOT EXISTS idx_tasks_agent_id ON tasks(agent_id);
CREATE INDEX IF NOT EXISTS idx_tasks_mode ON tasks(mode);

DROP TRIGGER IF EXISTS tasks_updated_at ON tasks;
CREATE TRIGGER tasks_updated_at
    BEFORE UPDATE ON tasks FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ==========================================================================
-- 2. runs — 任务执行记录
-- ==========================================================================
CREATE TABLE IF NOT EXISTS runs (
    id VARCHAR(20) PRIMARY KEY,
    task_id VARCHAR(20) NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'queued',
    node_id VARCHAR(50),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    snapshot JSONB,
    error TEXT,
    -- 层次化支持（015）
    parent_id VARCHAR(64),
    root_id VARCHAR(64),
    depth INT NOT NULL DEFAULT 0,
    phase VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_runs_task_id ON runs(task_id);
CREATE INDEX IF NOT EXISTS idx_runs_status ON runs(status);
CREATE INDEX IF NOT EXISTS idx_runs_node_id ON runs(node_id);
CREATE INDEX IF NOT EXISTS idx_runs_parent_id ON runs(parent_id);
CREATE INDEX IF NOT EXISTS idx_runs_root_id ON runs(root_id);
CREATE INDEX IF NOT EXISTS idx_runs_depth ON runs(depth);
CREATE INDEX IF NOT EXISTS idx_runs_phase ON runs(phase);

DROP TRIGGER IF EXISTS runs_updated_at ON runs;
CREATE TRIGGER runs_updated_at
    BEFORE UPDATE ON runs FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ==========================================================================
-- 3. events — 执行事件（CanonicalEvent）
-- ==========================================================================
CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    run_id VARCHAR(20) NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    seq INTEGER NOT NULL,
    type VARCHAR(50) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    payload JSONB,
    raw TEXT,
    UNIQUE(run_id, seq)
);

CREATE INDEX IF NOT EXISTS idx_events_run_id ON events(run_id);
CREATE INDEX IF NOT EXISTS idx_events_run_id_seq ON events(run_id, seq);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);

-- ==========================================================================
-- 4. nodes — 执行节点
-- ==========================================================================
CREATE TABLE IF NOT EXISTS nodes (
    id VARCHAR(50) PRIMARY KEY,
    status VARCHAR(20) NOT NULL DEFAULT 'offline',
    hostname VARCHAR(255) DEFAULT '',
    ips TEXT DEFAULT '',
    labels JSONB DEFAULT '{}',
    capacity JSONB DEFAULT '{}',
    last_heartbeat TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_nodes_status ON nodes(status);

DROP TRIGGER IF EXISTS nodes_updated_at ON nodes;
CREATE TRIGGER nodes_updated_at
    BEFORE UPDATE ON nodes FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ==========================================================================
-- 5. artifacts — 执行产物
-- ==========================================================================
CREATE TABLE IF NOT EXISTS artifacts (
    id BIGSERIAL PRIMARY KEY,
    run_id VARCHAR(20) NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    path VARCHAR(500) NOT NULL,
    size BIGINT,
    content_type VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ==========================================================================
-- 6. accounts — AI Agent 认证账号
-- ==========================================================================
CREATE TABLE IF NOT EXISTS accounts (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    agent_type_id VARCHAR(64) NOT NULL,
    node_id VARCHAR(64) NOT NULL,
    volume_name VARCHAR(255),
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_accounts_status ON accounts(status);
CREATE INDEX IF NOT EXISTS idx_accounts_agent_type ON accounts(agent_type_id);

DROP TRIGGER IF EXISTS accounts_updated_at ON accounts;
CREATE TRIGGER accounts_updated_at
    BEFORE UPDATE ON accounts FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ==========================================================================
-- 7. auth_tasks — 认证任务（控制面/数据面分离）
-- ==========================================================================
CREATE TABLE IF NOT EXISTS auth_tasks (
    id VARCHAR(64) PRIMARY KEY,
    account_id VARCHAR(64) NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    method VARCHAR(32) NOT NULL,
    node_id VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    terminal_port INT,
    terminal_url VARCHAR(255),
    container_name VARCHAR(255),
    message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_auth_tasks_status ON auth_tasks(status);
CREATE INDEX IF NOT EXISTS idx_auth_tasks_node_id ON auth_tasks(node_id);
CREATE INDEX IF NOT EXISTS idx_auth_tasks_account_id ON auth_tasks(account_id);

DROP TRIGGER IF EXISTS auth_tasks_updated_at ON auth_tasks;
CREATE TRIGGER auth_tasks_updated_at
    BEFORE UPDATE ON auth_tasks FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ==========================================================================
-- 8. instances — 容器实例
-- ==========================================================================
CREATE TABLE IF NOT EXISTS instances (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    account_id VARCHAR(64) NOT NULL,
    agent_type_id VARCHAR(32) NOT NULL,
    container_name VARCHAR(255),
    node_id VARCHAR(64),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_instances_account ON instances(account_id);
CREATE INDEX IF NOT EXISTS idx_instances_status ON instances(status);
CREATE INDEX IF NOT EXISTS idx_instances_node ON instances(node_id);

DROP TRIGGER IF EXISTS instances_updated_at ON instances;
CREATE TRIGGER instances_updated_at
    BEFORE UPDATE ON instances FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ==========================================================================
-- 9. terminal_sessions — 终端会话
-- ==========================================================================
CREATE TABLE IF NOT EXISTS terminal_sessions (
    id VARCHAR(64) PRIMARY KEY,
    instance_id VARCHAR(64),
    container_name VARCHAR(255) NOT NULL,
    node_id VARCHAR(64),
    port INTEGER,
    url VARCHAR(512),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_terminal_sessions_instance ON terminal_sessions(instance_id);
CREATE INDEX IF NOT EXISTS idx_terminal_sessions_status ON terminal_sessions(status);
CREATE INDEX IF NOT EXISTS idx_terminal_sessions_node ON terminal_sessions(node_id);

-- ==========================================================================
-- 10. proxies — 网络代理配置
-- ==========================================================================
CREATE TABLE IF NOT EXISTS proxies (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL DEFAULT 'http',
    host VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL,
    username VARCHAR(255),
    password VARCHAR(255),
    no_proxy TEXT,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_proxies_status ON proxies(status);
CREATE INDEX IF NOT EXISTS idx_proxies_is_default ON proxies(is_default);

-- ==========================================================================
-- 11. operations — 系统操作（认证、运行时管理等）
-- ==========================================================================
CREATE TABLE IF NOT EXISTS operations (
    id VARCHAR(64) PRIMARY KEY,
    type VARCHAR(32) NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    node_id VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_operations_type ON operations(type);
CREATE INDEX IF NOT EXISTS idx_operations_status ON operations(status);
CREATE INDEX IF NOT EXISTS idx_operations_node_id ON operations(node_id);
CREATE INDEX IF NOT EXISTS idx_operations_created_at ON operations(created_at DESC);

CREATE OR REPLACE FUNCTION update_operations_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_operations_updated_at ON operations;
CREATE TRIGGER trigger_operations_updated_at
    BEFORE UPDATE ON operations FOR EACH ROW EXECUTE FUNCTION update_operations_updated_at();

-- ==========================================================================
-- 12. actions — 操作执行实例
-- ==========================================================================
CREATE TABLE IF NOT EXISTS actions (
    id VARCHAR(64) PRIMARY KEY,
    operation_id VARCHAR(64) NOT NULL REFERENCES operations(id) ON DELETE CASCADE,
    status VARCHAR(32) NOT NULL DEFAULT 'assigned',
    progress INT NOT NULL DEFAULT 0,
    result JSONB,
    error TEXT,
    phase VARCHAR(64) DEFAULT '',
    message TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_actions_operation_id ON actions(operation_id);
CREATE INDEX IF NOT EXISTS idx_actions_status ON actions(status);
CREATE INDEX IF NOT EXISTS idx_actions_created_at ON actions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_actions_phase ON actions(phase) WHERE phase != '';

-- ==========================================================================
-- 13. agent_templates — 智能体模板
-- ==========================================================================
CREATE TABLE IF NOT EXISTS agent_templates (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(32) NOT NULL DEFAULT 'custom',
    role VARCHAR(255),
    description TEXT,
    personality JSONB,
    system_prompt TEXT,
    skills JSONB,
    tools JSONB,
    mcp_servers JSONB,
    documents JSONB,
    gambits JSONB,
    hooks JSONB,
    model VARCHAR(64),
    temperature DECIMAL(3,2) DEFAULT 0.7,
    max_context INT DEFAULT 128000,
    default_security_policy_id VARCHAR(64),
    is_builtin BOOLEAN NOT NULL DEFAULT FALSE,
    category VARCHAR(64),
    tags JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_templates_type ON agent_templates(type);
CREATE INDEX IF NOT EXISTS idx_agent_templates_is_builtin ON agent_templates(is_builtin);
CREATE INDEX IF NOT EXISTS idx_agent_templates_category ON agent_templates(category);

-- 内置 AgentTemplate
INSERT INTO agent_templates (id, name, type, role, description, personality, model, temperature, max_context, is_builtin, category)
VALUES
    ('builtin-claude-dev', 'Claude 开发助手', 'claude', '代码开发助手', '基于 Claude 的代码开发智能体', '["专业", "严谨", "乐于助人"]', 'claude-3-opus', 0.7, 128000, TRUE, 'development'),
    ('builtin-gemini-research', 'Gemini 研究助手', 'gemini', '技术研究助手', '基于 Gemini 的研究智能体', '["博学", "分析性强", "客观"]', 'gemini-pro', 0.5, 100000, TRUE, 'research'),
    ('builtin-qwen-code', 'Qwen 编程助手', 'qwen', '编程助手', '基于 Qwen 的编程智能体', '["高效", "精确", "实用"]', 'qwen-coder', 0.6, 64000, TRUE, 'development')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- ==========================================================================
-- 14. agents — 智能体实例
-- ==========================================================================
CREATE TABLE IF NOT EXISTS agents (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    template_id VARCHAR(64) REFERENCES agent_templates(id),
    type VARCHAR(32) NOT NULL DEFAULT 'custom',
    account_id VARCHAR(64) NOT NULL,
    runtime_id VARCHAR(64),
    node_id VARCHAR(64),
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    current_task_id VARCHAR(64),
    config_overrides JSONB,
    security_policy_id VARCHAR(64),
    memory_enabled BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_agents_template_id ON agents(template_id);
CREATE INDEX IF NOT EXISTS idx_agents_account_id ON agents(account_id);
CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);
CREATE INDEX IF NOT EXISTS idx_agents_node_id ON agents(node_id);

-- ==========================================================================
-- 15. runtimes — 运行时环境
-- ==========================================================================
CREATE TABLE IF NOT EXISTS runtimes (
    id VARCHAR(64) PRIMARY KEY,
    type VARCHAR(32) NOT NULL DEFAULT 'container',
    status VARCHAR(32) NOT NULL DEFAULT 'creating',
    agent_id VARCHAR(64) NOT NULL,
    node_id VARCHAR(64) NOT NULL,
    container_id VARCHAR(255),
    container_name VARCHAR(255),
    image VARCHAR(255),
    workspace_path TEXT,
    ip_address VARCHAR(45),
    ports JSONB,
    resources JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    stopped_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_runtimes_agent_id ON runtimes(agent_id);
CREATE INDEX IF NOT EXISTS idx_runtimes_node_id ON runtimes(node_id);
CREATE INDEX IF NOT EXISTS idx_runtimes_status ON runtimes(status);

-- ==========================================================================
-- 16. security_policies — 安全策略
-- ==========================================================================
CREATE TABLE IF NOT EXISTS security_policies (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    tool_permissions JSONB,
    resource_limits JSONB,
    network_policy JSONB,
    sandbox_policy JSONB,
    is_builtin BOOLEAN NOT NULL DEFAULT FALSE,
    category VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_security_policies_is_builtin ON security_policies(is_builtin);
CREATE INDEX IF NOT EXISTS idx_security_policies_category ON security_policies(category);

-- 内置安全策略
INSERT INTO security_policies (id, name, description, tool_permissions, resource_limits, network_policy, is_builtin, category)
VALUES
    ('builtin-strict', '严格策略', '最小权限原则', '[{"tool": "file_read", "permission": "allowed"}]', '{"max_cpu": "1.0", "max_memory": "2Gi"}', '{"allow_internet": false}', TRUE, 'security'),
    ('builtin-standard', '标准策略', '平衡安全与便利', '[{"tool": "file_*", "permission": "allowed"}]', '{"max_cpu": "2.0", "max_memory": "4Gi"}', '{"allow_internet": true}', TRUE, 'development'),
    ('builtin-permissive', '宽松策略', '较少限制', '[{"tool": "*", "permission": "allowed"}]', '{"max_cpu": "4.0", "max_memory": "8Gi"}', '{"allow_internet": true}', TRUE, 'trusted')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- ==========================================================================
-- 17. sandboxes — 沙箱实例
-- ==========================================================================
CREATE TABLE IF NOT EXISTS sandboxes (
    id VARCHAR(64) PRIMARY KEY,
    agent_id VARCHAR(64) NOT NULL,
    type VARCHAR(32) NOT NULL DEFAULT 'container',
    status VARCHAR(32) NOT NULL DEFAULT 'creating',
    isolation VARCHAR(64),
    fs_root TEXT,
    net_ns VARCHAR(255),
    resource_limits JSONB,
    runtime_id VARCHAR(64),
    node_id VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    destroyed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_sandboxes_agent_id ON sandboxes(agent_id);
CREATE INDEX IF NOT EXISTS idx_sandboxes_status ON sandboxes(status);
CREATE INDEX IF NOT EXISTS idx_sandboxes_node_id ON sandboxes(node_id);

-- ==========================================================================
-- 18. HITL 表（审批、反馈、干预、确认）
-- ==========================================================================
CREATE TABLE IF NOT EXISTS approval_requests (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL,
    type VARCHAR(64) NOT NULL,
    status VARCHAR(32) DEFAULT 'pending',
    operation TEXT NOT NULL,
    reason TEXT,
    context JSONB,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_approval_requests_run_id ON approval_requests(run_id);
CREATE INDEX IF NOT EXISTS idx_approval_requests_status ON approval_requests(status);

CREATE TABLE IF NOT EXISTS approval_decisions (
    id VARCHAR(64) PRIMARY KEY,
    request_id VARCHAR(64) NOT NULL REFERENCES approval_requests(id) ON DELETE CASCADE,
    decision VARCHAR(32) NOT NULL,
    decided_by VARCHAR(128) NOT NULL,
    comment TEXT,
    instructions TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_approval_decisions_request_id ON approval_decisions(request_id);

CREATE TABLE IF NOT EXISTS human_feedbacks (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL,
    type VARCHAR(32) NOT NULL,
    content TEXT NOT NULL,
    created_by VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_human_feedbacks_run_id ON human_feedbacks(run_id);

CREATE TABLE IF NOT EXISTS interventions (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL,
    action VARCHAR(32) NOT NULL,
    reason TEXT,
    parameters JSONB,
    created_by VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    executed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_interventions_run_id ON interventions(run_id);

CREATE TABLE IF NOT EXISTS confirmations (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL,
    type VARCHAR(32) NOT NULL,
    message TEXT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    options JSONB,
    selected_option VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_confirmations_run_id ON confirmations(run_id);
CREATE INDEX IF NOT EXISTS idx_confirmations_status ON confirmations(status);

-- ==========================================================================
-- 19. skills — 技能系统
-- ==========================================================================
CREATE TABLE IF NOT EXISTS skills (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(32) NOT NULL,
    level VARCHAR(32) NOT NULL,
    description TEXT,
    instructions TEXT,
    tools JSONB,
    examples JSONB,
    parameters JSONB,
    source VARCHAR(32) NOT NULL,
    author_id VARCHAR(64),
    registry_id VARCHAR(64),
    version VARCHAR(32) NOT NULL DEFAULT '1.0.0',
    is_builtin BOOLEAN NOT NULL DEFAULT FALSE,
    tags JSONB,
    use_count BIGINT NOT NULL DEFAULT 0,
    rating FLOAT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_skills_category ON skills(category);
CREATE INDEX IF NOT EXISTS idx_skills_is_builtin ON skills(is_builtin);

CREATE TABLE IF NOT EXISTS agent_skills (
    agent_id VARCHAR(64) NOT NULL,
    skill_id VARCHAR(64) NOT NULL REFERENCES skills(id),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    config JSONB,
    priority INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (agent_id, skill_id)
);

-- ==========================================================================
-- 20. MCP — Model Context Protocol
-- ==========================================================================
CREATE TABLE IF NOT EXISTS mcp_servers (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    source VARCHAR(32) NOT NULL,
    transport VARCHAR(32) NOT NULL,
    command VARCHAR(512),
    args JSONB,
    url VARCHAR(512),
    headers JSONB,
    capabilities JSONB,
    version VARCHAR(32) NOT NULL DEFAULT '1.0.0',
    author VARCHAR(128),
    repository VARCHAR(512),
    is_builtin BOOLEAN NOT NULL DEFAULT FALSE,
    tags JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_mcp_servers_source ON mcp_servers(source);
CREATE INDEX IF NOT EXISTS idx_mcp_servers_is_builtin ON mcp_servers(is_builtin);

CREATE TABLE IF NOT EXISTS agent_mcp_servers (
    agent_id VARCHAR(64) NOT NULL,
    mcp_server_id VARCHAR(64) NOT NULL REFERENCES mcp_servers(id),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    config JSONB,
    priority INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (agent_id, mcp_server_id)
);

-- ==========================================================================
-- 21. task_templates — 任务模板
-- ==========================================================================
CREATE TABLE IF NOT EXISTS task_templates (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(32) NOT NULL,
    description TEXT,
    prompt_template JSONB,
    default_workspace JSONB,
    default_security JSONB,
    default_labels JSONB,
    variables JSONB,
    is_builtin BOOLEAN NOT NULL DEFAULT FALSE,
    category VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_task_templates_type ON task_templates(type);
CREATE INDEX IF NOT EXISTS idx_task_templates_is_builtin ON task_templates(is_builtin);

-- ==========================================================================
-- 22. prompt_templates — 提示词模板
-- ==========================================================================
CREATE TABLE IF NOT EXISTS prompt_templates (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    content TEXT NOT NULL,
    variables JSONB DEFAULT '[]',
    category VARCHAR(64),
    tags JSONB DEFAULT '[]',
    source VARCHAR(32) DEFAULT 'custom',
    source_ref VARCHAR(200),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ==========================================================================
-- 23. memories — 记忆系统
-- ==========================================================================
CREATE TABLE IF NOT EXISTS memories (
    id VARCHAR(64) PRIMARY KEY,
    agent_id VARCHAR(64) NOT NULL,
    type VARCHAR(32) NOT NULL,
    content TEXT NOT NULL,
    metadata JSONB,
    importance FLOAT NOT NULL DEFAULT 0.5,
    tags JSONB,
    source_run_id VARCHAR(64),
    source_task_id VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_memories_agent_id ON memories(agent_id);
CREATE INDEX IF NOT EXISTS idx_memories_type ON memories(type);

-- ==========================================================================
-- 24. node_provisions — 节点部署记录
-- ==========================================================================
CREATE TABLE IF NOT EXISTS node_provisions (
    id              TEXT PRIMARY KEY,
    node_id         TEXT NOT NULL,
    host            TEXT NOT NULL,
    port            INTEGER NOT NULL DEFAULT 22,
    ssh_user        TEXT NOT NULL,
    auth_method     TEXT NOT NULL DEFAULT 'password',
    status          TEXT NOT NULL DEFAULT 'pending',
    error_message   TEXT,
    version         TEXT NOT NULL,
    github_repo     TEXT NOT NULL DEFAULT 'org/agents-admin',
    api_server_url  TEXT NOT NULL,
    tenant_id       VARCHAR(36) DEFAULT 'system',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_node_provisions_status ON node_provisions(status);
CREATE INDEX IF NOT EXISTS idx_node_provisions_node_id ON node_provisions(node_id);
CREATE INDEX IF NOT EXISTS idx_node_provisions_tenant ON node_provisions(tenant_id);

-- ==========================================================================
-- 25. users — 用户认证
-- ==========================================================================
CREATE TABLE IF NOT EXISTS users (
    id            VARCHAR(36) PRIMARY KEY,
    email         VARCHAR(255) NOT NULL UNIQUE,
    username      VARCHAR(100) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role          VARCHAR(20)  NOT NULL DEFAULT 'user',
    status        VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- 多租户: 给现有表添加 tenant_id 列
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(36) DEFAULT 'system';
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(36) DEFAULT 'system';
ALTER TABLE instances ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(36) DEFAULT 'system';
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(36) DEFAULT 'system';
ALTER TABLE operations ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(36) DEFAULT 'system';

CREATE INDEX IF NOT EXISTS idx_tasks_tenant ON tasks(tenant_id);
CREATE INDEX IF NOT EXISTS idx_accounts_tenant ON accounts(tenant_id);
CREATE INDEX IF NOT EXISTS idx_instances_tenant ON instances(tenant_id);
CREATE INDEX IF NOT EXISTS idx_proxies_tenant ON proxies(tenant_id);
CREATE INDEX IF NOT EXISTS idx_operations_tenant ON operations(tenant_id);

-- ==========================================================================
-- 26. schema_version — 数据库版本追踪
-- ==========================================================================
CREATE TABLE IF NOT EXISTS schema_version (
    id              INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    migration_id    INTEGER NOT NULL,
    version         VARCHAR(32) NOT NULL,
    description     TEXT,
    applied_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    applied_by      VARCHAR(128) DEFAULT current_user
);

INSERT INTO schema_version (migration_id, version, description)
VALUES (19, '1.0.0', '全量安装：合并 migrations 002-019')
ON CONFLICT (id) DO UPDATE SET
    migration_id = EXCLUDED.migration_id,
    version = EXCLUDED.version,
    description = EXCLUDED.description,
    applied_at = NOW();

-- ==========================================================================
-- 完成
-- ==========================================================================
-- 本脚本包含 26 张表，覆盖：
--   核心: tasks, runs, events, nodes, artifacts
--   账号: accounts, auth_tasks, instances, terminal_sessions
--   基础设施: proxies, operations, actions, node_provisions
--   智能体: agent_templates, agents, runtimes
--   安全: security_policies, sandboxes
--   HITL: approval_requests, approval_decisions, human_feedbacks, interventions, confirmations
--   扩展: skills, mcp_servers, task_templates, prompt_templates, memories
--   系统: users, schema_version

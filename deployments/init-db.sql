-- Agent Kanban 数据库初始化脚本
-- 版本: 2.0 - 支持任务类型和灵活工作空间

-- 启用 UUID 扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================
-- 任务表
-- 存储任务定义（TaskSpec），支持多种任务类型
-- ============================================================
CREATE TABLE IF NOT EXISTS tasks (
    id VARCHAR(20) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    -- 任务类型：general, development, operation, research, automation, review
    type VARCHAR(20) NOT NULL DEFAULT 'general',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    -- spec 存储完整的 TaskSpec JSON，包含 Agent、Workspace、Security 配置
    spec JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Run 表
-- 存储任务执行实例，包含运行时快照
-- ============================================================
CREATE TABLE IF NOT EXISTS runs (
    id VARCHAR(20) PRIMARY KEY,
    -- 使用 ON DELETE CASCADE 允许删除任务时自动删除关联的 runs
    task_id VARCHAR(20) NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'queued',
    node_id VARCHAR(50),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    -- snapshot 存储执行时的 TaskSpec 快照，用于审计和复现
    snapshot JSONB,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- 事件表
-- 存储 Run 执行过程中的 CanonicalEvent
-- ============================================================
CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    -- 使用 ON DELETE CASCADE 允许删除 run 时自动删除关联的事件
    run_id VARCHAR(20) NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    seq INTEGER NOT NULL,
    -- 事件类型：run_started, message, tool_use_start, file_write, error 等
    type VARCHAR(50) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    -- payload 存储事件载荷
    payload JSONB,
    -- raw 存储原始 CLI 输出（用于调试和回放）
    raw TEXT,
    UNIQUE(run_id, seq)
);

-- ============================================================
-- 节点表
-- 存储 Node Agent 注册信息和状态
-- ============================================================
CREATE TABLE IF NOT EXISTS nodes (
    id VARCHAR(50) PRIMARY KEY,
    -- 节点状态：online, offline, draining, disabled, maintenance
    status VARCHAR(20) NOT NULL DEFAULT 'offline',
    -- labels 存储节点标签（用于调度匹配）
    labels JSONB DEFAULT '{}',
    -- capacity 存储节点容量信息
    capacity JSONB DEFAULT '{}',
    last_heartbeat TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- 产物表
-- 存储 Run 执行产物（事件日志、代码变更、输出文件等）
-- ============================================================
CREATE TABLE IF NOT EXISTS artifacts (
    id BIGSERIAL PRIMARY KEY,
    -- 使用 ON DELETE CASCADE 允许删除 run 时自动删除关联的产物
    run_id VARCHAR(20) NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    path VARCHAR(500) NOT NULL,
    size BIGINT,
    content_type VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- 索引
-- ============================================================
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_type ON tasks(type);
CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_runs_task_id ON runs(task_id);
CREATE INDEX IF NOT EXISTS idx_runs_status ON runs(status);
CREATE INDEX IF NOT EXISTS idx_runs_node_id ON runs(node_id);
CREATE INDEX IF NOT EXISTS idx_events_run_id ON events(run_id);
CREATE INDEX IF NOT EXISTS idx_events_run_id_seq ON events(run_id, seq);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);
CREATE INDEX IF NOT EXISTS idx_nodes_status ON nodes(status);

-- ============================================================
-- 账号表
-- 存储 AI Agent 认证账号信息
-- ============================================================
CREATE TABLE IF NOT EXISTS accounts (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    agent_type_id VARCHAR(64) NOT NULL,
    node_id VARCHAR(64) NOT NULL,  -- 账号所属节点（当前阶段必填，未来共享存储后可选）
    volume_name VARCHAR(255),  -- Docker Volume 名称（由 Node Agent 创建后回填）
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    -- pending: 待认证
    -- authenticating: 认证中
    -- authenticated: 已认证
    -- expired: 认证过期
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ
);

-- ============================================================
-- 认证任务表
-- 存储认证任务的期望状态和当前状态（控制面/数据面分离）
-- ============================================================
CREATE TABLE IF NOT EXISTS auth_tasks (
    id VARCHAR(64) PRIMARY KEY,
    account_id VARCHAR(64) NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    
    -- 期望状态（由 API Server 设置）
    method VARCHAR(32) NOT NULL,  -- oauth, api_key
    
    -- 节点信息（由用户指定，不走 Scheduler）
    node_id VARCHAR(64) NOT NULL,
    
    -- 当前状态（由 Node Agent 上报）
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    -- pending: 待调度
    -- assigned: 已分配节点
    -- running: 执行中
    -- waiting_user: 等待用户操作（终端已启动）
    -- success: 认证成功
    -- failed: 认证失败
    -- timeout: 超时
    
    terminal_port INT,            -- Node Agent 上报的终端端口
    terminal_url VARCHAR(255),    -- 终端访问 URL
    container_name VARCHAR(255),  -- 容器名称
    message TEXT,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_accounts_status ON accounts(status);
CREATE INDEX IF NOT EXISTS idx_accounts_agent_type ON accounts(agent_type_id);
CREATE INDEX IF NOT EXISTS idx_auth_tasks_status ON auth_tasks(status);
CREATE INDEX IF NOT EXISTS idx_auth_tasks_node_id ON auth_tasks(node_id);
CREATE INDEX IF NOT EXISTS idx_auth_tasks_account_id ON auth_tasks(account_id);

-- 更新时间触发器
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tasks_updated_at
    BEFORE UPDATE ON tasks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER runs_updated_at
    BEFORE UPDATE ON runs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER nodes_updated_at
    BEFORE UPDATE ON nodes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER accounts_updated_at
    BEFORE UPDATE ON accounts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER auth_tasks_updated_at
    BEFORE UPDATE ON auth_tasks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

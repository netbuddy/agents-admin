-- Migration: 003_add_instances_terminals
-- Description: 添加 instances 和 terminal_sessions 表，支持状态持久化
-- Date: 2026-01-30

-- instances 表：存储容器实例
CREATE TABLE IF NOT EXISTS instances (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    account_id VARCHAR(64) NOT NULL,
    agent_type_id VARCHAR(32) NOT NULL,
    container_name VARCHAR(255),
    node_id VARCHAR(64),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_instances_account ON instances(account_id);
CREATE INDEX IF NOT EXISTS idx_instances_status ON instances(status);
CREATE INDEX IF NOT EXISTS idx_instances_node ON instances(node_id);

-- terminal_sessions 表：存储终端会话
CREATE TABLE IF NOT EXISTS terminal_sessions (
    id VARCHAR(64) PRIMARY KEY,
    instance_id VARCHAR(64),
    container_name VARCHAR(255) NOT NULL,
    node_id VARCHAR(64),
    port INTEGER,
    url VARCHAR(512),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_terminal_sessions_instance ON terminal_sessions(instance_id);
CREATE INDEX IF NOT EXISTS idx_terminal_sessions_status ON terminal_sessions(status);
CREATE INDEX IF NOT EXISTS idx_terminal_sessions_node ON terminal_sessions(node_id);

-- 触发器：自动更新 updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS update_instances_updated_at ON instances;
CREATE TRIGGER update_instances_updated_at
    BEFORE UPDATE ON instances
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

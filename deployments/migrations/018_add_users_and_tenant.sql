-- 018: 用户表 + 多租户 tenant_id 列
-- 创建 users 表用于用户认证，给现有业务表添加 tenant_id 列实现多租户隔离

BEGIN;

-- 1. 创建 users 表
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

-- 2. 给现有表添加 tenant_id 列（默认 'system'，后续归属到管理员）
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(36) DEFAULT 'system';
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(36) DEFAULT 'system';
ALTER TABLE instances ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(36) DEFAULT 'system';
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(36) DEFAULT 'system';
ALTER TABLE operations ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(36) DEFAULT 'system';
ALTER TABLE node_provisions ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(36) DEFAULT 'system';

-- 3. 给 tenant_id 列添加索引
CREATE INDEX IF NOT EXISTS idx_tasks_tenant ON tasks(tenant_id);
CREATE INDEX IF NOT EXISTS idx_accounts_tenant ON accounts(tenant_id);
CREATE INDEX IF NOT EXISTS idx_instances_tenant ON instances(tenant_id);
CREATE INDEX IF NOT EXISTS idx_proxies_tenant ON proxies(tenant_id);
CREATE INDEX IF NOT EXISTS idx_operations_tenant ON operations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_node_provisions_tenant ON node_provisions(tenant_id);

COMMIT;

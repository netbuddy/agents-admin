-- 014_operations_and_actions.sql
-- 创建 Operation 和 Action 表
-- 用于系统操作（认证、运行时管理等）的管理

-- ============================================================================
-- Operations 表 - 系统操作定义
-- ============================================================================

CREATE TABLE IF NOT EXISTS operations (
    id VARCHAR(64) PRIMARY KEY,
    type VARCHAR(32) NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    node_id VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    
    CONSTRAINT fk_operations_node FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE SET NULL
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_operations_type ON operations(type);
CREATE INDEX IF NOT EXISTS idx_operations_status ON operations(status);
CREATE INDEX IF NOT EXISTS idx_operations_node_id ON operations(node_id);
CREATE INDEX IF NOT EXISTS idx_operations_created_at ON operations(created_at DESC);

-- 注释
COMMENT ON TABLE operations IS '系统操作定义（认证、运行时管理等）';
COMMENT ON COLUMN operations.id IS '操作 ID，格式：op-{random}';
COMMENT ON COLUMN operations.type IS '操作类型：oauth, api_key, device_code, runtime_create, runtime_start, runtime_stop, runtime_destroy';
COMMENT ON COLUMN operations.config IS '操作配置（JSON）';
COMMENT ON COLUMN operations.status IS '操作状态：pending, in_progress, completed, failed, cancelled';
COMMENT ON COLUMN operations.node_id IS '目标节点 ID';

-- ============================================================================
-- Actions 表 - 操作执行实例
-- ============================================================================

CREATE TABLE IF NOT EXISTS actions (
    id VARCHAR(64) PRIMARY KEY,
    operation_id VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'assigned',
    progress INT NOT NULL DEFAULT 0,
    result JSONB,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    
    CONSTRAINT fk_actions_operation FOREIGN KEY (operation_id) REFERENCES operations(id) ON DELETE CASCADE
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_actions_operation_id ON actions(operation_id);
CREATE INDEX IF NOT EXISTS idx_actions_status ON actions(status);
CREATE INDEX IF NOT EXISTS idx_actions_created_at ON actions(created_at DESC);

-- 注释
COMMENT ON TABLE actions IS 'Operation 的执行实例';
COMMENT ON COLUMN actions.id IS 'Action ID，格式：act-{random}';
COMMENT ON COLUMN actions.operation_id IS '关联的 Operation ID';
COMMENT ON COLUMN actions.status IS '执行状态：assigned, running, waiting, success, failed, timeout, cancelled';
COMMENT ON COLUMN actions.progress IS '执行进度 (0-100)';
COMMENT ON COLUMN actions.result IS '执行结果（JSON）';
COMMENT ON COLUMN actions.error IS '错误信息';

-- ============================================================================
-- 更新触发器
-- ============================================================================

-- Operations 更新时间触发器
CREATE OR REPLACE FUNCTION update_operations_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_operations_updated_at
    BEFORE UPDATE ON operations
    FOR EACH ROW
    EXECUTE FUNCTION update_operations_updated_at();

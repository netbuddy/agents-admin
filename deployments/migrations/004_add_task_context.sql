-- 添加任务上下文和父任务字段
-- 版本: 004
-- 日期: 2026-01-30

-- ============================================================
-- 添加 parent_id 字段（支持任务层级）
-- ============================================================
ALTER TABLE tasks 
ADD COLUMN IF NOT EXISTS parent_id VARCHAR(20) REFERENCES tasks(id) ON DELETE SET NULL;

-- ============================================================
-- 添加 context 字段（存储 TaskContext JSON）
-- ============================================================
ALTER TABLE tasks
ADD COLUMN IF NOT EXISTS context JSONB DEFAULT '{}';

-- ============================================================
-- 添加 instance_id 字段（关联执行实例）
-- ============================================================
ALTER TABLE tasks
ADD COLUMN IF NOT EXISTS instance_id VARCHAR(64) REFERENCES instances(id) ON DELETE SET NULL;

-- ============================================================
-- 索引
-- ============================================================
CREATE INDEX IF NOT EXISTS idx_tasks_parent_id ON tasks(parent_id);
CREATE INDEX IF NOT EXISTS idx_tasks_instance_id ON tasks(instance_id);

-- ============================================================
-- 添加 GIN 索引用于 context JSONB 查询
-- ============================================================
CREATE INDEX IF NOT EXISTS idx_tasks_context ON tasks USING GIN (context);

-- ============================================================
-- 注释
-- ============================================================
COMMENT ON COLUMN tasks.parent_id IS '父任务 ID，用于任务层级结构';
COMMENT ON COLUMN tasks.context IS '任务上下文 JSON，包含继承的上下文和对话历史';
COMMENT ON COLUMN tasks.instance_id IS '执行实例 ID，关联到 instances 表';

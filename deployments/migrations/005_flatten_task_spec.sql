-- ============================================================================
-- 005_flatten_task_spec.sql
-- 将 TaskSpec 字段扁平化合并到 Task 表中
-- ============================================================================

-- 1. 添加新字段
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS type VARCHAR(32) DEFAULT 'general';
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS prompt TEXT;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS workspace JSONB;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS security JSONB;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS labels JSONB;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS template_id VARCHAR(64);
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS agent_id VARCHAR(64);

-- 2. 迁移现有数据（从 spec JSONB 提取）
UPDATE tasks SET
    type = COALESCE(spec->>'type', 'general'),
    prompt = spec->>'prompt',
    workspace = spec->'workspace',
    security = spec->'security',
    labels = spec->'labels'
WHERE spec IS NOT NULL AND prompt IS NULL;

-- 3. 将 instance_id 迁移到 agent_id
UPDATE tasks SET agent_id = instance_id WHERE agent_id IS NULL AND instance_id IS NOT NULL;

-- 4. 创建索引
CREATE INDEX IF NOT EXISTS idx_tasks_type ON tasks(type);
CREATE INDEX IF NOT EXISTS idx_tasks_template_id ON tasks(template_id);
CREATE INDEX IF NOT EXISTS idx_tasks_agent_id ON tasks(agent_id);

-- 5. 注意：暂时保留 spec 和 instance_id 字段以便回滚
-- 在确认迁移成功后，可以在后续版本中删除：
-- ALTER TABLE tasks DROP COLUMN spec;
-- ALTER TABLE tasks DROP COLUMN instance_id;

-- ============================================================================
-- 回滚脚本（如需回滚，手动执行）
-- ============================================================================
-- ALTER TABLE tasks DROP COLUMN IF EXISTS type;
-- ALTER TABLE tasks DROP COLUMN IF EXISTS prompt;
-- ALTER TABLE tasks DROP COLUMN IF EXISTS workspace;
-- ALTER TABLE tasks DROP COLUMN IF EXISTS security;
-- ALTER TABLE tasks DROP COLUMN IF EXISTS labels;
-- ALTER TABLE tasks DROP COLUMN IF EXISTS template_id;
-- ALTER TABLE tasks DROP COLUMN IF EXISTS agent_id;
-- DROP INDEX IF EXISTS idx_tasks_type;
-- DROP INDEX IF EXISTS idx_tasks_template_id;
-- DROP INDEX IF EXISTS idx_tasks_agent_id;

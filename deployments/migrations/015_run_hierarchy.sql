-- 015_run_hierarchy.sql
-- 为 runs 表添加层次化支持字段
-- 支持父子任务关系和任务分解

-- ============================================================================
-- 为 runs 表添加层次化字段
-- ============================================================================

-- 添加父子关系字段
ALTER TABLE runs ADD COLUMN IF NOT EXISTS parent_id VARCHAR(64);
ALTER TABLE runs ADD COLUMN IF NOT EXISTS root_id VARCHAR(64);
ALTER TABLE runs ADD COLUMN IF NOT EXISTS depth INT NOT NULL DEFAULT 0;

-- 添加执行阶段字段
ALTER TABLE runs ADD COLUMN IF NOT EXISTS phase VARCHAR(64);

-- 添加外键约束（允许自引用）
ALTER TABLE runs ADD CONSTRAINT fk_runs_parent 
    FOREIGN KEY (parent_id) REFERENCES runs(id) ON DELETE SET NULL;

-- 索引
CREATE INDEX IF NOT EXISTS idx_runs_parent_id ON runs(parent_id);
CREATE INDEX IF NOT EXISTS idx_runs_root_id ON runs(root_id);
CREATE INDEX IF NOT EXISTS idx_runs_depth ON runs(depth);
CREATE INDEX IF NOT EXISTS idx_runs_phase ON runs(phase);

-- 注释
COMMENT ON COLUMN runs.parent_id IS '父 Run ID（如果是子任务）';
COMMENT ON COLUMN runs.root_id IS '根 Run ID（最顶层）';
COMMENT ON COLUMN runs.depth IS '嵌套深度（0 表示顶层）';
COMMENT ON COLUMN runs.phase IS '当前执行阶段（任务特定）';

-- ============================================================================
-- 为 tasks 表添加执行模式字段
-- ============================================================================

-- 添加任务执行模式
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS mode VARCHAR(32) NOT NULL DEFAULT 'simple';

-- 添加任务分解定义
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS decomposition JSONB;

-- 添加父关闭策略
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS parent_close_policy VARCHAR(32) NOT NULL DEFAULT 'terminate';

-- 索引
CREATE INDEX IF NOT EXISTS idx_tasks_mode ON tasks(mode);

-- 注释
COMMENT ON COLUMN tasks.mode IS '任务执行模式：simple, composite, interactive, long_running';
COMMENT ON COLUMN tasks.decomposition IS '任务分解定义（JSON，用于 composite 模式）';
COMMENT ON COLUMN tasks.parent_close_policy IS '父关闭策略：terminate, abandon, request_cancel';

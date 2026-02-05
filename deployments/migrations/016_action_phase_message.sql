-- 016_action_phase_message.sql
-- 为 actions 表添加语义阶段（phase）和消息（message）字段
-- 参考 Kubernetes Pod Lifecycle 的三层状态模型

ALTER TABLE actions ADD COLUMN IF NOT EXISTS phase VARCHAR(64) DEFAULT '';
ALTER TABLE actions ADD COLUMN IF NOT EXISTS message TEXT DEFAULT '';

-- 索引：按 phase 查询（如查看所有处于 waiting_oauth 阶段的 Action）
CREATE INDEX IF NOT EXISTS idx_actions_phase ON actions(phase) WHERE phase != '';

-- 注释
COMMENT ON COLUMN actions.phase IS '语义阶段：launching_container, waiting_oauth, extracting_token 等';
COMMENT ON COLUMN actions.message IS '人类可读的状态描述';

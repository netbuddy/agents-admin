-- 023: 重命名 instances 表为 agents，对齐领域模型
-- 设计决策：移除 Runner 概念，Agent 直接作为执行者
-- 参见：internal-docs/design/core-model/02-概念全景图.md

-- 重命名表
ALTER TABLE instances RENAME TO agents;

-- 添加 Agent 模型新字段（与 model.Agent 对齐）
ALTER TABLE agents ADD COLUMN IF NOT EXISTS type VARCHAR(50) DEFAULT '';
ALTER TABLE agents ADD COLUMN IF NOT EXISTS runtime_id VARCHAR(100);
ALTER TABLE agents ADD COLUMN IF NOT EXISTS current_task_id VARCHAR(100);
ALTER TABLE agents ADD COLUMN IF NOT EXISTS config_overrides JSONB;
ALTER TABLE agents ADD COLUMN IF NOT EXISTS security_policy_id VARCHAR(100);
ALTER TABLE agents ADD COLUMN IF NOT EXISTS memory_enabled BOOLEAN DEFAULT FALSE;
ALTER TABLE agents ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMP;

-- 从 agent_type_id 填充 type 字段（向后兼容）
UPDATE agents SET type = agent_type_id WHERE type = '' AND agent_type_id IS NOT NULL;

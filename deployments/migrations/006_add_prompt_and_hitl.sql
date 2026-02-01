-- 006_add_prompt_and_hitl.sql
-- 添加 Prompt 体系、TaskTemplate 和人在环路（HITL）相关表
--
-- 变更内容：
-- 1. 创建 prompt_templates 表（提示词模板）
-- 2. 创建 task_templates 表（任务模板）
-- 3. 更新 tasks 表：添加 description 字段，修改 prompt 为 JSONB 类型
-- 4. 创建 HITL 相关表：
--    - approval_requests（审批请求）
--    - approval_decisions（审批决定）
--    - human_feedbacks（人工反馈）
--    - interventions（干预）
--    - confirmations（确认请求）

-- ============================================================================
-- 1. 创建 prompt_templates 表
-- ============================================================================

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
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_prompt_templates_category ON prompt_templates(category);
CREATE INDEX IF NOT EXISTS idx_prompt_templates_source ON prompt_templates(source);
CREATE INDEX IF NOT EXISTS idx_prompt_templates_name ON prompt_templates(name);

COMMENT ON TABLE prompt_templates IS '提示词模板表';
COMMENT ON COLUMN prompt_templates.id IS '唯一标识';
COMMENT ON COLUMN prompt_templates.name IS '模板名称';
COMMENT ON COLUMN prompt_templates.description IS '模板描述';
COMMENT ON COLUMN prompt_templates.content IS '模板内容（支持变量插值）';
COMMENT ON COLUMN prompt_templates.variables IS '变量定义 JSONB';
COMMENT ON COLUMN prompt_templates.category IS '分类';
COMMENT ON COLUMN prompt_templates.tags IS '标签列表 JSONB';
COMMENT ON COLUMN prompt_templates.source IS '来源：builtin/custom/mcp';
COMMENT ON COLUMN prompt_templates.source_ref IS '来源引用（如 MCP Server ID）';

-- ============================================================================
-- 2. 创建 task_templates 表（任务模板）
-- ============================================================================

CREATE TABLE IF NOT EXISTS task_templates (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    type VARCHAR(32) DEFAULT 'general',
    prompt_template JSONB,
    prompt_template_id VARCHAR(64) REFERENCES prompt_templates(id),
    default_workspace JSONB,
    default_security JSONB,
    default_labels JSONB DEFAULT '{}',
    variables JSONB DEFAULT '[]',
    category VARCHAR(64),
    tags JSONB DEFAULT '[]',
    source VARCHAR(32) DEFAULT 'custom',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_task_templates_type ON task_templates(type);
CREATE INDEX IF NOT EXISTS idx_task_templates_category ON task_templates(category);
CREATE INDEX IF NOT EXISTS idx_task_templates_source ON task_templates(source);
CREATE INDEX IF NOT EXISTS idx_task_templates_name ON task_templates(name);

COMMENT ON TABLE task_templates IS '任务模板表';
COMMENT ON COLUMN task_templates.id IS '唯一标识';
COMMENT ON COLUMN task_templates.name IS '模板名称';
COMMENT ON COLUMN task_templates.description IS '模板描述';
COMMENT ON COLUMN task_templates.type IS '任务类型：general/development/research/operation/automation';
COMMENT ON COLUMN task_templates.prompt_template IS '内嵌的提示词模板 JSONB';
COMMENT ON COLUMN task_templates.prompt_template_id IS '引用的提示词模板 ID';
COMMENT ON COLUMN task_templates.default_workspace IS '默认工作空间配置 JSONB';
COMMENT ON COLUMN task_templates.default_security IS '默认安全配置 JSONB';
COMMENT ON COLUMN task_templates.default_labels IS '默认标签 JSONB';
COMMENT ON COLUMN task_templates.variables IS '变量定义 JSONB';
COMMENT ON COLUMN task_templates.source IS '来源：builtin/custom/shared';

-- ============================================================================
-- 3. 更新 tasks 表
-- ============================================================================

-- 添加 description 字段
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS description TEXT;

-- 备份旧的 prompt 字段（VARCHAR 类型）
-- 注意：如果 prompt 已经是 JSONB 类型，此操作可能失败，这是预期的
DO $$
BEGIN
    -- 检查 prompt 列是否存在且为字符类型
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'tasks' AND column_name = 'prompt' AND data_type IN ('character varying', 'text')
    ) THEN
        -- 临时列存储旧数据
        ALTER TABLE tasks ADD COLUMN IF NOT EXISTS prompt_text_backup TEXT;
        UPDATE tasks SET prompt_text_backup = prompt WHERE prompt IS NOT NULL;
        
        -- 删除旧列并创建新的 JSONB 列
        ALTER TABLE tasks DROP COLUMN prompt;
        ALTER TABLE tasks ADD COLUMN prompt JSONB;
        
        -- 迁移数据：将旧的文本 prompt 转换为 JSONB 格式
        UPDATE tasks
        SET prompt = jsonb_build_object(
            'content', prompt_text_backup,
            'description', ''
        )
        WHERE prompt_text_backup IS NOT NULL AND prompt IS NULL;
        
        -- 删除备份列
        ALTER TABLE tasks DROP COLUMN IF EXISTS prompt_text_backup;
    END IF;
END $$;

COMMENT ON COLUMN tasks.description IS '任务描述';
COMMENT ON COLUMN tasks.prompt IS '结构化提示词 JSONB';

-- ============================================================================
-- 4. 创建 approval_requests 表（审批请求）
-- ============================================================================

CREATE TABLE IF NOT EXISTS approval_requests (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    type VARCHAR(64) NOT NULL,
    status VARCHAR(32) DEFAULT 'pending',
    operation TEXT NOT NULL,
    reason TEXT,
    context JSONB,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP WITH TIME ZONE
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_approval_requests_run_id ON approval_requests(run_id);
CREATE INDEX IF NOT EXISTS idx_approval_requests_status ON approval_requests(status);
CREATE INDEX IF NOT EXISTS idx_approval_requests_type ON approval_requests(type);
CREATE INDEX IF NOT EXISTS idx_approval_requests_expires_at ON approval_requests(expires_at);

COMMENT ON TABLE approval_requests IS '审批请求表';
COMMENT ON COLUMN approval_requests.type IS '审批类型：dangerous_operation/sensitive_access/resource_exceeded/external_access';
COMMENT ON COLUMN approval_requests.status IS '状态：pending/approved/rejected/expired';
COMMENT ON COLUMN approval_requests.operation IS '请求的操作描述';
COMMENT ON COLUMN approval_requests.context IS '操作上下文 JSONB';

-- ============================================================================
-- 5. 创建 approval_decisions 表（审批决定）
-- ============================================================================

CREATE TABLE IF NOT EXISTS approval_decisions (
    id VARCHAR(64) PRIMARY KEY,
    request_id VARCHAR(64) NOT NULL REFERENCES approval_requests(id) ON DELETE CASCADE,
    decision VARCHAR(32) NOT NULL,
    decided_by VARCHAR(64) NOT NULL,
    comment TEXT,
    instructions TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_approval_decisions_request_id ON approval_decisions(request_id);
CREATE INDEX IF NOT EXISTS idx_approval_decisions_decided_by ON approval_decisions(decided_by);

COMMENT ON TABLE approval_decisions IS '审批决定表';
COMMENT ON COLUMN approval_decisions.decision IS '决策：approve/reject';
COMMENT ON COLUMN approval_decisions.instructions IS '附加指令';

-- ============================================================================
-- 6. 创建 human_feedbacks 表（人工反馈）
-- ============================================================================

CREATE TABLE IF NOT EXISTS human_feedbacks (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    type VARCHAR(32) NOT NULL,
    content TEXT NOT NULL,
    created_by VARCHAR(64) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP WITH TIME ZONE
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_human_feedbacks_run_id ON human_feedbacks(run_id);
CREATE INDEX IF NOT EXISTS idx_human_feedbacks_type ON human_feedbacks(type);
CREATE INDEX IF NOT EXISTS idx_human_feedbacks_created_by ON human_feedbacks(created_by);

COMMENT ON TABLE human_feedbacks IS '人工反馈表';
COMMENT ON COLUMN human_feedbacks.type IS '反馈类型：guidance/correction/clarification';
COMMENT ON COLUMN human_feedbacks.processed_at IS 'Agent 确认收到反馈的时间';

-- ============================================================================
-- 7. 创建 interventions 表（干预）
-- ============================================================================

CREATE TABLE IF NOT EXISTS interventions (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    action VARCHAR(32) NOT NULL,
    reason TEXT,
    parameters JSONB,
    created_by VARCHAR(64) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    executed_at TIMESTAMP WITH TIME ZONE
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_interventions_run_id ON interventions(run_id);
CREATE INDEX IF NOT EXISTS idx_interventions_action ON interventions(action);
CREATE INDEX IF NOT EXISTS idx_interventions_created_by ON interventions(created_by);

COMMENT ON TABLE interventions IS '干预表';
COMMENT ON COLUMN interventions.action IS '干预动作：pause/resume/cancel/modify';
COMMENT ON COLUMN interventions.parameters IS '参数（modify 时使用）';

-- ============================================================================
-- 8. 创建 confirmations 表（确认请求）
-- ============================================================================

CREATE TABLE IF NOT EXISTS confirmations (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    type VARCHAR(32) NOT NULL,
    message TEXT NOT NULL,
    status VARCHAR(32) DEFAULT 'pending',
    options JSONB DEFAULT '[]',
    selected_option VARCHAR(200),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP WITH TIME ZONE
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_confirmations_run_id ON confirmations(run_id);
CREATE INDEX IF NOT EXISTS idx_confirmations_type ON confirmations(type);
CREATE INDEX IF NOT EXISTS idx_confirmations_status ON confirmations(status);

COMMENT ON TABLE confirmations IS '确认请求表';
COMMENT ON COLUMN confirmations.type IS '确认类型：deployment/deletion/payment/irreversible';
COMMENT ON COLUMN confirmations.status IS '状态：pending/confirmed/cancelled';
COMMENT ON COLUMN confirmations.options IS '可选项列表 JSONB';

-- 009_hitl_tables.sql
-- HITL（人在环路）表结构
-- 阶段6：审批、反馈、干预、确认

-- ============================================================================
-- ApprovalRequests 表 - 审批请求
-- ============================================================================

CREATE TABLE IF NOT EXISTS approval_requests (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL,
    type VARCHAR(32) NOT NULL,           -- dangerous_operation, sensitive_access, resource_exceeded, external_access
    status VARCHAR(32) NOT NULL DEFAULT 'pending',  -- pending, approved, rejected, expired
    operation TEXT NOT NULL,              -- 请求的操作描述
    reason TEXT,                          -- 请求原因
    context JSONB,                        -- 操作上下文
    expires_at TIMESTAMP,                 -- 过期时间
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP                 -- 处理时间
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_approval_requests_run_id ON approval_requests(run_id);
CREATE INDEX IF NOT EXISTS idx_approval_requests_status ON approval_requests(status);
CREATE INDEX IF NOT EXISTS idx_approval_requests_expires_at ON approval_requests(expires_at);

-- ============================================================================
-- ApprovalDecisions 表 - 审批决定
-- ============================================================================

CREATE TABLE IF NOT EXISTS approval_decisions (
    id VARCHAR(64) PRIMARY KEY,
    request_id VARCHAR(64) NOT NULL REFERENCES approval_requests(id),
    decision VARCHAR(16) NOT NULL,        -- approve, reject
    decided_by VARCHAR(128) NOT NULL,     -- 决策者 UserID
    comment TEXT,                         -- 审批意见
    instructions TEXT,                    -- 附加指令
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_approval_decisions_request_id ON approval_decisions(request_id);

-- ============================================================================
-- HumanFeedbacks 表 - 人工反馈
-- ============================================================================

CREATE TABLE IF NOT EXISTS human_feedbacks (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL,
    type VARCHAR(32) NOT NULL,            -- guidance, correction, clarification
    content TEXT NOT NULL,                -- 反馈内容
    created_by VARCHAR(128) NOT NULL,     -- 创建者 UserID
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP                -- 处理时间
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_human_feedbacks_run_id ON human_feedbacks(run_id);
CREATE INDEX IF NOT EXISTS idx_human_feedbacks_processed ON human_feedbacks(processed_at);

-- ============================================================================
-- Interventions 表 - 干预记录
-- ============================================================================

CREATE TABLE IF NOT EXISTS interventions (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL,
    action VARCHAR(32) NOT NULL,          -- pause, resume, cancel, modify
    reason TEXT,                          -- 干预原因
    parameters JSONB,                     -- 参数（modify 时使用）
    created_by VARCHAR(128) NOT NULL,     -- 创建者 UserID
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    executed_at TIMESTAMP                 -- 执行时间
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_interventions_run_id ON interventions(run_id);
CREATE INDEX IF NOT EXISTS idx_interventions_action ON interventions(action);

-- ============================================================================
-- Confirmations 表 - 确认请求
-- ============================================================================

CREATE TABLE IF NOT EXISTS confirmations (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL,
    type VARCHAR(32) NOT NULL,            -- deployment, deletion, payment, irreversible
    message TEXT NOT NULL,                -- 确认消息
    status VARCHAR(32) NOT NULL DEFAULT 'pending',  -- pending, confirmed, cancelled
    options JSONB,                        -- 可选项
    selected_option VARCHAR(128),         -- 用户选择的选项
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP                 -- 处理时间
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_confirmations_run_id ON confirmations(run_id);
CREATE INDEX IF NOT EXISTS idx_confirmations_status ON confirmations(status);

-- ============================================================================
-- 添加 runs 表的 paused 状态支持
-- ============================================================================

-- 注意：确保 runs 表的 status 字段支持 'paused' 状态
-- 如果使用枚举类型，需要添加新值
-- ALTER TYPE run_status ADD VALUE IF NOT EXISTS 'paused';

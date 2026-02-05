-- 008_security_and_sandbox.sql
-- SecurityPolicy + Sandbox 表结构
-- 阶段5：安全策略与沙箱

-- ============================================================================
-- SecurityPolicies 表 - 安全策略
-- ============================================================================

CREATE TABLE IF NOT EXISTS security_policies (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- 权限配置
    tool_permissions JSONB,
    
    -- 资源限制
    resource_limits JSONB,
    
    -- 网络策略
    network_policy JSONB,
    
    -- 沙箱策略
    sandbox_policy JSONB,
    
    -- 元数据
    is_builtin BOOLEAN NOT NULL DEFAULT FALSE,
    category VARCHAR(64),
    
    -- 时间戳
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_security_policies_is_builtin ON security_policies(is_builtin);
CREATE INDEX IF NOT EXISTS idx_security_policies_category ON security_policies(category);

-- ============================================================================
-- Sandboxes 表 - 沙箱实例
-- ============================================================================

CREATE TABLE IF NOT EXISTS sandboxes (
    id VARCHAR(64) PRIMARY KEY,
    agent_id VARCHAR(64) NOT NULL,
    type VARCHAR(32) NOT NULL DEFAULT 'container',
    status VARCHAR(32) NOT NULL DEFAULT 'creating',
    
    -- 隔离配置
    isolation VARCHAR(64),
    fs_root TEXT,
    net_ns VARCHAR(255),
    
    -- 资源配置
    resource_limits JSONB,
    
    -- 关联
    runtime_id VARCHAR(64),
    node_id VARCHAR(64) NOT NULL,
    
    -- 生命周期
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    expires_at TIMESTAMP,
    destroyed_at TIMESTAMP
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_sandboxes_agent_id ON sandboxes(agent_id);
CREATE INDEX IF NOT EXISTS idx_sandboxes_status ON sandboxes(status);
CREATE INDEX IF NOT EXISTS idx_sandboxes_node_id ON sandboxes(node_id);
CREATE INDEX IF NOT EXISTS idx_sandboxes_expires_at ON sandboxes(expires_at);

-- ============================================================================
-- 插入内置安全策略
-- ============================================================================

INSERT INTO security_policies (id, name, description, tool_permissions, resource_limits, network_policy, is_builtin, category, created_at, updated_at)
VALUES 
    (
        'builtin-strict', 
        '严格策略', 
        '最小权限原则，禁止危险操作',
        '[{"tool": "file_read", "permission": "allowed"}, {"tool": "file_write", "permission": "approval_required", "approval_note": "文件写入需要审批"}, {"tool": "file_delete", "permission": "denied", "reason": "严格模式禁止删除文件"}, {"tool": "command_execute", "permission": "denied", "reason": "严格模式禁止执行命令"}]',
        '{"max_cpu": "1.0", "max_memory": "2Gi", "max_disk": "5Gi", "max_processes": 50, "max_open_files": 256}',
        '{"allow_internet": false}',
        TRUE, 
        'security', 
        CURRENT_TIMESTAMP, 
        CURRENT_TIMESTAMP
    ),
    (
        'builtin-standard', 
        '标准策略', 
        '平衡安全与便利，适用于开发环境',
        '[{"tool": "file_*", "permission": "allowed"}, {"tool": "command_execute", "permission": "approval_required", "approval_note": "命令执行需要确认"}]',
        '{"max_cpu": "2.0", "max_memory": "4Gi", "max_disk": "20Gi", "max_processes": 100, "max_open_files": 1024}',
        '{"allow_internet": true, "allowed_domains": ["github.com", "*.npmjs.org", "*.pypi.org"]}',
        TRUE, 
        'development', 
        CURRENT_TIMESTAMP, 
        CURRENT_TIMESTAMP
    ),
    (
        'builtin-permissive', 
        '宽松策略', 
        '较少限制，适用于受信环境',
        '[{"tool": "*", "permission": "allowed"}]',
        '{"max_cpu": "4.0", "max_memory": "8Gi", "max_disk": "50Gi", "max_processes": 500, "max_open_files": 4096}',
        '{"allow_internet": true}',
        TRUE, 
        'trusted', 
        CURRENT_TIMESTAMP, 
        CURRENT_TIMESTAMP
    )
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    tool_permissions = EXCLUDED.tool_permissions,
    resource_limits = EXCLUDED.resource_limits,
    network_policy = EXCLUDED.network_policy,
    updated_at = CURRENT_TIMESTAMP;

-- ============================================================================
-- 添加 agents 表的 security_policy_id 外键（如果需要）
-- ============================================================================

-- 注意：根据实际情况决定是否添加外键约束
-- ALTER TABLE agents ADD CONSTRAINT fk_agents_security_policy 
--     FOREIGN KEY (security_policy_id) REFERENCES security_policies(id);

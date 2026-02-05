-- 012_task_templates.sql
-- TaskTemplate 表结构
-- 阶段9：模板存储层

-- ============================================================================
-- TaskTemplates 表 - 任务模板
-- ============================================================================

CREATE TABLE IF NOT EXISTS task_templates (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(32) NOT NULL,            -- code_gen, test_gen, review, refactor, doc_gen, debug, etc.
    description TEXT,
    
    -- 模板内容
    prompt_template JSONB,                -- PromptTemplate
    default_workspace JSONB,              -- WorkspaceConfig
    default_security JSONB,               -- SecurityConfig
    default_labels JSONB,                 -- 默认标签
    variables JSONB,                      -- TemplateVariable 列表
    
    -- 元数据
    is_builtin BOOLEAN NOT NULL DEFAULT FALSE,
    category VARCHAR(64),                 -- development, testing, documentation, etc.
    
    -- 时间戳
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_task_templates_type ON task_templates(type);
CREATE INDEX IF NOT EXISTS idx_task_templates_is_builtin ON task_templates(is_builtin);
CREATE INDEX IF NOT EXISTS idx_task_templates_category ON task_templates(category);

-- ============================================================================
-- 插入内置任务模板
-- ============================================================================

INSERT INTO task_templates (id, name, type, description, prompt_template, is_builtin, category, created_at, updated_at)
VALUES 
    (
        'builtin-code-gen', 
        '代码生成', 
        'code_gen',
        '根据需求生成代码',
        '{"id": "builtin-code-gen-prompt", "name": "代码生成提示词", "template": "请根据以下需求生成代码：\n\n{{requirement}}\n\n要求：\n- 代码清晰可读\n- 包含必要注释\n- 遵循最佳实践", "variables": [{"name": "requirement", "type": "string", "required": true, "description": "需求描述"}]}',
        TRUE,
        'development',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        'builtin-test-gen', 
        '测试生成', 
        'test_gen',
        '为代码生成单元测试',
        '{"id": "builtin-test-gen-prompt", "name": "测试生成提示词", "template": "请为以下代码生成单元测试：\n\n{{code}}\n\n要求：\n- 覆盖主要功能\n- 包含边界条件\n- 使用合适的断言", "variables": [{"name": "code", "type": "string", "required": true, "description": "待测代码"}]}',
        TRUE,
        'testing',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        'builtin-code-review', 
        '代码审查', 
        'review',
        '审查代码质量',
        '{"id": "builtin-review-prompt", "name": "代码审查提示词", "template": "请审查以下代码：\n\n{{code}}\n\n关注点：\n- 代码质量\n- 潜在问题\n- 改进建议", "variables": [{"name": "code", "type": "string", "required": true, "description": "待审代码"}]}',
        TRUE,
        'development',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        'builtin-doc-gen', 
        '文档生成', 
        'doc_gen',
        '生成技术文档',
        '{"id": "builtin-doc-gen-prompt", "name": "文档生成提示词", "template": "请为以下代码生成文档：\n\n{{code}}\n\n文档类型：{{doc_type}}\n\n要求：\n- 清晰准确\n- 包含示例", "variables": [{"name": "code", "type": "string", "required": true}, {"name": "doc_type", "type": "string", "default": "API文档"}]}',
        TRUE,
        'documentation',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    )
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    prompt_template = EXCLUDED.prompt_template,
    updated_at = CURRENT_TIMESTAMP;

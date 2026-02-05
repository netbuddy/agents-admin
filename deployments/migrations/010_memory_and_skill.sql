-- 010_memory_and_skill.sql
-- Memory + Skill 表结构
-- 阶段7：记忆系统和技能系统

-- ============================================================================
-- Memories 表 - 记忆
-- ============================================================================

CREATE TABLE IF NOT EXISTS memories (
    id VARCHAR(64) PRIMARY KEY,
    agent_id VARCHAR(64) NOT NULL,
    type VARCHAR(32) NOT NULL,           -- working, episodic, semantic, procedural
    
    -- 内容
    content TEXT NOT NULL,
    embedding VECTOR(1536),              -- 向量表示（需要 pgvector 扩展）
    
    -- 元数据
    metadata JSONB,
    importance FLOAT NOT NULL DEFAULT 0.5,
    tags JSONB,
    
    -- 关联
    source_run_id VARCHAR(64),
    source_task_id VARCHAR(64),
    
    -- 生命周期
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    accessed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_memories_agent_id ON memories(agent_id);
CREATE INDEX IF NOT EXISTS idx_memories_type ON memories(type);
CREATE INDEX IF NOT EXISTS idx_memories_importance ON memories(importance DESC);
CREATE INDEX IF NOT EXISTS idx_memories_expires_at ON memories(expires_at);
CREATE INDEX IF NOT EXISTS idx_memories_agent_type ON memories(agent_id, type);

-- 向量索引（需要 pgvector 扩展，可选）
-- CREATE INDEX IF NOT EXISTS idx_memories_embedding ON memories USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- ============================================================================
-- Skills 表 - 技能
-- ============================================================================

CREATE TABLE IF NOT EXISTS skills (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(32) NOT NULL,       -- coding, writing, research, analysis, design, data, devops, other
    level VARCHAR(32) NOT NULL,          -- basic, advanced, expert
    description TEXT,
    
    -- 技能定义
    instructions TEXT,
    tools JSONB,
    examples JSONB,
    parameters JSONB,
    
    -- 来源
    source VARCHAR(32) NOT NULL,         -- builtin, user, community
    author_id VARCHAR(64),
    registry_id VARCHAR(64),
    
    -- 元数据
    version VARCHAR(32) NOT NULL DEFAULT '1.0.0',
    is_builtin BOOLEAN NOT NULL DEFAULT FALSE,
    tags JSONB,
    
    -- 统计
    use_count BIGINT NOT NULL DEFAULT 0,
    rating FLOAT NOT NULL DEFAULT 0,
    
    -- 时间戳
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_skills_category ON skills(category);
CREATE INDEX IF NOT EXISTS idx_skills_level ON skills(level);
CREATE INDEX IF NOT EXISTS idx_skills_source ON skills(source);
CREATE INDEX IF NOT EXISTS idx_skills_is_builtin ON skills(is_builtin);
CREATE INDEX IF NOT EXISTS idx_skills_rating ON skills(rating DESC);

-- ============================================================================
-- SkillRegistries 表 - 技能市场
-- ============================================================================

CREATE TABLE IF NOT EXISTS skill_registries (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(32) NOT NULL,           -- builtin, user, community
    owner_id VARCHAR(64),
    is_public BOOLEAN NOT NULL DEFAULT FALSE,
    skill_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_skill_registries_type ON skill_registries(type);
CREATE INDEX IF NOT EXISTS idx_skill_registries_is_public ON skill_registries(is_public);

-- ============================================================================
-- AgentSkills 表 - Agent 与 Skill 关联（N:M）
-- ============================================================================

CREATE TABLE IF NOT EXISTS agent_skills (
    agent_id VARCHAR(64) NOT NULL,
    skill_id VARCHAR(64) NOT NULL REFERENCES skills(id),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    config JSONB,
    priority INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (agent_id, skill_id)
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_agent_skills_agent_id ON agent_skills(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_skills_skill_id ON agent_skills(skill_id);

-- ============================================================================
-- 插入内置技能
-- ============================================================================

INSERT INTO skills (id, name, category, level, description, instructions, source, version, is_builtin, tags, created_at, updated_at)
VALUES 
    (
        'builtin-code-review', 
        '代码审查', 
        'coding', 
        'advanced',
        '审查代码质量、发现问题、提出改进建议',
        '作为代码审查专家，你需要：
1. 检查代码风格和规范一致性
2. 识别潜在的 bug 和安全问题
3. 评估代码的可读性和可维护性
4. 提出具体的改进建议
5. 关注性能和资源使用',
        'builtin',
        '1.0.0',
        TRUE,
        '["code", "review", "quality"]',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        'builtin-unit-test', 
        '单元测试编写', 
        'coding', 
        'advanced',
        '编写高质量的单元测试，确保代码覆盖率',
        '作为测试专家，你需要：
1. 分析被测代码的功能和边界条件
2. 编写测试用例覆盖正常路径和异常路径
3. 使用合适的断言和 mock
4. 确保测试的独立性和可重复性
5. 追求高代码覆盖率',
        'builtin',
        '1.0.0',
        TRUE,
        '["test", "unit-test", "coverage"]',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        'builtin-doc-writing', 
        '技术文档编写', 
        'writing', 
        'advanced',
        '编写清晰、准确的技术文档',
        '作为技术文档专家，你需要：
1. 理解目标读者和使用场景
2. 组织清晰的文档结构
3. 使用准确的技术术语
4. 提供代码示例和图表
5. 保持文档的可维护性',
        'builtin',
        '1.0.0',
        TRUE,
        '["documentation", "writing", "technical"]',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        'builtin-debugging', 
        '问题调试', 
        'coding', 
        'expert',
        '系统性地定位和解决代码问题',
        '作为调试专家，你需要：
1. 收集和分析错误信息
2. 复现问题并隔离变量
3. 使用日志和调试工具定位问题
4. 分析根本原因
5. 提出修复方案并验证',
        'builtin',
        '1.0.0',
        TRUE,
        '["debug", "troubleshoot", "problem-solving"]',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    )
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    instructions = EXCLUDED.instructions,
    updated_at = CURRENT_TIMESTAMP;

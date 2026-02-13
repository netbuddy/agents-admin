# 数据库升级策略

## 概述

本项目采用 **全量 + 增量** 双轨数据库管理策略：

| 场景 | 使用脚本 | 说明 |
|------|---------|------|
| **新安装** | `init-db.sql` | 一次性创建完整 schema，设置版本号 |
| **版本升级** | `migrations/NNN_xxx.sql` | 按顺序执行未应用的增量脚本 |

## 版本追踪

数据库通过 `schema_version` 表（单行）记录当前版本：

```sql
SELECT * FROM schema_version;
-- migration_id | version | description           | applied_at
-- 19           | 1.0.0   | 全量安装：合并 002-019  | 2025-07-14 ...
```

## 新安装

```bash
# 创建数据库
createdb -U postgres agents_admin

# 执行全量脚本
psql "postgres://agents:password@localhost:5432/agents_admin" \
  -f deployments/init-db.sql
```

## 版本升级

### 1. 查看当前版本

```bash
psql "postgres://agents:password@localhost:5432/agents_admin" \
  -c "SELECT migration_id, version FROM schema_version;"
```

### 2. 执行未应用的增量脚本

例如当前 `migration_id = 19`，需要升级到包含 020、021 的新版本：

```bash
psql "postgres://agents:password@localhost:5432/agents_admin" \
  -f deployments/migrations/020_xxx.sql
psql "postgres://agents:password@localhost:5432/agents_admin" \
  -f deployments/migrations/021_xxx.sql
```

### 3. 验证

```bash
psql "postgres://agents:password@localhost:5432/agents_admin" \
  -c "SELECT * FROM schema_version;"
```

## 发布流程（面向开发者）

每次发布新版本时，需要同时维护两套脚本：

### 增量迁移脚本（开发阶段）

1. 在 `deployments/migrations/` 中创建新的迁移脚本，编号递增
2. 脚本末尾**必须**更新 `schema_version`：

```sql
BEGIN;

-- ... 你的 DDL/DML 变更 ...

-- 更新版本号
INSERT INTO schema_version (migration_id, version, description)
VALUES (20, '1.1.0', '添加 xxx 功能')
ON CONFLICT (id) DO UPDATE SET
    migration_id = EXCLUDED.migration_id,
    version = EXCLUDED.version,
    description = EXCLUDED.description,
    applied_at = NOW();

COMMIT;
```

### 全量脚本同步（发布阶段）

发布前，将新增的 DDL 变更合并到 `init-db.sql`：

1. 将新表/列/索引添加到 `init-db.sql` 的对应位置
2. 更新文件头的版本信息和合并范围
3. 更新末尾的 `schema_version` INSERT 语句
4. 更新完成注释中的表计数

### 验证清单

- [ ] `init-db.sql` 在空数据库上执行成功
- [ ] 所有增量脚本按顺序在旧版数据库上执行成功
- [ ] 两种路径执行后的 schema 完全一致（可用 `pg_dump --schema-only` 对比）
- [ ] `schema_version` 表的版本号一致

## 迁移脚本编写规范

### 命名规则

```
NNN_简短描述.sql
```

- `NNN` = 三位数字，顺序递增（如 020, 021）
- 描述使用下划线分隔的英文小写

### 编写要求

1. **幂等性**：使用 `IF NOT EXISTS`、`IF EXISTS`，允许重复执行
2. **事务包裹**：用 `BEGIN; ... COMMIT;` 包裹，保证原子性
3. **向后兼容**：`ALTER TABLE ADD COLUMN IF NOT EXISTS` + 合理默认值
4. **注释说明**：文件头说明变更目的和影响
5. **版本更新**：末尾必须更新 `schema_version`

### 示例

```sql
-- 020: 添加 xxx 表
-- 背景: 支持 xxx 功能
-- 影响: 新增 1 张表，无破坏性变更

BEGIN;

CREATE TABLE IF NOT EXISTS xxx (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_xxx_name ON xxx(name);

-- 更新 schema 版本
INSERT INTO schema_version (migration_id, version, description)
VALUES (20, '1.1.0', '添加 xxx 表')
ON CONFLICT (id) DO UPDATE SET
    migration_id = EXCLUDED.migration_id,
    version = EXCLUDED.version,
    description = EXCLUDED.description,
    applied_at = NOW();

COMMIT;
```

## 迁移历史

| migration_id | 版本 | 文件 | 说明 |
|:---:|:---:|------|------|
| 002 | - | `002_add_proxies.sql` | 代理表 |
| 003 | - | `003_add_instances_terminals.sql` | 实例和终端会话 |
| 004 | - | `004_add_task_context.sql` | 任务上下文 |
| 005 | - | `005_flatten_task_spec.sql` | 任务字段扁平化 |
| 006 | - | `006_add_prompt_and_hitl.sql` | Prompt 和 HITL |
| 007 | - | `007_agent_template_and_agent.sql` | 智能体模板和实例 |
| 008 | - | `008_security_and_sandbox.sql` | 安全策略和沙箱 |
| 009 | - | `009_hitl_tables.sql` | HITL 审批表 |
| 010 | - | `010_memory_and_skill.sql` | 记忆和技能系统 |
| 011 | - | `011_mcp.sql` | MCP 服务器 |
| 012 | - | `012_task_templates.sql` | 任务模板 |
| 013 | - | `013_task_status_rename.sql` | Task 状态重命名 |
| 014 | - | `014_operations_and_actions.sql` | 操作和动作 |
| 015 | - | `015_run_hierarchy.sql` | Run 层次化 |
| 016 | - | `016_action_phase_message.sql` | Action 阶段消息 |
| 017 | - | `017_node_provisions.sql` | 节点部署记录 |
| 018 | - | `018_add_users_and_tenant.sql` | 用户表和多租户 |
| **019** | **1.0.0** | `019_schema_version.sql` | 版本追踪表（首次发布） |

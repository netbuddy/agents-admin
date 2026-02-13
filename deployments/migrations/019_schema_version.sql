-- 019: 数据库版本追踪表
--
-- 用途：
--   记录数据库 schema 的当前版本，支持全量/增量升级策略。
--   - 新安装：init-db.sql 自动设置为最新版本
--   - 升级：每次增量迁移后更新版本号
--
-- 版本号规则：
--   migration_id = 迁移脚本编号（如 019）
--   version = 语义化版本号（如 1.0.0），与发布版本对应

BEGIN;

CREATE TABLE IF NOT EXISTS schema_version (
    id              INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),  -- 单行表
    migration_id    INTEGER NOT NULL,                               -- 最后应用的迁移编号
    version         VARCHAR(32) NOT NULL,                           -- 对应的发布版本号
    description     TEXT,                                           -- 版本说明
    applied_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),             -- 最后更新时间
    applied_by      VARCHAR(128) DEFAULT current_user               -- 执行者
);

-- 设置当前版本：迁移 019，对应首次发布版本 1.0.0
INSERT INTO schema_version (migration_id, version, description)
VALUES (19, '1.0.0', '首次发布：合并 migrations 002-019')
ON CONFLICT (id) DO UPDATE SET
    migration_id = EXCLUDED.migration_id,
    version = EXCLUDED.version,
    description = EXCLUDED.description,
    applied_at = NOW();

COMMIT;

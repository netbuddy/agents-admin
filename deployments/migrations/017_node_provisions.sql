-- 017: 节点部署记录表
-- 记录通过 SSH 远程部署 node-manager 的历史

BEGIN;

CREATE TABLE IF NOT EXISTS node_provisions (
    id              TEXT PRIMARY KEY,
    node_id         TEXT NOT NULL,
    host            TEXT NOT NULL,
    port            INTEGER NOT NULL DEFAULT 22,
    ssh_user        TEXT NOT NULL,
    auth_method     TEXT NOT NULL DEFAULT 'password',
    status          TEXT NOT NULL DEFAULT 'pending',
    error_message   TEXT,
    version         TEXT NOT NULL,
    github_repo     TEXT NOT NULL DEFAULT 'org/agents-admin',
    api_server_url  TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_node_provisions_status ON node_provisions(status);
CREATE INDEX IF NOT EXISTS idx_node_provisions_node_id ON node_provisions(node_id);

COMMIT;

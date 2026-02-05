// Package sqlite SQLite 数据库驱动
//
// 提供 SQLite 连接管理、方言实现和自动 Schema 迁移。
// 适用于开发、测试和轻量级部署场景。
package sqlite

import (
	"database/sql"
	"fmt"

	"agents-admin/internal/shared/storage/dbutil"

	_ "modernc.org/sqlite"
)

// Dialect SQLite 方言实现
type Dialect struct{}

var _ dbutil.Dialect = (*Dialect)(nil)

func (d *Dialect) DriverType() dbutil.DriverType {
	return dbutil.DriverSQLite
}

func (d *Dialect) Rebind(query string) string {
	return dbutil.StripPgCasts(dbutil.RebindToQuestion(query))
}

func (d *Dialect) CurrentTimestamp() string {
	return "datetime('now')"
}

func (d *Dialect) BooleanLiteral(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func (d *Dialect) UpsertConflict(conflictColumn string, updateExprs []string) string {
	result := fmt.Sprintf("ON CONFLICT (%s) DO UPDATE SET ", conflictColumn)
	for i, expr := range updateExprs {
		if i > 0 {
			result += ", "
		}
		result += expr
	}
	return result
}

func (d *Dialect) SupportsNullsLast() bool {
	return false
}

func (d *Dialect) NullsLastClause() string {
	return ""
}

func (d *Dialect) SupportsRecursiveCTE() bool {
	return true
}

func (d *Dialect) AutoMigrate(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}

// Open 创建 SQLite 数据库连接
// dsn 示例: "file:test.db?cache=shared&mode=rwc" 或 ":memory:"
func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	// SQLite 优化设置
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return nil, fmt.Errorf("failed to set pragma %s: %w", p, err)
		}
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite: %w", err)
	}

	return db, nil
}

// NewDialect 创建 SQLite 方言
func NewDialect() *Dialect {
	return &Dialect{}
}

// schema SQLite 完整建表语句（等价于 PostgreSQL 迁移文件）
const schema = `
-- tasks
CREATE TABLE IF NOT EXISTS tasks (
    id VARCHAR(64) PRIMARY KEY,
    parent_id VARCHAR(64),
    name VARCHAR(200),
    status VARCHAR(32) DEFAULT 'pending',
    spec TEXT,
    type VARCHAR(32) DEFAULT 'general',
    prompt TEXT,
    workspace TEXT,
    security TEXT,
    labels TEXT DEFAULT '{}',
    context TEXT,
    template_id VARCHAR(64),
    agent_id VARCHAR(64),
    created_at DATETIME DEFAULT (datetime('now')),
    updated_at DATETIME DEFAULT (datetime('now'))
);

-- runs
CREATE TABLE IF NOT EXISTS runs (
    id VARCHAR(64) PRIMARY KEY,
    task_id VARCHAR(64) NOT NULL REFERENCES tasks(id),
    status VARCHAR(32) DEFAULT 'queued',
    node_id VARCHAR(64),
    started_at DATETIME,
    finished_at DATETIME,
    snapshot TEXT,
    error TEXT,
    created_at DATETIME DEFAULT (datetime('now')),
    updated_at DATETIME DEFAULT (datetime('now'))
);

-- events
CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id VARCHAR(64) NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    seq INTEGER NOT NULL,
    type VARCHAR(64),
    timestamp DATETIME,
    payload TEXT,
    raw TEXT
);

-- nodes
CREATE TABLE IF NOT EXISTS nodes (
    id VARCHAR(64) PRIMARY KEY,
    status VARCHAR(32) DEFAULT 'online',
    labels TEXT DEFAULT '{}',
    capacity TEXT DEFAULT '{}',
    last_heartbeat DATETIME,
    created_at DATETIME DEFAULT (datetime('now')),
    updated_at DATETIME DEFAULT (datetime('now'))
);

-- accounts
CREATE TABLE IF NOT EXISTS accounts (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(200),
    agent_type_id VARCHAR(64),
    node_id VARCHAR(64),
    volume_name VARCHAR(200),
    status VARCHAR(32) DEFAULT 'pending',
    created_at DATETIME DEFAULT (datetime('now')),
    updated_at DATETIME DEFAULT (datetime('now')),
    last_used_at DATETIME
);

-- auth_tasks
CREATE TABLE IF NOT EXISTS auth_tasks (
    id VARCHAR(64) PRIMARY KEY,
    account_id VARCHAR(64) NOT NULL REFERENCES accounts(id),
    method VARCHAR(32),
    node_id VARCHAR(64),
    status VARCHAR(32) DEFAULT 'pending',
    terminal_port INTEGER,
    terminal_url TEXT,
    container_name VARCHAR(200),
    message TEXT,
    created_at DATETIME DEFAULT (datetime('now')),
    updated_at DATETIME DEFAULT (datetime('now')),
    expires_at DATETIME
);

-- operations
CREATE TABLE IF NOT EXISTS operations (
    id VARCHAR(64) PRIMARY KEY,
    type VARCHAR(64),
    config TEXT,
    status VARCHAR(32) DEFAULT 'pending',
    node_id VARCHAR(64),
    created_at DATETIME DEFAULT (datetime('now')),
    updated_at DATETIME DEFAULT (datetime('now')),
    finished_at DATETIME
);

-- actions
CREATE TABLE IF NOT EXISTS actions (
    id VARCHAR(64) PRIMARY KEY,
    operation_id VARCHAR(64) NOT NULL REFERENCES operations(id),
    status VARCHAR(32) DEFAULT 'pending',
    phase VARCHAR(64),
    message TEXT,
    progress INTEGER DEFAULT 0,
    result TEXT,
    error TEXT,
    created_at DATETIME DEFAULT (datetime('now')),
    started_at DATETIME,
    finished_at DATETIME
);

-- proxies
CREATE TABLE IF NOT EXISTS proxies (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(200),
    type VARCHAR(32),
    host VARCHAR(200),
    port INTEGER,
    username VARCHAR(200),
    password VARCHAR(200),
    no_proxy TEXT,
    is_default INTEGER DEFAULT 0,
    status VARCHAR(32) DEFAULT 'active',
    created_at DATETIME DEFAULT (datetime('now')),
    updated_at DATETIME DEFAULT (datetime('now'))
);

-- instances
CREATE TABLE IF NOT EXISTS instances (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(200),
    account_id VARCHAR(64),
    agent_type_id VARCHAR(64),
    container_name VARCHAR(200),
    node_id VARCHAR(64),
    status VARCHAR(32) DEFAULT 'pending',
    created_at DATETIME DEFAULT (datetime('now')),
    updated_at DATETIME DEFAULT (datetime('now'))
);

-- terminal_sessions
CREATE TABLE IF NOT EXISTS terminal_sessions (
    id VARCHAR(64) PRIMARY KEY,
    instance_id VARCHAR(64),
    container_name VARCHAR(200),
    node_id VARCHAR(64),
    port INTEGER,
    url TEXT,
    status VARCHAR(32) DEFAULT 'pending',
    created_at DATETIME DEFAULT (datetime('now')),
    expires_at DATETIME
);

-- approval_requests
CREATE TABLE IF NOT EXISTS approval_requests (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    type VARCHAR(64) NOT NULL,
    status VARCHAR(32) DEFAULT 'pending',
    operation TEXT NOT NULL,
    reason TEXT,
    context TEXT,
    expires_at DATETIME,
    created_at DATETIME DEFAULT (datetime('now')),
    resolved_at DATETIME
);

-- approval_decisions
CREATE TABLE IF NOT EXISTS approval_decisions (
    id VARCHAR(64) PRIMARY KEY,
    request_id VARCHAR(64) NOT NULL REFERENCES approval_requests(id) ON DELETE CASCADE,
    decision VARCHAR(32) NOT NULL,
    decided_by VARCHAR(64) NOT NULL,
    comment TEXT,
    instructions TEXT,
    created_at DATETIME DEFAULT (datetime('now'))
);

-- human_feedbacks
CREATE TABLE IF NOT EXISTS human_feedbacks (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    type VARCHAR(32) NOT NULL,
    content TEXT NOT NULL,
    created_by VARCHAR(64) NOT NULL,
    created_at DATETIME DEFAULT (datetime('now')),
    processed_at DATETIME
);

-- interventions
CREATE TABLE IF NOT EXISTS interventions (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    action VARCHAR(32) NOT NULL,
    reason TEXT,
    parameters TEXT,
    created_by VARCHAR(64) NOT NULL,
    created_at DATETIME DEFAULT (datetime('now')),
    executed_at DATETIME
);

-- confirmations
CREATE TABLE IF NOT EXISTS confirmations (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    type VARCHAR(32) NOT NULL,
    message TEXT NOT NULL,
    status VARCHAR(32) DEFAULT 'pending',
    options TEXT DEFAULT '[]',
    selected_option VARCHAR(200),
    created_at DATETIME DEFAULT (datetime('now')),
    resolved_at DATETIME
);

-- task_templates
CREATE TABLE IF NOT EXISTS task_templates (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    type VARCHAR(32) DEFAULT 'general',
    description TEXT,
    prompt_template TEXT,
    default_workspace TEXT,
    default_security TEXT,
    default_labels TEXT DEFAULT '{}',
    variables TEXT DEFAULT '[]',
    is_builtin INTEGER DEFAULT 0,
    category VARCHAR(64),
    created_at DATETIME DEFAULT (datetime('now')),
    updated_at DATETIME DEFAULT (datetime('now'))
);

-- agent_templates
CREATE TABLE IF NOT EXISTS agent_templates (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    type VARCHAR(32),
    role VARCHAR(64),
    description TEXT,
    personality TEXT,
    model VARCHAR(100),
    temperature REAL,
    max_context INTEGER,
    skills TEXT,
    mcp_servers TEXT,
    is_builtin INTEGER DEFAULT 0,
    category VARCHAR(64),
    created_at DATETIME DEFAULT (datetime('now')),
    updated_at DATETIME DEFAULT (datetime('now'))
);

-- skills
CREATE TABLE IF NOT EXISTS skills (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    category VARCHAR(64),
    level VARCHAR(32),
    description TEXT,
    instructions TEXT,
    tools TEXT,
    examples TEXT,
    parameters TEXT,
    source VARCHAR(32) DEFAULT 'custom',
    author_id VARCHAR(64),
    registry_id VARCHAR(64),
    version VARCHAR(32),
    is_builtin INTEGER DEFAULT 0,
    tags TEXT DEFAULT '[]',
    use_count INTEGER DEFAULT 0,
    rating REAL DEFAULT 0,
    created_at DATETIME DEFAULT (datetime('now')),
    updated_at DATETIME DEFAULT (datetime('now'))
);

-- mcp_servers
CREATE TABLE IF NOT EXISTS mcp_servers (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    source VARCHAR(32) DEFAULT 'custom',
    transport VARCHAR(32),
    command TEXT,
    args TEXT,
    url TEXT,
    headers TEXT,
    capabilities TEXT,
    version VARCHAR(32),
    author VARCHAR(200),
    repository VARCHAR(500),
    is_builtin INTEGER DEFAULT 0,
    tags TEXT DEFAULT '[]',
    created_at DATETIME DEFAULT (datetime('now')),
    updated_at DATETIME DEFAULT (datetime('now'))
);

-- security_policies
CREATE TABLE IF NOT EXISTS security_policies (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    tool_permissions TEXT,
    resource_limits TEXT,
    network_policy TEXT,
    sandbox_policy TEXT,
    is_builtin INTEGER DEFAULT 0,
    category VARCHAR(64),
    created_at DATETIME DEFAULT (datetime('now')),
    updated_at DATETIME DEFAULT (datetime('now'))
);

-- artifacts (referenced by task delete cascade)
CREATE TABLE IF NOT EXISTS artifacts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id VARCHAR(64) NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    name VARCHAR(200),
    type VARCHAR(64),
    data TEXT,
    created_at DATETIME DEFAULT (datetime('now'))
);
`

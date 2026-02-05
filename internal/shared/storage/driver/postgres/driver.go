// Package postgres PostgreSQL 数据库驱动
//
// 提供 PostgreSQL 连接管理和方言实现。
package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"agents-admin/internal/shared/storage/dbutil"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Dialect PostgreSQL 方言实现
type Dialect struct{}

var _ dbutil.Dialect = (*Dialect)(nil)

func (d *Dialect) DriverType() dbutil.DriverType {
	return dbutil.DriverPostgres
}

func (d *Dialect) Rebind(query string) string {
	return dbutil.RebindToPositional(query)
}

func (d *Dialect) CurrentTimestamp() string {
	return "NOW()"
}

func (d *Dialect) BooleanLiteral(b bool) string {
	if b {
		return "TRUE"
	}
	return "FALSE"
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
	return true
}

func (d *Dialect) NullsLastClause() string {
	return "NULLS LAST"
}

func (d *Dialect) SupportsRecursiveCTE() bool {
	return true
}

func (d *Dialect) AutoMigrate(db *sql.DB) error {
	// PostgreSQL 使用外部迁移文件，不在代码中自动建表
	return nil
}

// Open 创建 PostgreSQL 数据库连接
func Open(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return db, nil
}

// NewDialect 创建 PostgreSQL 方言
func NewDialect() *Dialect {
	return &Dialect{}
}

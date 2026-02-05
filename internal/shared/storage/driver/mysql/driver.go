// Package mysql MySQL 数据库驱动（预留）
//
// 提供 MySQL 连接管理和方言实现。
// 当前为 stub 实现，后续可完善。
package mysql

import (
	"database/sql"
	"fmt"

	"agents-admin/internal/shared/storage/dbutil"
)

// Dialect MySQL 方言实现
type Dialect struct{}

var _ dbutil.Dialect = (*Dialect)(nil)

func (d *Dialect) DriverType() dbutil.DriverType {
	return dbutil.DriverMySQL
}

func (d *Dialect) Rebind(query string) string {
	return dbutil.StripPgCasts(dbutil.RebindToQuestion(query))
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
	// MySQL 使用 ON DUPLICATE KEY UPDATE 语法
	result := "ON DUPLICATE KEY UPDATE "
	for i, expr := range updateExprs {
		if i > 0 {
			result += ", "
		}
		// 将 EXCLUDED.col 替换为 VALUES(col) 格式
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
	return true // MySQL 8.0+
}

func (d *Dialect) AutoMigrate(db *sql.DB) error {
	return fmt.Errorf("mysql auto-migrate not implemented yet")
}

// NewDialect 创建 MySQL 方言
func NewDialect() *Dialect {
	return &Dialect{}
}

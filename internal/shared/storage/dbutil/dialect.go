// Package dbutil 提供数据库方言抽象和工具函数
//
// 通过 Dialect 接口屏蔽不同数据库（PostgreSQL、SQLite、MySQL）的 SQL 差异，
// 使 repository 层可以编写与数据库无关的业务逻辑。
package dbutil

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// DriverType 数据库驱动类型
type DriverType string

const (
	DriverPostgres DriverType = "postgres"
	DriverSQLite   DriverType = "sqlite"
	DriverMySQL    DriverType = "mysql"
)

// Dialect 数据库方言接口
//
// 不同数据库的 SQL 语法差异通过该接口屏蔽：
//   - 占位符：PostgreSQL 用 $1, $2；MySQL/SQLite 用 ?
//   - 时间函数：PostgreSQL/MySQL 用 NOW()；SQLite 用 datetime('now')
//   - UPSERT：各数据库语法不同
//   - 类型转换：PostgreSQL 有 ::type 语法
type Dialect interface {
	// DriverType 返回驱动类型标识
	DriverType() DriverType

	// Rebind 将 PostgreSQL 风格的占位符 ($1, $2, ...) 转换为目标数据库的占位符格式
	Rebind(query string) string

	// CurrentTimestamp 返回当前时间戳的 SQL 表达式
	CurrentTimestamp() string

	// BooleanLiteral 返回布尔字面量
	BooleanLiteral(b bool) string

	// UpsertConflict 生成 UPSERT 的冲突处理子句
	// conflictColumn: 冲突检测列
	// updateExprs: 更新表达式列表，如 "status = EXCLUDED.status"
	UpsertConflict(conflictColumn string, updateExprs []string) string

	// SupportsNullsLast 是否支持 ORDER BY ... NULLS LAST
	SupportsNullsLast() bool

	// NullsLastClause 返回 NULLS LAST 子句（不支持时返回空字符串）
	NullsLastClause() string

	// SupportsRecursiveCTE 是否支持 WITH RECURSIVE
	SupportsRecursiveCTE() bool

	// AutoMigrate 自动创建/迁移数据库 Schema
	AutoMigrate(db *sql.DB) error
}

// pgPlaceholderRe 匹配 PostgreSQL 风格占位符 $1, $2, ...
var pgPlaceholderRe = regexp.MustCompile(`\$(\d+)`)

// pgCastRe 匹配 PostgreSQL 类型转换 ::type
var pgCastRe = regexp.MustCompile(`::(\w+)`)

// RebindToPositional 保持 $N 占位符不变（PostgreSQL 专用）
func RebindToPositional(query string) string {
	return query
}

// RebindToQuestion 将 $N 占位符转换为 ? （MySQL/SQLite 专用）
func RebindToQuestion(query string) string {
	return pgPlaceholderRe.ReplaceAllString(query, "?")
}

// StripPgCasts 去除 PostgreSQL 类型转换 (::varchar, ::text 等)
func StripPgCasts(query string) string {
	return pgCastRe.ReplaceAllString(query, "")
}

// ReplaceNow 替换 NOW() 为目标数据库的时间函数
func ReplaceNow(query string, replacement string) string {
	re := regexp.MustCompile(`(?i)\bNOW\(\)`)
	return re.ReplaceAllString(query, replacement)
}

// Itoa 简单的整数转字符串
func Itoa(n int) string {
	return strconv.Itoa(n)
}

// BuildDynamicQuery 构建动态 WHERE 条件的查询
// 根据方言自动调整占位符
func BuildDynamicQuery(d Dialect, baseQuery string, conditions []string, args []interface{}) (string, []interface{}) {
	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}
	return d.Rebind(baseQuery), args
}

// PlaceholderList 生成指定数量的占位符列表，如 "$1, $2, $3"
func PlaceholderList(d Dialect, start, count int) string {
	parts := make([]string, count)
	for i := 0; i < count; i++ {
		parts[i] = fmt.Sprintf("$%d", start+i)
	}
	result := strings.Join(parts, ", ")
	return d.Rebind(result)
}

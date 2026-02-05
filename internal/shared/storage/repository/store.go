// Package repository 数据库无关的业务逻辑存储层
//
// 通过 dbutil.Dialect 接口屏蔽不同数据库的 SQL 差异，
// 所有 SQL 以 PostgreSQL 风格编写，运行时由 Dialect.Rebind() 转换。
package repository

import (
	"database/sql"
	"encoding/json"

	"agents-admin/internal/shared/storage/dbutil"
)

// Store 通用存储实现
// 实现了 storage.PersistentStore 接口
type Store struct {
	db      *sql.DB
	dialect dbutil.Dialect
}

// NewStore 创建通用存储
func NewStore(db *sql.DB, dialect dbutil.Dialect) *Store {
	return &Store{db: db, dialect: dialect}
}

// Close 关闭数据库连接
func (s *Store) Close() error {
	return s.db.Close()
}

// DB 返回底层数据库连接（仅用于测试）
func (s *Store) DB() *sql.DB {
	return s.db
}

// Dialect 返回当前方言
func (s *Store) Dialect() dbutil.Dialect {
	return s.dialect
}

// rebind 快捷方法：将 PG 风格 SQL 转换为当前方言
func (s *Store) rebind(query string) string {
	return s.dialect.Rebind(query)
}

// now 返回当前时间戳 SQL 表达式
func (s *Store) now() string {
	return s.dialect.CurrentTimestamp()
}

// NullableJSON 用于安全扫描可能为 NULL 的 JSON 字段
// database/sql 无法直接将 NULL scan 到 json.RawMessage，需要通过 *[]byte 中间变量
type NullableJSON struct {
	Data *[]byte
}

// Value 返回 json.RawMessage（如果非 NULL）
func (n *NullableJSON) Value() json.RawMessage {
	if n.Data != nil {
		return json.RawMessage(*n.Data)
	}
	return nil
}

// Package postgres PostgreSQL 存储实现（向后兼容包装层）
//
// 本包现在是 repository.Store + driver/postgres.Dialect 的组合包装。
// 新代码请直接使用 repository.NewStore() + driver/postgres 或 driver/sqlite。
package postgres

import (
	"database/sql"

	pgdriver "agents-admin/internal/shared/storage/driver/postgres"
	"agents-admin/internal/shared/storage/repository"
)

// Store PostgreSQL 存储（向后兼容）
// 内部委托给 repository.Store
type Store = repository.Store

// NewStore 创建 PostgreSQL 存储（向后兼容）
func NewStore(databaseURL string) (*Store, error) {
	db, err := pgdriver.Open(databaseURL)
	if err != nil {
		return nil, err
	}
	dialect := pgdriver.NewDialect()
	return repository.NewStore(db, dialect), nil
}

// NewStoreFromDB 从已有的 *sql.DB 创建 PostgreSQL 存储
func NewStoreFromDB(db *sql.DB) *Store {
	dialect := pgdriver.NewDialect()
	return repository.NewStore(db, dialect)
}

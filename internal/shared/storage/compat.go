// Package storage 兼容层
//
// 提供类型别名和构造函数，保持向后兼容。
// 建议：新代码应直接使用接口类型，而非这些具体类型。
//
// 多数据库支持：
//   - 使用 NewPersistentStore(driverType, dsn) 创建支持多种数据库的存储
//   - 或直接使用 postgres.NewStore() / NewSQLiteStore() 创建特定数据库存储
//
// 迁移指南：
//  1. 将字段类型从 *storage.PostgresStore 改为 storage.PersistentStore（接口类型）
//  2. 将字段类型从 *storage.RedisStore 改为 storage.AuthSessionCache（或其他接口）
//  3. 在 main.go 中使用 NewPersistentStore() 或具体驱动包创建实例并注入
package storage

import (
	"fmt"

	"agents-admin/internal/shared/storage/dbutil"
	sqlitedriver "agents-admin/internal/shared/storage/driver/sqlite"
	"agents-admin/internal/shared/storage/etcd"
	"agents-admin/internal/shared/storage/postgres"
	"agents-admin/internal/shared/storage/redis"
	"agents-admin/internal/shared/storage/repository"
)

// ============================================================================
// PostgreSQL 类型别名（向后兼容）
// ============================================================================

// PostgresStore 是 postgres.Store 的类型别名
// Deprecated: 建议使用接口类型（如 TaskStore, RunStore 等）
type PostgresStore = postgres.Store

// NewPostgresStore 创建 PostgreSQL 存储
// Deprecated: 建议使用 postgres.NewStore
var NewPostgresStore = postgres.NewStore

// ============================================================================
// Redis 类型别名（向后兼容）
// ============================================================================

// RedisStore 是 redis.Store 的类型别名
// Deprecated: 建议使用接口类型（如 AuthSessionCache, RunEventBus 等）
type RedisStore = redis.Store

// NewRedisStore 创建 Redis 存储
// Deprecated: 建议使用 redis.NewStore
var NewRedisStore = redis.NewStore

// NewRedisStoreFromURL 从 URL 创建 Redis 存储
// Deprecated: 建议使用 redis.NewStoreFromURL
var NewRedisStoreFromURL = redis.NewStoreFromURL

// ============================================================================
// etcd 类型别名（向后兼容）
// ============================================================================

// EtcdStore 是 etcd.Store 的类型别名
// Deprecated: 建议使用接口类型 EtcdNodeHeartbeat
type EtcdStore = etcd.Store

// EtcdConfig 是 etcd.Config 的类型别名
type EtcdConfig = etcd.Config

// NewEtcdStore 创建 etcd 存储
// Deprecated: 建议使用 etcd.NewStore
var NewEtcdStore = etcd.NewStore

// EtcdEventBus 是 etcd.EventBus 的类型别名
type EtcdEventBus = etcd.EventBus

// NewEtcdEventBus 创建 etcd 事件总线
var NewEtcdEventBus = etcd.NewEventBus

// NewEtcdEventBusFromStore 从 Store 创建事件总线
var NewEtcdEventBusFromStore = etcd.NewEventBusFromStore

// ============================================================================
// 旧类型名兼容（redis.go 中原有的类型）
// ============================================================================

// NodeHeartbeat 是 EtcdHeartbeat 的别名（历史兼容）
// Deprecated: 使用 EtcdHeartbeat
type NodeHeartbeat = EtcdHeartbeat

// ============================================================================
// 多数据库工厂函数
// ============================================================================

// RepositoryStore 是 repository.Store 的类型别名
type RepositoryStore = repository.Store

// NewSQLiteStore 创建 SQLite 存储（含自动建表）
func NewSQLiteStore(dsn string) (*RepositoryStore, error) {
	db, err := sqlitedriver.Open(dsn)
	if err != nil {
		return nil, err
	}
	dialect := sqlitedriver.NewDialect()
	if err := dialect.AutoMigrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite auto-migrate failed: %w", err)
	}
	return repository.NewStore(db, dialect), nil
}

// NewPersistentStoreFromDSN 根据驱动类型和 DSN 创建持久化存储
// 支持的驱动类型：postgres, sqlite, mongodb
func NewPersistentStoreFromDSN(driver dbutil.DriverType, dsn string) (PersistentStore, error) {
	switch driver {
	case dbutil.DriverPostgres:
		return NewPostgresStore(dsn)
	case dbutil.DriverSQLite:
		return NewSQLiteStore(dsn)
	case dbutil.DriverMongoDB:
		return nil, fmt.Errorf("mongodb driver requires NewMongoStore(uri, dbName); use NewPersistentStoreFromConfig instead")
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}
}

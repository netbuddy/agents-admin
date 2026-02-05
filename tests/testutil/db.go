package testutil

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"agents-admin/internal/config"
	"agents-admin/internal/shared/infra"
)

// TestConfig 返回测试环境配置
// 自动加载 configs/test.yaml
func TestConfig(t *testing.T) *config.Config {
	t.Helper()
	// 确保使用测试环境
	os.Setenv("APP_ENV", "test")
	return config.Load()
}

// TestDB 返回测试数据库连接
// 使用前请先运行: ./scripts/test-env.sh setup
func TestDB(t *testing.T) *sql.DB {
	t.Helper()

	os.Setenv("APP_ENV", "test")
	cfg := config.Load()
	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("无法连接测试数据库: %v\n请先运行: ./scripts/test-env.sh setup", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("测试数据库连接失败: %v\n请先运行: ./scripts/test-env.sh setup", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	t.Logf("测试数据库: %s", cfg.String())
	return db
}

// TestRedis 返回测试 Redis 连接
func TestRedis(t *testing.T) *infra.RedisInfra {
	t.Helper()

	os.Setenv("APP_ENV", "test")
	cfg := config.Load()
	redisInfra, err := infra.NewRedisInfra(cfg.RedisURL)
	if err != nil {
		t.Fatalf("无法连接测试 Redis: %v", err)
	}

	t.Cleanup(func() {
		redisInfra.Close()
	})

	t.Logf("测试 Redis: %s", cfg.RedisURL)
	return redisInfra
}

// CleanupTables 清空指定表的数据
func CleanupTables(t *testing.T, db *sql.DB, tables ...string) {
	t.Helper()
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			t.Logf("警告: 清空表 %s 失败: %v", table, err)
		}
	}
}

// CleanupAllTables 清空所有表的数据
func CleanupAllTables(t *testing.T, db *sql.DB) {
	t.Helper()

	rows, err := db.Query(`
		SELECT tablename FROM pg_tables 
		WHERE schemaname = 'public'
	`)
	if err != nil {
		t.Fatalf("获取表列表失败: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			continue
		}
		tables = append(tables, table)
	}

	CleanupTables(t, db, tables...)
}

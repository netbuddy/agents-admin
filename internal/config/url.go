package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// buildDatabaseURL 根据驱动类型构建数据库连接字符串
func buildDatabaseURL(db DatabaseConfig, password string) string {
	switch strings.ToLower(db.Driver) {
	case "sqlite":
		dbPath := db.Path
		if dbPath == "" {
			dbPath = "/var/lib/agents-admin/agents-admin.db"
		}
		return fmt.Sprintf("file:%s?cache=shared&mode=rwc", dbPath)
	case "mongodb":
		if db.URI != "" {
			return db.URI
		}
		if db.User != "" && password != "" {
			return fmt.Sprintf("mongodb://%s:%s@%s:%d", db.User, password, db.Host, db.Port)
		}
		return fmt.Sprintf("mongodb://%s:%d", db.Host, db.Port)
	default: // postgres
		return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			db.User, password, db.Host, db.Port, db.Name, db.SSLMode)
	}
}

// detectDatabaseDriver 检测数据库驱动类型
// 优先级：YAML driver 字段 > DATABASE_URL 前缀自动检测 > 默认 mongodb
func detectDatabaseDriver(yamlDriver, databaseURL string) string {
	// 1. YAML 显式指定
	if d := strings.ToLower(yamlDriver); d == "sqlite" || d == "postgres" || d == "mongodb" {
		return d
	}
	// 2. 从 DATABASE_URL 前缀自动检测
	if strings.HasPrefix(databaseURL, "file:") || strings.HasPrefix(databaseURL, "sqlite:") {
		return "sqlite"
	}
	if strings.HasPrefix(databaseURL, "postgres://") || strings.HasPrefix(databaseURL, "postgresql://") {
		return "postgres"
	}
	if strings.HasPrefix(databaseURL, "mongodb://") || strings.HasPrefix(databaseURL, "mongodb+srv://") {
		return "mongodb"
	}
	// 3. 默认 mongodb
	return "mongodb"
}

// buildRedisURL 构建 Redis 连接字符串
// 如果 URL 字段非空（Node Manager 风格），直接使用；否则从 host/port/db/password 构建
func buildRedisURL(redis RedisConfig) string {
	if redis.URL != "" {
		return redis.URL
	}
	if redis.Password != "" {
		return fmt.Sprintf("redis://:%s@%s:%d/%d", redis.Password, redis.Host, redis.Port, redis.DB)
	}
	return fmt.Sprintf("redis://%s:%d/%d", redis.Host, redis.Port, redis.DB)
}

// maskPassword 隐藏密码
func maskPassword(url string) string {
	re := regexp.MustCompile(`(://[^:]+:)([^@]+)(@)`)
	return re.ReplaceAllString(url, "${1}***${3}")
}

// parseEnv 解析环境字符串
func parseEnv(env string) Environment {
	switch strings.ToLower(env) {
	case "test":
		return EnvTest
	case "prod", "production":
		return EnvProduction
	default:
		return EnvDevelopment
	}
}

// firstEnv 返回第一个非空的环境变量值（用于兼容多种 Docker Compose 变量名）
func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

// getEnv 获取环境变量，支持默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// IsTest 是否为测试环境
func (c *Config) IsTest() bool {
	return c.Env == EnvTest
}

// String 返回配置摘要（隐藏密码）
func (c *Config) String() string {
	return fmt.Sprintf("Config{Env: %s, Driver: %s, DB: %s, Redis: %s}",
		c.Env, c.DatabaseDriver, maskPassword(c.DatabaseURL), c.RedisURL)
}

// validate 验证并填充调度器默认值
func (s *SchedulerConfig) validate() {
	if s.NodeID == "" {
		s.NodeID = "scheduler-default"
	}
	if s.Strategy.Default == "" {
		s.Strategy.Default = "label_match"
	}
	if len(s.Strategy.Chain) == 0 {
		s.Strategy.Chain = []string{"affinity", "label_match"}
	}
	if s.Redis.ReadTimeout == 0 {
		s.Redis.ReadTimeout = 5 * time.Second
	}
	if s.Redis.ReadCount == 0 {
		s.Redis.ReadCount = 10
	}
	if s.Fallback.Interval == 0 {
		s.Fallback.Interval = 5 * time.Minute
	}
	if s.Fallback.StaleThreshold == 0 {
		s.Fallback.StaleThreshold = 5 * time.Minute
	}
	if s.Requeue.OfflineThreshold == 0 {
		s.Requeue.OfflineThreshold = 30 * time.Second
	}
}

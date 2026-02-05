// Package config 统一配置管理
//
// 配置加载策略：
//  1. 从 .env 加载敏感信息（密码、密钥）和 APP_ENV
//  2. 根据 APP_ENV 加载对应的 configs/{env}.yaml 配置文件
//  3. 环境变量可覆盖 YAML 配置
//
// 使用方式：
//   - 开发环境: APP_ENV=dev (默认)
//   - 测试环境: APP_ENV=test
//   - 生产环境: APP_ENV=prod
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Environment 环境类型
type Environment string

const (
	EnvProduction  Environment = "prod"
	EnvTest        Environment = "test" // 测试环境（集成测试 + E2E 共用）
	EnvDevelopment Environment = "dev"
)

// YAMLConfig YAML 配置文件结构
type YAMLConfig struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Redis     RedisConfig     `yaml:"redis"`
	Etcd      EtcdConfig      `yaml:"etcd"`
	Scheduler SchedulerConfig `yaml:"scheduler"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
}

type DatabaseConfig struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	User    string `yaml:"user"`
	Name    string `yaml:"name"`
	SSLMode string `yaml:"sslmode"`
}

type RedisConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	DB   int    `yaml:"db"`
}

type EtcdConfig struct {
	Endpoints []string `yaml:"endpoints"`
	Prefix    string   `yaml:"prefix"`
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	NodeID   string                  `yaml:"node_id"`
	Strategy SchedulerStrategyConfig `yaml:"strategy"`
	Redis    SchedulerRedisConfig    `yaml:"redis"`
	Fallback SchedulerFallbackConfig `yaml:"fallback"`
	Requeue  SchedulerRequeueConfig  `yaml:"requeue"`
}

type SchedulerStrategyConfig struct {
	Default    string                    `yaml:"default"`
	Chain      []string                  `yaml:"chain"`
	LabelMatch SchedulerLabelMatchConfig `yaml:"label_match"`
}

type SchedulerLabelMatchConfig struct {
	LoadBalance bool `yaml:"load_balance"`
}

type SchedulerRedisConfig struct {
	ReadTimeout time.Duration `yaml:"read_timeout"`
	ReadCount   int           `yaml:"read_count"`
}

type SchedulerFallbackConfig struct {
	Interval       time.Duration `yaml:"interval"`
	StaleThreshold time.Duration `yaml:"stale_threshold"`
}

type SchedulerRequeueConfig struct {
	OfflineThreshold time.Duration `yaml:"offline_threshold"`
}

// Config 应用配置（最终使用的配置）
type Config struct {
	Env           Environment
	DatabaseURL   string
	RedisURL      string
	EtcdEndpoints string
	EtcdPrefix    string
	APIPort       string
	Scheduler     SchedulerConfig
}

var configPaths = []string{
	"configs",
	"../configs",
	"../../configs",
	"../../../configs",
}

var envPaths = []string{
	".env",
	"../.env",
	"../../.env",
	"../../../.env",
}

// Load 加载配置
// 1. 加载 .env（敏感信息 + APP_ENV）
// 2. 根据 APP_ENV 加载 configs/{env}.yaml
// 3. 构建最终配置
func Load() *Config {
	// 加载 .env
	for _, p := range envPaths {
		if err := godotenv.Load(p); err == nil {
			break
		}
	}

	// 解析环境
	env := parseEnv(getEnv("APP_ENV", "dev"))

	// 加载 YAML 配置
	yamlCfg := loadYAMLConfig(env)

	// 从环境变量获取敏感信息
	dbPassword := getEnv("DB_PASSWORD", "agents_dev_password")

	// 构建最终配置
	cfg := &Config{
		Env:           env,
		DatabaseURL:   buildDatabaseURL(yamlCfg.Database, dbPassword),
		RedisURL:      buildRedisURL(yamlCfg.Redis),
		EtcdEndpoints: strings.Join(yamlCfg.Etcd.Endpoints, ","),
		EtcdPrefix:    yamlCfg.Etcd.Prefix,
		APIPort:       yamlCfg.Server.Port,
		Scheduler:     yamlCfg.Scheduler,
	}

	// 验证并填充调度器默认值
	cfg.Scheduler.validate()

	return cfg
}

// loadYAMLConfig 加载 YAML 配置文件
// 加载顺序：默认值 → common.yaml → {env}.yaml
func loadYAMLConfig(env Environment) *YAMLConfig {
	// 1. 初始化默认值
	cfg := &YAMLConfig{
		Server:   ServerConfig{Port: "8080"},
		Database: DatabaseConfig{Host: "localhost", Port: 5432, User: "agents", Name: "agents_admin", SSLMode: "disable"},
		Redis:    RedisConfig{Host: "localhost", Port: 6380, DB: 0},
		Etcd:     EtcdConfig{Endpoints: []string{"localhost:2379"}, Prefix: "/agents"},
		Scheduler: SchedulerConfig{
			NodeID: "scheduler-default",
			Strategy: SchedulerStrategyConfig{
				Default:    "label_match",
				Chain:      []string{"direct", "affinity", "label_match"},
				LabelMatch: SchedulerLabelMatchConfig{LoadBalance: true},
			},
			Redis:    SchedulerRedisConfig{ReadTimeout: 5 * time.Second, ReadCount: 10},
			Fallback: SchedulerFallbackConfig{Interval: 5 * time.Minute, StaleThreshold: 5 * time.Minute},
			Requeue:  SchedulerRequeueConfig{OfflineThreshold: 30 * time.Second},
		},
	}

	// 2. 加载 common.yaml（公共配置）
	for _, base := range configPaths {
		path := filepath.Join(base, "common.yaml")
		if data, err := os.ReadFile(path); err == nil {
			yaml.Unmarshal(data, cfg)
			break
		}
	}

	// 3. 加载 {env}.yaml（环境特定配置，覆盖公共配置）
	filename := fmt.Sprintf("%s.yaml", env)
	for _, base := range configPaths {
		path := filepath.Join(base, filename)
		if data, err := os.ReadFile(path); err == nil {
			yaml.Unmarshal(data, cfg)
			break
		}
	}

	return cfg
}

// buildDatabaseURL 构建 PostgreSQL 连接字符串
func buildDatabaseURL(db DatabaseConfig, password string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		db.User, password, db.Host, db.Port, db.Name, db.SSLMode)
}

// buildRedisURL 构建 Redis 连接字符串
func buildRedisURL(redis RedisConfig) string {
	return fmt.Sprintf("redis://%s:%d/%d", redis.Host, redis.Port, redis.DB)
}

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
	return fmt.Sprintf("Config{Env: %s, DB: %s, Redis: %s}",
		c.Env, maskPassword(c.DatabaseURL), c.RedisURL)
}

// maskPassword 隐藏密码
func maskPassword(url string) string {
	re := regexp.MustCompile(`(://[^:]+:)([^@]+)(@)`)
	return re.ReplaceAllString(url, "${1}***${3}")
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

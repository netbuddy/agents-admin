package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Load 加载配置
//
// 流程：
//  1. 加载 .env 和 .env.{env}（凭据单一数据源）
//  2. 根据 APP_ENV 加载 YAML 配置文件
//  3. 环境变量覆盖 YAML 配置
//  4. 构建最终 Config
func Load() *Config {
	env := parseEnv(getEnv("APP_ENV", "dev"))
	loadEnvFiles(env)

	yamlCfg := loadYAMLConfig(env)
	dbPassword := applyEnvOverrides(yamlCfg)

	// 构建数据库和 Redis URL（环境变量 DATABASE_URL/REDIS_URL 仍可覆盖）
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = buildDatabaseURL(yamlCfg.Database, dbPassword)
	}
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = buildRedisURL(yamlCfg.Redis)
	}

	cfg := &Config{
		Env:            env,
		DatabaseDriver: detectDatabaseDriver(yamlCfg.Database.Driver, databaseURL),
		DatabaseURL:    databaseURL,
		DatabaseDBName: yamlCfg.Database.Name,
		RedisURL:       redisURL,
		APIPort:        yamlCfg.APIServer.Port,
		Scheduler:      yamlCfg.Scheduler,
		TLS:            yamlCfg.TLS,
		Auth:           yamlCfg.Auth,
		MinIO:          yamlCfg.MinIO,
		APIServer:      yamlCfg.APIServer,
		Node:           yamlCfg.Node,
		ConfigFilePath: yamlCfg.loadedFrom,
	}
	cfg.Scheduler.validate()
	return cfg
}

// LoadNodeManager 加载 Node Manager 配置（供 cmd/nodemanager 使用）
// 使用与 API Server 相同的 YAML schema，只读取 node/redis/tls 章节
func LoadNodeManager() *Config {
	env := parseEnv(getEnv("APP_ENV", "dev"))
	loadEnvFiles(env)

	yamlCfg := loadYAMLConfig(env)

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = buildRedisURL(yamlCfg.Redis)
	}

	return &Config{
		Env:            env,
		RedisURL:       redisURL,
		TLS:            yamlCfg.TLS,
		APIServer:      yamlCfg.APIServer,
		Node:           yamlCfg.Node,
		ConfigFilePath: yamlCfg.loadedFrom,
	}
}

// ConfigFileName 返回当前环境对应的 YAML 配置文件名（{env}.yaml）
func ConfigFileName() string {
	env := parseEnv(getEnv("APP_ENV", "dev"))
	return fmt.Sprintf("%s.yaml", env)
}

// EnvFileName 返回当前环境对应的环境变量文件名（{env}.env）
func EnvFileName() string {
	env := parseEnv(getEnv("APP_ENV", "dev"))
	return fmt.Sprintf("%s.env", env)
}

// loadYAMLConfig 加载 YAML 配置文件
//
// 搜索候选名：{env}.yaml（如 dev.yaml、test.yaml、prod.yaml）
// 默认值全部硬编码在此函数中，不依赖外部文件
func loadYAMLConfig(env Environment) *yamlConfigInternal {
	cfg := &yamlConfigInternal{
		YAMLConfig: YAMLConfig{
			APIServer: APIServerConfig{Port: "8080"},
			Database:  DatabaseConfig{Driver: "mongodb", Host: "localhost", Port: 27017, Name: "agents_admin"},
			Redis:     RedisConfig{Host: "localhost", Port: 6380, DB: 0},
			MinIO:     MinIOConfig{Endpoint: "localhost:9000", AccessKey: "minioadmin", SecretKey: "minioadmin_dev_123", Bucket: "agents-admin"},
			Node:      NodeConfig{WorkspaceDir: "/tmp/agents-workspaces"},
			Scheduler: SchedulerConfig{
				NodeID: "api-server",
				Strategy: SchedulerStrategyConfig{
					Default:    "label_match",
					Chain:      []string{"direct", "affinity", "label_match"},
					LabelMatch: SchedulerLabelMatchConfig{LoadBalance: true},
				},
				Redis:    SchedulerRedisConfig{ReadTimeout: 5 * time.Second, ReadCount: 10},
				Fallback: SchedulerFallbackConfig{Interval: 5 * time.Minute, StaleThreshold: 5 * time.Minute},
				Requeue:  SchedulerRequeueConfig{OfflineThreshold: 30 * time.Second},
			},
		},
	}

	candidates := []string{fmt.Sprintf("%s.yaml", env)}
	for _, name := range candidates {
		for _, base := range effectiveConfigPaths() {
			path := filepath.Join(base, name)
			if data, err := os.ReadFile(path); err == nil {
				yaml.Unmarshal(data, &cfg.YAMLConfig)
				cfg.loadedFrom = path
				return cfg
			}
		}
	}
	return cfg
}

// applyEnvOverrides 从环境变量加载所有凭据（密码/密钥）
//
// 凭据只从环境变量读取，YAML 中不存储任何密码。
// 环境变量来源：.env 文件（godotenv）、systemd EnvironmentFile、shell 环境。
func applyEnvOverrides(yamlCfg *yamlConfigInternal) string {
	// 数据库用户名：MONGO_ROOT_USERNAME / POSTGRES_USER（与 Docker Compose 共用）
	if v := firstEnv("MONGO_ROOT_USERNAME", "POSTGRES_USER"); v != "" {
		yamlCfg.Database.User = v
	}
	// 数据库名：MONGO_DB_NAME / POSTGRES_DB（与 Docker Compose 共用，避免 YAML 重复）
	if v := firstEnv("MONGO_DB_NAME", "POSTGRES_DB"); v != "" {
		yamlCfg.Database.Name = v
	}
	// 数据库端口：MONGO_PORT / POSTGRES_PORT（与 Docker Compose 共用，避免 YAML 重复）
	if v := firstEnv("MONGO_PORT", "POSTGRES_PORT"); v != "" {
		var port int
		if _, err := fmt.Sscanf(v, "%d", &port); err == nil && port > 0 {
			yamlCfg.Database.Port = port
		}
	}

	// 数据库密码：DB_PASSWORD > MONGO_ROOT_PASSWORD > POSTGRES_PASSWORD > 硬编码默认值
	dbPassword := firstEnv("DB_PASSWORD", "MONGO_ROOT_PASSWORD", "POSTGRES_PASSWORD")
	if dbPassword == "" {
		dbPassword = "agents_dev_password"
	}

	// Redis 密码
	yamlCfg.Redis.Password = os.Getenv("REDIS_PASSWORD")
	// Redis 端口：REDIS_PORT（与 Docker Compose 共用，避免 YAML 重复）
	if v := os.Getenv("REDIS_PORT"); v != "" {
		var port int
		if _, err := fmt.Sscanf(v, "%d", &port); err == nil && port > 0 {
			yamlCfg.Redis.Port = port
		}
	}

	// MinIO 凭据（兼容 Docker Compose 和直接配置两种变量名）
	if v := os.Getenv("MINIO_ENDPOINT"); v != "" {
		yamlCfg.MinIO.Endpoint = v
	}
	if v := os.Getenv("MINIO_ROOT_USER"); v != "" {
		yamlCfg.MinIO.AccessKey = v
	} else if v := os.Getenv("MINIO_ACCESS_KEY"); v != "" {
		yamlCfg.MinIO.AccessKey = v
	}
	if v := os.Getenv("MINIO_ROOT_PASSWORD"); v != "" {
		yamlCfg.MinIO.SecretKey = v
	} else if v := os.Getenv("MINIO_SECRET_KEY"); v != "" {
		yamlCfg.MinIO.SecretKey = v
	}

	// Auth 凭据（只从环境变量读取）
	yamlCfg.Auth.JWTSecret = os.Getenv("JWT_SECRET")
	yamlCfg.Auth.AdminEmail = os.Getenv("ADMIN_EMAIL")
	yamlCfg.Auth.AdminPassword = os.Getenv("ADMIN_PASSWORD")
	yamlCfg.Auth.NodeToken = os.Getenv("NODE_TOKEN")

	return dbPassword
}

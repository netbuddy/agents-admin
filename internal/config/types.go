// Package config 统一配置管理
//
// 配置文件格式统一：API Server 和 Node Manager 共用同一 YAML schema，
// 通过不同章节（section）区分各组件的配置。
//
// 配置加载优先级（高→低）：
//  1. 环境变量（通过 .env 文件或 shell/systemd 注入）
//  2. YAML 配置文件（{env}.yaml，如 dev.yaml、test.yaml、prod.yaml）
//  3. 代码硬编码默认值
//
// 凭据单一数据源：
//
//	密码/密钥只存在 .env 文件中（YAML 中不存储任何密码）。
//	.env 文件同时被 Docker Compose（--env-file）、Go 应用（godotenv）、
//	systemd（EnvironmentFile=）共用，确保单一数据源。
//
// 配置路径确定策略：
//  1. --config 命令行参数（显式路径）
//  2. CONFIG_DIR 环境变量
//  3. 按 APP_ENV 选择默认路径：
//     - prod → /etc/agents-admin/
//     - dev/test → ./configs/
//
// 环境：
//   - 开发: APP_ENV=dev → configs/dev.yaml + .env.dev
//   - 测试: APP_ENV=test → configs/test.yaml + .env.test
//   - 生产: APP_ENV=prod → /etc/agents-admin/prod.yaml + prod.env
package config

import "time"

// Environment 环境类型
type Environment string

const (
	EnvProduction  Environment = "prod"
	EnvTest        Environment = "test" // 测试环境（集成测试 + E2E 共用）
	EnvDevelopment Environment = "dev"
)

// YAMLConfig 统一 YAML 配置文件结构
// API Server 和 Node Manager 共用此格式，通过章节区分
type YAMLConfig struct {
	APIServer APIServerConfig `yaml:"api_server"` // API Server（端口 + URL）
	Database  DatabaseConfig  `yaml:"database"`   // 数据库（API Server）
	Redis     RedisConfig     `yaml:"redis"`      // Redis（共享）
	MinIO     MinIOConfig     `yaml:"minio"`      // MinIO 对象存储
	Node      NodeConfig      `yaml:"node"`       // 节点共性配置（Node Manager）
	Scheduler SchedulerConfig `yaml:"scheduler"`  // 调度器（API Server）
	TLS       TLSConfig       `yaml:"tls"`        // TLS（共享）
	Auth      AuthConfig      `yaml:"auth"`       // 认证（API Server）
}

// AuthConfig 认证配置
// 注意：JWTSecret/AdminEmail/AdminPassword 只从环境变量读取，不存储在 YAML 中
type AuthConfig struct {
	JWTSecret       string `yaml:"-"`                 // 只从 JWT_SECRET 环境变量读取
	AccessTokenTTL  string `yaml:"access_token_ttl"`  // 例如 "15m"
	RefreshTokenTTL string `yaml:"refresh_token_ttl"` // 例如 "168h"
	AdminEmail      string `yaml:"-"`                 // 只从 ADMIN_EMAIL 环境变量读取
	AdminPassword   string `yaml:"-"`                 // 只从 ADMIN_PASSWORD 环境变量读取
	NodeToken       string `yaml:"-"`                 // 只从 NODE_TOKEN 环境变量读取（NodeManager 共享密钥）
}

// APIServerConfig API Server 配置
type APIServerConfig struct {
	Port string `yaml:"port"` // 监听端口
	URL  string `yaml:"url"`  // API Server 完整 URL（Node Manager 连接用）
}

// TLSConfig TLS/HTTPS 配置
type TLSConfig struct {
	Enabled      bool       `yaml:"enabled"`
	CertFile     string     `yaml:"cert_file"`     // 服务端证书
	KeyFile      string     `yaml:"key_file"`      // 服务端私钥
	CAFile       string     `yaml:"ca_file"`       // CA 证书（用于验证客户端/服务端）
	CertDir      string     `yaml:"cert_dir"`      // 证书目录（auto_generate 时使用，默认 /etc/agents-admin/certs）
	AutoGenerate bool       `yaml:"auto_generate"` // 启用时若证书不存在则自动生成自签名证书
	Hosts        string     `yaml:"hosts"`         // 证书 SANs（逗号分隔的 IP/域名，自动包含 localhost）
	ACME         ACMEConfig `yaml:"acme"`          // Let's Encrypt 自动证书（互联网域名）
}

// ACMEConfig Let's Encrypt / ACME 自动证书配置
type ACMEConfig struct {
	Enabled  bool     `yaml:"enabled"`   // 启用 ACME 自动证书
	Domains  []string `yaml:"domains"`   // 域名列表，如 ["admin.example.com"]
	Email    string   `yaml:"email"`     // 联系邮箱（Let's Encrypt 要求）
	CacheDir string   `yaml:"cache_dir"` // 证书缓存目录（默认 /etc/agents-admin/certs/acme）
}

type DatabaseConfig struct {
	Driver   string `yaml:"driver"` // "postgres", "sqlite", or "mongodb"（默认 mongodb）
	Path     string `yaml:"path"`   // SQLite 文件路径
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"-"` // 只从环境变量读取（DB_PASSWORD / MONGO_ROOT_PASSWORD）
	Name     string `yaml:"name"`
	SSLMode  string `yaml:"sslmode"`
	URI      string `yaml:"uri"` // MongoDB 连接 URI（优先于 host/port，如 mongodb://localhost:27017）
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	DB       int    `yaml:"db"`
	Password string `yaml:"-"`   // 只从 REDIS_PASSWORD 环境变量读取
	URL      string `yaml:"url"` // 直接指定 URL（Node Manager 使用，优先于 host/port/db）
}

// MinIOConfig MinIO 对象存储配置
type MinIOConfig struct {
	Endpoint  string `yaml:"endpoint"` // 例如 localhost:9000
	AccessKey string `yaml:"-"`        // 只从 MINIO_ROOT_USER 环境变量读取
	SecretKey string `yaml:"-"`        // 只从 MINIO_ROOT_PASSWORD 环境变量读取
	UseSSL    bool   `yaml:"use_ssl"`  // 是否使用 HTTPS
	Bucket    string `yaml:"bucket"`   // 默认 bucket 名称
}

// NodeConfig 节点共性配置（Node Manager 使用）
type NodeConfig struct {
	ID           string            `yaml:"id"`
	WorkspaceDir string            `yaml:"workspace_dir"`
	Labels       map[string]string `yaml:"labels"`
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
	Env            Environment
	DatabaseDriver string // "postgres", "sqlite", or "mongodb"
	DatabaseURL    string
	DatabaseDBName string // MongoDB 数据库名称
	RedisURL       string
	APIPort        string
	Scheduler      SchedulerConfig
	TLS            TLSConfig
	Auth           AuthConfig
	MinIO          MinIOConfig     // MinIO 对象存储配置
	APIServer      APIServerConfig // API Server 配置（端口 + URL）
	Node           NodeConfig      // 节点共性配置（Node Manager 使用）
	ConfigFilePath string          // 实际加载的配置文件路径（用于配置管理 API）
}

// yamlConfigInternal 内部包装，记录配置文件来源（不参与 YAML 序列化）
type yamlConfigInternal struct {
	YAMLConfig `yaml:",inline"`
	loadedFrom string
}

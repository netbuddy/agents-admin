package setup

import "sync"

// ========== Request/Response Types ==========

// InfoResponse /setup/api/info 响应
type InfoResponse struct {
	Hostname string   `json:"hostname"`
	IPs      []string `json:"ips"`
	IsRoot   bool     `json:"is_root"`
}

// ValidateRequest /setup/api/validate 请求
type ValidateRequest struct {
	Database DatabaseConfig `json:"database"`
	Redis    RedisConfig    `json:"redis"`
	TLS      TLSConfig      `json:"tls"`
	Auth     AuthConfig     `json:"auth"`
	Server   ServerConfig   `json:"server"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type     string `json:"type"` // "postgres", "sqlite", or "mongodb"
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode"`
	Path     string `json:"path"` // SQLite 文件路径
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
}

// SetupMinIOConfig MinIO 配置（Setup Wizard 使用）
type SetupMinIOConfig struct {
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}

// TLSConfig TLS 配置
type TLSConfig struct {
	Enabled     bool   `json:"enabled"`
	Mode        string `json:"mode"` // auto_generate | acme | manual
	Hosts       string `json:"hosts,omitempty"`
	AcmeDomains string `json:"acme_domains,omitempty"`
	AcmeEmail   string `json:"acme_email,omitempty"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	AdminEmail    string `json:"admin_email"`
	AdminPassword string `json:"admin_password"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string `json:"port"`
}

// CheckResult 检查结果
type CheckResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// ValidateResponse 验证响应
type ValidateResponse struct {
	Valid  bool                   `json:"valid"`
	Checks map[string]CheckResult `json:"checks"`
}

// ApplyResponse 应用响应
type ApplyResponse struct {
	Success        bool   `json:"success"`
	ConfigDir      string `json:"config_dir"`
	Message        string `json:"message"`
	SystemdInstall bool   `json:"systemd_install,omitempty"`
}

// InitDBRequest 数据库初始化请求
type InitDBRequest struct {
	Database       DatabaseConfig `json:"database"`
	ConfirmDestroy bool           `json:"confirm_destroy"` // 用户必须确认
}

// InitDBResponse 数据库初始化响应
type InitDBResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ========== Infrastructure Types ==========

// InfraGenerateRequest 基础设施生成请求
type InfraGenerateRequest struct {
	MongoPort        int `json:"mongo_port"`
	RedisPort        int `json:"redis_port"`
	MinIOAPIPort     int `json:"minio_api_port"`
	MinIOConsolePort int `json:"minio_console_port"`
}

// InfraGenerateResponse 基础设施生成响应
type InfraGenerateResponse struct {
	Success          bool   `json:"success"`
	Message          string `json:"message"`
	MongoUser        string `json:"mongo_user"`
	MongoPassword    string `json:"mongo_password"`
	MongoPort        int    `json:"mongo_port"`
	RedisPassword    string `json:"redis_password"`
	RedisPort        int    `json:"redis_port"`
	MinIOUser        string `json:"minio_user"`
	MinIOPassword    string `json:"minio_password"`
	MinIOAPIPort     int    `json:"minio_api_port"`
	MinIOConsolePort int    `json:"minio_console_port"`
	ComposeFile      string `json:"compose_file"`
	EnvFile          string `json:"env_file"`
}

// InfraDeployResponse 基础设施部署响应
type InfraDeployResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Output  string `json:"output,omitempty"`
}

// InfraStatusResponse 基础设施状态响应
type InfraStatusResponse struct {
	Status   string                   `json:"status"` // "not_deployed", "starting", "healthy", "unhealthy", "error"
	Services map[string]ServiceStatus `json:"services"`
	Message  string                   `json:"message,omitempty"`
}

// ServiceStatus 单个服务状态
type ServiceStatus struct {
	Running bool   `json:"running"`
	Health  string `json:"health"` // "healthy", "unhealthy", "starting", "none"
}

// infraState 基础设施部署状态（Server 级别）
type infraState struct {
	mu           sync.Mutex
	generated    bool
	deploying    bool
	composeFile  string
	envFile      string
	deployOutput string
	deployErr    string
}

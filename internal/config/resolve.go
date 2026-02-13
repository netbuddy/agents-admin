package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// configDir 由外部通过 SetConfigDir 指定，优先级最高
var configDir string

// envSearchDirs .env 文件搜索目录（仅 dev/test 使用，生产环境由 systemd 注入）
var envSearchDirs = []string{
	".",
	"..",
}

// SetConfigDir 设置配置文件目录（用于 --config 命令行参数）
// 调用后 Load 将优先从该目录加载配置文件
func SetConfigDir(dir string) {
	configDir = dir
}

// configPathsForEnv 根据环境返回配置文件搜索路径
func configPathsForEnv(env Environment) []string {
	if env == EnvProduction {
		return []string{"/etc/agents-admin"}
	}
	// dev/test: 项目根目录的 configs/
	return []string{"configs", "../configs"}
}

// GetConfigDir 返回当前配置目录（首次运行向导用于确定配置文件保存位置）
//
// 优先级：
//  1. --config 命令行参数
//  2. root 用户 → /etc/agents-admin（与 deb 包一致）
//  3. /etc/agents-admin 已存在且可写
//  4. 开发环境回退到 configs/
func GetConfigDir() string {
	if configDir != "" {
		return configDir
	}
	// root 用户（sudo 运行或 deb systemd）→ 统一使用 /etc/agents-admin
	if IsRoot() {
		return "/etc/agents-admin"
	}
	// 非 root 但 /etc/agents-admin 存在且可写（例如 agents-admin 用户）
	if info, err := os.Stat("/etc/agents-admin"); err == nil && info.IsDir() {
		testFile := "/etc/agents-admin/.write_test"
		if err := os.WriteFile(testFile, []byte("test"), 0644); err == nil {
			os.Remove(testFile)
			return "/etc/agents-admin"
		}
	}
	// 开发环境回退到 configs/
	return "configs"
}

// GetConfigFilePath 返回当前加载的配置文件路径
func GetConfigFilePath() string {
	env := parseEnv(getEnv("APP_ENV", "dev"))
	cfg := loadYAMLConfig(env)
	return cfg.loadedFrom
}

// ConfigExists 检测配置文件是否存在（用于首次运行检测）
//
// 搜索 {APP_ENV}.yaml（如 dev.yaml、prod.yaml），存在即视为已配置。
// API Server 和 Node Manager 共用同一份配置文件，仅读取各自关心的章节。
func ConfigExists() bool {
	return findConfigFile() != ""
}

// NodeManagerConfigExists 检测 Node Manager 配置文件是否存在
// 与 ConfigExists 逻辑一致：API Server 和 Node Manager 共用 {env}.yaml。
func NodeManagerConfigExists() bool {
	return ConfigExists()
}

// IsRoot 检测当前进程是否以 root 身份运行
func IsRoot() bool {
	return os.Getuid() == 0
}

// ReadConfigFile 读取当前配置文件的原始 YAML 内容（用于配置管理 API）
func ReadConfigFile() ([]byte, string, error) {
	path := GetConfigFilePath()
	if path == "" {
		return nil, "", fmt.Errorf("no config file found")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, path, err
	}
	return data, path, nil
}

// WriteConfigFile 写入配置文件内容（用于配置管理 API）
func WriteConfigFile(content []byte) (string, error) {
	path := GetConfigFilePath()
	if path == "" {
		// 如果没有现有文件，写入默认位置
		path = filepath.Join(GetConfigDir(), ConfigFileName())
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return path, err
	}
	return path, os.WriteFile(path, content, 0640)
}

// findConfigFile 在搜索路径中查找第一个存在的配置文件
func findConfigFile(extraNames ...string) string {
	env := parseEnv(getEnv("APP_ENV", "dev"))
	names := []string{fmt.Sprintf("%s.yaml", env)}
	names = append(names, extraNames...)
	paths := effectiveConfigPaths()
	for _, base := range paths {
		for _, name := range names {
			p := filepath.Join(base, name)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	return ""
}

// effectiveConfigPaths 返回实际搜索路径
//
// 优先级：
//  1. --config 命令行参数（SetConfigDir）
//  2. CONFIG_DIR 环境变量
//  3. 按 APP_ENV 选择默认路径
func effectiveConfigPaths() []string {
	if configDir != "" {
		return []string{configDir}
	}
	if dir := os.Getenv("CONFIG_DIR"); dir != "" {
		return []string{dir}
	}
	env := parseEnv(getEnv("APP_ENV", "dev"))
	return configPathsForEnv(env)
}

// loadEnvFiles 加载 .env 文件
//
// 生产环境不搜索 .env 文件（密码由 systemd EnvironmentFile 或 shell 环境注入）。
// dev/test 环境加载 .env.{env} 文件（凭据单一数据源，与 Docker Compose 共用）。
func loadEnvFiles(env Environment) {
	// 生产环境：不搜索 .env 文件
	if env == EnvProduction {
		return
	}

	// 加载 .env.{env}（dev/test 凭据文件，与 Docker Compose 共用）
	// godotenv.Load 不覆盖已有环境变量，优先级低于 shell 环境变量
	envFileName := fmt.Sprintf(".env.%s", string(env))
	for _, dir := range envSearchDirs {
		if err := godotenv.Load(filepath.Join(dir, envFileName)); err == nil {
			break
		}
	}
}

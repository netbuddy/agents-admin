// Package sysinstall 提供系统级安装工具
//
// 支持二进制和 deb 包统一安装路径：
//   - 创建 agents-admin 系统用户
//   - 创建必要目录并设置权限
//   - 安装 systemd service 文件
//   - 检测运行环境（root、systemd）
package sysinstall

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

const (
	// ServiceUser 系统服务用户名
	ServiceUser = "agents-admin"
	// ConfigDir 统一配置目录
	ConfigDir = "/etc/agents-admin"
	// DataDir 数据目录
	DataDir = "/var/lib/agents-admin"
	// LogDir 日志目录
	LogDir = "/var/log/agents-admin"
	// CertsSubDir 证书子目录
	CertsSubDir = "certs"
)

// EnsureSystemUser 创建系统用户（如果不存在）
func EnsureSystemUser() error {
	if _, err := user.Lookup(ServiceUser); err == nil {
		return nil // 用户已存在
	}

	cmd := exec.Command("useradd",
		"--system",
		"--no-create-home",
		"--shell", "/usr/sbin/nologin",
		ServiceUser,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create system user %s: %v (%s)", ServiceUser, err, strings.TrimSpace(string(output)))
	}

	log.Printf("Created system user: %s", ServiceUser)
	return nil
}

// EnsureDirectories 创建必要的系统目录并设置权限
func EnsureDirectories() error {
	dirs := []struct {
		path string
		perm os.FileMode
	}{
		{ConfigDir, 0755},
		{filepath.Join(ConfigDir, CertsSubDir), 0755},
		{DataDir, 0755},
		{filepath.Join(DataDir, "workspaces"), 0755},
		{LogDir, 0755},
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d.path, d.perm); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", d.path, err)
		}
	}

	// 设置目录所有权为 agents-admin 用户
	if u, err := user.Lookup(ServiceUser); err == nil {
		for _, d := range dirs {
			chownRecursive(d.path, u.Uid, u.Gid)
		}
	}

	return nil
}

// chownRecursive 递归设置目录所有权
func chownRecursive(path, uid, gid string) {
	cmd := exec.Command("chown", "-R", uid+":"+gid, path)
	cmd.Run() // 忽略错误（可能部分文件无法修改）
}

// InstallSystemdService 安装 systemd service 文件
// serviceName: 如 "agents-admin-api-server"
// serviceContent: systemd unit file 内容
func InstallSystemdService(serviceName, serviceContent string) error {
	servicePath := filepath.Join("/etc/systemd/system", serviceName+".service")

	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	// daemon-reload
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("systemctl daemon-reload failed: %w", err)
	}

	// enable
	if err := exec.Command("systemctl", "enable", serviceName).Run(); err != nil {
		return fmt.Errorf("systemctl enable failed: %w", err)
	}

	log.Printf("Installed systemd service: %s", servicePath)
	return nil
}

// GenerateServiceFile 生成 systemd service 文件内容
// binaryPath: 二进制文件的实际路径（通过 os.Executable() 获取）
// serviceName: 服务名
// description: 服务描述
// envFile: 环境变量文件路径（可选）
// extraAfter: 额外的 After 依赖（如 "postgresql.service redis.service"）
func GenerateServiceFile(binaryPath, serviceName, description, envFile, extraAfter string) string {
	after := "network-online.target"
	if extraAfter != "" {
		after += " " + extraAfter
	}

	envFileLine := ""
	if envFile != "" {
		envFileLine = fmt.Sprintf("EnvironmentFile=-%s\n", envFile)
	}

	readWritePaths := fmt.Sprintf("%s %s %s", ConfigDir, DataDir, LogDir)

	return fmt.Sprintf(`[Unit]
Description=%s
After=%s
Wants=network-online.target

[Service]
Type=simple
User=%s
Group=%s
%sExecStart=%s --config %s
Restart=always
RestartSec=5
StartLimitBurst=5
StartLimitIntervalSec=60

# 安全加固
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=%s /tmp
PrivateTmp=true

# 日志
StandardOutput=journal
StandardError=journal
SyslogIdentifier=%s

[Install]
WantedBy=multi-user.target
`, description, after, ServiceUser, ServiceUser, envFileLine, binaryPath, ConfigDir, readWritePaths, serviceName)
}

// GetExecutablePath 获取当前二进制的实际路径
func GetExecutablePath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	// 解析符号链接
	real, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return exe
	}
	return real
}

// IsRoot 检测是否以 root 运行
func IsRoot() bool {
	return os.Getuid() == 0
}

// IsUnderSystemd 检测是否在 systemd 下运行
func IsUnderSystemd() bool {
	if os.Getenv("INVOCATION_ID") != "" {
		return true
	}
	return os.Getppid() == 1
}

// HasSystemd 检测系统是否有 systemd
func HasSystemd() bool {
	_, err := exec.LookPath("systemctl")
	return err == nil
}

// SetFileOwnership 设置文件所有权为 agents-admin 用户
func SetFileOwnership(path string) {
	if u, err := user.Lookup(ServiceUser); err == nil {
		chownRecursive(path, u.Uid, u.Gid)
	}
}

// SetSecureFilePermissions 设置敏感文件权限（如 .env 文件）
func SetSecureFilePermissions(path string) {
	os.Chmod(path, 0640)
	if u, err := user.Lookup(ServiceUser); err == nil {
		// root:agents-admin 0640
		exec.Command("chown", "root:"+u.Gid, path).Run()
	}
}

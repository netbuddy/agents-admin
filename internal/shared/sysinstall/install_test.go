package sysinstall

import (
	"os"
	"strings"
	"testing"
)

func TestConstants(t *testing.T) {
	if ServiceUser != "agents-admin" {
		t.Errorf("ServiceUser = %q, want agents-admin", ServiceUser)
	}
	if ConfigDir != "/etc/agents-admin" {
		t.Errorf("ConfigDir = %q, want /etc/agents-admin", ConfigDir)
	}
	if DataDir != "/var/lib/agents-admin" {
		t.Errorf("DataDir = %q, want /var/lib/agents-admin", DataDir)
	}
	if LogDir != "/var/log/agents-admin" {
		t.Errorf("LogDir = %q, want /var/log/agents-admin", LogDir)
	}
	if CertsSubDir != "certs" {
		t.Errorf("CertsSubDir = %q, want certs", CertsSubDir)
	}
}

func TestIsRoot(t *testing.T) {
	// 在普通测试环境中，不应该是 root
	if os.Getuid() != 0 && IsRoot() {
		t.Error("IsRoot() should return false for non-root user")
	}
}

func TestIsUnderSystemd(t *testing.T) {
	original := os.Getenv("INVOCATION_ID")
	defer os.Setenv("INVOCATION_ID", original)

	os.Unsetenv("INVOCATION_ID")
	if os.Getppid() != 1 && IsUnderSystemd() {
		t.Error("IsUnderSystemd() should return false in test environment")
	}

	os.Setenv("INVOCATION_ID", "test-invocation")
	if !IsUnderSystemd() {
		t.Error("IsUnderSystemd() should return true when INVOCATION_ID is set")
	}
}

func TestHasSystemd(t *testing.T) {
	// 大多数 Linux 系统有 systemctl
	result := HasSystemd()
	t.Logf("HasSystemd() = %v", result)
}

func TestGetExecutablePath(t *testing.T) {
	path := GetExecutablePath()
	if path == "" {
		t.Error("GetExecutablePath() should return non-empty path")
	}
	t.Logf("Executable path: %s", path)
}

func TestGenerateServiceFile(t *testing.T) {
	tests := []struct {
		name        string
		binaryPath  string
		serviceName string
		description string
		envFile     string
		extraAfter  string
		wantParts   []string
		dontWant    []string
	}{
		{
			name:        "API Server with env and extra after",
			binaryPath:  "/usr/local/bin/agents-admin-api-server",
			serviceName: "agents-admin-api-server",
			description: "Agents Admin API Server",
			envFile:     "/etc/agents-admin/prod.env",
			extraAfter:  "postgresql.service redis.service",
			wantParts: []string{
				"Description=Agents Admin API Server",
				"After=network-online.target postgresql.service redis.service",
				"User=agents-admin",
				"Group=agents-admin",
				"EnvironmentFile=-/etc/agents-admin/prod.env",
				"ExecStart=/usr/local/bin/agents-admin-api-server --config /etc/agents-admin",
				"Restart=always",
				"NoNewPrivileges=true",
				"ProtectSystem=strict",
				"SyslogIdentifier=agents-admin-api-server",
				"WantedBy=multi-user.target",
				"ReadWritePaths=/etc/agents-admin /var/lib/agents-admin /var/log/agents-admin",
			},
		},
		{
			name:        "Node Manager without env",
			binaryPath:  "/usr/local/bin/agents-admin-node-manager",
			serviceName: "agents-admin-node-manager",
			description: "Agents Admin Node Manager",
			envFile:     "",
			extraAfter:  "",
			wantParts: []string{
				"Description=Agents Admin Node Manager",
				"After=network-online.target",
				"ExecStart=/usr/local/bin/agents-admin-node-manager --config /etc/agents-admin",
				"SyslogIdentifier=agents-admin-node-manager",
			},
			dontWant: []string{
				"EnvironmentFile",
				"postgresql.service",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateServiceFile(tt.binaryPath, tt.serviceName, tt.description, tt.envFile, tt.extraAfter)

			for _, want := range tt.wantParts {
				if !strings.Contains(result, want) {
					t.Errorf("GenerateServiceFile should contain %q, got:\n%s", want, result)
				}
			}

			for _, dontWant := range tt.dontWant {
				if strings.Contains(result, dontWant) {
					t.Errorf("GenerateServiceFile should NOT contain %q", dontWant)
				}
			}
		})
	}
}

func TestGenerateServiceFileFormat(t *testing.T) {
	result := GenerateServiceFile("/usr/local/bin/test", "test-svc", "Test Service", "", "")

	// 验证基本的 systemd unit file 格式
	if !strings.HasPrefix(result, "[Unit]") {
		t.Error("service file should start with [Unit]")
	}
	if !strings.Contains(result, "[Service]") {
		t.Error("service file should contain [Service] section")
	}
	if !strings.Contains(result, "[Install]") {
		t.Error("service file should contain [Install] section")
	}
}

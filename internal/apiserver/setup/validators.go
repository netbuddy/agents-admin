package setup

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agents-admin/internal/shared/sysinstall"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// ========== Connection Tests ==========

// testSQLite 测试 SQLite 文件路径是否可写
func testSQLite(cfg DatabaseConfig, configDir string) CheckResult {
	dbPath := cfg.Path
	if dbPath == "" {
		dbPath = filepath.Join(sysinstall.DataDir, "agents-admin.db")
	}

	// 确保目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return CheckResult{OK: false, Message: fmt.Sprintf("Cannot create directory %s: %v", dir, err)}
	}

	// 测试写入权限
	testFile := filepath.Join(dir, ".write_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return CheckResult{OK: false, Message: fmt.Sprintf("Directory not writable: %v", err)}
	}
	os.Remove(testFile)

	return CheckResult{OK: true, Message: fmt.Sprintf("SQLite path: %s (writable)", dbPath)}
}

func testPostgreSQL(cfg DatabaseConfig) CheckResult {
	if cfg.Host == "" {
		return CheckResult{OK: false, Message: "Host is required"}
	}
	if cfg.Port == 0 {
		cfg.Port = 5432
	}
	if cfg.User == "" {
		return CheckResult{OK: false, Message: "User is required"}
	}
	if cfg.DBName == "" {
		return CheckResult{OK: false, Message: "Database name is required"}
	}
	if cfg.SSLMode == "" {
		cfg.SSLMode = "disable"
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s&connect_timeout=5",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return CheckResult{OK: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return CheckResult{OK: false, Message: fmt.Sprintf("Ping failed: %v", err)}
	}

	return CheckResult{OK: true, Message: "Connected successfully"}
}

func testRedis(cfg RedisConfig) CheckResult {
	if cfg.Host == "" {
		return CheckResult{OK: false, Message: "Host is required"}
	}
	if cfg.Port == 0 {
		cfg.Port = 6379
	}

	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return CheckResult{OK: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// AUTH if password is set
	if cfg.Password != "" {
		fmt.Fprintf(conn, "*2\r\n$4\r\nAUTH\r\n$%d\r\n%s\r\n", len(cfg.Password), cfg.Password)
		buf := make([]byte, 64)
		n, err := conn.Read(buf)
		if err != nil {
			return CheckResult{OK: false, Message: fmt.Sprintf("AUTH failed: %v", err)}
		}
		resp := string(buf[:n])
		if !strings.HasPrefix(resp, "+OK") {
			return CheckResult{OK: false, Message: fmt.Sprintf("AUTH failed: %s", strings.TrimSpace(resp))}
		}
	}

	// PING
	fmt.Fprintf(conn, "*1\r\n$4\r\nPING\r\n")

	buf := make([]byte, 64)
	n, err := conn.Read(buf)
	if err != nil {
		return CheckResult{OK: false, Message: fmt.Sprintf("PING failed: %v", err)}
	}

	if strings.Contains(string(buf[:n]), "PONG") {
		return CheckResult{OK: true, Message: "Connected (PONG)"}
	}

	return CheckResult{OK: false, Message: fmt.Sprintf("Unexpected: %s", strings.TrimSpace(string(buf[:n])))}
}

// testMongoDB 测试 MongoDB 连接
func testMongoDB(cfg DatabaseConfig) CheckResult {
	if cfg.Host == "" {
		return CheckResult{OK: false, Message: "Host is required"}
	}
	if cfg.Port == 0 {
		cfg.Port = 27017
	}

	// TCP 连通性检测
	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return CheckResult{OK: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	conn.Close()

	return CheckResult{OK: true, Message: fmt.Sprintf("Connected to %s", addr)}
}

func validateAuth(cfg AuthConfig) CheckResult {
	if cfg.AdminEmail == "" {
		return CheckResult{OK: false, Message: "Admin email is required"}
	}
	if !strings.Contains(cfg.AdminEmail, "@") {
		return CheckResult{OK: false, Message: "Invalid email format"}
	}
	if len(cfg.AdminPassword) < 8 {
		return CheckResult{OK: false, Message: "Password must be at least 8 characters"}
	}
	return CheckResult{OK: true, Message: "Valid"}
}

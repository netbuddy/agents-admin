package setup

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"agents-admin/deployments"
	"agents-admin/internal/config"
	"agents-admin/internal/shared/sysinstall"
)

// ========== Core Handlers ==========

// handleInfo GET /setup/api/info
func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	hostname, _ := os.Hostname()
	resp := InfoResponse{
		Hostname: hostname,
		IPs:      getLocalIPs(),
		IsRoot:   sysinstall.IsRoot(),
	}
	jsonResp(w, http.StatusOK, resp)
}

// handleValidate POST /setup/api/validate
func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResp(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON: " + err.Error()})
		return
	}

	checks := make(map[string]CheckResult)
	allValid := true

	// 1. Database (MongoDB, PostgreSQL, or SQLite)
	var dbCheck CheckResult
	switch req.Database.Type {
	case "sqlite":
		dbCheck = testSQLite(req.Database, s.configDir)
	case "mongodb":
		dbCheck = testMongoDB(req.Database)
	default:
		dbCheck = testPostgreSQL(req.Database)
	}
	checks["database"] = dbCheck
	if !dbCheck.OK {
		allValid = false
	}

	// 2. Redis
	redisCheck := testRedis(req.Redis)
	checks["redis"] = redisCheck
	if !redisCheck.OK {
		allValid = false
	}

	// 3. Auth 验证
	authCheck := validateAuth(req.Auth)
	checks["auth"] = authCheck
	if !authCheck.OK {
		allValid = false
	}

	jsonResp(w, http.StatusOK, ValidateResponse{Valid: allValid, Checks: checks})
}

// handleApply POST /setup/api/apply
func (s *Server) handleApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResp(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON: " + err.Error()})
		return
	}

	// 填充默认值
	if req.Database.Type == "" {
		req.Database.Type = "postgres"
	}
	if req.Database.Type == "postgres" {
		if req.Database.Port == 0 {
			req.Database.Port = 5432
		}
		if req.Database.SSLMode == "" {
			req.Database.SSLMode = "disable"
		}
	}
	if req.Database.Type == "sqlite" && req.Database.Path == "" {
		req.Database.Path = filepath.Join(sysinstall.DataDir, "agents-admin.db")
	}
	if req.Redis.Port == 0 {
		req.Redis.Port = 6379
	}
	if req.Server.Port == "" {
		req.Server.Port = "8080"
	}

	// 生成 JWT Secret
	jwtSecret := generateRandomString(32)

	// 如果以 root 运行，执行系统级安装
	systemdInstalled := false
	if sysinstall.IsRoot() {
		// 创建系统用户
		if err := sysinstall.EnsureSystemUser(); err != nil {
			log.Printf("WARNING: failed to create system user: %v", err)
		}
		// 创建必要目录
		if err := sysinstall.EnsureDirectories(); err != nil {
			log.Printf("WARNING: failed to create directories: %v", err)
		}
	}

	// 确保配置目录存在
	if err := os.MkdirAll(s.configDir, 0755); err != nil {
		jsonResp(w, http.StatusInternalServerError, map[string]string{"error": "Cannot create config dir: " + err.Error()})
		return
	}

	// 写 {env}.yaml（合并已有配置）
	if err := writeYAMLConfig(s.configDir, req); err != nil {
		jsonResp(w, http.StatusInternalServerError, map[string]string{"error": "Failed to write yaml: " + err.Error()})
		return
	}

	// 写 {env}.env（仅敏感信息）
	if err := writeEnvConfig(s.configDir, req, jwtSecret); err != nil {
		jsonResp(w, http.StatusInternalServerError, map[string]string{"error": "Failed to write env: " + err.Error()})
		return
	}

	// root 安装：设置文件权限并安装 systemd 服务
	if sysinstall.IsRoot() {
		sysinstall.SetFileOwnership(s.configDir)
		sysinstall.SetSecureFilePermissions(filepath.Join(s.configDir, config.EnvFileName()))

		// 安装 systemd service
		if sysinstall.HasSystemd() && !isUnderSystemd() {
			exePath := sysinstall.GetExecutablePath()
			serviceContent := sysinstall.GenerateServiceFile(
				exePath,
				"agents-admin-api-server",
				"Agents Admin API Server",
				filepath.Join(s.configDir, config.EnvFileName()),
				"postgresql.service redis.service",
			)
			if err := sysinstall.InstallSystemdService("agents-admin-api-server", serviceContent); err != nil {
				log.Printf("WARNING: failed to install systemd service: %v", err)
			} else {
				systemdInstalled = true
			}
		}
	}

	underSystemd := isUnderSystemd()
	message := "Configuration saved to " + s.configDir + "/."
	if systemdInstalled {
		message += " Systemd service installed. Use 'sudo systemctl start agents-admin-api-server' to start."
	} else if underSystemd {
		message += " The service will restart automatically."
	} else {
		message += " Please restart the program manually."
	}

	jsonResp(w, http.StatusOK, ApplyResponse{
		Success:        true,
		ConfigDir:      s.configDir,
		Message:        message,
		SystemdInstall: systemdInstalled,
	})

	log.Printf("Configuration saved to %s/", s.configDir)

	go func() {
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}()
}

// handleInitDB POST /setup/api/init-db — 初始化数据库（全新安装）
func (s *Server) handleInitDB(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req InitDBRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResp(w, http.StatusBadRequest, InitDBResponse{Message: "Invalid JSON: " + err.Error()})
		return
	}

	if !req.ConfirmDestroy {
		jsonResp(w, http.StatusBadRequest, InitDBResponse{
			Message: "Database initialization requires confirmation. This will DELETE ALL existing data.",
		})
		return
	}

	if req.Database.Type == "sqlite" {
		// SQLite: 数据库文件会在程序启动时由 AutoMigrate 自动创建
		jsonResp(w, http.StatusOK, InitDBResponse{
			Success: true,
			Message: "SQLite database will be auto-initialized on first start.",
		})
		return
	}

	// PostgreSQL: 执行嵌入的 init-db.sql
	if req.Database.Port == 0 {
		req.Database.Port = 5432
	}
	if req.Database.SSLMode == "" {
		req.Database.SSLMode = "disable"
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s&connect_timeout=10",
		req.Database.User, req.Database.Password,
		req.Database.Host, req.Database.Port,
		req.Database.DBName, req.Database.SSLMode)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		jsonResp(w, http.StatusOK, InitDBResponse{Message: fmt.Sprintf("Connection failed: %v", err)})
		return
	}
	defer db.Close()

	// 执行全量初始化脚本
	if _, err := db.Exec(deployments.InitDBSQL); err != nil {
		jsonResp(w, http.StatusOK, InitDBResponse{
			Message: fmt.Sprintf("Database initialization failed: %v", err),
		})
		return
	}

	log.Println("Database initialized successfully with init-db.sql")
	jsonResp(w, http.StatusOK, InitDBResponse{
		Success: true,
		Message: "Database initialized successfully (26 tables created).",
	})
}

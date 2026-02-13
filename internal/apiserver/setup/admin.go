package setup

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"agents-admin/internal/apiserver/auth"
	"agents-admin/internal/shared/storage"
	"agents-admin/internal/shared/storage/mongostore"

	"agents-admin/internal/shared/storage/dbutil"
)

// CreateAdminRequest 创建管理员请求
type CreateAdminRequest struct {
	Database DatabaseConfig `json:"database"`
	Email    string         `json:"email"`
	Password string         `json:"password"`
}

// CreateAdminResponse 创建管理员响应
type CreateAdminResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// handleCreateAdmin POST /setup/api/create-admin
// 在已初始化的数据库中创建管理员用户
func (s *Server) handleCreateAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResp(w, http.StatusBadRequest, CreateAdminResponse{Message: "Invalid JSON: " + err.Error()})
		return
	}

	// 验证必填字段
	if req.Email == "" || req.Password == "" {
		jsonResp(w, http.StatusBadRequest, CreateAdminResponse{Message: "email and password are required"})
		return
	}
	if len(req.Password) < 8 {
		jsonResp(w, http.StatusBadRequest, CreateAdminResponse{Message: "password must be at least 8 characters"})
		return
	}

	// 临时连接数据库
	store, err := openTempStore(req.Database)
	if err != nil {
		jsonResp(w, http.StatusOK, CreateAdminResponse{
			Message: fmt.Sprintf("Database connection failed: %v", err),
		})
		return
	}
	defer store.Close()

	// 创建管理员用户
	if err := auth.EnsureAdminUser(store, req.Email, req.Password); err != nil {
		jsonResp(w, http.StatusOK, CreateAdminResponse{
			Message: fmt.Sprintf("Failed to create admin user: %v", err),
		})
		return
	}

	log.Printf("Admin user created/ensured: %s", req.Email)
	jsonResp(w, http.StatusOK, CreateAdminResponse{
		Success: true,
		Message: fmt.Sprintf("Admin user created: %s", req.Email),
	})
}

// openTempStore 根据数据库配置临时打开一个存储连接
func openTempStore(cfg DatabaseConfig) (storage.PersistentStore, error) {
	switch cfg.Type {
	case "mongodb":
		if cfg.Port == 0 {
			cfg.Port = 27017
		}
		uri := fmt.Sprintf("mongodb://%s:%d", cfg.Host, cfg.Port)
		if cfg.User != "" && cfg.Password != "" {
			uri = fmt.Sprintf("mongodb://%s:%s@%s:%d", cfg.User, cfg.Password, cfg.Host, cfg.Port)
		}
		dbName := cfg.DBName
		if dbName == "" {
			dbName = "agents_admin"
		}
		return mongostore.NewStore(uri, dbName)

	case "sqlite":
		dsn := cfg.Path
		if dsn == "" {
			return nil, fmt.Errorf("sqlite path is required")
		}
		return storage.NewPersistentStoreFromDSN(dbutil.DriverSQLite, dsn)

	default: // postgres
		if cfg.Port == 0 {
			cfg.Port = 5432
		}
		if cfg.SSLMode == "" {
			cfg.SSLMode = "disable"
		}
		dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s&connect_timeout=10",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode)
		return storage.NewPersistentStoreFromDSN(dbutil.DriverPostgres, dsn)
	}
}

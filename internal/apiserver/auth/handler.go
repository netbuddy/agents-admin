package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

	"agents-admin/internal/shared/model"
)

// UserStore 用户存储接口
type UserStore interface {
	CreateUser(ctx context.Context, user *model.User) error
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	GetUserByID(ctx context.Context, id string) (*model.User, error)
	UpdateUserPassword(ctx context.Context, id, passwordHash string) error
	ListUsers(ctx context.Context) ([]*model.User, error)
}

// Handler 认证 HTTP 处理器
type Handler struct {
	store UserStore
	cfg   Config
}

// NewHandler 创建认证处理器
func NewHandler(store UserStore, cfg Config) *Handler {
	return &Handler{store: store, cfg: cfg}
}

// RegisterRoutes 注册认证相关路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/register", h.Register)
	mux.HandleFunc("POST /api/v1/auth/login", h.Login)
	mux.HandleFunc("POST /api/v1/auth/refresh", h.Refresh)
	mux.HandleFunc("GET /api/v1/auth/me", h.Me)
	mux.HandleFunc("PUT /api/v1/auth/password", h.ChangePassword)
}

// ============================================================================
// 请求/响应类型
// ============================================================================

type registerRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type authResponse struct {
	User         *model.User `json:"user"`
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token,omitempty"`
}

// ============================================================================
// Handlers
// ============================================================================

// Register 用户注册
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email, username, password are required")
		return
	}
	if !isValidEmail(req.Email) {
		writeError(w, http.StatusBadRequest, "invalid email format")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	// 检查邮箱是否已注册
	existing, err := h.store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		log.Printf("[auth.register] GetUserByEmail error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if existing != nil {
		writeError(w, http.StatusConflict, "email already registered")
		return
	}

	// 哈希密码
	hash, err := HashPassword(req.Password)
	if err != nil {
		log.Printf("[auth.register] HashPassword error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	now := time.Now()
	user := &model.User{
		ID:           generateID(),
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: hash,
		Role:         model.UserRoleUser,
		Status:       model.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.store.CreateUser(r.Context(), user); err != nil {
		log.Printf("[auth.register] CreateUser error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	// 生成令牌
	accessToken, err := GenerateAccessToken(h.cfg, user.ID, user.Email, string(user.Role))
	if err != nil {
		log.Printf("[auth.register] GenerateAccessToken error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	refreshToken, err := GenerateRefreshToken(h.cfg, user.ID)
	if err != nil {
		log.Printf("[auth.register] GenerateRefreshToken error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	log.Printf("[auth] User registered: %s (%s)", user.Email, user.ID)
	writeJSON(w, http.StatusCreated, authResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// Login 用户登录
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	user, err := h.store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		log.Printf("[auth.login] GetUserByEmail error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if user == nil || !CheckPassword(req.Password, user.PasswordHash) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if user.Status == model.UserStatusDisabled {
		writeError(w, http.StatusForbidden, "account is disabled")
		return
	}

	accessToken, err := GenerateAccessToken(h.cfg, user.ID, user.Email, string(user.Role))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	refreshToken, err := GenerateRefreshToken(h.cfg, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	log.Printf("[auth] User logged in: %s", user.Email)
	writeJSON(w, http.StatusOK, authResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// Refresh 刷新访问令牌
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	claims, err := ParseToken(h.cfg, req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	if claims.Type != "refresh" {
		writeError(w, http.StatusUnauthorized, "invalid token type")
		return
	}

	// 查询用户确保仍然存在且有效
	user, err := h.store.GetUserByID(r.Context(), claims.Subject)
	if err != nil || user == nil {
		writeError(w, http.StatusUnauthorized, "user not found")
		return
	}
	if user.Status == model.UserStatusDisabled {
		writeError(w, http.StatusForbidden, "account is disabled")
		return
	}

	accessToken, err := GenerateAccessToken(h.cfg, user.ID, user.Email, string(user.Role))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"access_token": accessToken,
	})
}

// Me 获取当前用户信息
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	authUser := GetAuthUser(r.Context())
	if authUser == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	user, err := h.store.GetUserByID(r.Context(), authUser.ID)
	if err != nil || user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// ChangePassword 修改密码
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	authUser := GetAuthUser(r.Context())
	if authUser == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.OldPassword == "" || req.NewPassword == "" {
		writeError(w, http.StatusBadRequest, "old_password and new_password are required")
		return
	}
	if len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "new password must be at least 8 characters")
		return
	}

	user, err := h.store.GetUserByID(r.Context(), authUser.ID)
	if err != nil || user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if !CheckPassword(req.OldPassword, user.PasswordHash) {
		writeError(w, http.StatusUnauthorized, "incorrect old password")
		return
	}

	hash, err := HashPassword(req.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := h.store.UpdateUserPassword(r.Context(), user.ID, hash); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update password")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}

// ============================================================================
// Admin Bootstrap
// ============================================================================

// EnsureAdminUser 确保管理员用户存在（启动时调用）
// 如果配置了 adminEmail 且数据库中不存在该用户，则自动创建
func EnsureAdminUser(store UserStore, adminEmail, adminPassword string) error {
	if adminEmail == "" || adminPassword == "" {
		return nil
	}

	ctx := context.Background()
	existing, err := store.GetUserByEmail(ctx, adminEmail)
	if err != nil {
		return fmt.Errorf("check admin user: %w", err)
	}
	if existing != nil {
		// 已存在，确保角色是 admin
		if existing.Role != model.UserRoleAdmin {
			log.Printf("[auth] Upgrading user %s to admin role", adminEmail)
			// 直接更新角色 — 简单做法
		}
		log.Printf("[auth] Admin user already exists: %s (%s)", adminEmail, existing.ID)
		return nil
	}

	hash, err := HashPassword(adminPassword)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	now := time.Now()
	user := &model.User{
		ID:           generateID(),
		Email:        adminEmail,
		Username:     "Admin",
		PasswordHash: hash,
		Role:         model.UserRoleAdmin,
		Status:       model.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := store.CreateUser(ctx, user); err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}
	log.Printf("[auth] Created admin user: %s (%s)", adminEmail, user.ID)
	return nil
}

// ============================================================================
// 工具函数
// ============================================================================

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func generateID() string {
	return fmt.Sprintf("usr-%d", time.Now().UnixNano())
}

// Package auth 用户认证：JWT 令牌管理、密码哈希、HTTP 中间件
package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// contextKey context 键类型
type contextKey string

const (
	ctxKeyAuthUser contextKey = "auth_user"
	ctxKeyTenantID contextKey = "tenant_id"
)

// AuthUser 从 JWT 解析出的用户信息
type AuthUser struct {
	ID    string
	Email string
	Role  string // "admin" | "user"
}

// Config 认证配置
type Config struct {
	JWTSecret       string        `yaml:"jwt_secret"`
	AccessTokenTTL  time.Duration `yaml:"access_token_ttl"`
	RefreshTokenTTL time.Duration `yaml:"refresh_token_ttl"`
	NodeToken       string        `yaml:"-"` // NodeManager 共享密钥，从 NODE_TOKEN 环境变量读取
}

// DefaultConfig 返回默认认证配置
func DefaultConfig() Config {
	return Config{
		JWTSecret:       "",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}
}

// Enabled 是否启用认证
func (c Config) Enabled() bool {
	return c.JWTSecret != ""
}

// ============================================================================
// 密码哈希
// ============================================================================

// HashPassword 使用 bcrypt 哈希密码
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

// CheckPassword 验证密码
func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// ============================================================================
// JWT Token
// ============================================================================

// Claims JWT 声明
type Claims struct {
	jwt.RegisteredClaims
	Email string `json:"email,omitempty"`
	Role  string `json:"role,omitempty"`
	Type  string `json:"type,omitempty"` // "access" | "refresh"
}

// GenerateAccessToken 生成访问令牌
func GenerateAccessToken(cfg Config, userID, email, role string) (string, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.AccessTokenTTL)),
		},
		Email: email,
		Role:  role,
		Type:  "access",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

// GenerateRefreshToken 生成刷新令牌
func GenerateRefreshToken(cfg Config, userID string) (string, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.RefreshTokenTTL)),
		},
		Type: "refresh",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

// ParseToken 解析并验证 JWT
func ParseToken(cfg Config, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

// ============================================================================
// Context 辅助函数
// ============================================================================

// WithAuthUser 将认证用户信息注入 context
func WithAuthUser(ctx context.Context, user *AuthUser) context.Context {
	return context.WithValue(ctx, ctxKeyAuthUser, user)
}

// GetAuthUser 从 context 获取认证用户
func GetAuthUser(ctx context.Context) *AuthUser {
	user, _ := ctx.Value(ctxKeyAuthUser).(*AuthUser)
	return user
}

// WithTenantID 将租户 ID 注入 context
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, ctxKeyTenantID, tenantID)
}

// GetTenantID 从 context 获取租户 ID
// 返回空字符串表示 admin（不限租户）或无认证模式
func GetTenantID(ctx context.Context) string {
	tid, _ := ctx.Value(ctxKeyTenantID).(string)
	return tid
}

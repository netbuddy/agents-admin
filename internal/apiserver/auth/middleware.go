package auth

import (
	"log"
	"net/http"
	"strings"
)

// 公开路由前缀（无需任何认证）
var publicPrefixes = []string{
	"/api/v1/auth/register",
	"/api/v1/auth/login",
	"/api/v1/auth/refresh",
	"/api/v1/node-bootstrap",
	"/health",
	"/metrics",
	"/ws/",
}

// isPublicRoute 判断是否为完全公开的路由（无需任何认证）
func isPublicRoute(method, path string) bool {
	for _, prefix := range publicPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	// 心跳是 NodeManager 注册的第一个请求，必须公开
	if method == "POST" && path == "/api/v1/nodes/heartbeat" {
		return true
	}
	return false
}

// isValidNodeToken 检查请求中的 X-Node-Token 是否有效
func isValidNodeToken(r *http.Request, nodeToken string) bool {
	if nodeToken == "" {
		return false
	}
	token := r.Header.Get("X-Node-Token")
	return token != "" && token == nodeToken
}

// Middleware 创建认证中间件
//
// 认证策略（优先级从高到低）：
//  1. 公开路由（login/register/health/heartbeat）：直接放行
//  2. X-Node-Token header：NodeManager 共享密钥认证，匹配则放行
//  3. JWT（Bearer token 或 Cookie）：用户认证
//
// 如果 cfg.Enabled() == false，直接放行所有请求（无认证模式）
func Middleware(cfg Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 无认证模式：直接放行
			if !cfg.Enabled() {
				next.ServeHTTP(w, r)
				return
			}

			// 公开路由：直接放行
			if isPublicRoute(r.Method, r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// 静态资源放行
			if strings.HasPrefix(r.URL.Path, "/_next/") || strings.HasPrefix(r.URL.Path, "/favicon") {
				next.ServeHTTP(w, r)
				return
			}

			// NodeManager Token 认证：X-Node-Token header 匹配则放行
			if isValidNodeToken(r, cfg.NodeToken) {
				next.ServeHTTP(w, r)
				return
			}

			// 提取 Bearer Token（优先 Authorization header，回退 Cookie）
			tokenString := ""
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
					tokenString = parts[1]
				}
			}
			if tokenString == "" {
				if c, err := r.Cookie("access_token"); err == nil && c.Value != "" {
					tokenString = c.Value
				}
			}
			if tokenString == "" {
				http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			// 解析 JWT
			claims, err := ParseToken(cfg, tokenString)
			if err != nil {
				log.Printf("[auth] token parse error: %v", err)
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			if claims.Type != "access" {
				http.Error(w, `{"error":"invalid token type"}`, http.StatusUnauthorized)
				return
			}

			// 注入 auth user 到 context
			user := &AuthUser{
				ID:    claims.Subject,
				Email: claims.Email,
				Role:  claims.Role,
			}
			ctx := WithAuthUser(r.Context(), user)

			// 注入 tenant_id
			if user.Role == "admin" {
				ctx = WithTenantID(ctx, "") // admin 不限租户
			} else {
				ctx = WithTenantID(ctx, user.ID)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AdminOnly 管理员专属路由中间件
func AdminOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := GetAuthUser(r.Context())
		if user == nil || user.Role != string(UserRoleAdmin) {
			http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

// UserRoleAdmin 管理员角色常量（避免 model 包循环引用）
const UserRoleAdmin = "admin"

package auth

import (
	"log"
	"net/http"
	"strings"
)

// 免认证路由白名单（前缀匹配）
var publicPrefixes = []string{
	"/api/v1/auth/register",
	"/api/v1/auth/login",
	"/api/v1/auth/refresh",
	"/health",
	"/metrics",
	"/ws/",
}

// 免认证路由精确匹配
var publicExact = map[string]bool{
	"POST /api/v1/nodes/heartbeat": true,
}

// 节点通信路由前缀（node-manager 调用，不走 JWT）
var nodePrefixes = []string{
	"/api/v1/nodes/heartbeat",
	"/api/v1/runs/",
	"/api/v1/actions/",
}

func isPublicRoute(method, path string) bool {
	for _, prefix := range publicPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	if publicExact[method+" "+path] {
		return true
	}
	// 节点通信路由：POST events, PATCH actions, GET nodes/{id}/runs 等
	if strings.HasPrefix(path, "/api/v1/nodes/") && (strings.HasSuffix(path, "/runs") ||
		strings.HasSuffix(path, "/actions") ||
		strings.HasSuffix(path, "/instances") ||
		strings.Contains(path, "/env-config")) {
		return true
	}
	if method == "POST" && strings.HasSuffix(path, "/events") {
		return true
	}
	if method == "PATCH" && strings.HasPrefix(path, "/api/v1/actions/") {
		return true
	}
	return false
}

// Middleware 创建 JWT 认证中间件
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

			// 提取 Bearer Token
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				http.Error(w, `{"error":"invalid authorization header"}`, http.StatusUnauthorized)
				return
			}

			// 解析 JWT
			claims, err := ParseToken(cfg, parts[1])
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

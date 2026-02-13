package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// newDevHandler 开发模式 handler：API 路由由 Go 处理，其余反向代理到 Next.js dev server
//
// 架构：
//
//	Browser → https://IP:8080 (Go, HTTPS)
//	          ├── /api/*   → Go handlers
//	          ├── /ws/*    → Go WebSocket
//	          ├── /health  → Go
//	          ├── /metrics → Go
//	          └── /*       → reverse proxy → http://localhost:3002 (Next.js dev)
func newDevHandler(apiHandler http.Handler, nextjsAddr string) http.Handler {
	target, err := url.Parse(nextjsAddr)
	if err != nil {
		log.Fatalf("Invalid Next.js dev server address: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// 自定义 Director：保留原始 Host header 以支持 Next.js HMR
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
	}

	// WebSocket 支持（Next.js HMR 使用 /_next/webpack-hmr）
	proxy.ModifyResponse = func(resp *http.Response) error {
		return nil
	}

	log.Printf("[dev] Reverse proxy: non-API routes → %s", nextjsAddr)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// API、WebSocket、监控端点 → Go 处理
		if strings.HasPrefix(path, "/api/") ||
			strings.HasPrefix(path, "/ws/") ||
			strings.HasPrefix(path, "/ttyd/") ||
			strings.HasPrefix(path, "/terminal/") ||
			strings.HasPrefix(path, "/spec") ||
			strings.HasPrefix(path, "/docs") ||
			path == "/health" ||
			path == "/metrics" {
			apiHandler.ServeHTTP(w, r)
			return
		}

		// 其余所有请求（页面、静态资源、HMR WebSocket）→ Next.js
		proxy.ServeHTTP(w, r)
	})
}

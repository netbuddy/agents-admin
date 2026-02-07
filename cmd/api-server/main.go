// Package main API Server 入口
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agents-admin/internal/apiserver/auth"
	"agents-admin/internal/apiserver/server"
	"agents-admin/internal/config"
	"agents-admin/internal/shared/infra"
	"agents-admin/internal/shared/storage"
	"agents-admin/internal/tlsutil"
	"agents-admin/web"
)

func main() {
	configDir := flag.String("config", "", "配置文件目录（默认搜索 configs/）")
	flag.Parse()

	if *configDir != "" {
		config.SetConfigDir(*configDir)
	}

	// 加载配置（自动加载 .env，根据 APP_ENV 切换数据库和 Redis）
	cfg := config.Load()

	log.Printf("Starting API Server... [env=%s]", cfg.Env)
	log.Printf("Config: %s", cfg.String())

	// 初始化 PostgreSQL（持久化业务数据）
	store, err := storage.NewPostgresStore(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer store.Close()
	log.Println("Connected to PostgreSQL")

	// 初始化 Redis（缓存、事件总线、消息队列）
	redisInfra, err := infra.NewRedisInfra(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisInfra.Close()
	log.Println("Connected to Redis")

	// 初始化 Handler（心跳缓存由 Redis 提供，etcd 已弃用）
	h := server.NewHandler(store, redisInfra)

	// 设置认证配置
	authCfg := server.AuthConfigCompat{
		JWTSecret:     cfg.Auth.JWTSecret,
		AdminEmail:    cfg.Auth.AdminEmail,
		AdminPassword: cfg.Auth.AdminPassword,
	}
	if d, err := time.ParseDuration(cfg.Auth.AccessTokenTTL); err == nil && d > 0 {
		authCfg.AccessTokenTTL = d
	} else {
		authCfg.AccessTokenTTL = 15 * time.Minute
	}
	if d, err := time.ParseDuration(cfg.Auth.RefreshTokenTTL); err == nil && d > 0 {
		authCfg.RefreshTokenTTL = d
	} else {
		authCfg.RefreshTokenTTL = 7 * 24 * time.Hour
	}
	h.SetAuthConfig(authCfg)

	// 初始化管理员用户
	if err := auth.EnsureAdminUser(store, cfg.Auth.AdminEmail, cfg.Auth.AdminPassword); err != nil {
		log.Printf("WARNING: Failed to ensure admin user: %v", err)
	}

	// 启动调度器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.StartScheduler(ctx)

	// 确定最终 handler：生产模式嵌入前端静态文件，开发模式仅 API
	var handler http.Handler = h.Router()
	if web.IsEmbedded() {
		staticFS, err := web.StaticFS()
		if err != nil {
			log.Fatalf("Failed to get embedded static FS: %v", err)
		}
		handler = newSPAHandler(handler, staticFS)
		log.Println("Frontend embedded: serving static files from binary")
	} else {
		log.Println("Frontend not embedded: dev mode, use Next.js dev server separately")
	}

	srv := &http.Server{
		Addr:         ":" + cfg.APIPort,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 优雅关闭
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	if cfg.TLS.Enabled {
		// TLS 自动生成：如果启用了 auto_generate 且证书不存在，自动生成自签名证书
		if cfg.TLS.AutoGenerate {
			opts := tlsutil.DefaultGenerateOptions()
			if cfg.TLS.Hosts != "" {
				opts.Hosts = cfg.TLS.Hosts
			}
			certs, err := tlsutil.EnsureCerts(opts)
			if err != nil {
				log.Fatalf("Failed to auto-generate TLS certs: %v", err)
			}
			// 用自动生成的路径填充空配置
			if cfg.TLS.CertFile == "" {
				cfg.TLS.CertFile = certs.CertFile
			}
			if cfg.TLS.KeyFile == "" {
				cfg.TLS.KeyFile = certs.KeyFile
			}
			if cfg.TLS.CAFile == "" {
				cfg.TLS.CAFile = certs.CAFile
			}
		}
		log.Printf("API Server listening on :%s (TLS)", cfg.APIPort)
		if err := srv.ListenAndServeTLS(cfg.TLS.CertFile, cfg.TLS.KeyFile); err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	} else {
		log.Printf("API Server listening on :%s", cfg.APIPort)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}

	fmt.Println("Server stopped")
}

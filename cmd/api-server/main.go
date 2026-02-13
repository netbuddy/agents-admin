// Package main API Server 入口
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agents-admin/internal/apiserver/auth"
	"agents-admin/internal/apiserver/server"
	"agents-admin/internal/apiserver/setup"
	"agents-admin/internal/config"
	"agents-admin/internal/shared/infra"
	objstore "agents-admin/internal/shared/minio"
	"agents-admin/internal/shared/storage"
	"agents-admin/internal/shared/storage/dbutil"
	"agents-admin/internal/shared/storage/mongostore"
	"agents-admin/internal/tlsutil"
	"agents-admin/web"

	"golang.org/x/crypto/acme/autocert"
)

func main() {
	configDir := flag.String("config", "", "配置文件目录（默认搜索 configs/）")
	reconfigure := flag.Bool("reconfigure", false, "强制重新进入配置向导")
	setupPort := flag.Int("setup-port", 15800, "Setup 向导监听端口")
	setupListen := flag.String("setup-listen", "0.0.0.0", "Setup 向导监听地址")
	flag.Parse()

	if *configDir != "" {
		config.SetConfigDir(*configDir)
	}

	// 环境变量覆盖命令行参数
	if p := os.Getenv("SETUP_PORT"); p != "" {
		fmt.Sscanf(p, "%d", setupPort)
	}
	if l := os.Getenv("SETUP_LISTEN"); l != "" {
		*setupListen = l
	}

	// 首次运行检测或 --reconfigure：启动 Setup 向导
	if *reconfigure || !config.ConfigExists() {
		log.Printf("No configuration found. Starting Setup Wizard on port %d...", *setupPort)
		setupSrv := setup.NewServer(config.GetConfigDir(), *setupListen, *setupPort)
		setupSrv.Run()
		// 向导完成后进程退出，systemd 会自动重启并加载新配置
		return
	}

	// 加载配置（自动加载 .env，根据 APP_ENV 切换数据库和 Redis）
	cfg := config.Load()

	log.Printf("Starting API Server... [env=%s]", cfg.Env)
	log.Printf("Config: %s", cfg.String())

	// 初始化数据库（根据配置自动选择 MongoDB、PostgreSQL 或 SQLite）
	var store storage.PersistentStore
	var err error
	if dbutil.DriverType(cfg.DatabaseDriver) == dbutil.DriverMongoDB {
		store, err = mongostore.NewStore(cfg.DatabaseURL, cfg.DatabaseDBName)
		if err != nil {
			log.Fatalf("Failed to connect to MongoDB: %v", err)
		}
	} else {
		store, err = storage.NewPersistentStoreFromDSN(dbutil.DriverType(cfg.DatabaseDriver), cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("Failed to connect to database (%s): %v", cfg.DatabaseDriver, err)
		}
	}
	defer store.Close()
	log.Printf("Connected to database (%s)", cfg.DatabaseDriver)

	// 初始化 Redis（缓存、事件总线、消息队列）
	redisInfra, err := infra.NewRedisInfra(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisInfra.Close()
	log.Println("Connected to Redis")

	// 初始化 Handler（心跳缓存由 Redis 提供，etcd 已弃用）
	h := server.NewHandler(store, redisInfra)

	// 初始化 MinIO 客户端（可选，用于 volume archive）
	if cfg.MinIO.Endpoint != "" && cfg.MinIO.AccessKey != "" {
		mc, err := objstore.NewClient(cfg.MinIO)
		if err != nil {
			log.Printf("WARNING: Failed to create MinIO client: %v (volume archive disabled)", err)
		} else {
			if err := mc.EnsureBucket(context.Background()); err != nil {
				log.Printf("WARNING: Failed to ensure MinIO bucket: %v (volume archive disabled)", err)
			} else {
				h.SetMinIOClient(mc)
				log.Println("Connected to MinIO object storage")
			}
		}
	} else {
		log.Println("MinIO not configured, volume archive disabled")
	}

	// 设置认证配置
	authCfg := server.AuthConfigCompat{
		JWTSecret:     cfg.Auth.JWTSecret,
		AdminEmail:    cfg.Auth.AdminEmail,
		AdminPassword: cfg.Auth.AdminPassword,
		NodeToken:     cfg.Auth.NodeToken,
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

	// 设置 Node Manager 引导配置（零配置安装）
	h.SetBootstrapConfig(server.BootstrapConfig{
		TLSEnabled: cfg.TLS.Enabled,
	})

	// 初始化管理员用户
	if err := auth.EnsureAdminUser(store, cfg.Auth.AdminEmail, cfg.Auth.AdminPassword); err != nil {
		log.Printf("WARNING: Failed to ensure admin user: %v", err)
	}

	// 启动调度器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.StartScheduler(ctx)

	// 确定最终 handler：生产模式嵌入前端，开发模式反向代理到 Next.js
	var handler http.Handler = h.Router()
	if web.IsEmbedded() {
		staticFS, err := web.StaticFS()
		if err != nil {
			log.Fatalf("Failed to get embedded static FS: %v", err)
		}
		handler = newSPAHandler(handler, staticFS)
		log.Println("Frontend embedded: serving static files from binary")
	} else {
		// 开发模式：反向代理非 API 请求到 Next.js dev server
		nextjsAddr := os.Getenv("NEXTJS_DEV_URL")
		if nextjsAddr == "" {
			nextjsAddr = "http://localhost:3002"
		}
		handler = newDevHandler(handler, nextjsAddr)
		log.Printf("Dev mode: proxying frontend to %s", nextjsAddr)
	}

	srv := &http.Server{
		Addr:         ":" + cfg.APIPort,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
		ErrorLog:     newTLSFilteredLogger(),
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

	// ============================================================
	// 启动服务器（三种 TLS 模式）
	// ============================================================
	if cfg.TLS.Enabled && cfg.TLS.ACME.Enabled {
		// 模式 A：ACME / Let's Encrypt（互联网域名自动证书）
		startWithACME(srv, cfg)
	} else if cfg.TLS.Enabled {
		// 模式 B：自签名证书（内网 IP 访问）
		startWithSelfSignedTLS(srv, cfg)
	} else {
		// 模式 C：纯 HTTP（不推荐）
		log.Printf("API Server listening on :%s (HTTP, insecure)", cfg.APIPort)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}

	fmt.Println("Server stopped")
}

// startWithSelfSignedTLS 自签名证书模式（本地开发 / 内网）
func startWithSelfSignedTLS(srv *http.Server, cfg *config.Config) {
	if cfg.TLS.AutoGenerate {
		opts := tlsutil.DefaultGenerateOptions()
		if cfg.TLS.CertDir != "" {
			opts.CertDir = cfg.TLS.CertDir
		}
		if cfg.TLS.Hosts != "" {
			opts.Hosts = cfg.TLS.Hosts
		}
		certs, err := tlsutil.EnsureCerts(opts)
		if err != nil {
			log.Fatalf("Failed to auto-generate TLS certs: %v", err)
		}
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

	// 注入 /ca.pem 端点，供客户端下载并信任 CA 证书
	srv.Handler = withCACertEndpoint(srv.Handler, cfg.TLS.CAFile)

	log.Printf("API Server listening on :%s (TLS, self-signed)", cfg.APIPort)
	log.Printf("  cert: %s", cfg.TLS.CertFile)
	log.Printf("  key:  %s", cfg.TLS.KeyFile)

	// 使用自定义 listener：同端口自动检测 HTTP 并重定向到 HTTPS
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", srv.Addr, err)
	}
	redirectLn := &httpOnTLSListener{Listener: ln}
	if err := srv.ServeTLS(redirectLn, cfg.TLS.CertFile, cfg.TLS.KeyFile); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

// startWithACME Let's Encrypt 自动证书模式（互联网域名）
func startWithACME(srv *http.Server, cfg *config.Config) {
	acmeCfg := cfg.TLS.ACME
	cacheDir := acmeCfg.CacheDir
	if cacheDir == "" {
		cacheDir = "/etc/agents-admin/certs/acme"
	}

	m := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(acmeCfg.Domains...),
		Cache:      autocert.DirCache(cacheDir),
		Email:      acmeCfg.Email,
	}

	srv.TLSConfig = &tls.Config{
		GetCertificate: m.GetCertificate,
		NextProtos:     []string{"h2", "http/1.1", "acme-tls/1"},
	}
	srv.Addr = ":443"

	// HTTP :80 → HTTPS 重定向 + ACME HTTP-01 challenge
	go func() {
		httpSrv := &http.Server{
			Addr:    ":80",
			Handler: m.HTTPHandler(redirectToHTTPS()),
		}
		log.Println("HTTP :80 → HTTPS redirect + ACME challenge")
		if err := httpSrv.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("HTTP redirect server error: %v", err)
		}
	}()

	log.Printf("API Server listening on :443 (TLS, ACME/Let's Encrypt)")
	log.Printf("  domains: %v", acmeCfg.Domains)
	if err := srv.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

// redirectToHTTPS 返回 HTTP→HTTPS 301 重定向 handler
func redirectToHTTPS() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := "https://" + r.Host + r.URL.RequestURI()
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
}

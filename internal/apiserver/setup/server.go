// Package setup API Server 首次运行配置向导
//
// 当检测不到配置文件时，启动一个轻量 HTTP 服务器，
// 通过 Web UI 引导用户完成 PostgreSQL、Redis、TLS、Auth 配置。
// 完成后生成 {env}.yaml + {env}.env 文件并退出。
package setup

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

//go:embed static
var staticFiles embed.FS

// Server 配置向导服务器
type Server struct {
	configDir  string // 配置文件目录（如 /etc/agents-admin）
	listenAddr string
	port       int
	token      string
	infra      infraState // 基础设施部署状态
}

// NewServer 创建配置向导服务器
func NewServer(configDir, listenAddr string, port int) *Server {
	return &Server{
		configDir:  configDir,
		listenAddr: listenAddr,
		port:       port,
		token:      generateToken(),
	}
}

// Run 启动配置向导（阻塞直到配置完成或超时）
func (s *Server) Run() {
	mux := http.NewServeMux()

	// API 路由
	mux.HandleFunc("/setup/api/info", s.handleInfo)
	mux.HandleFunc("/setup/api/validate", s.handleValidate)
	mux.HandleFunc("/setup/api/apply", s.handleApply)
	mux.HandleFunc("/setup/api/init-db", s.handleInitDB)
	mux.HandleFunc("/setup/api/create-admin", s.handleCreateAdmin)
	mux.HandleFunc("/setup/api/generate-infra", s.handleGenerateInfra)
	mux.HandleFunc("/setup/api/deploy-infra", s.handleDeployInfra)
	mux.HandleFunc("/setup/api/infra-status", s.handleInfraStatus)

	// 前端静态文件
	mux.HandleFunc("/setup", s.handleIndex)
	mux.HandleFunc("/setup/", s.handleStatic)

	// 根路径重定向
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/setup?token="+s.token, http.StatusFound)
	})

	// Token 认证中间件
	handler := s.tokenAuthMiddleware(mux)

	addr := fmt.Sprintf("%s:%d", s.listenAddr, s.port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// 打印访问信息
	s.printBanner()

	// 优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Setup wizard interrupted")
		cancel()
		srv.Shutdown(context.Background())
	}()

	// 30 分钟超时
	go func() {
		select {
		case <-time.After(30 * time.Minute):
			log.Println("Setup wizard timed out (30 minutes), exiting...")
			srv.Shutdown(context.Background())
		case <-ctx.Done():
		}
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Setup server error: %v", err)
	}
}

func (s *Server) printBanner() {
	ips := getLocalIPs()

	log.Println("====================================================")
	log.Println("  API Server Setup Wizard")
	log.Println("====================================================")
	log.Printf("  Config will be saved to: %s/", s.configDir)
	log.Println()
	log.Println("  Access URLs:")
	for _, ip := range ips {
		log.Printf("    http://%s:%d/setup?token=%s", ip, s.port, s.token)
	}
	log.Printf("    http://localhost:%d/setup?token=%s", s.port, s.token)
	log.Println()
	log.Printf("  Token: %s", s.token)
	log.Printf("  Timeout: 30 minutes")
	log.Println("====================================================")
}

// handleIndex 返回 index.html
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

// handleStatic 处理静态文件
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	sub, _ := fs.Sub(staticFiles, "static")
	path := strings.TrimPrefix(r.URL.Path, "/setup/")
	if path == "" || path == "/" {
		s.handleIndex(w, r)
		return
	}
	http.StripPrefix("/setup/", http.FileServer(http.FS(sub))).ServeHTTP(w, r)
}

// tokenAuthMiddleware Token 认证中间件
func (s *Server) tokenAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 静态文件放行（已通过 URL token 进入页面）
		if strings.HasPrefix(r.URL.Path, "/setup/") && !strings.HasPrefix(r.URL.Path, "/setup/api/") {
			next.ServeHTTP(w, r)
			return
		}

		// 提取 token
		token := r.URL.Query().Get("token")
		if token == "" {
			if c, err := r.Cookie("setup_token"); err == nil {
				token = c.Value
			}
		}
		if token == "" {
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(auth, "token ") {
				token = strings.TrimPrefix(auth, "token ")
			}
		}

		if token != s.token {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}

		// 设置 cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "setup_token",
			Value:    s.token,
			Path:     "/",
			HttpOnly: true,
			MaxAge:   3600,
		})

		next.ServeHTTP(w, r)
	})
}

// getLocalIPs 获取本机非回环 IP
func getLocalIPs() []string {
	var ips []string
	addrs, _ := net.InterfaceAddrs()
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
			ips = append(ips, ipNet.IP.String())
		}
	}
	return ips
}

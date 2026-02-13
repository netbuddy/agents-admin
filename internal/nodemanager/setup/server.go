// Package setup 提供 Node Manager 首次运行配置向导
package setup

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// Server 配置向导 HTTP 服务器
type Server struct {
	configPath string // 配置文件写入路径
	listenAddr string // 监听地址
	port       int    // 监听端口
	token      string // Setup Token
}

// NewServer 创建配置向导服务器
func NewServer(configPath, listenAddr string, port int) *Server {
	token := generateToken()
	return &Server{
		configPath: configPath,
		listenAddr: listenAddr,
		port:       port,
		token:      token,
	}
}

// Run 启动配置向导服务器（阻塞直到配置完成或收到信号）
func (s *Server) Run() {
	mux := http.NewServeMux()

	// API 路由
	mux.HandleFunc("/setup/api/info", s.handleInfo)
	mux.HandleFunc("/setup/api/validate", s.handleValidate)
	mux.HandleFunc("/setup/api/apply", s.handleApply)
	mux.HandleFunc("/setup/api/download-ca", s.handleDownloadCA)
	mux.HandleFunc("/setup/api/bootstrap", s.handleBootstrap)

	// 前端静态文件
	mux.HandleFunc("/setup", s.handleIndex)
	mux.HandleFunc("/setup/", s.handleStatic)

	// 根路径重定向到 /setup
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

	// 打印启动信息
	s.printBanner()

	// 30 分钟无操作超时
	setupTimeout := 30 * time.Minute
	timeoutTimer := time.NewTimer(setupTimeout)

	// 优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigChan:
			log.Println("Setup wizard interrupted")
			cancel()
			srv.Shutdown(context.Background())
		case <-timeoutTimer.C:
			log.Println("Setup wizard timed out (30 minutes), exiting...")
			cancel()
			srv.Shutdown(context.Background())
		case <-ctx.Done():
		}
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Setup server error: %v", err)
	}
}

// printBanner 打印启动 Banner，包含 Token 和访问 URL
func (s *Server) printBanner() {
	ips := getLocalIPs()
	log.Println("====================================================")
	log.Println("  Node Manager Setup Wizard")
	log.Println("====================================================")
	log.Println("  Config will be saved to:", s.configPath)
	log.Println("")
	log.Println("  Access URLs:")
	for _, ip := range ips {
		log.Printf("    http://%s:%d/setup?token=%s", ip, s.port, s.token)
	}
	log.Printf("    http://localhost:%d/setup?token=%s", s.port, s.token)
	log.Println("")
	log.Println("  Token:", s.token)
	log.Println("  Timeout: 30 minutes")
	log.Println("====================================================")
}

// tokenAuthMiddleware Token 认证中间件
func (s *Server) tokenAuthMiddleware(next http.Handler) http.Handler {
	const cookieName = "setup_token"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查 URL query 参数
		if r.URL.Query().Get("token") == s.token {
			// Token 正确，设置 Cookie
			http.SetCookie(w, &http.Cookie{
				Name:     cookieName,
				Value:    s.token,
				Path:     "/",
				MaxAge:   3600, // 1 小时
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
			next.ServeHTTP(w, r)
			return
		}

		// 检查 Cookie
		cookie, err := r.Cookie(cookieName)
		if err == nil && cookie.Value == s.token {
			next.ServeHTTP(w, r)
			return
		}

		// 检查 Authorization header
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "token ") && strings.TrimPrefix(authHeader, "token ") == s.token {
			next.ServeHTTP(w, r)
			return
		}

		// 未认证
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid or missing setup token. Check the terminal/journal for the access URL.",
		})
	})
}

// generateToken 生成 32 字节随机 Token（hex 编码后 64 字符）
func generateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("Failed to generate setup token: %v", err)
	}
	return hex.EncodeToString(b)
}

// getLocalIPs 获取本机物理网卡的 IPv4 地址（过滤 docker/veth/bridge 等虚拟网卡）
func getLocalIPs() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var ips []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if isVirtualInterface(iface.Name) {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	return ips
}

// isVirtualInterface 判断是否为虚拟网卡
func isVirtualInterface(name string) bool {
	// Linux: 物理网卡在 sysfs 中有 /device 符号链接
	if _, err := os.Stat("/sys/class/net/" + name + "/device"); err == nil {
		return false
	}
	// 回退：已知物理网卡命名前缀（systemd predictable naming + legacy）
	for _, prefix := range []string{"en", "eth", "wl", "ww"} {
		if strings.HasPrefix(name, prefix) {
			return false
		}
	}
	return true
}

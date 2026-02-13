// Package main 节点管理器入口
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"agents-admin/internal/config"
	"agents-admin/internal/nodemanager"
	"agents-admin/internal/nodemanager/adapter/claude"
	"agents-admin/internal/nodemanager/adapter/gemini"
	"agents-admin/internal/nodemanager/adapter/qwencode"
	"agents-admin/internal/nodemanager/setup"
)

func main() {
	configDirFlag := flag.String("config", "", "配置文件目录（或 YAML 文件路径）")
	reconfigure := flag.Bool("reconfigure", false, "强制重新进入配置向导")
	setupPort := flag.Int("setup-port", 15700, "Setup 向导监听端口")
	setupListen := flag.String("setup-listen", "0.0.0.0", "Setup 向导监听地址")
	flag.Parse()

	// 环境变量覆盖命令行参数
	if p := os.Getenv("SETUP_PORT"); p != "" {
		fmt.Sscanf(p, "%d", setupPort)
	}
	if l := os.Getenv("SETUP_LISTEN"); l != "" {
		*setupListen = l
	}

	// 设置配置目录（复用 config 包的统一路径策略）
	if *configDirFlag != "" {
		dir := *configDirFlag
		// 支持直接指定 YAML 文件路径
		if strings.HasSuffix(dir, ".yaml") || strings.HasSuffix(dir, ".yml") {
			dir = filepath.Dir(dir)
		}
		config.SetConfigDir(dir)
	}

	// 配置文件路径（Setup Wizard 写入位置）
	configPath := filepath.Join(config.GetConfigDir(), config.ConfigFileName())

	// 判断运行模式：Setup Mode vs Worker Mode
	// 简单规则：--reconfigure 强制安装，或 {env}.yaml 不存在时进入安装向导
	// 配置文件存在即视为已配置，缺失字段由 runWorkerMode() 中的回退逻辑处理
	needSetup := *reconfigure || !config.NodeManagerConfigExists()
	if needSetup {
		if *reconfigure {
			log.Println("Reconfigure mode: ignoring existing config file")
		}
		// Setup Mode: 启动配置向导
		runSetupMode(configPath, *setupListen, *setupPort)
		return
	}

	// Worker Mode: 正常运行
	runWorkerMode()
}

// runSetupMode 启动配置向导 Web 服务
func runSetupMode(configPath, listenAddr string, port int) {
	srv := setup.NewServer(configPath, listenAddr, port)
	srv.Run()
}

// runWorkerMode 正常工作模式
func runWorkerMode() {
	log.Println("Starting NodeManager...")

	// 通过统一的 config 包加载配置
	appCfg := config.LoadNodeManager()

	// 环境变量 > yaml 配置 > 默认值
	cfg := nodemanager.Config{
		NodeID:       firstNonEmpty(os.Getenv("NODE_ID"), appCfg.Node.ID, nodemanager.GenerateNodeID()),
		APIServerURL: firstNonEmpty(os.Getenv("API_SERVER_URL"), appCfg.APIServer.URL, "http://localhost:8080"),
		WorkspaceDir: firstNonEmpty(os.Getenv("WORKSPACE_DIR"), appCfg.Node.WorkspaceDir, "/tmp/workspaces"),
		Labels:       appCfg.Node.Labels,
		NodeToken:    firstNonEmpty(os.Getenv("NODE_TOKEN"), appCfg.Auth.NodeToken),
	}
	if len(cfg.Labels) == 0 {
		cfg.Labels = map[string]string{"os": "linux"}
	}

	// TLS 客户端配置：环境变量 > yaml 配置 > 自动检测 HTTPS URL
	tlsCAFile := firstNonEmpty(os.Getenv("TLS_CA_FILE"), appCfg.TLS.CAFile)
	tlsEnabled := appCfg.TLS.Enabled || strings.HasPrefix(cfg.APIServerURL, "https://")

	if tlsEnabled && tlsCAFile != "" {
		tlsClient, err := buildTLSClient(tlsCAFile)
		if err != nil {
			log.Fatalf("Failed to load TLS CA: %v", err)
		}
		cfg.HTTPClient = tlsClient
		log.Printf("TLS enabled, CA: %s", tlsCAFile)
	} else if tlsEnabled {
		// HTTPS URL 但无 CA 文件：跳过证书验证（开发便利，生产应提供 CA）
		log.Println("WARNING: TLS enabled but no CA file, skipping certificate verification")
		cfg.HTTPClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	}

	log.Printf("Node ID: %s", cfg.NodeID)
	log.Printf("API Server: %s", cfg.APIServerURL)
	log.Printf("Workspace Dir: %s", cfg.WorkspaceDir)

	if err := os.MkdirAll(cfg.WorkspaceDir, 0755); err != nil {
		log.Fatalf("Failed to create workspace dir: %v", err)
	}

	mgr, err := nodemanager.NewNodeManager(cfg)
	if err != nil {
		log.Fatalf("Failed to create node manager: %v", err)
	}
	mgr.RegisterAdapter(qwencode.New()) // 优先：免费 2000 请求/天
	mgr.RegisterAdapter(gemini.New())
	mgr.RegisterAdapter(claude.New())

	// HTTP-Only 架构：所有通信通过 HTTPS 与 API Server 交互，无需直连 Redis
	log.Println("HTTP-Only mode: task polling via API Server")

	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down NodeManager...")
		cancel()
	}()

	mgr.Start(ctx)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// buildTLSClient 构建带自定义 CA 证书的 HTTP 客户端
func buildTLSClient(caFile string) (*http.Client, error) {
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caPool,
			},
		},
	}, nil
}

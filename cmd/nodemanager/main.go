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
	"syscall"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"

	"agents-admin/internal/nodemanager"
	"agents-admin/internal/nodemanager/adapter/claude"
	"agents-admin/internal/nodemanager/adapter/gemini"
	"agents-admin/internal/nodemanager/adapter/qwencode"
	"agents-admin/internal/shared/infra"
	"agents-admin/internal/shared/storage"
)

// nodeManagerYAML YAML 配置文件结构
type nodeManagerYAML struct {
	Node struct {
		ID           string            `yaml:"id"`
		APIServerURL string            `yaml:"api_server_url"`
		WorkspaceDir string            `yaml:"workspace_dir"`
		Labels       map[string]string `yaml:"labels"`
	} `yaml:"node"`
	Redis struct {
		URL string `yaml:"url"`
	} `yaml:"redis"`
	Etcd struct {
		Endpoints string `yaml:"endpoints"`
		Prefix    string `yaml:"prefix"`
	} `yaml:"etcd"`
	TLS struct {
		Enabled bool   `yaml:"enabled"`
		CAFile  string `yaml:"ca_file"`
	} `yaml:"tls"`
}

func main() {
	configDir := flag.String("config", "", "配置文件目录（默认搜索 configs/）")
	flag.Parse()

	// 加载 .env 文件（如果存在）
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	log.Println("Starting NodeManager...")

	// 加载 YAML 配置
	yamlCfg := loadNodeManagerConfig(*configDir)

	cfg := nodemanager.Config{
		NodeID:       firstNonEmpty(yamlCfg.Node.ID, getEnv("NODE_ID", "node-001")),
		APIServerURL: firstNonEmpty(yamlCfg.Node.APIServerURL, getEnv("API_SERVER_URL", "http://localhost:8080")),
		WorkspaceDir: firstNonEmpty(yamlCfg.Node.WorkspaceDir, getEnv("WORKSPACE_DIR", "/tmp/workspaces")),
		Labels:       yamlCfg.Node.Labels,
	}
	if len(cfg.Labels) == 0 {
		cfg.Labels = map[string]string{"os": "linux"}
	}

	// 如果启用 TLS，配置自定义 CA 的 HTTP 客户端
	if yamlCfg.TLS.Enabled && yamlCfg.TLS.CAFile != "" {
		tlsClient, err := buildTLSClient(yamlCfg.TLS.CAFile)
		if err != nil {
			log.Fatalf("Failed to load TLS CA: %v", err)
		}
		cfg.HTTPClient = tlsClient
		log.Printf("TLS enabled, CA: %s", yamlCfg.TLS.CAFile)
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

	// 初始化 Redis（用于任务分发事件驱动）
	redisURL := firstNonEmpty(yamlCfg.Redis.URL, getEnv("REDIS_URL", "redis://localhost:6379/0"))
	redisInfra, err := infra.NewRedisInfra(redisURL)
	if err != nil {
		log.Printf("WARNING: Redis not available, using HTTP polling mode: %v", err)
	} else {
		defer redisInfra.Close()
		mgr.SetNodeQueue(redisInfra)
		log.Println("Node queue enabled (event-driven task dispatch)")
	}

	// 初始化 etcd EventBus（可选，用于事件驱动模式）
	etcdEndpoints := firstNonEmpty(yamlCfg.Etcd.Endpoints, getEnv("ETCD_ENDPOINTS", "localhost:2379"))
	etcdPrefix := firstNonEmpty(yamlCfg.Etcd.Prefix, "/agents")
	etcdStore, err := storage.NewEtcdStore(storage.EtcdConfig{
		Endpoints: []string{etcdEndpoints},
		Prefix:    etcdPrefix,
	})
	if err != nil {
		log.Printf("WARNING: etcd not available, using HTTP mode: %v", err)
	} else {
		defer etcdStore.Close()
		eventBus := storage.NewEtcdEventBusFromStore(etcdStore)
		mgr.SetEventBus(eventBus)
		log.Println("EventBus enabled (event-driven mode)")
	}

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

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// loadNodeManagerConfig 加载 nodemanager.yaml 配置
// 搜索顺序: --config 指定目录 → configs/ → ../configs/ → 环境变量兜底
func loadNodeManagerConfig(cfgDir string) *nodeManagerYAML {
	cfg := &nodeManagerYAML{}

	searchPaths := []string{"configs", "../configs", "../../configs"}
	if cfgDir != "" {
		searchPaths = []string{cfgDir}
	}

	for _, base := range searchPaths {
		path := filepath.Join(base, "nodemanager.yaml")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			log.Printf("WARNING: failed to parse %s: %v", path, err)
			continue
		}
		log.Printf("Loaded config from %s", path)
		return cfg
	}

	log.Println("No nodemanager.yaml found, using environment variables")
	return cfg
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

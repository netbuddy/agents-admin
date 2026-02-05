// Package main 节点管理器入口
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"agents-admin/internal/nodemanager"
	"agents-admin/internal/nodemanager/adapter/claude"
	"agents-admin/internal/nodemanager/adapter/gemini"
	"agents-admin/internal/nodemanager/adapter/qwencode"
	"agents-admin/internal/shared/infra"
	"agents-admin/internal/shared/storage"
)

func main() {
	// 加载 .env 文件（如果存在）
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	log.Println("Starting NodeManager...")

	cfg := nodemanager.Config{
		NodeID:       getEnv("NODE_ID", "node-001"),
		APIServerURL: getEnv("API_SERVER_URL", "http://localhost:8080"),
		WorkspaceDir: getEnv("WORKSPACE_DIR", "/tmp/workspaces"),
		Labels: map[string]string{
			"os": "linux",
		},
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
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379/0")
	redisInfra, err := infra.NewRedisInfra(redisURL)
	if err != nil {
		log.Printf("WARNING: Redis not available, using HTTP polling mode: %v", err)
	} else {
		defer redisInfra.Close()
		mgr.SetNodeQueue(redisInfra)
		log.Println("Node queue enabled (event-driven task dispatch)")
	}

	// 初始化 etcd EventBus（可选，用于事件驱动模式）
	etcdEndpoints := getEnv("ETCD_ENDPOINTS", "localhost:2379")
	etcdStore, err := storage.NewEtcdStore(storage.EtcdConfig{
		Endpoints: []string{etcdEndpoints},
		Prefix:    "/agents",
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

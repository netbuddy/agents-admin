// Package main 执行器入口
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"agents-admin/internal/executor"
	"agents-admin/internal/storage"
	"agents-admin/pkg/driver/claude"
	"agents-admin/pkg/driver/gemini"
	"agents-admin/pkg/driver/qwencode"
)

func main() {
	log.Println("Starting Executor...")

	cfg := executor.Config{
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

	exec, err := executor.NewExecutor(cfg)
	if err != nil {
		log.Fatalf("Failed to create executor: %v", err)
	}
	exec.RegisterDriver(qwencode.New()) // 优先：免费 2000 请求/天
	exec.RegisterDriver(gemini.New())
	exec.RegisterDriver(claude.New())

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
		exec.SetEventBus(eventBus)
		log.Println("EventBus enabled (event-driven mode)")
	}

	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down Executor...")
		cancel()
	}()

	exec.Start(ctx)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

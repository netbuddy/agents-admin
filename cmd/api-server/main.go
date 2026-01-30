// Package main API Server 入口
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agents-admin/internal/api"
	"agents-admin/internal/storage"
)

func main() {
	log.Println("Starting API Server...")

	port := getEnv("PORT", "8080")
	databaseURL := getEnv("DATABASE_URL", "postgres://agents:agents_dev_password@localhost:5432/agents_admin?sslmode=disable")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6380")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	etcdEndpoints := getEnv("ETCD_ENDPOINTS", "localhost:2379")

	// 初始化 PostgreSQL（持久化业务数据）
	store, err := storage.NewPostgresStore(databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer store.Close()
	log.Println("Connected to PostgreSQL")

	// 初始化 Redis（运行时状态、事件流）
	redisStore, err := storage.NewRedisStore(redisAddr, redisPassword, 0)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisStore.Close()
	log.Println("Connected to Redis")

	// 初始化 etcd（可选，已弃用，保留兼容）
	var etcdStore *storage.EtcdStore
	etcdStore, err = storage.NewEtcdStore(storage.EtcdConfig{
		Endpoints: []string{etcdEndpoints},
		Prefix:    "/agents",
	})
	if err != nil {
		log.Printf("WARNING: etcd not available (deprecated): %v", err)
		etcdStore = nil
	} else {
		defer etcdStore.Close()
		log.Println("Connected to etcd (deprecated, kept for compatibility)")
	}

	// 初始化 Handler
	handler := api.NewHandler(store, redisStore, etcdStore)

	// 初始化 EventBus（已弃用，保留兼容）
	if etcdStore != nil {
		eventBus := storage.NewEtcdEventBusFromStore(etcdStore)
		handler.SetEventBus(eventBus)
		log.Println("Legacy EventBus enabled (deprecated)")
	}

	// 启动调度器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go handler.StartScheduler(ctx)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler.Router(),
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

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("API Server listening on :%s", port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	fmt.Println("Server stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

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

	"agents-admin/internal/apiserver/server"
	"agents-admin/internal/config"
	"agents-admin/internal/shared/infra"
	"agents-admin/internal/shared/storage"
)

func main() {
	// 加载配置（自动加载 .env，根据 TEST_ENV 切换数据库和 Redis）
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

	// 启动调度器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.StartScheduler(ctx)

	srv := &http.Server{
		Addr:         ":" + cfg.APIPort,
		Handler:      h.Router(),
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

	log.Printf("API Server listening on :%s", cfg.APIPort)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	fmt.Println("Server stopped")
}

// Package redis Redis 缓存实现
package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Store Redis 缓存存储
type Store struct {
	client *redis.Client
}

// NewStore 创建 Redis 缓存实例
func NewStore(addr, password string, db int) (*Store, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Printf("[Redis/Cache] Connected to %s", addr)
	return &Store{client: client}, nil
}

// NewStoreFromURL 从 URL 创建 Redis 缓存实例
func NewStoreFromURL(redisURL string) (*Store, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Printf("[Redis/Cache] Connected to %s", opts.Addr)
	return &Store{client: client}, nil
}

// NewStoreFromClient 从现有 Redis 客户端创建缓存实例
func NewStoreFromClient(client *redis.Client) *Store {
	return &Store{client: client}
}

// Close 关闭 Redis 连接
func (s *Store) Close() error {
	return s.client.Close()
}

// Client 返回底层 Redis 客户端
func (s *Store) Client() *redis.Client {
	return s.client
}

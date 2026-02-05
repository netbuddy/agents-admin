// Package storage 提供存储层抽象
//
// mock.go 提供用于测试的 NoOp 实现
package storage

import (
	"agents-admin/internal/shared/cache"
	"agents-admin/internal/shared/eventbus"
	"agents-admin/internal/shared/queue"
)

// ============================================================================
// NoOpCacheStore - 空操作的 CacheStore 实现（用于测试）
// ============================================================================

// NoOpCacheStore 是一个组合了 cache、eventbus、queue 的空操作实现
// Deprecated: 建议直接使用 cache.NoOpCache, eventbus.NoOpEventBus, queue.NoOpQueue
type NoOpCacheStore struct {
	*cache.NoOpCache
	*eventbus.NoOpEventBus
	*queue.NoOpQueue
}

// NewNoOpCacheStore 创建 NoOpCacheStore 实例
func NewNoOpCacheStore() *NoOpCacheStore {
	return &NoOpCacheStore{
		NoOpCache:    cache.NewNoOpCache(),
		NoOpEventBus: eventbus.NewNoOpEventBus(),
		NoOpQueue:    queue.NewNoOpQueue(),
	}
}

// Close 关闭存储
func (s *NoOpCacheStore) Close() error {
	return nil
}

// 确保 NoOpCacheStore 实现了 CacheStore 接口
var _ CacheStore = (*NoOpCacheStore)(nil)

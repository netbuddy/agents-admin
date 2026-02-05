// Package infra 基础设施聚合层
//
// 提供统一的基础设施初始化和依赖注入，包括：
//   - Storage：持久化存储（PostgreSQL）
//   - Cache：缓存（Redis），包含节点心跳缓存
//   - EventBus：事件总线（Redis Streams/Pub-Sub）
//   - Queue：消息队列（Redis Streams）
package infra

import (
	"agents-admin/internal/shared/cache"
	"agents-admin/internal/shared/eventbus"
	"agents-admin/internal/shared/queue"
	"agents-admin/internal/shared/storage"
)

// Infrastructure 基础设施聚合结构
type Infrastructure struct {
	// Storage 持久化存储（PostgreSQL）
	Storage storage.PersistentStore

	// Cache 缓存（Redis），包含节点心跳缓存（NodeHeartbeatCache）
	Cache cache.Cache

	// EventBus 事件总线（Redis）
	EventBus eventbus.EventBus

	// Queue 消息队列（Redis）
	Queue queue.Queue
}

// Close 关闭所有基础设施连接
func (i *Infrastructure) Close() error {
	var lastErr error

	if i.Storage != nil {
		if err := i.Storage.Close(); err != nil {
			lastErr = err
		}
	}

	if i.Cache != nil {
		if err := i.Cache.Close(); err != nil {
			lastErr = err
		}
	}

	if i.EventBus != nil {
		if err := i.EventBus.Close(); err != nil {
			lastErr = err
		}
	}

	if i.Queue != nil {
		if err := i.Queue.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// NewNoOpInfrastructure 创建空操作的基础设施（用于测试）
func NewNoOpInfrastructure() *Infrastructure {
	return &Infrastructure{
		Cache:    cache.NewNoOpCache(),
		EventBus: eventbus.NewNoOpEventBus(),
		Queue:    queue.NewNoOpQueue(),
	}
}

// Package redis NodeHeartbeat 缓存操作
package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"agents-admin/internal/shared/cache"
)

// UpdateNodeHeartbeat 更新节点心跳
func (s *Store) UpdateNodeHeartbeat(ctx context.Context, nodeID string, status *cache.NodeStatus) error {
	key := cache.KeyNodeHeartbeat + nodeID

	status.UpdatedAt = time.Now()
	data, err := json.Marshal(status)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, key, data, cache.TTLNodeHeartbeat).Err()
}

// GetNodeHeartbeat 获取节点心跳
func (s *Store) GetNodeHeartbeat(ctx context.Context, nodeID string) (*cache.NodeStatus, error) {
	key := cache.KeyNodeHeartbeat + nodeID

	data, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var status cache.NodeStatus
	if err := json.Unmarshal([]byte(data), &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// DeleteNodeHeartbeat 删除节点心跳缓存
func (s *Store) DeleteNodeHeartbeat(ctx context.Context, nodeID string) error {
	key := cache.KeyNodeHeartbeat + nodeID
	return s.client.Del(ctx, key).Err()
}

// ListOnlineNodes 列出在线节点
//
// 使用 SCAN 替代 KEYS，避免在节点数量大时阻塞 Redis
func (s *Store) ListOnlineNodes(ctx context.Context) ([]string, error) {
	pattern := cache.KeyNodeHeartbeat + "*"
	var nodeIDs []string
	iter := s.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		nodeID := key[len(cache.KeyNodeHeartbeat):]
		nodeIDs = append(nodeIDs, nodeID)
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return nodeIDs, nil
}

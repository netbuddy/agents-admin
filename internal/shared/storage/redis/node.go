// Package redis NodeHeartbeat 和 NodeTasks 相关操作
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"agents-admin/internal/shared/storagetypes"
)

// === NodeHeartbeat ===

// UpdateNodeHeartbeat 更新节点心跳
func (s *Store) UpdateNodeHeartbeat(ctx context.Context, nodeID string, status *storagetypes.NodeStatus) error {
	key := storagetypes.KeyNodeHeartbeat + nodeID

	status.UpdatedAt = time.Now()
	data, err := json.Marshal(status)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, key, data, storagetypes.TTLNodeHeartbeat).Err()
}

// GetNodeHeartbeat 获取节点心跳
func (s *Store) GetNodeHeartbeat(ctx context.Context, nodeID string) (*storagetypes.NodeStatus, error) {
	key := storagetypes.KeyNodeHeartbeat + nodeID

	data, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var status storagetypes.NodeStatus
	if err := json.Unmarshal([]byte(data), &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// DeleteNodeHeartbeat 删除节点心跳缓存
func (s *Store) DeleteNodeHeartbeat(ctx context.Context, nodeID string) error {
	key := storagetypes.KeyNodeHeartbeat + nodeID
	return s.client.Del(ctx, key).Err()
}

// ListOnlineNodes 列出在线节点
//
// 使用 SCAN 替代 KEYS，避免在节点数量大时阻塞 Redis
func (s *Store) ListOnlineNodes(ctx context.Context) ([]string, error) {
	pattern := storagetypes.KeyNodeHeartbeat + "*"
	var nodeIDs []string
	iter := s.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		nodeID := key[len(storagetypes.KeyNodeHeartbeat):]
		nodeIDs = append(nodeIDs, nodeID)
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return nodeIDs, nil
}

// === NodeTasks ===

func nodeTasksKey(nodeID string) string {
	return storagetypes.KeyNodeTasks + nodeID + storagetypes.KeyNodeTasksSuffix
}

// PublishTaskToNode 发布任务到节点的 Stream
func (s *Store) PublishTaskToNode(ctx context.Context, nodeID, runID, taskID string) (string, error) {
	key := nodeTasksKey(nodeID)

	args := &redis.XAddArgs{
		Stream: key,
		MaxLen: 1000,
		Approx: true,
		Values: map[string]interface{}{
			"run_id":      runID,
			"task_id":     taskID,
			"assigned_at": time.Now().Format(time.RFC3339Nano),
		},
	}

	msgID, err := s.client.XAdd(ctx, args).Result()
	if err != nil {
		return "", fmt.Errorf("failed to publish task to node %s: %w", nodeID, err)
	}

	log.Printf("[Redis] Published task to node: node=%s run=%s task=%s msg_id=%s", nodeID, runID, taskID, msgID)
	return msgID, nil
}

// CreateNodeConsumerGroup 创建节点消费者组
func (s *Store) CreateNodeConsumerGroup(ctx context.Context, nodeID string) error {
	key := nodeTasksKey(nodeID)

	err := s.client.XGroupCreateMkStream(ctx, key, storagetypes.NodeManagerConsumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group for node %s: %w", nodeID, err)
	}

	log.Printf("[Redis] Created consumer group for node: %s", nodeID)
	return nil
}

// ConsumeNodeTasks 消费节点任务
func (s *Store) ConsumeNodeTasks(ctx context.Context, nodeID, consumerID string, count int64, blockTimeout time.Duration) ([]*storagetypes.NodeTaskMessage, error) {
	key := nodeTasksKey(nodeID)

	streams, err := s.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    storagetypes.NodeManagerConsumerGroup,
		Consumer: consumerID,
		Streams:  []string{key, ">"},
		Count:    count,
		Block:    blockTimeout,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to consume node tasks: %w", err)
	}

	var messages []*storagetypes.NodeTaskMessage
	for _, stream := range streams {
		for _, msg := range stream.Messages {
			m := &storagetypes.NodeTaskMessage{
				ID: msg.ID,
			}
			if runID, ok := msg.Values["run_id"].(string); ok {
				m.RunID = runID
			}
			if taskID, ok := msg.Values["task_id"].(string); ok {
				m.TaskID = taskID
			}
			if assignedAt, ok := msg.Values["assigned_at"].(string); ok {
				if t, err := time.Parse(time.RFC3339Nano, assignedAt); err == nil {
					m.AssignedAt = t
				}
			}
			messages = append(messages, m)
		}
	}

	if len(messages) > 0 {
		log.Printf("[Redis] Consumed %d tasks for node: %s", len(messages), nodeID)
	}

	return messages, nil
}

// AckNodeTask 确认节点任务已处理
func (s *Store) AckNodeTask(ctx context.Context, nodeID, messageID string) error {
	key := nodeTasksKey(nodeID)
	return s.client.XAck(ctx, key, storagetypes.NodeManagerConsumerGroup, messageID).Err()
}

// GetNodeTasksQueueLength 获取节点任务队列长度
func (s *Store) GetNodeTasksQueueLength(ctx context.Context, nodeID string) (int64, error) {
	key := nodeTasksKey(nodeID)
	return s.client.XLen(ctx, key).Result()
}

// GetNodeTasksPendingCount 获取节点未确认消息数量
func (s *Store) GetNodeTasksPendingCount(ctx context.Context, nodeID string) (int64, error) {
	key := nodeTasksKey(nodeID)
	pending, err := s.client.XPending(ctx, key, storagetypes.NodeManagerConsumerGroup).Result()
	if err != nil {
		return 0, err
	}
	return pending.Count, nil
}

// Package redis NodeRunQueue 操作
package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"agents-admin/internal/shared/queue"
)

func nodeRunsKey(nodeID string) string {
	return queue.KeyNodeRuns + nodeID + queue.KeyNodeRunsSuffix
}

// PublishRunToNode 将 Run 分配给指定节点
func (s *Store) PublishRunToNode(ctx context.Context, nodeID, runID, taskID string) (string, error) {
	key := nodeRunsKey(nodeID)

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
		return "", fmt.Errorf("failed to publish run to node %s: %w", nodeID, err)
	}

	log.Printf("[Redis/Queue] Published run to node: node=%s run=%s task=%s msg_id=%s", nodeID, runID, taskID, msgID)
	return msgID, nil
}

// CreateNodeConsumerGroup 创建节点消费者组
func (s *Store) CreateNodeConsumerGroup(ctx context.Context, nodeID string) error {
	key := nodeRunsKey(nodeID)

	err := s.client.XGroupCreateMkStream(ctx, key, queue.NodeManagerConsumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group for node %s: %w", nodeID, err)
	}

	log.Printf("[Redis/Queue] Created consumer group for node: %s", nodeID)
	return nil
}

// ConsumeNodeRuns 消费节点分配的 Run
func (s *Store) ConsumeNodeRuns(ctx context.Context, nodeID, consumerID string, count int64, blockTimeout time.Duration) ([]*queue.NodeRunMessage, error) {
	key := nodeRunsKey(nodeID)

	streams, err := s.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    queue.NodeManagerConsumerGroup,
		Consumer: consumerID,
		Streams:  []string{key, ">"},
		Count:    count,
		Block:    blockTimeout,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to consume node runs: %w", err)
	}

	var messages []*queue.NodeRunMessage
	for _, stream := range streams {
		for _, msg := range stream.Messages {
			m := &queue.NodeRunMessage{
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
		log.Printf("[Redis/Queue] Consumed %d runs for node: %s", len(messages), nodeID)
	}

	return messages, nil
}

// AckNodeRun 确认节点 Run 消息已处理
func (s *Store) AckNodeRun(ctx context.Context, nodeID, messageID string) error {
	key := nodeRunsKey(nodeID)
	return s.client.XAck(ctx, key, queue.NodeManagerConsumerGroup, messageID).Err()
}

// GetNodeRunsQueueLength 获取节点 Run 队列长度
func (s *Store) GetNodeRunsQueueLength(ctx context.Context, nodeID string) (int64, error) {
	key := nodeRunsKey(nodeID)
	return s.client.XLen(ctx, key).Result()
}

// GetNodeRunsPendingCount 获取节点未确认 Run 消息数量
func (s *Store) GetNodeRunsPendingCount(ctx context.Context, nodeID string) (int64, error) {
	key := nodeRunsKey(nodeID)
	pending, err := s.client.XPending(ctx, key, queue.NodeManagerConsumerGroup).Result()
	if err != nil {
		return 0, err
	}
	return pending.Count, nil
}

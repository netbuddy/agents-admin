// Package redis SchedulerQueue 相关操作
package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"agents-admin/internal/shared/storagetypes"
)

// ScheduleRun 将 Run 加入调度队列（等待分配节点）
func (s *Store) ScheduleRun(ctx context.Context, runID, taskID string) (string, error) {
	args := &redis.XAddArgs{
		Stream: storagetypes.KeySchedulerRuns,
		MaxLen: 10000,
		Approx: true,
		Values: map[string]interface{}{
			"run_id":     runID,
			"task_id":    taskID,
			"created_at": time.Now().Format(time.RFC3339Nano),
		},
	}

	return s.client.XAdd(ctx, args).Result()
}

// CreateSchedulerConsumerGroup 创建调度器消费者组
func (s *Store) CreateSchedulerConsumerGroup(ctx context.Context) error {
	err := s.client.XGroupCreateMkStream(ctx, storagetypes.KeySchedulerRuns, storagetypes.SchedulerConsumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

// ConsumeSchedulerRuns 消费调度队列中的 Run
func (s *Store) ConsumeSchedulerRuns(ctx context.Context, consumerID string, count int64, blockTimeout time.Duration) ([]*storagetypes.SchedulerMessage, error) {
	streams, err := s.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    storagetypes.SchedulerConsumerGroup,
		Consumer: consumerID,
		Streams:  []string{storagetypes.KeySchedulerRuns, ">"},
		Count:    count,
		Block:    blockTimeout,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var messages []*storagetypes.SchedulerMessage
	for _, stream := range streams {
		for _, msg := range stream.Messages {
			m := &storagetypes.SchedulerMessage{
				ID: msg.ID,
			}
			if runID, ok := msg.Values["run_id"].(string); ok {
				m.RunID = runID
			}
			if taskID, ok := msg.Values["task_id"].(string); ok {
				m.TaskID = taskID
			}
			if createdAt, ok := msg.Values["created_at"].(string); ok {
				if t, err := time.Parse(time.RFC3339Nano, createdAt); err == nil {
					m.CreatedAt = t
				}
			}
			messages = append(messages, m)
		}
	}

	return messages, nil
}

// AckSchedulerRun 确认 Run 调度消息已处理
func (s *Store) AckSchedulerRun(ctx context.Context, messageID string) error {
	return s.client.XAck(ctx, storagetypes.KeySchedulerRuns, storagetypes.SchedulerConsumerGroup, messageID).Err()
}

// GetSchedulerQueueLength 获取调度队列长度
func (s *Store) GetSchedulerQueueLength(ctx context.Context) (int64, error) {
	return s.client.XLen(ctx, storagetypes.KeySchedulerRuns).Result()
}

// GetSchedulerPendingCount 获取未确认消息数量
func (s *Store) GetSchedulerPendingCount(ctx context.Context) (int64, error) {
	pending, err := s.client.XPending(ctx, storagetypes.KeySchedulerRuns, storagetypes.SchedulerConsumerGroup).Result()
	if err != nil {
		return 0, err
	}
	return pending.Count, nil
}

// GetStreamLength 获取 Stream 长度
func (s *Store) GetStreamLength(ctx context.Context, stream string) (int64, error) {
	return s.client.XLen(ctx, stream).Result()
}

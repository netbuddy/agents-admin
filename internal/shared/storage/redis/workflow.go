// Package redis WorkflowState 和 WorkflowEvents 相关操作
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"agents-admin/internal/shared/storagetypes"
)

// === WorkflowState ===

// SetWorkflowState 设置工作流状态
func (s *Store) SetWorkflowState(ctx context.Context, wfType, wfID string, state *storagetypes.RedisWorkflowState) error {
	key := fmt.Sprintf("%s%s:%s", storagetypes.KeyWorkflowState, wfType, wfID)

	data := map[string]interface{}{
		"state":        state.State,
		"progress":     state.Progress,
		"current_step": state.CurrentStep,
		"error":        state.Error,
	}

	pipe := s.client.Pipeline()
	pipe.HSet(ctx, key, data)
	pipe.Expire(ctx, key, storagetypes.TTLWorkflowState)
	_, err := pipe.Exec(ctx)

	return err
}

// GetWorkflowState 获取工作流状态
func (s *Store) GetWorkflowState(ctx context.Context, wfType, wfID string) (*storagetypes.RedisWorkflowState, error) {
	key := fmt.Sprintf("%s%s:%s", storagetypes.KeyWorkflowState, wfType, wfID)

	result, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	state := &storagetypes.RedisWorkflowState{
		State:       result["state"],
		CurrentStep: result["current_step"],
		Error:       result["error"],
	}

	if progress, err := strconv.Atoi(result["progress"]); err == nil {
		state.Progress = progress
	}

	return state, nil
}

// DeleteWorkflowState 删除工作流状态
func (s *Store) DeleteWorkflowState(ctx context.Context, wfType, wfID string) error {
	key := fmt.Sprintf("%s%s:%s", storagetypes.KeyWorkflowState, wfType, wfID)
	return s.client.Del(ctx, key).Err()
}

// === WorkflowEvents ===

// PublishEvent 发布工作流事件
func (s *Store) PublishEvent(ctx context.Context, wfType, wfID string, event *storagetypes.RedisWorkflowEvent) error {
	key := fmt.Sprintf("%s%s:%s", storagetypes.KeyWorkflowEvents, wfType, wfID)

	dataJSON, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	args := &redis.XAddArgs{
		Stream: key,
		MaxLen: storagetypes.MaxStreamLength,
		Approx: true,
		Values: map[string]interface{}{
			"type":      event.Type,
			"timestamp": event.Timestamp.Format(time.RFC3339Nano),
			"data":      string(dataJSON),
		},
	}

	id, err := s.client.XAdd(ctx, args).Result()
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	log.Printf("[Redis] Published event: %s/%s seq=%s type=%s", wfType, wfID, id, event.Type)
	return nil
}

// GetEvents 获取工作流事件列表
func (s *Store) GetEvents(ctx context.Context, wfType, wfID string, fromID string, count int64) ([]*storagetypes.RedisWorkflowEvent, error) {
	key := fmt.Sprintf("%s%s:%s", storagetypes.KeyWorkflowEvents, wfType, wfID)

	if fromID == "" {
		fromID = "0"
	}

	msgs, err := s.client.XRange(ctx, key, fromID, "+").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	var events []*storagetypes.RedisWorkflowEvent
	for i, msg := range msgs {
		event := &storagetypes.RedisWorkflowEvent{
			ID:   msg.ID,
			Seq:  i + 1,
			Type: msg.Values["type"].(string),
		}

		if ts, ok := msg.Values["timestamp"].(string); ok {
			if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
				event.Timestamp = t
			}
		}

		if dataStr, ok := msg.Values["data"].(string); ok {
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(dataStr), &data); err == nil {
				event.Data = data
			}
		}

		events = append(events, event)

		if count > 0 && int64(len(events)) >= count {
			break
		}
	}

	return events, nil
}

// GetEventCount 获取事件数量
func (s *Store) GetEventCount(ctx context.Context, wfType, wfID string) (int64, error) {
	key := fmt.Sprintf("%s%s:%s", storagetypes.KeyWorkflowEvents, wfType, wfID)
	return s.client.XLen(ctx, key).Result()
}

// SubscribeEvents 订阅工作流事件
func (s *Store) SubscribeEvents(ctx context.Context, wfType, wfID string) (<-chan *storagetypes.RedisWorkflowEvent, error) {
	key := fmt.Sprintf("%s%s:%s", storagetypes.KeyWorkflowEvents, wfType, wfID)
	ch := make(chan *storagetypes.RedisWorkflowEvent, 100)

	go func() {
		defer close(ch)
		lastID := "$"

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			streams, err := s.client.XRead(ctx, &redis.XReadArgs{
				Streams: []string{key, lastID},
				Count:   10,
				Block:   5 * time.Second,
			}).Result()

			if err != nil {
				if err == redis.Nil {
					continue
				}
				log.Printf("[Redis] Event subscription error: %v", err)
				return
			}

			for _, stream := range streams {
				for _, msg := range stream.Messages {
					event := &storagetypes.RedisWorkflowEvent{
						ID:   msg.ID,
						Type: msg.Values["type"].(string),
					}

					if ts, ok := msg.Values["timestamp"].(string); ok {
						if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
							event.Timestamp = t
						}
					}

					if dataStr, ok := msg.Values["data"].(string); ok {
						var data map[string]interface{}
						if err := json.Unmarshal([]byte(dataStr), &data); err == nil {
							event.Data = data
						}
					}

					select {
					case ch <- event:
						lastID = msg.ID
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return ch, nil
}

// DeleteEvents 删除工作流事件流
func (s *Store) DeleteEvents(ctx context.Context, wfType, wfID string) error {
	key := fmt.Sprintf("%s%s:%s", storagetypes.KeyWorkflowEvents, wfType, wfID)
	return s.client.Del(ctx, key).Err()
}

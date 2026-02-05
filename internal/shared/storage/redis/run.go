// Package redis RunEvent 相关操作
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

// PublishRunEvent 发布 Run 事件
func (s *Store) PublishRunEvent(ctx context.Context, runID string, event *storagetypes.RunEvent) error {
	key := storagetypes.KeyRunEvents + runID

	payloadJSON, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	values := map[string]interface{}{
		"seq":       event.Seq,
		"type":      event.Type,
		"timestamp": event.Timestamp.Format(time.RFC3339Nano),
		"payload":   string(payloadJSON),
	}

	if event.Raw != "" {
		values["raw"] = event.Raw
	}

	args := &redis.XAddArgs{
		Stream: key,
		MaxLen: storagetypes.MaxStreamLength,
		Approx: true,
		Values: values,
	}

	id, err := s.client.XAdd(ctx, args).Result()
	if err != nil {
		return fmt.Errorf("failed to publish run event: %w", err)
	}

	event.ID = id
	log.Printf("[Redis] Published run event: run=%s seq=%d type=%s", runID, event.Seq, event.Type)
	return nil
}

// GetRunEvents 获取 Run 事件列表
func (s *Store) GetRunEvents(ctx context.Context, runID string, fromSeq int, count int64) ([]*storagetypes.RunEvent, error) {
	key := storagetypes.KeyRunEvents + runID

	msgs, err := s.client.XRange(ctx, key, "-", "+").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get run events: %w", err)
	}

	var events []*storagetypes.RunEvent
	for _, msg := range msgs {
		event, err := parseRunEvent(runID, msg)
		if err != nil {
			continue
		}

		if event.Seq <= fromSeq {
			continue
		}

		events = append(events, event)

		if count > 0 && int64(len(events)) >= count {
			break
		}
	}

	return events, nil
}

// GetRunEventCount 获取 Run 事件数量
func (s *Store) GetRunEventCount(ctx context.Context, runID string) (int64, error) {
	key := storagetypes.KeyRunEvents + runID
	return s.client.XLen(ctx, key).Result()
}

// SubscribeRunEvents 订阅 Run 事件
func (s *Store) SubscribeRunEvents(ctx context.Context, runID string) (<-chan *storagetypes.RunEvent, error) {
	key := storagetypes.KeyRunEvents + runID
	ch := make(chan *storagetypes.RunEvent, 100)

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
				log.Printf("[Redis] Run event subscription error: %v", err)
				return
			}

			for _, stream := range streams {
				for _, msg := range stream.Messages {
					event, err := parseRunEvent(runID, msg)
					if err != nil {
						continue
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

// DeleteRunEvents 删除 Run 事件流
func (s *Store) DeleteRunEvents(ctx context.Context, runID string) error {
	key := storagetypes.KeyRunEvents + runID
	return s.client.Del(ctx, key).Err()
}

func parseRunEvent(runID string, msg redis.XMessage) (*storagetypes.RunEvent, error) {
	event := &storagetypes.RunEvent{
		ID:    msg.ID,
		RunID: runID,
	}

	if seqStr, ok := msg.Values["seq"].(string); ok {
		if seq, err := strconv.Atoi(seqStr); err == nil {
			event.Seq = seq
		}
	}

	if eventType, ok := msg.Values["type"].(string); ok {
		event.Type = eventType
	}

	if ts, ok := msg.Values["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			event.Timestamp = t
		}
	}

	if payloadStr, ok := msg.Values["payload"].(string); ok {
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(payloadStr), &payload); err == nil {
			event.Payload = payload
		}
	}

	if raw, ok := msg.Values["raw"].(string); ok {
		event.Raw = raw
	}

	return event, nil
}

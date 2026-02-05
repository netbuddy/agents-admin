// Package etcd etcd 事件总线实现
//
// Deprecated: P2-3 架构重构后，事件总线统一使用 Redis Streams。
// 本实现保留用于兼容，新功能请使用 redis.Store 的事件流 API
package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"agents-admin/internal/shared/storagetypes"
)

// EventBus etcd 事件总线实现
type EventBus struct {
	client *clientv3.Client
	prefix string
}

// NewEventBus 创建 etcd 事件总线
func NewEventBus(client *clientv3.Client, prefix string) *EventBus {
	if prefix == "" {
		prefix = "/agents"
	}
	return &EventBus{
		client: client,
		prefix: prefix,
	}
}

// NewEventBusFromStore 从 Store 创建事件总线
func NewEventBusFromStore(store *Store) *EventBus {
	return &EventBus{
		client: store.client,
		prefix: store.prefix,
	}
}

func (e *EventBus) eventKey(workflowType, workflowID string, seq int64) string {
	return fmt.Sprintf("%s/events/%s/%s/%06d", e.prefix, workflowType, workflowID, seq)
}

func (e *EventBus) eventPrefix(workflowType, workflowID string) string {
	return fmt.Sprintf("%s/events/%s/%s/", e.prefix, workflowType, workflowID)
}

func (e *EventBus) stateKey(workflowType, workflowID string) string {
	return fmt.Sprintf("%s/state/%s/%s", e.prefix, workflowType, workflowID)
}

func (e *EventBus) statePrefix(workflowType string) string {
	return fmt.Sprintf("%s/state/%s/", e.prefix, workflowType)
}

// Publish 发布事件
func (e *EventBus) Publish(ctx context.Context, workflowType, workflowID string, event *storagetypes.WorkflowEvent) error {
	seq, err := e.GetNextSeq(ctx, workflowType, workflowID)
	if err != nil {
		return fmt.Errorf("failed to get next seq: %w", err)
	}
	event.Seq = seq
	event.WorkflowID = workflowID

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	eventKey := e.eventKey(workflowType, workflowID, seq)
	lease, err := e.client.Grant(ctx, 24*60*60)
	if err != nil {
		return fmt.Errorf("failed to create lease: %w", err)
	}

	_, err = e.client.Put(ctx, eventKey, string(eventData), clientv3.WithLease(lease.ID))
	if err != nil {
		return fmt.Errorf("failed to put event: %w", err)
	}

	log.Printf("[EventBus] Published event: %s/%s seq=%d type=%s", workflowType, workflowID, seq, event.Type)
	return nil
}

// PublishWithState 发布事件并更新状态
func (e *EventBus) PublishWithState(ctx context.Context, workflowType, workflowID string, event *storagetypes.WorkflowEvent, state *storagetypes.WorkflowStateData) error {
	seq, err := e.GetNextSeq(ctx, workflowType, workflowID)
	if err != nil {
		return fmt.Errorf("failed to get next seq: %w", err)
	}
	event.Seq = seq
	event.WorkflowID = workflowID

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	state.UpdatedAt = time.Now()

	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	stateData, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	eventKey := e.eventKey(workflowType, workflowID, seq)
	stateKey := e.stateKey(workflowType, workflowID)

	lease, err := e.client.Grant(ctx, 24*60*60)
	if err != nil {
		return fmt.Errorf("failed to create lease: %w", err)
	}

	txn := e.client.Txn(ctx)
	_, err = txn.Then(
		clientv3.OpPut(eventKey, string(eventData), clientv3.WithLease(lease.ID)),
		clientv3.OpPut(stateKey, string(stateData)),
	).Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("[EventBus] Published event with state: %s/%s seq=%d type=%s state=%s",
		workflowType, workflowID, seq, event.Type, state.State)
	return nil
}

// Subscribe 订阅事件
func (e *EventBus) Subscribe(ctx context.Context, workflowType, workflowID string, fromSeq int64) (<-chan *storagetypes.WorkflowEvent, error) {
	eventCh := make(chan *storagetypes.WorkflowEvent, 100)

	go func() {
		defer close(eventCh)

		prefix := e.eventPrefix(workflowType, workflowID)
		resp, err := e.client.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
		if err != nil {
			log.Printf("[EventBus] Failed to get history events: %v", err)
			return
		}

		for _, kv := range resp.Kvs {
			var event storagetypes.WorkflowEvent
			if err := json.Unmarshal(kv.Value, &event); err != nil {
				continue
			}
			if event.Seq > fromSeq {
				select {
				case eventCh <- &event:
				case <-ctx.Done():
					return
				}
			}
		}

		watchCh := e.client.Watch(ctx, prefix, clientv3.WithPrefix(), clientv3.WithRev(resp.Header.Revision+1))
		for watchResp := range watchCh {
			for _, ev := range watchResp.Events {
				if ev.Type != clientv3.EventTypePut {
					continue
				}
				var event storagetypes.WorkflowEvent
				if err := json.Unmarshal(ev.Kv.Value, &event); err != nil {
					continue
				}
				if event.Seq > fromSeq {
					select {
					case eventCh <- &event:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return eventCh, nil
}

// GetState 获取当前状态
func (e *EventBus) GetState(ctx context.Context, workflowType, workflowID string) (*storagetypes.WorkflowStateData, error) {
	key := e.stateKey(workflowType, workflowID)
	resp, err := e.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	var state storagetypes.WorkflowStateData
	if err := json.Unmarshal(resp.Kvs[0].Value, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// SetState 更新状态
func (e *EventBus) SetState(ctx context.Context, workflowType, workflowID string, state *storagetypes.WorkflowStateData) error {
	state.UpdatedAt = time.Now()
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	key := e.stateKey(workflowType, workflowID)
	_, err = e.client.Put(ctx, key, string(data))
	if err != nil {
		return fmt.Errorf("failed to put state: %w", err)
	}

	log.Printf("[EventBus] Set state: %s/%s state=%s", workflowType, workflowID, state.State)
	return nil
}

// WatchState Watch 状态变更
func (e *EventBus) WatchState(ctx context.Context, workflowType, workflowID string) (<-chan *storagetypes.WorkflowStateData, error) {
	stateCh := make(chan *storagetypes.WorkflowStateData, 10)

	go func() {
		defer close(stateCh)

		key := e.stateKey(workflowType, workflowID)

		state, err := e.GetState(ctx, workflowType, workflowID)
		if err == nil && state != nil {
			select {
			case stateCh <- state:
			case <-ctx.Done():
				return
			}
		}

		watchCh := e.client.Watch(ctx, key)
		for watchResp := range watchCh {
			for _, ev := range watchResp.Events {
				if ev.Type == clientv3.EventTypeDelete {
					continue
				}
				var state storagetypes.WorkflowStateData
				if err := json.Unmarshal(ev.Kv.Value, &state); err != nil {
					continue
				}
				select {
				case stateCh <- &state:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return stateCh, nil
}

// WatchAllStates Watch 所有状态变更
func (e *EventBus) WatchAllStates(ctx context.Context, workflowType string) (<-chan *storagetypes.WorkflowStateData, error) {
	stateCh := make(chan *storagetypes.WorkflowStateData, 100)

	go func() {
		defer close(stateCh)

		prefix := e.statePrefix(workflowType)
		watchCh := e.client.Watch(ctx, prefix, clientv3.WithPrefix())
		for watchResp := range watchCh {
			for _, ev := range watchResp.Events {
				if ev.Type == clientv3.EventTypeDelete {
					continue
				}
				var state storagetypes.WorkflowStateData
				if err := json.Unmarshal(ev.Kv.Value, &state); err != nil {
					continue
				}
				select {
				case stateCh <- &state:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return stateCh, nil
}

// GetEvents 获取事件列表
func (e *EventBus) GetEvents(ctx context.Context, workflowType, workflowID string, fromSeq, limit int64) ([]*storagetypes.WorkflowEvent, error) {
	prefix := e.eventPrefix(workflowType, workflowID)
	resp, err := e.client.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	var events []*storagetypes.WorkflowEvent
	for _, kv := range resp.Kvs {
		var event storagetypes.WorkflowEvent
		if err := json.Unmarshal(kv.Value, &event); err != nil {
			continue
		}
		if event.Seq > fromSeq {
			events = append(events, &event)
			if limit > 0 && int64(len(events)) >= limit {
				break
			}
		}
	}

	return events, nil
}

// GetNextSeq 获取下一个序列号
func (e *EventBus) GetNextSeq(ctx context.Context, workflowType, workflowID string) (int64, error) {
	prefix := e.eventPrefix(workflowType, workflowID)
	resp, err := e.client.Get(ctx, prefix,
		clientv3.WithPrefix(),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortDescend),
		clientv3.WithLimit(1))
	if err != nil {
		return 0, fmt.Errorf("failed to get last event: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return 1, nil
	}

	key := string(resp.Kvs[0].Key)
	parts := strings.Split(key, "/")
	if len(parts) == 0 {
		return 1, nil
	}

	lastSeqStr := parts[len(parts)-1]
	lastSeq, err := strconv.ParseInt(lastSeqStr, 10, 64)
	if err != nil {
		return 1, nil
	}

	return lastSeq + 1, nil
}

// DeleteWorkflow 删除工作流的所有事件和状态
func (e *EventBus) DeleteWorkflow(ctx context.Context, workflowType, workflowID string) error {
	eventPrefix := e.eventPrefix(workflowType, workflowID)
	_, err := e.client.Delete(ctx, eventPrefix, clientv3.WithPrefix())
	if err != nil {
		return fmt.Errorf("failed to delete events: %w", err)
	}

	stateKey := e.stateKey(workflowType, workflowID)
	_, err = e.client.Delete(ctx, stateKey)
	if err != nil {
		return fmt.Errorf("failed to delete state: %w", err)
	}

	log.Printf("[EventBus] Deleted workflow: %s/%s", workflowType, workflowID)
	return nil
}

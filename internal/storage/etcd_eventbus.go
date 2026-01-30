// Package storage etcd 事件总线实现
//
// 基于 etcd 实现事件驱动协同架构，提供：
//   - 事件发布/订阅
//   - 状态同步
//   - Watch 通知
//
// 详细设计见 docs/design/event-driven-architecture.md
package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// WorkflowState 工作流状态
type WorkflowState string

const (
	WorkflowStatePending   WorkflowState = "pending"
	WorkflowStateRunning   WorkflowState = "running"
	WorkflowStateWaiting   WorkflowState = "waiting"
	WorkflowStateCompleted WorkflowState = "completed"
	WorkflowStateFailed    WorkflowState = "failed"
	WorkflowStateCancelled WorkflowState = "cancelled"
)

// WorkflowEvent 工作流事件
type WorkflowEvent struct {
	ID         string                 `json:"id"`
	WorkflowID string                 `json:"workflow_id"`
	Type       string                 `json:"type"`
	Seq        int64                  `json:"seq"`
	Data       map[string]interface{} `json:"data"`
	ProducerID string                 `json:"producer_id"`
	Timestamp  time.Time              `json:"timestamp"`
}

// WorkflowStateData 工作流状态数据
type WorkflowStateData struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	State     WorkflowState          `json:"state"`
	Data      map[string]interface{} `json:"data"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// EventBus 事件总线接口
type EventBus interface {
	// Publish 发布事件
	Publish(ctx context.Context, workflowType, workflowID string, event *WorkflowEvent) error

	// Subscribe 订阅事件（从指定序列号开始）
	Subscribe(ctx context.Context, workflowType, workflowID string, fromSeq int64) (<-chan *WorkflowEvent, error)

	// GetState 获取当前状态
	GetState(ctx context.Context, workflowType, workflowID string) (*WorkflowStateData, error)

	// SetState 更新状态
	SetState(ctx context.Context, workflowType, workflowID string, state *WorkflowStateData) error

	// WatchState Watch 状态变更
	WatchState(ctx context.Context, workflowType, workflowID string) (<-chan *WorkflowStateData, error)

	// WatchAllStates Watch 所有状态变更（用于 API Server）
	WatchAllStates(ctx context.Context, workflowType string) (<-chan *WorkflowStateData, error)

	// GetEvents 获取事件列表
	GetEvents(ctx context.Context, workflowType, workflowID string, fromSeq, limit int64) ([]*WorkflowEvent, error)

	// GetNextSeq 获取下一个序列号
	GetNextSeq(ctx context.Context, workflowType, workflowID string) (int64, error)
}

// EtcdEventBus etcd 事件总线实现
type EtcdEventBus struct {
	client *clientv3.Client
	prefix string
}

// NewEtcdEventBus 创建 etcd 事件总线
func NewEtcdEventBus(client *clientv3.Client, prefix string) *EtcdEventBus {
	if prefix == "" {
		prefix = "/agents"
	}
	return &EtcdEventBus{
		client: client,
		prefix: prefix,
	}
}

// NewEtcdEventBusFromStore 从 EtcdStore 创建事件总线
func NewEtcdEventBusFromStore(store *EtcdStore) *EtcdEventBus {
	return &EtcdEventBus{
		client: store.client,
		prefix: store.prefix,
	}
}

// eventKey 生成事件键
func (e *EtcdEventBus) eventKey(workflowType, workflowID string, seq int64) string {
	return fmt.Sprintf("%s/events/%s/%s/%06d", e.prefix, workflowType, workflowID, seq)
}

// eventPrefix 生成事件前缀
func (e *EtcdEventBus) eventPrefix(workflowType, workflowID string) string {
	return fmt.Sprintf("%s/events/%s/%s/", e.prefix, workflowType, workflowID)
}

// stateKey 生成状态键
func (e *EtcdEventBus) stateKey(workflowType, workflowID string) string {
	return fmt.Sprintf("%s/state/%s/%s", e.prefix, workflowType, workflowID)
}

// statePrefix 生成状态前缀
func (e *EtcdEventBus) statePrefix(workflowType string) string {
	return fmt.Sprintf("%s/state/%s/", e.prefix, workflowType)
}

// Publish 发布事件
func (e *EtcdEventBus) Publish(ctx context.Context, workflowType, workflowID string, event *WorkflowEvent) error {
	// 1. 获取下一个序列号
	seq, err := e.GetNextSeq(ctx, workflowType, workflowID)
	if err != nil {
		return fmt.Errorf("failed to get next seq: %w", err)
	}
	event.Seq = seq
	event.WorkflowID = workflowID

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// 2. 序列化事件
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// 3. 写入事件（带 24 小时 TTL）
	eventKey := e.eventKey(workflowType, workflowID, seq)
	lease, err := e.client.Grant(ctx, 24*60*60) // 24 小时
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

// PublishWithState 发布事件并更新状态（原子操作）
func (e *EtcdEventBus) PublishWithState(ctx context.Context, workflowType, workflowID string, event *WorkflowEvent, state *WorkflowStateData) error {
	// 1. 获取下一个序列号
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

	// 2. 序列化
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	stateData, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// 3. 事务写入
	eventKey := e.eventKey(workflowType, workflowID, seq)
	stateKey := e.stateKey(workflowType, workflowID)

	// 事件带 24 小时 TTL
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
func (e *EtcdEventBus) Subscribe(ctx context.Context, workflowType, workflowID string, fromSeq int64) (<-chan *WorkflowEvent, error) {
	eventCh := make(chan *WorkflowEvent, 100)

	go func() {
		defer close(eventCh)

		// 1. 先获取历史事件
		prefix := e.eventPrefix(workflowType, workflowID)
		resp, err := e.client.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
		if err != nil {
			log.Printf("[EventBus] Failed to get history events: %v", err)
			return
		}

		for _, kv := range resp.Kvs {
			var event WorkflowEvent
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

		// 2. Watch 新事件
		watchCh := e.client.Watch(ctx, prefix, clientv3.WithPrefix(), clientv3.WithRev(resp.Header.Revision+1))
		for watchResp := range watchCh {
			for _, ev := range watchResp.Events {
				if ev.Type != clientv3.EventTypePut {
					continue
				}
				var event WorkflowEvent
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
func (e *EtcdEventBus) GetState(ctx context.Context, workflowType, workflowID string) (*WorkflowStateData, error) {
	key := e.stateKey(workflowType, workflowID)
	resp, err := e.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	var state WorkflowStateData
	if err := json.Unmarshal(resp.Kvs[0].Value, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// SetState 更新状态
func (e *EtcdEventBus) SetState(ctx context.Context, workflowType, workflowID string, state *WorkflowStateData) error {
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
func (e *EtcdEventBus) WatchState(ctx context.Context, workflowType, workflowID string) (<-chan *WorkflowStateData, error) {
	stateCh := make(chan *WorkflowStateData, 10)

	go func() {
		defer close(stateCh)

		key := e.stateKey(workflowType, workflowID)

		// 先获取当前状态
		state, err := e.GetState(ctx, workflowType, workflowID)
		if err == nil && state != nil {
			select {
			case stateCh <- state:
			case <-ctx.Done():
				return
			}
		}

		// Watch 变更
		watchCh := e.client.Watch(ctx, key)
		for watchResp := range watchCh {
			for _, ev := range watchResp.Events {
				if ev.Type == clientv3.EventTypeDelete {
					continue
				}
				var state WorkflowStateData
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
func (e *EtcdEventBus) WatchAllStates(ctx context.Context, workflowType string) (<-chan *WorkflowStateData, error) {
	stateCh := make(chan *WorkflowStateData, 100)

	go func() {
		defer close(stateCh)

		prefix := e.statePrefix(workflowType)
		watchCh := e.client.Watch(ctx, prefix, clientv3.WithPrefix())
		for watchResp := range watchCh {
			for _, ev := range watchResp.Events {
				if ev.Type == clientv3.EventTypeDelete {
					continue
				}
				var state WorkflowStateData
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
func (e *EtcdEventBus) GetEvents(ctx context.Context, workflowType, workflowID string, fromSeq, limit int64) ([]*WorkflowEvent, error) {
	prefix := e.eventPrefix(workflowType, workflowID)
	resp, err := e.client.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	var events []*WorkflowEvent
	for _, kv := range resp.Kvs {
		var event WorkflowEvent
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
func (e *EtcdEventBus) GetNextSeq(ctx context.Context, workflowType, workflowID string) (int64, error) {
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

	// 从 key 中解析序列号
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
func (e *EtcdEventBus) DeleteWorkflow(ctx context.Context, workflowType, workflowID string) error {
	// 删除事件
	eventPrefix := e.eventPrefix(workflowType, workflowID)
	_, err := e.client.Delete(ctx, eventPrefix, clientv3.WithPrefix())
	if err != nil {
		return fmt.Errorf("failed to delete events: %w", err)
	}

	// 删除状态
	stateKey := e.stateKey(workflowType, workflowID)
	_, err = e.client.Delete(ctx, stateKey)
	if err != nil {
		return fmt.Errorf("failed to delete state: %w", err)
	}

	log.Printf("[EventBus] Deleted workflow: %s/%s", workflowType, workflowID)
	return nil
}

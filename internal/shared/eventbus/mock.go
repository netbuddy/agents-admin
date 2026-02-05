// Package eventbus 事件总线 mock 实现
package eventbus

import (
	"context"
)

// ============================================================================
// NoOpEventBus - 空操作的 EventBus 实现（用于测试）
// ============================================================================

// NoOpEventBus 是一个不做任何操作的 EventBus 实现
type NoOpEventBus struct{}

// NewNoOpEventBus 创建 NoOpEventBus 实例
func NewNoOpEventBus() *NoOpEventBus {
	return &NoOpEventBus{}
}

// Close 关闭事件总线
func (e *NoOpEventBus) Close() error {
	return nil
}

// WorkflowEventBus 方法

func (e *NoOpEventBus) PublishEvent(ctx context.Context, wfType, wfID string, event *WorkflowEvent) error {
	return nil
}
func (e *NoOpEventBus) GetEvents(ctx context.Context, wfType, wfID string, fromID string, count int64) ([]*WorkflowEvent, error) {
	return []*WorkflowEvent{}, nil
}
func (e *NoOpEventBus) GetEventCount(ctx context.Context, wfType, wfID string) (int64, error) {
	return 0, nil
}
func (e *NoOpEventBus) SubscribeEvents(ctx context.Context, wfType, wfID string) (<-chan *WorkflowEvent, error) {
	ch := make(chan *WorkflowEvent)
	close(ch)
	return ch, nil
}
func (e *NoOpEventBus) DeleteEvents(ctx context.Context, wfType, wfID string) error {
	return nil
}

// RunEventBus 方法

func (e *NoOpEventBus) PublishRunEvent(ctx context.Context, runID string, event *RunEvent) error {
	return nil
}
func (e *NoOpEventBus) GetRunEvents(ctx context.Context, runID string, fromSeq int, count int64) ([]*RunEvent, error) {
	return []*RunEvent{}, nil
}
func (e *NoOpEventBus) GetRunEventCount(ctx context.Context, runID string) (int64, error) {
	return 0, nil
}
func (e *NoOpEventBus) SubscribeRunEvents(ctx context.Context, runID string) (<-chan *RunEvent, error) {
	ch := make(chan *RunEvent)
	close(ch)
	return ch, nil
}
func (e *NoOpEventBus) DeleteRunEvents(ctx context.Context, runID string) error {
	return nil
}

// 确保 NoOpEventBus 实现了 EventBus 接口
var _ EventBus = (*NoOpEventBus)(nil)

// Package eventbus 事件总线抽象接口
//
// 提供事件的发布/订阅能力，当前由 Redis Streams/Pub-Sub 实现。
package eventbus

import (
	"context"
)

// ============================================================================
// 事件总线接口定义
// ============================================================================

// WorkflowEventBus 工作流事件总线接口
type WorkflowEventBus interface {
	PublishEvent(ctx context.Context, wfType, wfID string, event *WorkflowEvent) error
	GetEvents(ctx context.Context, wfType, wfID string, fromID string, count int64) ([]*WorkflowEvent, error)
	GetEventCount(ctx context.Context, wfType, wfID string) (int64, error)
	SubscribeEvents(ctx context.Context, wfType, wfID string) (<-chan *WorkflowEvent, error)
	DeleteEvents(ctx context.Context, wfType, wfID string) error
}

// RunEventBus Run 事件总线接口
type RunEventBus interface {
	PublishRunEvent(ctx context.Context, runID string, event *RunEvent) error
	GetRunEvents(ctx context.Context, runID string, fromSeq int, count int64) ([]*RunEvent, error)
	GetRunEventCount(ctx context.Context, runID string) (int64, error)
	SubscribeRunEvents(ctx context.Context, runID string) (<-chan *RunEvent, error)
	DeleteRunEvents(ctx context.Context, runID string) error
}

// ============================================================================
// 组合接口
// ============================================================================

// EventBus 事件总线组合接口
type EventBus interface {
	WorkflowEventBus
	RunEventBus
	Close() error
}

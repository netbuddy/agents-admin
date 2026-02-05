// Package queue 消息队列抽象接口
//
// 提供任务分发和消费的队列能力，当前由 Redis Streams 实现。
package queue

import (
	"context"
	"time"
)

// ============================================================================
// 队列接口定义
// ============================================================================

// SchedulerQueue 调度队列接口
type SchedulerQueue interface {
	// ScheduleRun 将 Run 加入调度队列（等待分配节点）
	ScheduleRun(ctx context.Context, runID, taskID string) (string, error)
	CreateSchedulerConsumerGroup(ctx context.Context) error
	ConsumeSchedulerRuns(ctx context.Context, consumerID string, count int64, blockTimeout time.Duration) ([]*SchedulerMessage, error)
	AckSchedulerRun(ctx context.Context, messageID string) error
	GetSchedulerQueueLength(ctx context.Context) (int64, error)
	GetSchedulerPendingCount(ctx context.Context) (int64, error)
}

// NodeRunQueue 节点 Run 队列接口
type NodeRunQueue interface {
	// PublishRunToNode 将 Run 分配给指定节点
	PublishRunToNode(ctx context.Context, nodeID, runID, taskID string) (string, error)
	CreateNodeConsumerGroup(ctx context.Context, nodeID string) error
	ConsumeNodeRuns(ctx context.Context, nodeID, consumerID string, count int64, blockTimeout time.Duration) ([]*NodeRunMessage, error)
	AckNodeRun(ctx context.Context, nodeID, messageID string) error
	GetNodeRunsQueueLength(ctx context.Context, nodeID string) (int64, error)
	GetNodeRunsPendingCount(ctx context.Context, nodeID string) (int64, error)
}

// NodeTaskQueue 别名，向后兼容
// Deprecated: 使用 NodeRunQueue
type NodeTaskQueue = NodeRunQueue

// ============================================================================
// 组合接口
// ============================================================================

// Queue 消息队列组合接口
type Queue interface {
	SchedulerQueue
	NodeRunQueue
	Close() error
}

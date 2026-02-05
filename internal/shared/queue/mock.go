// Package queue 消息队列 mock 实现
package queue

import (
	"context"
	"time"
)

// ============================================================================
// NoOpQueue - 空操作的 Queue 实现（用于测试）
// ============================================================================

// NoOpQueue 是一个不做任何操作的 Queue 实现
type NoOpQueue struct{}

// NewNoOpQueue 创建 NoOpQueue 实例
func NewNoOpQueue() *NoOpQueue {
	return &NoOpQueue{}
}

// Close 关闭队列
func (q *NoOpQueue) Close() error {
	return nil
}

// SchedulerQueue 方法

func (q *NoOpQueue) ScheduleRun(ctx context.Context, runID, taskID string) (string, error) {
	return "", nil
}
func (q *NoOpQueue) CreateSchedulerConsumerGroup(ctx context.Context) error {
	return nil
}
func (q *NoOpQueue) ConsumeSchedulerRuns(ctx context.Context, consumerID string, count int64, blockTimeout time.Duration) ([]*SchedulerMessage, error) {
	return []*SchedulerMessage{}, nil
}
func (q *NoOpQueue) AckSchedulerRun(ctx context.Context, messageID string) error {
	return nil
}
func (q *NoOpQueue) GetSchedulerQueueLength(ctx context.Context) (int64, error) {
	return 0, nil
}
func (q *NoOpQueue) GetSchedulerPendingCount(ctx context.Context) (int64, error) {
	return 0, nil
}

// NodeRunQueue 方法

func (q *NoOpQueue) PublishRunToNode(ctx context.Context, nodeID, runID, taskID string) (string, error) {
	return "", nil
}
func (q *NoOpQueue) CreateNodeConsumerGroup(ctx context.Context, nodeID string) error {
	return nil
}
func (q *NoOpQueue) ConsumeNodeRuns(ctx context.Context, nodeID, consumerID string, count int64, blockTimeout time.Duration) ([]*NodeRunMessage, error) {
	return []*NodeRunMessage{}, nil
}
func (q *NoOpQueue) AckNodeRun(ctx context.Context, nodeID, messageID string) error {
	return nil
}
func (q *NoOpQueue) GetNodeRunsQueueLength(ctx context.Context, nodeID string) (int64, error) {
	return 0, nil
}
func (q *NoOpQueue) GetNodeRunsPendingCount(ctx context.Context, nodeID string) (int64, error) {
	return 0, nil
}

// 确保 NoOpQueue 实现了 Queue 接口
var _ Queue = (*NoOpQueue)(nil)

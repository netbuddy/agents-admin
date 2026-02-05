// Package queue 消息队列类型定义
package queue

import (
	"time"
)

// ============================================================================
// 消息类型
// ============================================================================

// SchedulerMessage 调度器消息
type SchedulerMessage struct {
	ID        string
	RunID     string
	TaskID    string
	CreatedAt time.Time
}

// NodeRunMessage 节点 Run 消息（原 NodeTaskMessage）
type NodeRunMessage struct {
	ID         string
	RunID      string
	TaskID     string
	AssignedAt time.Time
}

// NodeTaskMessage 别名，向后兼容
// Deprecated: 使用 NodeRunMessage
type NodeTaskMessage = NodeRunMessage

// ============================================================================
// Key 前缀和常量
// ============================================================================

const (
	// 调度器队列 - 存放待调度的 Run
	KeySchedulerRuns = "scheduler:runs"

	// 节点队列 - 存放分配给节点的 Run
	KeyNodeRuns       = "nodes:"
	KeyNodeRunsSuffix = ":runs"

	// 废弃常量，向后兼容
	// Deprecated: 使用 KeySchedulerRuns
	KeyTasksPending = KeySchedulerRuns
	// Deprecated: 使用 KeyNodeRuns
	KeyNodeTasks = KeyNodeRuns
	// Deprecated: 使用 KeyNodeRunsSuffix
	KeyNodeTasksSuffix = KeyNodeRunsSuffix

	// 消费者组
	SchedulerConsumerGroup   = "schedulers"
	NodeManagerConsumerGroup = "node_managers"
)

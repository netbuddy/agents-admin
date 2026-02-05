// Package storagetypes 定义存储层共享数据类型
//
// 独立包，避免循环导入
package storagetypes

import (
	"time"
)

// ============================================================================
// Redis 相关类型
// ============================================================================

// AuthSession 认证会话数据（存储在 Redis）
//
// Deprecated: AuthSession 已废弃，请使用 model.Operation 和 model.Action 替代。
// 新的认证流程使用 PostgreSQL 存储 Operation/Action，不再使用 Redis。
// 此类型保留仅为兼容旧代码，将在后续版本中移除。
type AuthSession struct {
	TaskID        string    `json:"task_id" redis:"task_id"`
	AccountID     string    `json:"account_id" redis:"account_id"`
	Method        string    `json:"method" redis:"method"`
	NodeID        string    `json:"node_id" redis:"node_id"`
	Status        string    `json:"status" redis:"status"`
	ProxyID       string    `json:"proxy_id,omitempty" redis:"proxy_id"`
	TerminalPort  int       `json:"terminal_port,omitempty" redis:"terminal_port"`
	TerminalURL   string    `json:"terminal_url,omitempty" redis:"terminal_url"`
	ContainerName string    `json:"container_name,omitempty" redis:"container_name"`
	OAuthURL      string    `json:"oauth_url,omitempty" redis:"oauth_url"`
	UserCode      string    `json:"user_code,omitempty" redis:"user_code"`
	Message       string    `json:"message,omitempty" redis:"message"`
	Executed      bool      `json:"executed" redis:"executed"`
	ExecutedAt    time.Time `json:"executed_at,omitempty" redis:"executed_at"`
	CreatedAt     time.Time `json:"created_at" redis:"created_at"`
	ExpiresAt     time.Time `json:"expires_at" redis:"expires_at"`
}

// RedisWorkflowState 工作流运行时状态（存储在 Redis）
type RedisWorkflowState struct {
	State       string `json:"state" redis:"state"`
	Progress    int    `json:"progress" redis:"progress"`
	CurrentStep string `json:"current_step" redis:"current_step"`
	Error       string `json:"error,omitempty" redis:"error"`
}

// NodeStatus 节点状态（Redis 心跳）
type NodeStatus struct {
	Status    string         `json:"status"`
	Capacity  map[string]int `json:"capacity"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// RedisWorkflowEvent 工作流事件（Redis Streams）
type RedisWorkflowEvent struct {
	ID        string                 `json:"id"`
	Seq       int                    `json:"seq"`
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// RunEvent Run 执行事件（Redis Streams）
type RunEvent struct {
	ID        string                 `json:"id"`
	RunID     string                 `json:"run_id"`
	Seq       int                    `json:"seq"`
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
	Raw       string                 `json:"raw,omitempty"`
}

// SchedulerMessage 调度器消息（Redis Streams）
type SchedulerMessage struct {
	ID        string
	RunID     string
	TaskID    string
	CreatedAt time.Time
}

// NodeTaskMessage 节点任务消息（Redis Streams）
type NodeTaskMessage struct {
	ID         string
	RunID      string
	TaskID     string
	AssignedAt time.Time
}

// ============================================================================
// etcd 相关类型
// ============================================================================

// EtcdHeartbeat 节点心跳数据（存储在 etcd）
type EtcdHeartbeat struct {
	NodeID        string                 `json:"node_id"`
	Status        string                 `json:"status"`
	LastHeartbeat time.Time              `json:"last_heartbeat"`
	Capacity      map[string]interface{} `json:"capacity"`
}

// WorkflowState 工作流状态枚举
type WorkflowState string

const (
	WorkflowStatePending   WorkflowState = "pending"
	WorkflowStateRunning   WorkflowState = "running"
	WorkflowStateWaiting   WorkflowState = "waiting"
	WorkflowStateCompleted WorkflowState = "completed"
	WorkflowStateFailed    WorkflowState = "failed"
	WorkflowStateCancelled WorkflowState = "cancelled"
)

// WorkflowEvent 工作流事件（etcd EventBus）
type WorkflowEvent struct {
	ID         string                 `json:"id"`
	WorkflowID string                 `json:"workflow_id"`
	Type       string                 `json:"type"`
	Seq        int64                  `json:"seq"`
	Data       map[string]interface{} `json:"data"`
	ProducerID string                 `json:"producer_id"`
	Timestamp  time.Time              `json:"timestamp"`
}

// WorkflowStateData 工作流状态数据（etcd EventBus）
type WorkflowStateData struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	State     WorkflowState          `json:"state"`
	Data      map[string]interface{} `json:"data"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// ============================================================================
// Redis Key 前缀和 TTL 常量
// ============================================================================

const (
	// Key 前缀
	KeyAuthSession          = "auth_session:"
	KeyAuthSessionByAccount = "auth_session_idx:"
	KeyWorkflowState        = "workflow_state:"
	KeyNodeHeartbeat        = "node_heartbeat:"
	KeyOnlineNodes          = "online_nodes"
	KeyWorkflowEvents       = "workflow_events:"
	KeyRunEvents            = "run_events:"
	// 调度器队列 - 存放待调度的 Run
	KeySchedulerRuns = "scheduler:runs"
	// 节点队列 - 存放分配给节点的 Run
	KeyNodeRuns       = "nodes:"
	KeyNodeRunsSuffix = ":runs"

	// 废弃常量，向后兼容
	KeyTasksPending    = KeySchedulerRuns
	KeyNodeTasks       = KeyNodeRuns
	KeyNodeTasksSuffix = KeyNodeRunsSuffix

	// TTL 常量
	TTLAuthSession   = 10 * time.Minute
	TTLWorkflowState = 1 * time.Hour
	TTLNodeHeartbeat = 30 * time.Second
	MaxStreamLength  = 1000

	// 消费者组
	SchedulerConsumerGroup   = "schedulers"
	NodeManagerConsumerGroup = "node_managers"
)

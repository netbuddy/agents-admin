// Package cache 缓存层类型定义
package cache

import (
	"time"
)

// ============================================================================
// 缓存数据类型
// ============================================================================

// AuthSession 认证会话数据
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

// WorkflowState 工作流运行时状态
type WorkflowState struct {
	State       string `json:"state" redis:"state"`
	Progress    int    `json:"progress" redis:"progress"`
	CurrentStep string `json:"current_step" redis:"current_step"`
	Error       string `json:"error,omitempty" redis:"error"`
}

// NodeStatus 节点状态
type NodeStatus struct {
	Status    string         `json:"status"`
	Capacity  map[string]int `json:"capacity"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// ============================================================================
// Key 前缀和 TTL 常量
// ============================================================================

const (
	// Key 前缀
	KeyAuthSession          = "auth_session:"
	KeyAuthSessionByAccount = "auth_session_idx:"
	KeyWorkflowState        = "workflow_state:"
	KeyNodeHeartbeat        = "node_heartbeat:"
	KeyOnlineNodes          = "online_nodes"

	// TTL 常量
	TTLAuthSession   = 10 * time.Minute
	TTLWorkflowState = 1 * time.Hour
	TTLNodeHeartbeat = 30 * time.Second
)

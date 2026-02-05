// Package cache 缓存层抽象接口
//
// 提供临时状态和缓存的存取能力，当前由 Redis 实现。
package cache

import (
	"context"
)

// ============================================================================
// 缓存接口定义
// ============================================================================

// AuthSessionCache 认证会话缓存接口
//
// Deprecated: 此接口已废弃，新认证流程使用 Operation/Action 模型。
// 保留仅为兼容旧代码。
type AuthSessionCache interface {
	CreateAuthSession(ctx context.Context, session *AuthSession) error
	GetAuthSession(ctx context.Context, taskID string) (*AuthSession, error)
	GetAuthSessionByAccountID(ctx context.Context, accountID string) (*AuthSession, error)
	UpdateAuthSession(ctx context.Context, taskID string, updates map[string]interface{}) error
	DeleteAuthSession(ctx context.Context, taskID string) error
	ListAuthSessions(ctx context.Context) ([]*AuthSession, error)
	ListAuthSessionsByNode(ctx context.Context, nodeID string) ([]*AuthSession, error)
}

// WorkflowStateCache 工作流状态缓存接口
type WorkflowStateCache interface {
	SetWorkflowState(ctx context.Context, wfType, wfID string, state *WorkflowState) error
	GetWorkflowState(ctx context.Context, wfType, wfID string) (*WorkflowState, error)
	DeleteWorkflowState(ctx context.Context, wfType, wfID string) error
}

// NodeHeartbeatCache 节点心跳缓存接口
type NodeHeartbeatCache interface {
	UpdateNodeHeartbeat(ctx context.Context, nodeID string, status *NodeStatus) error
	GetNodeHeartbeat(ctx context.Context, nodeID string) (*NodeStatus, error)
	DeleteNodeHeartbeat(ctx context.Context, nodeID string) error
	ListOnlineNodes(ctx context.Context) ([]string, error)
}

// ============================================================================
// 组合接口
// ============================================================================

// Cache 缓存组合接口
type Cache interface {
	AuthSessionCache
	WorkflowStateCache
	NodeHeartbeatCache
	Close() error
}

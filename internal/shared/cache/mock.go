// Package cache 缓存层 mock 实现
package cache

import (
	"context"
)

// ============================================================================
// NoOpCache - 空操作的 Cache 实现（用于测试）
// ============================================================================

// NoOpCache 是一个不做任何操作的 Cache 实现
type NoOpCache struct{}

// NewNoOpCache 创建 NoOpCache 实例
func NewNoOpCache() *NoOpCache {
	return &NoOpCache{}
}

// Close 关闭缓存
func (c *NoOpCache) Close() error {
	return nil
}

// AuthSessionCache 方法

func (c *NoOpCache) CreateAuthSession(ctx context.Context, session *AuthSession) error {
	return nil
}
func (c *NoOpCache) GetAuthSession(ctx context.Context, taskID string) (*AuthSession, error) {
	return nil, nil
}
func (c *NoOpCache) GetAuthSessionByAccountID(ctx context.Context, accountID string) (*AuthSession, error) {
	return nil, nil
}
func (c *NoOpCache) UpdateAuthSession(ctx context.Context, taskID string, updates map[string]interface{}) error {
	return nil
}
func (c *NoOpCache) DeleteAuthSession(ctx context.Context, taskID string) error {
	return nil
}
func (c *NoOpCache) ListAuthSessions(ctx context.Context) ([]*AuthSession, error) {
	return []*AuthSession{}, nil
}
func (c *NoOpCache) ListAuthSessionsByNode(ctx context.Context, nodeID string) ([]*AuthSession, error) {
	return []*AuthSession{}, nil
}

// WorkflowStateCache 方法

func (c *NoOpCache) SetWorkflowState(ctx context.Context, wfType, wfID string, state *WorkflowState) error {
	return nil
}
func (c *NoOpCache) GetWorkflowState(ctx context.Context, wfType, wfID string) (*WorkflowState, error) {
	return nil, nil
}
func (c *NoOpCache) DeleteWorkflowState(ctx context.Context, wfType, wfID string) error {
	return nil
}

// NodeHeartbeatCache 方法

func (c *NoOpCache) UpdateNodeHeartbeat(ctx context.Context, nodeID string, status *NodeStatus) error {
	return nil
}
func (c *NoOpCache) GetNodeHeartbeat(ctx context.Context, nodeID string) (*NodeStatus, error) {
	return nil, nil
}
func (c *NoOpCache) DeleteNodeHeartbeat(ctx context.Context, nodeID string) error {
	return nil
}
func (c *NoOpCache) ListOnlineNodes(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

// 确保 NoOpCache 实现了 Cache 接口
var _ Cache = (*NoOpCache)(nil)

// Package redis AuthSession 相关操作
package redis

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"agents-admin/internal/shared/storagetypes"
)

// CreateAuthSession 创建认证会话
func (s *Store) CreateAuthSession(ctx context.Context, session *storagetypes.AuthSession) error {
	key := storagetypes.KeyAuthSession + session.TaskID

	data := map[string]interface{}{
		"task_id":        session.TaskID,
		"account_id":     session.AccountID,
		"method":         session.Method,
		"node_id":        session.NodeID,
		"status":         session.Status,
		"proxy_id":       session.ProxyID,
		"terminal_port":  session.TerminalPort,
		"terminal_url":   session.TerminalURL,
		"container_name": session.ContainerName,
		"oauth_url":      session.OAuthURL,
		"user_code":      session.UserCode,
		"message":        session.Message,
		"executed":       session.Executed,
		"executed_at":    session.ExecutedAt.Format(time.RFC3339),
		"created_at":     session.CreatedAt.Format(time.RFC3339),
		"expires_at":     session.ExpiresAt.Format(time.RFC3339),
	}

	pipe := s.client.Pipeline()
	pipe.HSet(ctx, key, data)
	pipe.Expire(ctx, key, storagetypes.TTLAuthSession)

	idxKey := storagetypes.KeyAuthSessionByAccount + session.AccountID
	pipe.Set(ctx, idxKey, session.TaskID, storagetypes.TTLAuthSession)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create auth session: %w", err)
	}

	log.Printf("[Redis] Created auth session: %s (account=%s, status=%s)", session.TaskID, session.AccountID, session.Status)
	return nil
}

// GetAuthSession 获取认证会话
func (s *Store) GetAuthSession(ctx context.Context, taskID string) (*storagetypes.AuthSession, error) {
	key := storagetypes.KeyAuthSession + taskID

	result, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get auth session: %w", err)
	}

	if len(result) == 0 {
		return nil, nil
	}

	return parseAuthSession(result)
}

// GetAuthSessionByAccountID 根据账号 ID 获取最新的认证会话
func (s *Store) GetAuthSessionByAccountID(ctx context.Context, accountID string) (*storagetypes.AuthSession, error) {
	idxKey := storagetypes.KeyAuthSessionByAccount + accountID

	taskID, err := s.client.Get(ctx, idxKey).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get auth session index: %w", err)
	}

	return s.GetAuthSession(ctx, taskID)
}

// UpdateAuthSession 更新认证会话
func (s *Store) UpdateAuthSession(ctx context.Context, taskID string, updates map[string]interface{}) error {
	key := storagetypes.KeyAuthSession + taskID

	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to check auth session existence: %w", err)
	}
	if exists == 0 {
		return fmt.Errorf("auth session not found: %s", taskID)
	}

	if err := s.client.HSet(ctx, key, updates).Err(); err != nil {
		return fmt.Errorf("failed to update auth session: %w", err)
	}

	log.Printf("[Redis] Updated auth session: %s, updates=%v", taskID, updates)
	return nil
}

// DeleteAuthSession 删除认证会话
func (s *Store) DeleteAuthSession(ctx context.Context, taskID string) error {
	key := storagetypes.KeyAuthSession + taskID

	accountID, _ := s.client.HGet(ctx, key, "account_id").Result()

	pipe := s.client.Pipeline()
	pipe.Del(ctx, key)
	if accountID != "" {
		pipe.Del(ctx, storagetypes.KeyAuthSessionByAccount+accountID)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// ListAuthSessions 列出所有认证会话
func (s *Store) ListAuthSessions(ctx context.Context) ([]*storagetypes.AuthSession, error) {
	pattern := storagetypes.KeyAuthSession + "*"
	keys, err := s.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list auth session keys: %w", err)
	}

	var sessions []*storagetypes.AuthSession
	for _, key := range keys {
		result, err := s.client.HGetAll(ctx, key).Result()
		if err != nil {
			continue
		}
		session, err := parseAuthSession(result)
		if err != nil {
			continue
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// ListAuthSessionsByNode 列出指定节点的认证会话
func (s *Store) ListAuthSessionsByNode(ctx context.Context, nodeID string) ([]*storagetypes.AuthSession, error) {
	allSessions, err := s.ListAuthSessions(ctx)
	if err != nil {
		return nil, err
	}

	var nodeSessions []*storagetypes.AuthSession
	for _, session := range allSessions {
		if session.NodeID == nodeID {
			nodeSessions = append(nodeSessions, session)
		}
	}

	return nodeSessions, nil
}

func parseAuthSession(data map[string]string) (*storagetypes.AuthSession, error) {
	session := &storagetypes.AuthSession{
		TaskID:        data["task_id"],
		AccountID:     data["account_id"],
		Method:        data["method"],
		NodeID:        data["node_id"],
		Status:        data["status"],
		ProxyID:       data["proxy_id"],
		TerminalURL:   data["terminal_url"],
		ContainerName: data["container_name"],
		OAuthURL:      data["oauth_url"],
		UserCode:      data["user_code"],
		Message:       data["message"],
	}

	if port, err := strconv.Atoi(data["terminal_port"]); err == nil {
		session.TerminalPort = port
	}

	if executed, ok := data["executed"]; ok {
		session.Executed = executed == "1" || executed == "true"
	}

	if t, err := time.Parse(time.RFC3339, data["executed_at"]); err == nil {
		session.ExecutedAt = t
	}

	if t, err := time.Parse(time.RFC3339, data["created_at"]); err == nil {
		session.CreatedAt = t
	}

	if t, err := time.Parse(time.RFC3339, data["expires_at"]); err == nil {
		session.ExpiresAt = t
	}

	return session, nil
}

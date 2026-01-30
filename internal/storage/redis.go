// Package storage 提供数据存储层
package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore Redis 存储层，用于临时状态、会话和事件流
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore 创建 Redis 存储实例
func NewRedisStore(addr, password string, db int) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Printf("[Redis] Connected to %s", addr)
	return &RedisStore{client: client}, nil
}

// Close 关闭 Redis 连接
func (s *RedisStore) Close() error {
	return s.client.Close()
}

// Client 返回底层 Redis 客户端
func (s *RedisStore) Client() *redis.Client {
	return s.client
}

// === Key 前缀常量 ===

const (
	// 认证会话 Hash
	keyAuthSession = "auth_session:"
	// 账号到会话的索引
	keyAuthSessionByAccount = "auth_session_idx:"
	// 工作流状态 Hash
	keyWorkflowState = "workflow_state:"
	// 节点心跳 String
	keyNodeHeartbeat = "node_heartbeat:"
	// 在线节点集合
	keyOnlineNodes = "online_nodes"
	// 工作流事件 Stream
	keyWorkflowEvents = "workflow_events:"
)

// === TTL 常量 ===

const (
	// 认证会话 TTL: 10 分钟
	ttlAuthSession = 10 * time.Minute
	// 工作流状态 TTL: 1 小时
	ttlWorkflowState = 1 * time.Hour
	// 节点心跳 TTL: 30 秒
	ttlNodeHeartbeat = 30 * time.Second
	// 事件流最大长度
	maxStreamLength = 1000
)

// === 认证会话 ===

// AuthSession 认证会话数据
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
	Executed      bool      `json:"executed" redis:"executed"`             // 是否已执行
	ExecutedAt    time.Time `json:"executed_at,omitempty" redis:"executed_at"` // 执行时间
	CreatedAt     time.Time `json:"created_at" redis:"created_at"`
	ExpiresAt     time.Time `json:"expires_at" redis:"expires_at"`
}

// CreateAuthSession 创建认证会话
func (s *RedisStore) CreateAuthSession(ctx context.Context, session *AuthSession) error {
	key := keyAuthSession + session.TaskID

	// 转换为 map
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
	pipe.Expire(ctx, key, ttlAuthSession)

	// 创建账号到会话的索引
	idxKey := keyAuthSessionByAccount + session.AccountID
	pipe.Set(ctx, idxKey, session.TaskID, ttlAuthSession)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create auth session: %w", err)
	}

	log.Printf("[Redis] Created auth session: %s (account=%s, status=%s)", session.TaskID, session.AccountID, session.Status)
	return nil
}

// GetAuthSession 获取认证会话
func (s *RedisStore) GetAuthSession(ctx context.Context, taskID string) (*AuthSession, error) {
	key := keyAuthSession + taskID

	result, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get auth session: %w", err)
	}

	if len(result) == 0 {
		return nil, nil // 不存在
	}

	return parseAuthSession(result)
}

// GetAuthSessionByAccountID 根据账号 ID 获取最新的认证会话
func (s *RedisStore) GetAuthSessionByAccountID(ctx context.Context, accountID string) (*AuthSession, error) {
	idxKey := keyAuthSessionByAccount + accountID

	taskID, err := s.client.Get(ctx, idxKey).Result()
	if err == redis.Nil {
		return nil, nil // 不存在
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get auth session index: %w", err)
	}

	return s.GetAuthSession(ctx, taskID)
}

// UpdateAuthSession 更新认证会话
func (s *RedisStore) UpdateAuthSession(ctx context.Context, taskID string, updates map[string]interface{}) error {
	key := keyAuthSession + taskID

	// 检查是否存在
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to check auth session existence: %w", err)
	}
	if exists == 0 {
		return fmt.Errorf("auth session not found: %s", taskID)
	}

	// 更新字段
	if err := s.client.HSet(ctx, key, updates).Err(); err != nil {
		return fmt.Errorf("failed to update auth session: %w", err)
	}

	log.Printf("[Redis] Updated auth session: %s, updates=%v", taskID, updates)
	return nil
}

// DeleteAuthSession 删除认证会话
func (s *RedisStore) DeleteAuthSession(ctx context.Context, taskID string) error {
	key := keyAuthSession + taskID

	// 先获取 accountID 以删除索引
	accountID, _ := s.client.HGet(ctx, key, "account_id").Result()

	pipe := s.client.Pipeline()
	pipe.Del(ctx, key)
	if accountID != "" {
		pipe.Del(ctx, keyAuthSessionByAccount+accountID)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// ListAuthSessions 列出所有认证会话（用于监控）
func (s *RedisStore) ListAuthSessions(ctx context.Context) ([]*AuthSession, error) {
	pattern := keyAuthSession + "*"
	keys, err := s.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list auth session keys: %w", err)
	}

	var sessions []*AuthSession
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
func (s *RedisStore) ListAuthSessionsByNode(ctx context.Context, nodeID string) ([]*AuthSession, error) {
	allSessions, err := s.ListAuthSessions(ctx)
	if err != nil {
		return nil, err
	}

	var nodeSessions []*AuthSession
	for _, session := range allSessions {
		if session.NodeID == nodeID {
			nodeSessions = append(nodeSessions, session)
		}
	}

	return nodeSessions, nil
}

func parseAuthSession(data map[string]string) (*AuthSession, error) {
	session := &AuthSession{
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

	// 解析 executed 字段
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

// === 工作流状态 ===

// RedisWorkflowState 工作流运行时状态
type RedisWorkflowState struct {
	State       string `json:"state" redis:"state"`
	Progress    int    `json:"progress" redis:"progress"`
	CurrentStep string `json:"current_step" redis:"current_step"`
	Error       string `json:"error,omitempty" redis:"error"`
}

// SetWorkflowState 设置工作流状态
func (s *RedisStore) SetWorkflowState(ctx context.Context, wfType, wfID string, state *RedisWorkflowState) error {
	key := fmt.Sprintf("%s%s:%s", keyWorkflowState, wfType, wfID)

	data := map[string]interface{}{
		"state":        state.State,
		"progress":     state.Progress,
		"current_step": state.CurrentStep,
		"error":        state.Error,
	}

	pipe := s.client.Pipeline()
	pipe.HSet(ctx, key, data)
	pipe.Expire(ctx, key, ttlWorkflowState)
	_, err := pipe.Exec(ctx)

	return err
}

// GetWorkflowState 获取工作流状态
func (s *RedisStore) GetWorkflowState(ctx context.Context, wfType, wfID string) (*RedisWorkflowState, error) {
	key := fmt.Sprintf("%s%s:%s", keyWorkflowState, wfType, wfID)

	result, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	state := &RedisWorkflowState{
		State:       result["state"],
		CurrentStep: result["current_step"],
		Error:       result["error"],
	}

	if progress, err := strconv.Atoi(result["progress"]); err == nil {
		state.Progress = progress
	}

	return state, nil
}

// DeleteWorkflowState 删除工作流状态
func (s *RedisStore) DeleteWorkflowState(ctx context.Context, wfType, wfID string) error {
	key := fmt.Sprintf("%s%s:%s", keyWorkflowState, wfType, wfID)
	return s.client.Del(ctx, key).Err()
}

// === 节点心跳 ===

// NodeStatus 节点状态
type NodeStatus struct {
	Status    string         `json:"status"`
	Capacity  map[string]int `json:"capacity"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// UpdateNodeHeartbeat 更新节点心跳
func (s *RedisStore) UpdateNodeHeartbeat(ctx context.Context, nodeID string, status *NodeStatus) error {
	key := keyNodeHeartbeat + nodeID

	status.UpdatedAt = time.Now()
	data, err := json.Marshal(status)
	if err != nil {
		return err
	}

	pipe := s.client.Pipeline()
	pipe.Set(ctx, key, data, ttlNodeHeartbeat)
	pipe.SAdd(ctx, keyOnlineNodes, nodeID)
	pipe.Expire(ctx, keyOnlineNodes, ttlNodeHeartbeat*2)
	_, err = pipe.Exec(ctx)

	return err
}

// GetNodeHeartbeat 获取节点心跳
func (s *RedisStore) GetNodeHeartbeat(ctx context.Context, nodeID string) (*NodeStatus, error) {
	key := keyNodeHeartbeat + nodeID

	data, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var status NodeStatus
	if err := json.Unmarshal([]byte(data), &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// ListOnlineNodes 列出在线节点
func (s *RedisStore) ListOnlineNodes(ctx context.Context) ([]string, error) {
	// 扫描所有心跳 key
	pattern := keyNodeHeartbeat + "*"
	keys, err := s.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	var nodeIDs []string
	for _, key := range keys {
		nodeID := key[len(keyNodeHeartbeat):]
		nodeIDs = append(nodeIDs, nodeID)
	}

	return nodeIDs, nil
}

// === 工作流事件 (Redis Streams) ===

// RedisWorkflowEvent 工作流事件
type RedisWorkflowEvent struct {
	ID        string                 `json:"id"`
	Seq       int                    `json:"seq"`
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// PublishEvent 发布工作流事件
func (s *RedisStore) PublishEvent(ctx context.Context, wfType, wfID string, event *RedisWorkflowEvent) error {
	key := fmt.Sprintf("%s%s:%s", keyWorkflowEvents, wfType, wfID)

	// 序列化 data
	dataJSON, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	// 使用 XADD 添加事件
	args := &redis.XAddArgs{
		Stream: key,
		MaxLen: maxStreamLength,
		Approx: true,
		Values: map[string]interface{}{
			"type":      event.Type,
			"timestamp": event.Timestamp.Format(time.RFC3339Nano),
			"data":      string(dataJSON),
		},
	}

	id, err := s.client.XAdd(ctx, args).Result()
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	log.Printf("[Redis] Published event: %s/%s seq=%s type=%s", wfType, wfID, id, event.Type)
	return nil
}

// GetEvents 获取工作流事件列表
func (s *RedisStore) GetEvents(ctx context.Context, wfType, wfID string, fromID string, count int64) ([]*RedisWorkflowEvent, error) {
	key := fmt.Sprintf("%s%s:%s", keyWorkflowEvents, wfType, wfID)

	if fromID == "" {
		fromID = "0"
	}

	// 使用 XRANGE 获取事件
	msgs, err := s.client.XRange(ctx, key, fromID, "+").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	var events []*RedisWorkflowEvent
	for i, msg := range msgs {
		event := &RedisWorkflowEvent{
			ID:   msg.ID,
			Seq:  i + 1,
			Type: msg.Values["type"].(string),
		}

		if ts, ok := msg.Values["timestamp"].(string); ok {
			if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
				event.Timestamp = t
			}
		}

		if dataStr, ok := msg.Values["data"].(string); ok {
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(dataStr), &data); err == nil {
				event.Data = data
			}
		}

		events = append(events, event)

		if count > 0 && int64(len(events)) >= count {
			break
		}
	}

	return events, nil
}

// GetEventCount 获取事件数量
func (s *RedisStore) GetEventCount(ctx context.Context, wfType, wfID string) (int64, error) {
	key := fmt.Sprintf("%s%s:%s", keyWorkflowEvents, wfType, wfID)
	return s.client.XLen(ctx, key).Result()
}

// SubscribeEvents 订阅工作流事件（实时推送）
func (s *RedisStore) SubscribeEvents(ctx context.Context, wfType, wfID string) (<-chan *RedisWorkflowEvent, error) {
	key := fmt.Sprintf("%s%s:%s", keyWorkflowEvents, wfType, wfID)
	ch := make(chan *RedisWorkflowEvent, 100)

	go func() {
		defer close(ch)
		lastID := "$" // 只获取新事件

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// 使用 XREAD BLOCK 等待新事件
			streams, err := s.client.XRead(ctx, &redis.XReadArgs{
				Streams: []string{key, lastID},
				Count:   10,
				Block:   5 * time.Second,
			}).Result()

			if err != nil {
				if err == redis.Nil {
					continue // 超时，继续等待
				}
				log.Printf("[Redis] Event subscription error: %v", err)
				return
			}

			for _, stream := range streams {
				for _, msg := range stream.Messages {
					event := &RedisWorkflowEvent{
						ID:   msg.ID,
						Type: msg.Values["type"].(string),
					}

					if ts, ok := msg.Values["timestamp"].(string); ok {
						if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
							event.Timestamp = t
						}
					}

					if dataStr, ok := msg.Values["data"].(string); ok {
						var data map[string]interface{}
						if err := json.Unmarshal([]byte(dataStr), &data); err == nil {
							event.Data = data
						}
					}

					select {
					case ch <- event:
						lastID = msg.ID
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return ch, nil
}

// DeleteEvents 删除工作流事件流
func (s *RedisStore) DeleteEvents(ctx context.Context, wfType, wfID string) error {
	key := fmt.Sprintf("%s%s:%s", keyWorkflowEvents, wfType, wfID)
	return s.client.Del(ctx, key).Err()
}

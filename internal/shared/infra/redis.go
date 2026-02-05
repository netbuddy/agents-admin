// Package infra Redis 基础设施初始化
package infra

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"agents-admin/internal/shared/cache"
	cacheredis "agents-admin/internal/shared/cache/redis"
	"agents-admin/internal/shared/eventbus"
	eventbusredis "agents-admin/internal/shared/eventbus/redis"
	"agents-admin/internal/shared/queue"
	queueredis "agents-admin/internal/shared/queue/redis"
	"agents-admin/internal/shared/storage"
)

// RedisInfra Redis 基础设施（实现 storage.CacheStore 接口）
//
// 组合 Cache、EventBus、Queue 实现完整的 CacheStore 接口
type RedisInfra struct {
	// 组件（显式命名避免冲突）
	cacheStore    *cacheredis.Store
	eventBusStore *eventbusredis.Store
	queueStore    *queueredis.Store

	// 底层连接
	client *redis.Client
}

// NewRedisInfra 从 URL 创建 Redis 基础设施
func NewRedisInfra(redisURL string) (*RedisInfra, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Printf("[Redis/Infra] Connected to %s", opts.Addr)

	return &RedisInfra{
		client:        client,
		cacheStore:    cacheredis.NewStoreFromClient(client),
		eventBusStore: eventbusredis.NewStoreFromClient(client),
		queueStore:    queueredis.NewStoreFromClient(client),
	}, nil
}

// NewRedisInfraFromAddr 从地址创建 Redis 基础设施
func NewRedisInfraFromAddr(addr, password string, db int) (*RedisInfra, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Printf("[Redis/Infra] Connected to %s", addr)

	return &RedisInfra{
		client:        client,
		cacheStore:    cacheredis.NewStoreFromClient(client),
		eventBusStore: eventbusredis.NewStoreFromClient(client),
		queueStore:    queueredis.NewStoreFromClient(client),
	}, nil
}

// Cache 返回缓存组件接口
func (r *RedisInfra) Cache() cache.Cache {
	return r.cacheStore
}

// EventBus 返回事件总线组件接口
func (r *RedisInfra) EventBus() eventbus.EventBus {
	return r.eventBusStore
}

// Queue 返回消息队列组件接口
func (r *RedisInfra) Queue() queue.Queue {
	return r.queueStore
}

// Client 返回底层 Redis 客户端
func (r *RedisInfra) Client() *redis.Client {
	return r.client
}

// Close 关闭 Redis 连接
func (r *RedisInfra) Close() error {
	return r.client.Close()
}

// ============================================================================
// cache.Cache 接口委托实现
// ============================================================================

func (r *RedisInfra) CreateAuthSession(ctx context.Context, session *cache.AuthSession) error {
	return r.cacheStore.CreateAuthSession(ctx, session)
}
func (r *RedisInfra) GetAuthSession(ctx context.Context, taskID string) (*cache.AuthSession, error) {
	return r.cacheStore.GetAuthSession(ctx, taskID)
}
func (r *RedisInfra) GetAuthSessionByAccountID(ctx context.Context, accountID string) (*cache.AuthSession, error) {
	return r.cacheStore.GetAuthSessionByAccountID(ctx, accountID)
}
func (r *RedisInfra) UpdateAuthSession(ctx context.Context, taskID string, updates map[string]interface{}) error {
	return r.cacheStore.UpdateAuthSession(ctx, taskID, updates)
}
func (r *RedisInfra) DeleteAuthSession(ctx context.Context, taskID string) error {
	return r.cacheStore.DeleteAuthSession(ctx, taskID)
}
func (r *RedisInfra) ListAuthSessions(ctx context.Context) ([]*cache.AuthSession, error) {
	return r.cacheStore.ListAuthSessions(ctx)
}
func (r *RedisInfra) ListAuthSessionsByNode(ctx context.Context, nodeID string) ([]*cache.AuthSession, error) {
	return r.cacheStore.ListAuthSessionsByNode(ctx, nodeID)
}
func (r *RedisInfra) SetWorkflowState(ctx context.Context, wfType, wfID string, state *cache.WorkflowState) error {
	return r.cacheStore.SetWorkflowState(ctx, wfType, wfID, state)
}
func (r *RedisInfra) GetWorkflowState(ctx context.Context, wfType, wfID string) (*cache.WorkflowState, error) {
	return r.cacheStore.GetWorkflowState(ctx, wfType, wfID)
}
func (r *RedisInfra) DeleteWorkflowState(ctx context.Context, wfType, wfID string) error {
	return r.cacheStore.DeleteWorkflowState(ctx, wfType, wfID)
}
func (r *RedisInfra) UpdateNodeHeartbeat(ctx context.Context, nodeID string, status *cache.NodeStatus) error {
	return r.cacheStore.UpdateNodeHeartbeat(ctx, nodeID, status)
}
func (r *RedisInfra) GetNodeHeartbeat(ctx context.Context, nodeID string) (*cache.NodeStatus, error) {
	return r.cacheStore.GetNodeHeartbeat(ctx, nodeID)
}
func (r *RedisInfra) DeleteNodeHeartbeat(ctx context.Context, nodeID string) error {
	return r.cacheStore.DeleteNodeHeartbeat(ctx, nodeID)
}
func (r *RedisInfra) ListOnlineNodes(ctx context.Context) ([]string, error) {
	return r.cacheStore.ListOnlineNodes(ctx)
}

// ============================================================================
// eventbus.EventBus 接口委托实现
// ============================================================================

func (r *RedisInfra) PublishEvent(ctx context.Context, wfType, wfID string, event *eventbus.WorkflowEvent) error {
	return r.eventBusStore.PublishEvent(ctx, wfType, wfID, event)
}
func (r *RedisInfra) GetEvents(ctx context.Context, wfType, wfID string, fromID string, count int64) ([]*eventbus.WorkflowEvent, error) {
	return r.eventBusStore.GetEvents(ctx, wfType, wfID, fromID, count)
}
func (r *RedisInfra) GetEventCount(ctx context.Context, wfType, wfID string) (int64, error) {
	return r.eventBusStore.GetEventCount(ctx, wfType, wfID)
}
func (r *RedisInfra) SubscribeEvents(ctx context.Context, wfType, wfID string) (<-chan *eventbus.WorkflowEvent, error) {
	return r.eventBusStore.SubscribeEvents(ctx, wfType, wfID)
}
func (r *RedisInfra) DeleteEvents(ctx context.Context, wfType, wfID string) error {
	return r.eventBusStore.DeleteEvents(ctx, wfType, wfID)
}
func (r *RedisInfra) PublishRunEvent(ctx context.Context, runID string, event *eventbus.RunEvent) error {
	return r.eventBusStore.PublishRunEvent(ctx, runID, event)
}
func (r *RedisInfra) GetRunEvents(ctx context.Context, runID string, fromSeq int, count int64) ([]*eventbus.RunEvent, error) {
	return r.eventBusStore.GetRunEvents(ctx, runID, fromSeq, count)
}
func (r *RedisInfra) GetRunEventCount(ctx context.Context, runID string) (int64, error) {
	return r.eventBusStore.GetRunEventCount(ctx, runID)
}
func (r *RedisInfra) SubscribeRunEvents(ctx context.Context, runID string) (<-chan *eventbus.RunEvent, error) {
	return r.eventBusStore.SubscribeRunEvents(ctx, runID)
}
func (r *RedisInfra) DeleteRunEvents(ctx context.Context, runID string) error {
	return r.eventBusStore.DeleteRunEvents(ctx, runID)
}

// ============================================================================
// queue.Queue 接口委托实现
// ============================================================================

func (r *RedisInfra) ScheduleRun(ctx context.Context, runID, taskID string) (string, error) {
	return r.queueStore.ScheduleRun(ctx, runID, taskID)
}
func (r *RedisInfra) CreateSchedulerConsumerGroup(ctx context.Context) error {
	return r.queueStore.CreateSchedulerConsumerGroup(ctx)
}
func (r *RedisInfra) ConsumeSchedulerRuns(ctx context.Context, consumerID string, count int64, blockTimeout time.Duration) ([]*queue.SchedulerMessage, error) {
	return r.queueStore.ConsumeSchedulerRuns(ctx, consumerID, count, blockTimeout)
}
func (r *RedisInfra) AckSchedulerRun(ctx context.Context, messageID string) error {
	return r.queueStore.AckSchedulerRun(ctx, messageID)
}
func (r *RedisInfra) GetSchedulerQueueLength(ctx context.Context) (int64, error) {
	return r.queueStore.GetSchedulerQueueLength(ctx)
}
func (r *RedisInfra) GetSchedulerPendingCount(ctx context.Context) (int64, error) {
	return r.queueStore.GetSchedulerPendingCount(ctx)
}
func (r *RedisInfra) PublishRunToNode(ctx context.Context, nodeID, runID, taskID string) (string, error) {
	return r.queueStore.PublishRunToNode(ctx, nodeID, runID, taskID)
}
func (r *RedisInfra) CreateNodeConsumerGroup(ctx context.Context, nodeID string) error {
	return r.queueStore.CreateNodeConsumerGroup(ctx, nodeID)
}
func (r *RedisInfra) ConsumeNodeRuns(ctx context.Context, nodeID, consumerID string, count int64, blockTimeout time.Duration) ([]*queue.NodeRunMessage, error) {
	return r.queueStore.ConsumeNodeRuns(ctx, nodeID, consumerID, count, blockTimeout)
}
func (r *RedisInfra) AckNodeRun(ctx context.Context, nodeID, messageID string) error {
	return r.queueStore.AckNodeRun(ctx, nodeID, messageID)
}
func (r *RedisInfra) GetNodeRunsQueueLength(ctx context.Context, nodeID string) (int64, error) {
	return r.queueStore.GetNodeRunsQueueLength(ctx, nodeID)
}
func (r *RedisInfra) GetNodeRunsPendingCount(ctx context.Context, nodeID string) (int64, error) {
	return r.queueStore.GetNodeRunsPendingCount(ctx, nodeID)
}

// 确保 RedisInfra 实现了 storage.CacheStore 接口
var _ storage.CacheStore = (*RedisInfra)(nil)

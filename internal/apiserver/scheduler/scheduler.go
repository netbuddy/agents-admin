// Package scheduler 调度器核心实现
//
// 调度器负责将 queued 状态的 Run 分配到可用的 Node 执行。
// 架构：Redis Streams 事件驱动 + PostgreSQL 保底轮询
//
// 主路径：Redis Streams 消费（实时、低延迟）
// 保底路径：PostgreSQL 轮询（处理 Redis 写入失败的情况）
package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"agents-admin/internal/apiserver/node"
	"agents-admin/internal/shared/model"
	"agents-admin/internal/shared/queue"
	"agents-admin/internal/shared/storage"
)

// Scheduler 任务调度器
//
// 调度器是 Control Plane 的核心组件，负责：
//   - 消费 Redis Streams 中的任务事件（主路径）
//   - 定期扫描 PostgreSQL 处理遗漏的任务（保底路径）
//   - 使用可配置的策略链选择合适的 Node
//   - 更新 Run 状态和分配信息
type Scheduler struct {
	config         *Config
	store          storage.PersistentStore // PostgreSQL 存储层
	schedulerQueue queue.SchedulerQueue    // 调度队列（消费待调度的 Run）
	nodeQueue      queue.NodeRunQueue      // 节点队列（分配 Run 到节点）
	nodeManager    *node.Manager
	strategyChain  *StrategyChain

	mu             sync.Mutex    // 保护 running 状态
	running        bool          // 调度器运行状态
	stopCh         chan struct{} // 停止信号通道
	fallbackEvery  time.Duration
	staleThreshold time.Duration
}

// NewScheduler 创建调度器实例
//
// 参数：
//   - store: PostgreSQL 存储层
//   - schedulerQueue: 调度队列（可为 nil，将只使用保底轮询）
//   - nodeQueue: 节点队列（可为 nil，将不通知节点）
//   - nodeID: 当前节点 ID
//
// 返回：
//   - 初始化完成的调度器实例
func NewScheduler(store storage.PersistentStore, schedulerQueue queue.SchedulerQueue, nodeQueue queue.NodeRunQueue, nodeID string) *Scheduler {
	config := DefaultConfig()
	if nodeID != "" {
		config.NodeID = nodeID
	}
	config.Validate()

	return &Scheduler{
		config:         config,
		store:          store,
		schedulerQueue: schedulerQueue,
		nodeQueue:      nodeQueue,
		nodeManager:    node.NewManager(store),
		strategyChain:  config.BuildStrategyChain(),
		stopCh:         make(chan struct{}),
		fallbackEvery:  config.Fallback.Interval,
		staleThreshold: config.Fallback.StaleThreshold,
	}
}

// NewSchedulerWithConfig 使用自定义配置创建调度器
func NewSchedulerWithConfig(store storage.PersistentStore, schedulerQueue queue.SchedulerQueue, nodeQueue queue.NodeRunQueue, config *Config) *Scheduler {
	if config == nil {
		config = DefaultConfig()
	}
	config.Validate()

	return &Scheduler{
		config:         config,
		store:          store,
		schedulerQueue: schedulerQueue,
		nodeQueue:      nodeQueue,
		nodeManager:    node.NewManager(store),
		strategyChain:  config.BuildStrategyChain(),
		stopCh:         make(chan struct{}),
		fallbackEvery:  config.Fallback.Interval,
		staleThreshold: config.Fallback.StaleThreshold,
	}
}

func (s *Scheduler) SetFallbackConfig(every time.Duration, staleThreshold time.Duration) {
	if every > 0 {
		s.fallbackEvery = every
	}
	if staleThreshold > 0 {
		s.staleThreshold = staleThreshold
	}
}

// SetStrategyChain 设置自定义策略链
func (s *Scheduler) SetStrategyChain(chain *StrategyChain) {
	s.strategyChain = chain
}

// GetConfig 获取当前配置
func (s *Scheduler) GetConfig() *Config {
	return s.config
}

// Start 启动调度器
//
// 调度器启动后会运行两个并行循环：
//  1. Redis Streams 消费循环（主路径，实时响应）
//  2. PostgreSQL 保底轮询循环（保底路径，处理遗漏）
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	log.Printf("[scheduler.start] node_id=%s queue_enabled=%v strategies=%v",
		s.config.NodeID, s.schedulerQueue != nil, s.config.Strategy.Chain)

	var wg sync.WaitGroup

	// 主路径：队列消费
	if s.schedulerQueue != nil {
		if err := s.schedulerQueue.CreateSchedulerConsumerGroup(ctx); err != nil {
			log.Printf("[scheduler.redis.group.failed] error=%v", err)
		} else {
			log.Printf("[scheduler.redis.group.created] group=schedulers")
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			s.consumeRedisStream(ctx)
		}()
	}

	// 保底路径：PostgreSQL 轮询
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.fallbackPolling(ctx)
	}()

	wg.Wait()
	log.Printf("[scheduler.stopped] node_id=%s", s.config.NodeID)
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		close(s.stopCh)
		s.running = false
	}
}

// consumeRedisStream 消费 Redis Streams 中的任务事件
func (s *Scheduler) consumeRedisStream(ctx context.Context) {
	log.Printf("[scheduler.redis.start] consumer_id=%s", s.config.NodeID)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[scheduler.redis.stop] reason=context_cancelled")
			return
		case <-s.stopCh:
			log.Printf("[scheduler.redis.stop] reason=stop_signal")
			return
		default:
		}

		messages, err := s.schedulerQueue.ConsumeSchedulerRuns(ctx, s.config.NodeID,
			int64(s.config.Redis.ReadCount), s.config.Redis.ReadTimeout)
		if err != nil {
			log.Printf("[scheduler.redis.consume.failed] error=%v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		if len(messages) == 0 {
			continue
		}

		log.Printf("[scheduler.redis.received] count=%d", len(messages))

		for _, msg := range messages {
			startTime := time.Now()
			log.Printf("[scheduler.run.start] run_id=%s task_id=%s msg_id=%s source=redis",
				msg.RunID, msg.TaskID, msg.ID)

			if err := s.scheduleRunByID(ctx, msg.RunID); err != nil {
				log.Printf("[scheduler.run.failed] run_id=%s error=%v", msg.RunID, err)
				continue
			}

			if err := s.schedulerQueue.AckSchedulerRun(ctx, msg.ID); err != nil {
				log.Printf("[scheduler.redis.ack.failed] run_id=%s msg_id=%s error=%v",
					msg.RunID, msg.ID, err)
			}

			delay := time.Since(msg.CreatedAt)
			duration := time.Since(startTime)
			log.Printf("[scheduler.run.success] run_id=%s msg_id=%s delay_ms=%d duration_ms=%d",
				msg.RunID, msg.ID, delay.Milliseconds(), duration.Milliseconds())
		}
	}
}

// fallbackPolling 保底轮询
func (s *Scheduler) fallbackPolling(ctx context.Context) {
	// 启动时立即执行一次
	s.processFallbackRuns(ctx)

	ticker := time.NewTicker(s.fallbackEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[scheduler.fallback.stop] reason=context_cancelled")
			return
		case <-s.stopCh:
			log.Printf("[scheduler.fallback.stop] reason=stop_signal")
			return
		case <-ticker.C:
			s.processFallbackRuns(ctx)
		}
	}
}

// processFallbackRuns 处理保底轮询
func (s *Scheduler) processFallbackRuns(ctx context.Context) {
	// 查找状态是 queued 但超过阈值时间没被调度的 Run
	runs, err := s.store.ListStaleQueuedRuns(ctx, s.staleThreshold)
	if err != nil {
		log.Printf("[scheduler.fallback.query.failed] error=%v", err)
		return
	}

	if len(runs) == 0 {
		return
	}

	log.Printf("[scheduler.fallback.found] count=%d threshold=%s", len(runs), s.staleThreshold)

	for _, run := range runs {
		log.Printf("[scheduler.fallback.processing] run_id=%s created_at=%s source=fallback",
			run.ID, run.CreatedAt.Format(time.RFC3339))

		if err := s.scheduleRunByID(ctx, run.ID); err != nil {
			log.Printf("[scheduler.fallback.failed] run_id=%s error=%v", run.ID, err)
			continue
		}

		log.Printf("[scheduler.fallback.success] run_id=%s", run.ID)
	}
}

// scheduleRunByID 根据 Run ID 执行调度
func (s *Scheduler) scheduleRunByID(ctx context.Context, runID string) error {
	run, err := s.store.GetRun(ctx, runID)
	if err != nil {
		return err
	}
	if run == nil {
		log.Printf("[scheduler.run.not_found] run_id=%s", runID)
		return nil
	}

	if run.Status != model.RunStatusQueued {
		log.Printf("[scheduler.run.skip] run_id=%s status=%s reason=not_queued", runID, run.Status)
		return nil
	}

	return s.scheduleRun(ctx, run)
}

// scheduleRun 执行单个 Run 的调度
func (s *Scheduler) scheduleRun(ctx context.Context, run *model.Run) error {
	// 获取在线节点
	nodes, err := s.nodeManager.ListOnlineNodes(ctx)
	if err != nil {
		return err
	}
	if len(nodes) == 0 {
		log.Printf("[scheduler.run.no_nodes] run_id=%s", run.ID)
		return nil
	}

	// 构建在线节点 ID 集合
	onlineIDs := make(map[string]struct{}, len(nodes))
	for _, n := range nodes {
		onlineIDs[n.ID] = struct{}{}
	}

	// 处理离线节点上的僵尸 Run
	s.nodeManager.RequeueRunsAssignedToOfflineNodes(ctx, onlineIDs, s.config.Requeue.OfflineThreshold)

	// 刷新节点运行任务计数
	s.nodeManager.RefreshRunningCount(ctx, nodes)

	// 获取任务信息
	var task *model.Task
	if run.TaskID != "" {
		task, _ = s.store.GetTask(ctx, run.TaskID)
	}

	// 解析优先节点
	preferredNode := s.nodeManager.ResolvePreferredNodeID(ctx, run.TaskID, run.Snapshot)

	// 构建调度请求
	req := &ScheduleRequest{
		Run:            run,
		Task:           task,
		CandidateNodes: nodes,
		NodeRunning:    s.nodeManager.GetNodeRunning(),
		PreferredNode:  preferredNode,
	}

	// 使用策略链选择节点
	node, reason := s.strategyChain.SelectNode(ctx, req)
	if node == nil {
		log.Printf("[scheduler.run.no_match] run_id=%s reason=%s", run.ID, reason)
		return nil
	}

	// 更新 Run 状态
	nodeID := node.ID
	if err := s.store.UpdateRunStatus(ctx, run.ID, model.RunStatusAssigned, &nodeID); err != nil {
		return err
	}

	// 通知节点管理器
	s.publishTaskToNode(ctx, nodeID, run.ID, run.TaskID)

	s.nodeManager.IncrementRunning(nodeID)
	log.Printf("[scheduler.run.assigned] run_id=%s node_id=%s reason=%s", run.ID, nodeID, reason)
	return nil
}

// publishTaskToNode 发布任务到节点的 Redis Stream
func (s *Scheduler) publishTaskToNode(ctx context.Context, nodeID, runID, taskID string) {
	if s.nodeQueue == nil {
		return
	}

	msgID, err := s.nodeQueue.PublishRunToNode(ctx, nodeID, runID, taskID)
	if err != nil {
		log.Printf("[scheduler.notify.failed] node_id=%s run_id=%s error=%v", nodeID, runID, err)
		return
	}

	log.Printf("[scheduler.notify.success] node_id=%s run_id=%s msg_id=%s", nodeID, runID, msgID)
}

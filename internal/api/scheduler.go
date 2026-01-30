// Package api 调度器实现
//
// 调度器负责将 queued 状态的 Run 分配到可用的 Node 执行。
// 当前实现为简单的单机版调度器，采用轮询方式。
package api

import (
	"context"
	"log"
	"sync"
	"time"

	"agents-admin/internal/model"
	"agents-admin/internal/storage"
)

// Scheduler 任务调度器
//
// 调度器是 Control Plane 的核心组件，负责：
//   - 定期扫描 queued 状态的 Run
//   - 选择合适的 Node 执行任务
//   - 更新 Run 状态和分配信息
//
// 当前实现为简单调度器（单机版），未来可扩展为分布式调度器。
//
// 调度策略：
//   - 当前采用简单的首选节点策略
//   - 未来可扩展支持：负载均衡、标签匹配、亲和性等
type Scheduler struct {
	store   *storage.PostgresStore // 数据存储层
	handler *Handler               // API 处理器引用
	mu      sync.Mutex             // 保护 running 状态
	running bool                   // 调度器运行状态
	stopCh  chan struct{}          // 停止信号通道
}

// NewScheduler 创建调度器实例
//
// 参数：
//   - store: 数据存储层
//   - handler: API 处理器
//
// 返回：
//   - 初始化完成的调度器实例
func NewScheduler(store *storage.PostgresStore, handler *Handler) *Scheduler {
	return &Scheduler{
		store:   store,
		handler: handler,
		stopCh:  make(chan struct{}),
	}
}

// Start 启动调度器
//
// 调度器启动后会每 5 秒执行一次调度循环：
//  1. 获取所有 queued 状态的 Run
//  2. 获取所有 online 状态的 Node
//  3. 为每个 Run 选择一个 Node 并分配
//
// 参数：
//   - ctx: 上下文，用于控制调度器生命周期
//
// 调度器会在以下情况停止：
//   - ctx 被取消
//   - 收到 Stop() 调用
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	log.Println("Scheduler started")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Scheduler stopping (context cancelled)")
			return
		case <-s.stopCh:
			log.Println("Scheduler stopping (stop signal)")
			return
		case <-ticker.C:
			s.scheduleRuns(ctx)
			// 注意：认证任务不走 Scheduler，由用户指定节点
		}
	}
}

// Stop 停止调度器
//
// 发送停止信号，调度器会在当前调度循环结束后退出。
// 此方法是幂等的，多次调用不会产生副作用。
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		close(s.stopCh)
		s.running = false
	}
}

// scheduleRuns 执行一次调度循环
//
// 调度流程：
//  1. 获取最多 10 个 queued 状态的 Run
//  2. 获取所有 online 状态的 Node
//  3. 为每个 Run 选择节点并更新状态为 running
func (s *Scheduler) scheduleRuns(ctx context.Context) {
	runs, err := s.store.ListQueuedRuns(ctx, 10)
	if err != nil {
		log.Printf("Failed to list queued runs: %v", err)
		return
	}

	if len(runs) == 0 {
		return
	}

	nodes, err := s.store.ListOnlineNodes(ctx)
	if err != nil {
		log.Printf("Failed to list online nodes: %v", err)
		return
	}

	if len(nodes) == 0 {
		log.Println("No online nodes available")
		return
	}

	for _, run := range runs {
		node := s.selectNode(nodes)
		if node == nil {
			log.Println("No available node for run:", run.ID)
			continue
		}

		nodeID := node.ID
		if err := s.store.UpdateRunStatus(ctx, run.ID, model.RunStatusRunning, &nodeID); err != nil {
			log.Printf("Failed to update run status: %v", err)
			continue
		}

		log.Printf("Scheduled run %s to node %s", run.ID, nodeID)
	}
}

// selectNode 选择执行节点
//
// 当前实现采用简单策略：返回列表中的第一个节点。
// 未来可扩展支持更复杂的选择策略。
//
// 参数：
//   - nodes: 可用节点列表
//
// 返回：
//   - 选中的节点，如果列表为空则返回 nil
func (s *Scheduler) selectNode(nodes []*model.Node) *model.Node {
	if len(nodes) == 0 {
		return nil
	}
	return nodes[0]
}

// scheduleAuthTasks 调度待处理的认证任务
func (s *Scheduler) scheduleAuthTasks(ctx context.Context) {
	// 获取待调度的认证任务
	tasks, err := s.store.ListPendingAuthTasks(ctx, 10)
	if err != nil {
		log.Printf("[Scheduler] ListPendingAuthTasks error: %v", err)
		return
	}

	if len(tasks) == 0 {
		return
	}

	// 获取在线节点
	nodes, err := s.store.ListOnlineNodes(ctx)
	if err != nil {
		log.Printf("[Scheduler] ListOnlineNodes error: %v", err)
		return
	}

	if len(nodes) == 0 {
		log.Printf("[Scheduler] No online nodes available for auth tasks")
		return
	}

	// 简单轮询调度
	for i, task := range tasks {
		node := nodes[i%len(nodes)]

		if err := s.store.UpdateAuthTaskAssignment(ctx, task.ID, node.ID); err != nil {
			log.Printf("[Scheduler] UpdateAuthTaskAssignment error: %v", err)
			continue
		}

		log.Printf("[Scheduler] AuthTask %s assigned to node %s", task.ID, node.ID)
	}
}

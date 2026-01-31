// Package api 调度器实现
//
// 调度器负责将 queued 状态的 Run 分配到可用的 Node 执行。
// 支持标签匹配、负载均衡等调度策略。
package api

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"agents-admin/internal/model"
	"agents-admin/internal/storage"
)

const (
	// offlineRequeueThreshold 表示：Run 已被标记为 running（已分配节点），但节点在 etcd 中无心跳时，
	// 需要等待一段时间再判断是否可回退到 queued，避免短暂抖动导致误判。
	offlineRequeueThreshold = 30 * time.Second
)

// Scheduler 任务调度器
//
// 调度器是 Control Plane 的核心组件，负责：
//   - 定期扫描 queued 状态的 Run
//   - 根据标签匹配选择合适的 Node
//   - 支持负载均衡策略
//   - 更新 Run 状态和分配信息
//
// 调度策略：
//   - 标签匹配：Task labels 必须是 Node labels 的子集
//   - 负载均衡：优先选择可用容量最大的节点
//   - 容量检查：确保节点有足够的并发容量
type Scheduler struct {
	store       *storage.PostgresStore // 数据存储层
	handler     *Handler               // API 处理器引用
	mu          sync.Mutex             // 保护 running 状态
	running     bool                   // 调度器运行状态
	stopCh      chan struct{}          // 停止信号通道
	nodeRunning map[string]int         // 节点当前运行的任务数（内存缓存）
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
		store:       store,
		handler:     handler,
		stopCh:      make(chan struct{}),
		nodeRunning: make(map[string]int),
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
//  3. 刷新节点运行任务计数
//  4. 为每个 Run 根据标签匹配和负载均衡选择节点
func (s *Scheduler) scheduleRuns(ctx context.Context) {
	// 1) 获取在线节点（优先使用 etcd 心跳，避免 PostgreSQL 中的“僵尸 online 节点”被误调度）
	nodes, hbMap, err := s.listOnlineNodes(ctx)
	if err != nil {
		log.Printf("Failed to list online nodes: %v", err)
		return
	}
	if len(nodes) == 0 {
		log.Println("No online nodes available")
		return
	}

	onlineIDs := make(map[string]struct{}, len(nodes))
	nodesByID := make(map[string]*model.Node, len(nodes))
	for _, n := range nodes {
		onlineIDs[n.ID] = struct{}{}
		nodesByID[n.ID] = n
	}

	// 2) 处理“误分配到离线节点”的 running Run（无任何事件，说明 Executor 从未实际执行）
	// 仅在启用 etcd 心跳时做该修复：否则无法可靠判断在线状态。
	if hbMap != nil {
		s.requeueRunsAssignedToOfflineNodes(ctx, onlineIDs)
	}

	// 3) 获取 queued Run
	runs, err := s.store.ListQueuedRuns(ctx, 10)
	if err != nil {
		log.Printf("Failed to list queued runs: %v", err)
		return
	}
	if len(runs) == 0 {
		return
	}

	// 4) 刷新节点运行任务计数（只对在线节点统计）
	s.refreshNodeRunningCount(ctx, nodes)

	for _, run := range runs {
		// 优先固定到实例/账号所属节点，保证容器/卷在本机可达
		if preferredNodeID := s.resolvePreferredNodeID(ctx, run.TaskID, run.Snapshot); preferredNodeID != "" {
			node := nodesByID[preferredNodeID]
			if node == nil {
				log.Printf("Preferred node %s is not online for run %s", preferredNodeID, run.ID)
				continue
			}

			maxConcurrent := s.getNodeMaxConcurrent(node)
			currentRunning := s.nodeRunning[node.ID]
			if maxConcurrent-currentRunning <= 0 {
				log.Printf("Preferred node %s has no capacity for run %s", preferredNodeID, run.ID)
				continue
			}

			nodeID := node.ID
			if err := s.store.UpdateRunStatus(ctx, run.ID, model.RunStatusRunning, &nodeID); err != nil {
				log.Printf("Failed to update run status: %v", err)
				continue
			}

			s.nodeRunning[nodeID]++
			log.Printf("Scheduled run %s to preferred node %s", run.ID, nodeID)
			continue
		}

		// 获取任务的标签
		taskLabels := s.getTaskLabels(ctx, run.TaskID)

		// 根据标签匹配和负载均衡选择节点
		node := s.selectNodeWithLabels(nodes, taskLabels)
		if node == nil {
			log.Printf("No matching node for run %s (labels: %v)", run.ID, taskLabels)
			continue
		}

		nodeID := node.ID
		if err := s.store.UpdateRunStatus(ctx, run.ID, model.RunStatusRunning, &nodeID); err != nil {
			log.Printf("Failed to update run status: %v", err)
			continue
		}

		// 更新内存中的运行计数
		s.nodeRunning[nodeID]++

		log.Printf("Scheduled run %s to node %s (labels matched: %v)", run.ID, nodeID, taskLabels)
	}
}

// listOnlineNodes 获取在线节点列表。
//
// - 优先使用 etcd 心跳（带 TTL）：etcd 有心跳 = online，无心跳 = offline
// - 当 etcd 不可用/关闭时，回退到 PostgreSQL 的 online 状态（并用 LastHeartbeat 进行新鲜度过滤）
func (s *Scheduler) listOnlineNodes(ctx context.Context) ([]*model.Node, map[string]*storage.NodeHeartbeat, error) {
	// etcd 可用：以 etcd 心跳为准
	if s.handler != nil && s.handler.etcdStore != nil {
		nodes, err := s.store.ListAllNodes(ctx)
		if err != nil {
			return nil, nil, err
		}

		hbList, err := s.handler.etcdStore.ListNodeHeartbeats(ctx)
		if err != nil {
			// etcd 异常时不要直接误调度到“僵尸 online 节点”，回退到“近期仍在写 PostgreSQL 心跳”的节点。
			log.Printf("[Scheduler] WARNING: failed to list etcd heartbeats, fallback to postgres freshness filter: %v", err)
			return filterNodesByFreshHeartbeat(nodes, 45*time.Second), nil, nil
		}

		online, hbMap := mergeOnlineNodes(nodes, hbList)
		return online, hbMap, nil
	}

	// etcd 不可用：回退到 PostgreSQL 的 status=online
	nodes, err := s.store.ListOnlineNodes(ctx)
	return nodes, nil, err
}

func filterNodesByFreshHeartbeat(nodes []*model.Node, window time.Duration) []*model.Node {
	if len(nodes) == 0 {
		return nil
	}

	now := time.Now()
	var out []*model.Node
	for _, n := range nodes {
		if n.LastHeartbeat == nil {
			continue
		}
		if now.Sub(*n.LastHeartbeat) > window {
			continue
		}
		out = append(out, n)
	}
	return out
}

func mergeOnlineNodes(nodes []*model.Node, heartbeats []*storage.NodeHeartbeat) ([]*model.Node, map[string]*storage.NodeHeartbeat) {
	hbMap := make(map[string]*storage.NodeHeartbeat, len(heartbeats))
	for _, hb := range heartbeats {
		if hb == nil || hb.NodeID == "" {
			continue
		}
		hbMap[hb.NodeID] = hb
	}
	if len(hbMap) == 0 || len(nodes) == 0 {
		return nil, hbMap
	}

	online := make([]*model.Node, 0, len(hbMap))
	for _, node := range nodes {
		hb, ok := hbMap[node.ID]
		if !ok {
			continue
		}

		// 使用 etcd 中的 Capacity（实时）覆盖 PostgreSQL 中的历史值
		var capJSON []byte
		if hb.Capacity != nil {
			if b, err := json.Marshal(hb.Capacity); err == nil {
				capJSON = b
			}
		}

		n := *node // shallow copy
		if len(capJSON) > 0 {
			n.Capacity = capJSON
		}
		n.Status = model.NodeStatusOnline
		n.LastHeartbeat = &hb.LastHeartbeat
		online = append(online, &n)
	}
	return online, hbMap
}

func (s *Scheduler) requeueRunsAssignedToOfflineNodes(ctx context.Context, onlineIDs map[string]struct{}) {
	runs, err := s.store.ListRunningRuns(ctx, 200)
	if err != nil {
		log.Printf("[Scheduler] ListRunningRuns error: %v", err)
		return
	}

	now := time.Now()
	for _, run := range runs {
		if run == nil || run.NodeID == nil || *run.NodeID == "" {
			continue
		}

		if _, ok := onlineIDs[*run.NodeID]; ok {
			continue
		}

		if run.StartedAt == nil || now.Sub(*run.StartedAt) < offlineRequeueThreshold {
			continue
		}

		cnt, err := s.store.CountEventsByRun(ctx, run.ID)
		if err != nil {
			log.Printf("[Scheduler] CountEventsByRun error (run=%s): %v", run.ID, err)
			continue
		}
		if cnt > 0 {
			// 有事件说明 Executor 已经开始执行：不自动回退，避免重复执行风险
			continue
		}

		if err := s.store.ResetRunToQueued(ctx, run.ID); err != nil {
			log.Printf("[Scheduler] ResetRunToQueued error (run=%s): %v", run.ID, err)
			continue
		}
		log.Printf("[Scheduler] Requeued run %s (was assigned to offline node %s with no events)", run.ID, *run.NodeID)
	}
}

func (s *Scheduler) resolvePreferredNodeID(ctx context.Context, taskID string, snapshot json.RawMessage) string {
	instanceID, accountID := extractAgentIDs(snapshot)

	// 兼容旧/其它调用方：如果 instance_id 没放在 spec.agent.instance_id，而是放在 Task.InstanceID 字段，
	// 则从任务记录补齐 instanceID。
	if instanceID == "" && taskID != "" {
		task, err := s.store.GetTask(ctx, taskID)
		if err != nil {
			log.Printf("[Scheduler] GetTask error: %v", err)
		} else if task != nil && task.InstanceID != nil && *task.InstanceID != "" {
			instanceID = *task.InstanceID
		}
	}

	if instanceID != "" {
		inst, err := s.store.GetInstance(ctx, instanceID)
		if err != nil {
			log.Printf("[Scheduler] GetInstance error: %v", err)
		} else if inst != nil && inst.NodeID != nil && *inst.NodeID != "" {
			return *inst.NodeID
		}
	}

	if accountID != "" {
		acc, err := s.store.GetAccount(ctx, accountID)
		if err != nil {
			log.Printf("[Scheduler] GetAccount error: %v", err)
		} else if acc != nil && acc.NodeID != "" {
			return acc.NodeID
		}
	}

	return ""
}

func extractAgentIDs(snapshot json.RawMessage) (instanceID string, accountID string) {
	if len(snapshot) == 0 {
		return "", ""
	}

	var spec map[string]interface{}
	if err := json.Unmarshal(snapshot, &spec); err != nil {
		return "", ""
	}

	agentRaw, ok := spec["agent"]
	if !ok {
		return "", ""
	}

	agent, ok := agentRaw.(map[string]interface{})
	if !ok {
		return "", ""
	}

	if v, ok := agent["instance_id"].(string); ok {
		instanceID = v
	}
	if v, ok := agent["account_id"].(string); ok {
		accountID = v
	}
	return instanceID, accountID
}

// selectNode 选择执行节点（简单版本，无标签匹配）
//
// 当前实现采用简单策略：返回列表中的第一个节点。
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

// selectNodeWithLabels 根据标签匹配和负载均衡选择节点
//
// 调度策略：
//  1. 标签匹配：Task labels 必须是 Node labels 的子集
//  2. 容量检查：节点当前运行任务数 < 最大并发数
//  3. 负载均衡：选择可用容量最大的节点
//
// 参数：
//   - nodes: 可用节点列表
//   - taskLabels: 任务要求的标签
//
// 返回：
//   - 选中的节点，如果没有匹配的节点则返回 nil
func (s *Scheduler) selectNodeWithLabels(nodes []*model.Node, taskLabels map[string]string) *model.Node {
	if len(nodes) == 0 {
		return nil
	}

	var bestNode *model.Node
	var bestAvailable int = -1

	for _, node := range nodes {
		// 1. 标签匹配
		if !s.matchLabels(node, taskLabels) {
			continue
		}

		// 2. 容量检查
		maxConcurrent := s.getNodeMaxConcurrent(node)
		currentRunning := s.nodeRunning[node.ID]
		available := maxConcurrent - currentRunning

		if available <= 0 {
			continue
		}

		// 3. 选择可用容量最大的节点
		if available > bestAvailable {
			bestAvailable = available
			bestNode = node
		}
	}

	return bestNode
}

// matchLabels 检查节点是否满足任务的标签要求
//
// 匹配规则：任务的每个标签必须在节点上存在且值相等
func (s *Scheduler) matchLabels(node *model.Node, taskLabels map[string]string) bool {
	if len(taskLabels) == 0 {
		return true // 无标签要求，所有节点都匹配
	}

	// 解析节点标签
	var nodeLabels map[string]string
	if len(node.Labels) > 0 {
		if err := json.Unmarshal(node.Labels, &nodeLabels); err != nil {
			log.Printf("Failed to parse node labels for %s: %v", node.ID, err)
			return false
		}
	}

	// 检查每个任务标签
	for key, value := range taskLabels {
		if nodeValue, ok := nodeLabels[key]; !ok || nodeValue != value {
			return false
		}
	}

	return true
}

// getNodeMaxConcurrent 获取节点的最大并发数
func (s *Scheduler) getNodeMaxConcurrent(node *model.Node) int {
	if len(node.Capacity) == 0 {
		return 1 // 默认最大并发数为 1
	}

	var capacity map[string]interface{}
	if err := json.Unmarshal(node.Capacity, &capacity); err != nil {
		return 1
	}

	if maxConcurrent, ok := capacity["max_concurrent"]; ok {
		switch v := maxConcurrent.(type) {
		case float64:
			return int(v)
		case int:
			return v
		}
	}

	return 1
}

// getTaskLabels 获取任务的标签
func (s *Scheduler) getTaskLabels(ctx context.Context, taskID string) map[string]string {
	task, err := s.store.GetTask(ctx, taskID)
	if err != nil || task == nil {
		return nil
	}

	// 解析 TaskSpec 获取 labels
	var spec map[string]interface{}
	if err := json.Unmarshal(task.Spec, &spec); err != nil {
		return nil
	}

	labelsRaw, ok := spec["labels"]
	if !ok {
		return nil
	}

	labels, ok := labelsRaw.(map[string]interface{})
	if !ok {
		return nil
	}

	result := make(map[string]string)
	for k, v := range labels {
		if str, ok := v.(string); ok {
			result[k] = str
		}
	}

	return result
}

// refreshNodeRunningCount 刷新节点运行任务计数
func (s *Scheduler) refreshNodeRunningCount(ctx context.Context, nodes []*model.Node) {
	// 清空当前计数
	s.nodeRunning = make(map[string]int)

	// 从数据库获取每个节点的运行任务数
	for _, node := range nodes {
		runs, err := s.store.ListRunsByNode(ctx, node.ID)
		if err != nil {
			log.Printf("Failed to list runs for node %s: %v", node.ID, err)
			continue
		}
		s.nodeRunning[node.ID] = len(runs)
	}
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
	nodes, _, err := s.listOnlineNodes(ctx)
	if err != nil {
		log.Printf("[Scheduler] listOnlineNodes error: %v", err)
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

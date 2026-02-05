// Package scheduler 轮询调度策略
package scheduler

import (
	"context"
	"sync"

	nodemgr "agents-admin/internal/apiserver/node"
	"agents-admin/internal/shared/model"
)

// RoundRobinStrategy 轮询调度策略
//
// 按顺序轮流分配任务到各个节点，保证均匀分布。
// 使用内部计数器记录下一个节点索引。
//
// 场景：
//   - 需要严格均匀分布的场景
//   - 节点能力相近的集群
type RoundRobinStrategy struct {
	mu    sync.Mutex
	index int
}

// NewRoundRobinStrategy 创建轮询策略
func NewRoundRobinStrategy() *RoundRobinStrategy {
	return &RoundRobinStrategy{}
}

// Name 返回策略名称
func (s *RoundRobinStrategy) Name() string {
	return "round_robin"
}

// SelectNode 轮询选择下一个有容量的节点
func (s *RoundRobinStrategy) SelectNode(ctx context.Context, req *ScheduleRequest) (*model.Node, string) {
	if len(req.CandidateNodes) == 0 {
		return nil, ""
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 从当前索引开始，尝试找到一个有容量的节点
	n := len(req.CandidateNodes)
	for i := 0; i < n; i++ {
		idx := (s.index + i) % n
		node := req.CandidateNodes[idx]

		maxConcurrent := nodemgr.GetNodeMaxConcurrent(node)
		currentRunning := req.NodeRunning[node.ID]
		if maxConcurrent-currentRunning > 0 {
			s.index = (idx + 1) % n // 更新索引到下一个
			return node, "round_robin"
		}
	}

	return nil, ""
}

// Reset 重置轮询索引
func (s *RoundRobinStrategy) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.index = 0
}

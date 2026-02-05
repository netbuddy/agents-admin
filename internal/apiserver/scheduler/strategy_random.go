// Package scheduler 随机调度策略
package scheduler

import (
	"context"
	"math/rand"

	nodemgr "agents-admin/internal/apiserver/node"
	"agents-admin/internal/shared/model"
)

// RandomStrategy 随机调度策略
//
// 从所有可用节点中随机选择一个节点。
// 适用于无状态任务，提供简单的负载分散。
//
// 场景：
//   - 测试环境
//   - 无状态短任务
type RandomStrategy struct{}

// NewRandomStrategy 创建随机策略
func NewRandomStrategy() *RandomStrategy {
	return &RandomStrategy{}
}

// Name 返回策略名称
func (s *RandomStrategy) Name() string {
	return "random"
}

// SelectNode 随机选择一个有容量的节点
func (s *RandomStrategy) SelectNode(ctx context.Context, req *ScheduleRequest) (*model.Node, string) {
	if len(req.CandidateNodes) == 0 {
		return nil, ""
	}

	// 筛选有容量的节点
	var available []*model.Node
	for _, node := range req.CandidateNodes {
		maxConcurrent := nodemgr.GetNodeMaxConcurrent(node)
		currentRunning := req.NodeRunning[node.ID]
		if maxConcurrent-currentRunning > 0 {
			available = append(available, node)
		}
	}

	if len(available) == 0 {
		return nil, ""
	}

	// 随机选择
	idx := rand.Intn(len(available))
	return available[idx], "random"
}

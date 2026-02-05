// Package scheduler 负载均衡调度策略
package scheduler

import (
	"context"

	nodemgr "agents-admin/internal/apiserver/node"
	"agents-admin/internal/shared/model"
)

// LoadBalanceStrategy 负载均衡调度策略
//
// 在所有可用节点中选择当前负载最低（可用容量最大）的节点。
// 这是最常用的通用调度策略。
//
// 场景：
//   - 无特殊要求的任务
//   - 需要均匀分布负载的场景
type LoadBalanceStrategy struct{}

// NewLoadBalanceStrategy 创建负载均衡策略
func NewLoadBalanceStrategy() *LoadBalanceStrategy {
	return &LoadBalanceStrategy{}
}

// Name 返回策略名称
func (s *LoadBalanceStrategy) Name() string {
	return "load_balance"
}

// SelectNode 选择负载最低的节点
func (s *LoadBalanceStrategy) SelectNode(ctx context.Context, req *ScheduleRequest) (*model.Node, string) {
	if len(req.CandidateNodes) == 0 {
		return nil, ""
	}

	var bestNode *model.Node
	var bestAvailable int = -1

	for _, node := range req.CandidateNodes {
		maxConcurrent := nodemgr.GetNodeMaxConcurrent(node)
		currentRunning := req.NodeRunning[node.ID]
		available := maxConcurrent - currentRunning

		if available <= 0 {
			continue
		}

		if available > bestAvailable {
			bestAvailable = available
			bestNode = node
		}
	}

	if bestNode == nil {
		return nil, ""
	}

	return bestNode, "load_balance"
}

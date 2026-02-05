// Package scheduler 亲和性调度策略
package scheduler

import (
	"context"

	nodemgr "agents-admin/internal/apiserver/node"
	"agents-admin/internal/shared/model"
)

// AffinityStrategy 亲和性调度策略
//
// 将 Run 调度到与其关联的 Instance 或 Account 所在的节点。
// 这是最高优先级的调度策略，用于保证有状态任务的执行一致性。
//
// 场景：
//   - 浏览器实例固定在某节点上
//   - 账号登录状态保存在某节点上
type AffinityStrategy struct{}

// NewAffinityStrategy 创建亲和性策略
func NewAffinityStrategy() *AffinityStrategy {
	return &AffinityStrategy{}
}

// Name 返回策略名称
func (s *AffinityStrategy) Name() string {
	return "affinity"
}

// SelectNode 选择优先节点
//
// 如果请求中指定了 PreferredNode，且该节点在候选列表中且有容量，则选择该节点。
func (s *AffinityStrategy) SelectNode(ctx context.Context, req *ScheduleRequest) (*model.Node, string) {
	if req.PreferredNode == "" {
		return nil, ""
	}

	for _, node := range req.CandidateNodes {
		if node.ID == req.PreferredNode {
			// 检查容量
			maxConcurrent := nodemgr.GetNodeMaxConcurrent(node)
			currentRunning := req.NodeRunning[node.ID]
			if maxConcurrent-currentRunning > 0 {
				return node, "affinity"
			}
			// 节点存在但无容量
			return nil, "affinity_no_capacity"
		}
	}

	return nil, ""
}

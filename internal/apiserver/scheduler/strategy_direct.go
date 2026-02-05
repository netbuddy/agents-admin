// Package scheduler 直接指定节点调度策略
package scheduler

import (
	"context"
	"encoding/json"
	"log"

	nodemgr "agents-admin/internal/apiserver/node"
	"agents-admin/internal/shared/model"
)

// DirectStrategy 直接指定节点调度策略
//
// 用户创建任务时可以直接指定目标节点，这是最高优先级的调度策略。
// 从 Run.Snapshot 中读取 spec.node_id 字段。
//
// 场景：
//   - 用户明确知道任务应该在哪个节点执行
//   - 调试或测试特定节点
//   - 手动干预调度决策
type DirectStrategy struct{}

// NewDirectStrategy 创建直接指定策略
func NewDirectStrategy() *DirectStrategy {
	return &DirectStrategy{}
}

// Name 返回策略名称
func (s *DirectStrategy) Name() string {
	return "direct"
}

// SelectNode 选择直接指定的节点
//
// 从 Run.Snapshot 中读取 spec.node_id 字段，如果存在且节点可用，则选择该节点。
func (s *DirectStrategy) SelectNode(ctx context.Context, req *ScheduleRequest) (*model.Node, string) {
	if req.Run == nil {
		return nil, ""
	}

	// 从 Snapshot 中提取指定的节点 ID
	specifiedNodeID := extractSpecifiedNodeID(req.Run.Snapshot)
	if specifiedNodeID == "" {
		return nil, ""
	}

	// 在候选节点中查找
	for _, n := range req.CandidateNodes {
		if n.ID == specifiedNodeID {
			// 检查容量
			maxConcurrent := nodemgr.GetNodeMaxConcurrent(n)
			currentRunning := req.NodeRunning[n.ID]
			if maxConcurrent-currentRunning > 0 {
				return n, "direct"
			}
			log.Printf("[strategy.direct] node %s has no capacity", specifiedNodeID)
			return nil, "direct_no_capacity"
		}
	}

	log.Printf("[strategy.direct] specified node %s not found or offline", specifiedNodeID)
	return nil, "direct_node_unavailable"
}

// extractSpecifiedNodeID 从 Snapshot 中提取指定的节点 ID
func extractSpecifiedNodeID(snapshot json.RawMessage) string {
	if len(snapshot) == 0 {
		return ""
	}

	var spec map[string]interface{}
	if err := json.Unmarshal(snapshot, &spec); err != nil {
		return ""
	}

	// 支持两种格式：
	// 1. spec.node_id
	// 2. spec.target_node
	if nodeID, ok := spec["node_id"].(string); ok && nodeID != "" {
		return nodeID
	}
	if nodeID, ok := spec["target_node"].(string); ok && nodeID != "" {
		return nodeID
	}

	return ""
}

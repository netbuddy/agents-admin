// Package scheduler 标签匹配调度策略
package scheduler

import (
	"context"
	"encoding/json"
	"log"

	nodemgr "agents-admin/internal/apiserver/node"
	"agents-admin/internal/shared/model"
)

// LabelMatchStrategy 标签匹配调度策略
//
// 根据 Task 的标签要求，选择标签完全匹配的节点。
// 匹配规则：Task labels 必须是 Node labels 的子集。
//
// 场景：
//   - 任务要求 env=prod，只调度到生产环境节点
//   - 任务要求 gpu=true，只调度到有 GPU 的节点
type LabelMatchStrategy struct {
	// 是否启用负载均衡（在匹配的节点中选择容量最大的）
	loadBalance bool
}

// NewLabelMatchStrategy 创建标签匹配策略
//
// 参数：
//   - loadBalance: 是否在匹配的节点中启用负载均衡
func NewLabelMatchStrategy(loadBalance bool) *LabelMatchStrategy {
	return &LabelMatchStrategy{loadBalance: loadBalance}
}

// Name 返回策略名称
func (s *LabelMatchStrategy) Name() string {
	return "label_match"
}

// SelectNode 选择标签匹配的节点
func (s *LabelMatchStrategy) SelectNode(ctx context.Context, req *ScheduleRequest) (*model.Node, string) {
	taskLabels := getTaskLabelsFromRequest(req)

	var matchedNodes []*model.Node
	for _, node := range req.CandidateNodes {
		if matchLabels(node, taskLabels) {
			// 检查容量
			maxConcurrent := nodemgr.GetNodeMaxConcurrent(node)
			currentRunning := req.NodeRunning[node.ID]
			if maxConcurrent-currentRunning > 0 {
				matchedNodes = append(matchedNodes, node)
			}
		}
	}

	if len(matchedNodes) == 0 {
		return nil, ""
	}

	// 如果只有一个匹配节点，直接返回
	if len(matchedNodes) == 1 {
		return matchedNodes[0], "label_match"
	}

	// 多个匹配节点时，根据配置选择
	if s.loadBalance {
		return selectByLoadBalance(matchedNodes, req.NodeRunning), "label_match_lb"
	}

	// 默认返回第一个匹配的节点
	return matchedNodes[0], "label_match"
}

// getTaskLabelsFromRequest 从请求中获取任务标签
func getTaskLabelsFromRequest(req *ScheduleRequest) map[string]string {
	if req.Task == nil || req.Task.Labels == nil {
		return nil
	}
	return req.Task.Labels
}

// matchLabels 检查节点是否满足任务的标签要求
func matchLabels(node *model.Node, taskLabels map[string]string) bool {
	if len(taskLabels) == 0 {
		return true // 无标签要求，所有节点都匹配
	}

	// 解析节点标签
	var nodeLabels map[string]string
	if len(node.Labels) > 0 {
		if err := json.Unmarshal(node.Labels, &nodeLabels); err != nil {
			log.Printf("[strategy.label] failed to parse node labels for %s: %v", node.ID, err)
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

// selectByLoadBalance 在节点列表中选择可用容量最大的节点
func selectByLoadBalance(nodes []*model.Node, nodeRunning map[string]int) *model.Node {
	var bestNode *model.Node
	var bestAvailable int = -1

	for _, node := range nodes {
		maxConcurrent := nodemgr.GetNodeMaxConcurrent(node)
		currentRunning := nodeRunning[node.ID]
		available := maxConcurrent - currentRunning

		if available > bestAvailable {
			bestAvailable = available
			bestNode = node
		}
	}

	return bestNode
}

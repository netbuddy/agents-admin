// Package scheduler 调度策略接口和策略链
package scheduler

import (
	"context"

	"agents-admin/internal/shared/model"
)

// Strategy 调度策略接口
//
// 所有调度策略必须实现此接口。策略负责从候选节点列表中选择最合适的节点。
// 策略可以组合成策略链，按优先级依次尝试。
type Strategy interface {
	// Name 返回策略名称（用于日志和配置）
	Name() string

	// SelectNode 从候选节点中选择一个节点
	//
	// 参数：
	//   - ctx: 上下文
	//   - req: 调度请求，包含 Run、Task 信息和候选节点
	//
	// 返回：
	//   - 选中的节点，如果没有合适的节点则返回 nil
	//   - 选择原因（用于日志）
	SelectNode(ctx context.Context, req *ScheduleRequest) (*model.Node, string)
}

// ScheduleRequest 调度请求
//
// 封装调度所需的所有信息，传递给策略进行节点选择
type ScheduleRequest struct {
	Run            *model.Run             // 待调度的 Run
	Task           *model.Task            // 关联的 Task（可能为 nil）
	CandidateNodes []*model.Node          // 候选节点列表（已过滤在线且有容量的节点）
	NodeRunning    map[string]int         // 各节点当前运行任务数
	PreferredNode  string                 // 优先节点 ID（由亲和性策略使用）
}

// StrategyChain 策略链
//
// 按优先级组织多个策略，依次尝试直到找到合适的节点。
// 典型的策略链顺序：亲和性 → 标签匹配 → 负载均衡
type StrategyChain struct {
	strategies []Strategy
}

// NewStrategyChain 创建策略链
func NewStrategyChain(strategies ...Strategy) *StrategyChain {
	return &StrategyChain{strategies: strategies}
}

// SelectNode 按策略链顺序选择节点
func (c *StrategyChain) SelectNode(ctx context.Context, req *ScheduleRequest) (*model.Node, string) {
	for _, strategy := range c.strategies {
		if node, reason := strategy.SelectNode(ctx, req); node != nil {
			return node, reason
		}
	}
	return nil, "no_strategy_matched"
}

// Add 添加策略到链尾
func (c *StrategyChain) Add(s Strategy) {
	c.strategies = append(c.strategies, s)
}

// Prepend 添加策略到链首
func (c *StrategyChain) Prepend(s Strategy) {
	c.strategies = append([]Strategy{s}, c.strategies...)
}

// Strategies 返回当前策略列表（只读）
func (c *StrategyChain) Strategies() []Strategy {
	result := make([]Strategy, len(c.strategies))
	copy(result, c.strategies)
	return result
}

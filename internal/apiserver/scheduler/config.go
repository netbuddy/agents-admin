// Package scheduler 调度器配置
package scheduler

import "time"

// Config 调度器配置
type Config struct {
	// NodeID 当前节点 ID（作为消费者 ID）
	NodeID string `yaml:"node_id"`

	// Strategy 调度策略配置
	Strategy StrategyConfig `yaml:"strategy"`

	// Redis 消费者配置
	Redis RedisConfig `yaml:"redis"`

	// Fallback 保底轮询配置
	Fallback FallbackConfig `yaml:"fallback"`

	// Requeue 重新入队配置
	Requeue RequeueConfig `yaml:"requeue"`
}

// StrategyConfig 调度策略配置
type StrategyConfig struct {
	// Default 默认策略名称
	// 可选值: "label_match", "load_balance", "round_robin", "random"
	Default string `yaml:"default"`

	// Chain 策略链（按优先级排序）
	// 如果不配置，使用默认链：["affinity", "label_match"]
	Chain []string `yaml:"chain"`

	// LabelMatch 标签匹配策略配置
	LabelMatch LabelMatchConfig `yaml:"label_match"`
}

// LabelMatchConfig 标签匹配策略配置
type LabelMatchConfig struct {
	// LoadBalance 是否在匹配的节点中启用负载均衡
	LoadBalance bool `yaml:"load_balance"`
}

// RedisConfig Redis 消费者配置
type RedisConfig struct {
	// ReadTimeout XREADGROUP 阻塞超时
	ReadTimeout time.Duration `yaml:"read_timeout"`

	// ReadCount 每次读取消息数
	ReadCount int `yaml:"read_count"`
}

// FallbackConfig 保底轮询配置
type FallbackConfig struct {
	// Interval 轮询间隔
	Interval time.Duration `yaml:"interval"`

	// StaleThreshold 判定为"过期"的阈值
	StaleThreshold time.Duration `yaml:"stale_threshold"`
}

// RequeueConfig 重新入队配置
type RequeueConfig struct {
	// OfflineThreshold 节点离线后，多久才将其任务重新入队
	OfflineThreshold time.Duration `yaml:"offline_threshold"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		NodeID: "scheduler-default",
		Strategy: StrategyConfig{
			Default: "label_match",
			Chain:   []string{"affinity", "label_match"},
			LabelMatch: LabelMatchConfig{
				LoadBalance: true,
			},
		},
		Redis: RedisConfig{
			ReadTimeout: 5 * time.Second,
			ReadCount:   10,
		},
		Fallback: FallbackConfig{
			Interval:       5 * time.Minute,
			StaleThreshold: 5 * time.Minute,
		},
		Requeue: RequeueConfig{
			OfflineThreshold: 30 * time.Second,
		},
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.NodeID == "" {
		c.NodeID = "scheduler-default"
	}
	if c.Strategy.Default == "" {
		c.Strategy.Default = "label_match"
	}
	if len(c.Strategy.Chain) == 0 {
		c.Strategy.Chain = []string{"direct", "affinity", "label_match"}
	}
	if c.Redis.ReadTimeout == 0 {
		c.Redis.ReadTimeout = 5 * time.Second
	}
	if c.Redis.ReadCount == 0 {
		c.Redis.ReadCount = 10
	}
	if c.Fallback.Interval == 0 {
		c.Fallback.Interval = 5 * time.Minute
	}
	if c.Fallback.StaleThreshold == 0 {
		c.Fallback.StaleThreshold = 5 * time.Minute
	}
	if c.Requeue.OfflineThreshold == 0 {
		c.Requeue.OfflineThreshold = 30 * time.Second
	}
	return nil
}

// BuildStrategyChain 根据配置构建策略链
func (c *Config) BuildStrategyChain() *StrategyChain {
	chain := NewStrategyChain()

	for _, name := range c.Strategy.Chain {
		switch name {
		case "direct":
			chain.Add(NewDirectStrategy())
		case "affinity":
			chain.Add(NewAffinityStrategy())
		case "label_match":
			chain.Add(NewLabelMatchStrategy(c.Strategy.LabelMatch.LoadBalance))
		case "load_balance":
			chain.Add(NewLoadBalanceStrategy())
		case "round_robin":
			chain.Add(NewRoundRobinStrategy())
		case "random":
			chain.Add(NewRandomStrategy())
		}
	}

	// 如果链为空，添加默认策略
	if len(chain.strategies) == 0 {
		chain.Add(NewLabelMatchStrategy(true))
	}

	return chain
}

// Package model 定义核心数据模型
//
// memory.go 包含记忆相关的数据模型定义：
//   - Memory：记忆条目
//   - MemoryType：记忆类型枚举
//   - MemoryQuery：记忆查询
//
// 记忆类型：
//   - working：工作记忆（当前任务上下文）
//   - episodic：情景记忆（历史对话/事件）
//   - semantic：语义记忆（知识/事实）
//   - procedural：程序性记忆（技能/流程）
//
// 设计理念：
//   - Agent 的记忆系统，支持长期记忆和短期记忆
//   - 支持向量检索（通过 Embedding 字段）
//   - 支持重要性评分和过期机制
package model

import (
	"encoding/json"
	"time"
)

// ============================================================================
// MemoryType - 记忆类型枚举
// ============================================================================

// MemoryType 记忆类型
type MemoryType string

const (
	// MemoryTypeWorking 工作记忆：当前任务上下文
	// 短期记忆，任务结束后通常会清理
	MemoryTypeWorking MemoryType = "working"

	// MemoryTypeEpisodic 情景记忆：历史对话/事件
	// 记录与用户的交互历史
	MemoryTypeEpisodic MemoryType = "episodic"

	// MemoryTypeSemantic 语义记忆：知识/事实
	// 学习到的概念、规则、知识
	MemoryTypeSemantic MemoryType = "semantic"

	// MemoryTypeProcedural 程序性记忆：技能/流程
	// 如何完成特定任务的经验
	MemoryTypeProcedural MemoryType = "procedural"
)

// ============================================================================
// Memory - 记忆条目
// ============================================================================

// Memory 记忆条目
//
// Memory 是 Agent 的记忆单元，支持向量检索和重要性排序。
//
// 使用场景：
//   - 存储用户偏好和习惯
//   - 记录历史对话中的关键信息
//   - 保存学习到的知识和技能
//
// 生命周期：
//   - 创建时设置初始重要性
//   - 每次访问时更新 AccessedAt
//   - 可选设置 ExpiresAt 自动清理
type Memory struct {
	// === 基础字段 ===

	// ID 唯一标识
	ID string `json:"id" bson:"_id" db:"id"`

	// AgentID 所属 Agent
	AgentID string `json:"agent_id" bson:"agent_id" db:"agent_id"`

	// Type 记忆类型
	Type MemoryType `json:"type" bson:"type" db:"type"`

	// === 内容 ===

	// Content 记忆内容（文本形式）
	Content string `json:"content" bson:"content" db:"content"`

	// Embedding 向量表示（用于语义检索）
	// 维度根据使用的嵌入模型确定，如 OpenAI text-embedding-3-small 是 1536 维
	Embedding []float32 `json:"embedding,omitempty" bson:"embedding,omitempty" db:"embedding"`

	// === 元数据 ===

	// Metadata 额外元数据
	Metadata json.RawMessage `json:"metadata,omitempty" bson:"metadata,omitempty" db:"metadata"`

	// Importance 重要性评分（0-1）
	// 用于检索时排序和垃圾回收时选择
	Importance float64 `json:"importance" bson:"importance" db:"importance"`

	// Tags 标签（用于分类和检索）
	Tags []string `json:"tags,omitempty" bson:"tags,omitempty" db:"tags"`

	// === 关联 ===

	// SourceRunID 来源 Run（可选，记录记忆来源）
	SourceRunID *string `json:"source_run_id,omitempty" bson:"source_run_id,omitempty" db:"source_run_id"`

	// SourceTaskID 来源 Task（可选）
	SourceTaskID *string `json:"source_task_id,omitempty" bson:"source_task_id,omitempty" db:"source_task_id"`

	// === 生命周期 ===

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at" bson:"created_at" db:"created_at"`

	// AccessedAt 最后访问时间
	AccessedAt time.Time `json:"accessed_at" bson:"accessed_at" db:"accessed_at"`

	// ExpiresAt 过期时间（可选）
	ExpiresAt *time.Time `json:"expires_at,omitempty" bson:"expires_at,omitempty" db:"expires_at"`
}

// ============================================================================
// Memory 辅助方法
// ============================================================================

// IsExpired 判断记忆是否已过期
func (m *Memory) IsExpired() bool {
	if m.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*m.ExpiresAt)
}

// IsWorking 判断是否为工作记忆
func (m *Memory) IsWorking() bool {
	return m.Type == MemoryTypeWorking
}

// HasEmbedding 判断是否有向量表示
func (m *Memory) HasEmbedding() bool {
	return len(m.Embedding) > 0
}

// ============================================================================
// MemoryQuery - 记忆查询
// ============================================================================

// MemoryQuery 记忆查询参数
//
// 支持多种查询方式：
//   - 按类型过滤
//   - 语义查询（需要提供 Embedding）
//   - 按重要性过滤
//   - 按标签过滤
type MemoryQuery struct {
	// AgentID 查询的 Agent
	AgentID string `json:"agent_id"`

	// Types 记忆类型过滤
	Types []MemoryType `json:"types,omitempty"`

	// Query 语义查询文本（需要转换为 Embedding）
	Query string `json:"query,omitempty"`

	// QueryEmbedding 查询向量（直接提供 Embedding）
	QueryEmbedding []float32 `json:"query_embedding,omitempty"`

	// Tags 标签过滤（AND 逻辑）
	Tags []string `json:"tags,omitempty"`

	// MinImportance 最小重要性（0-1）
	MinImportance float64 `json:"min_importance,omitempty"`

	// Limit 返回数量限制
	Limit int `json:"limit,omitempty"`

	// IncludeExpired 是否包含已过期记忆
	IncludeExpired bool `json:"include_expired,omitempty"`
}

// ============================================================================
// MemoryStats - 记忆统计
// ============================================================================

// MemoryStats 记忆统计信息
type MemoryStats struct {
	// AgentID Agent ID
	AgentID string `json:"agent_id"`

	// TotalCount 记忆总数
	TotalCount int `json:"total_count"`

	// CountByType 按类型统计
	CountByType map[MemoryType]int `json:"count_by_type"`

	// TotalSize 总大小（字节）
	TotalSize int64 `json:"total_size"`

	// AvgImportance 平均重要性
	AvgImportance float64 `json:"avg_importance"`
}

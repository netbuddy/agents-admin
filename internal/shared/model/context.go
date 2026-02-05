// Package model 定义核心数据模型
//
// context.go 包含上下文相关的数据模型定义：
//   - TaskContext：任务上下文（持久化存储）
//   - ExecutionContext：执行上下文（运行时使用）
//   - ContextItem：上下文项
//   - ContextDocument：上下文文档
//   - Message：对话消息
//   - ConversationMessage：对话消息（别名）
package model

import "time"

// ============================================================================
// TaskContext - 任务上下文（持久化存储）
// ============================================================================

// TaskContext 任务上下文结构
//
// TaskContext 用于 Task 层级的上下文传递，持久化到数据库。
//
// 包含：
//   - InheritedContext：从父任务继承的上下文
//   - ProducedContext：本任务产出的上下文（供子任务使用）
//   - ConversationHistory：对话历史
type TaskContext struct {
	// InheritedContext 继承的上下文（来自父任务和兄弟任务）
	InheritedContext []ContextItem `json:"inherited_context,omitempty"`

	// ProducedContext 本任务产出的上下文（供子任务使用）
	ProducedContext []ContextItem `json:"produced_context,omitempty"`

	// ConversationHistory 对话历史
	ConversationHistory []Message `json:"conversation_history,omitempty"`
}

// ============================================================================
// ContextItem - 上下文项
// ============================================================================

// ContextItem 上下文项
//
// 上下文项是可复用的信息单元，可以是：
//   - file：文件内容
//   - summary：摘要
//   - reference：引用
type ContextItem struct {
	// Type 上下文类型（file, summary, reference）
	Type string `json:"type"`

	// Name 名称
	Name string `json:"name"`

	// Content 内容
	Content string `json:"content,omitempty"`

	// Source 来源任务 ID
	Source string `json:"source,omitempty"`
}

// ============================================================================
// Message - 对话消息
// ============================================================================

// Message 对话消息
//
// 用于记录对话历史，包含角色和内容。
type Message struct {
	// Role 角色（user, assistant, system）
	Role string `json:"role"`

	// Content 消息内容
	Content string `json:"content"`

	// Timestamp 时间戳
	Timestamp time.Time `json:"timestamp"`
}

// ============================================================================
// ExecutionContext - 执行上下文（运行时使用，从 pkg/driver 迁入）
// ============================================================================

// ExecutionContext 提供任务执行的运行时上下文
//
// 与 TaskContext 的区别：
//   - TaskContext：持久化存储，用于任务层级的上下文传递
//   - ExecutionContext：运行时使用，传递给 Driver 执行
//
// 用于传递：
//   - 相关文档和参考资料
//   - 前序任务的输出
//   - 继承的上下文信息
//   - 对话历史
type ExecutionContext struct {
	// Documents 相关文档列表
	Documents []ContextDocument `json:"documents,omitempty"`

	// PreviousResults 前序任务的结果
	PreviousResults map[string]interface{} `json:"previous_results,omitempty"`

	// Instructions 额外指令
	Instructions string `json:"instructions,omitempty"`

	// InheritedContext 从父任务继承的上下文
	InheritedContext []ContextItem `json:"inherited_context,omitempty"`

	// ConversationHistory 对话历史（来自 Session）
	ConversationHistory []ConversationMessage `json:"conversation_history,omitempty"`
}

// ContextDocument 上下文文档
type ContextDocument struct {
	// Name 文档名称
	Name string `json:"name"`

	// Content 文档内容
	Content string `json:"content"`

	// Type 文档类型（markdown/code/json 等）
	Type string `json:"type,omitempty"`
}

// ConversationMessage 对话消息（ExecutionContext 使用）
//
// 与 Message 类似，用于运行时上下文传递。
type ConversationMessage struct {
	// Role 角色（user, assistant, system）
	Role string `json:"role"`

	// Content 消息内容
	Content string `json:"content"`

	// Timestamp 时间戳
	Timestamp time.Time `json:"timestamp"`
}

// ToMessage 将 ConversationMessage 转换为 Message
func (cm *ConversationMessage) ToMessage() Message {
	return Message{
		Role:      cm.Role,
		Content:   cm.Content,
		Timestamp: cm.Timestamp,
	}
}

// FromMessage 从 Message 创建 ConversationMessage
func FromMessage(m Message) ConversationMessage {
	return ConversationMessage{
		Role:      m.Role,
		Content:   m.Content,
		Timestamp: m.Timestamp,
	}
}

// Package model 定义核心数据模型
//
// prompt.go 包含提示词相关的数据模型定义：
//   - PromptTemplate：提示词模板（支持变量插值）
//   - Prompt：提示词实例（填充后的内容）
//   - ContextStrategy：上下文增强策略
package model

import (
	"time"
)

// ============================================================================
// PromptSource - 提示词来源枚举
// ============================================================================

// PromptSource 提示词来源
type PromptSource string

const (
	// PromptSourceBuiltin 系统内置
	PromptSourceBuiltin PromptSource = "builtin"

	// PromptSourceCustom 用户自定义
	PromptSourceCustom PromptSource = "custom"

	// PromptSourceMCP 从 MCP Server 获取
	PromptSourceMCP PromptSource = "mcp"
)

// ============================================================================
// TemplateVariable - 模板变量定义
// ============================================================================

// TemplateVariable 模板变量定义
//
// 用于定义 PromptTemplate 和 TaskTemplate 中的可替换变量：
//   - Name：变量名，用于模板插值（如 {{.variable_name}}）
//   - Type：变量类型，用于校验和 UI 展示
//   - Description：变量描述，帮助用户理解变量用途
//   - Required：是否必填
//   - Default：默认值
type TemplateVariable struct {
	// Name 变量名
	Name string `json:"name"`

	// Type 变量类型：string/number/boolean/array/object
	Type string `json:"type"`

	// Description 变量描述
	Description string `json:"description"`

	// Required 是否必填
	Required bool `json:"required"`

	// Default 默认值
	Default interface{} `json:"default,omitempty"`

	// Options 可选值列表（用于枚举类型）
	Options []interface{} `json:"options,omitempty"`

	// Validation 校验规则（正则表达式）
	Validation string `json:"validation,omitempty"`
}

// ============================================================================
// PromptTemplate - 提示词模板
// ============================================================================

// PromptTemplate 提示词模板
//
// 提示词模板定义可复用的 Prompt 结构：
//   - 支持变量插值（使用 Go template 语法：{{.variable_name}}）
//   - 可以被多次实例化为 Prompt
//   - 可以来自系统内置、用户自定义或 MCP Server
//
// 使用场景：
//   - 系统预置的常用 Prompt 模板（如代码审查、Bug修复、重构等）
//   - 用户自定义的模板
//   - 从 MCP Server 的 Prompts 能力获取的模板
type PromptTemplate struct {
	// ID 唯一标识
	ID string `json:"id" db:"id"`

	// Name 模板名称
	Name string `json:"name" db:"name"`

	// Description 模板描述/说明
	Description string `json:"description" db:"description"`

	// Content 模板内容（支持变量插值，如 {{.variable_name}}）
	Content string `json:"content" db:"content"`

	// Variables 变量定义
	Variables []TemplateVariable `json:"variables,omitempty" db:"variables"`

	// Category 分类（如 development, testing, documentation）
	Category string `json:"category,omitempty" db:"category"`

	// Tags 标签
	Tags []string `json:"tags,omitempty" db:"tags"`

	// Source 来源（builtin/custom/mcp）
	Source PromptSource `json:"source,omitempty" db:"source"`

	// SourceRef 来源引用（如 MCP Server ID）
	SourceRef string `json:"source_ref,omitempty" db:"source_ref"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ============================================================================
// ContextStrategy - 上下文增强策略
// ============================================================================

// ContextStrategy 上下文增强策略
//
// 定义如何增强 Prompt 的上下文：
//   - 是否包含工作空间信息（文件结构、README 等）
//   - 是否包含父任务结果
//   - 上下文 token 限制
type ContextStrategy struct {
	// IncludeWorkspace 是否包含工作空间信息
	IncludeWorkspace bool `json:"include_workspace"`

	// IncludeParentResult 是否包含父任务结果
	IncludeParentResult bool `json:"include_parent_result"`

	// IncludeConversationHistory 是否包含对话历史
	IncludeConversationHistory bool `json:"include_conversation_history"`

	// MaxContextTokens 上下文 token 限制
	MaxContextTokens int `json:"max_context_tokens,omitempty"`

	// WorkspaceDepth 工作空间扫描深度
	WorkspaceDepth int `json:"workspace_depth,omitempty"`

	// FilePatterns 要包含的文件模式（如 ["*.go", "*.md"]）
	FilePatterns []string `json:"file_patterns,omitempty"`

	// ExcludePatterns 要排除的文件模式
	ExcludePatterns []string `json:"exclude_patterns,omitempty"`
}

// ============================================================================
// Prompt - 提示词实例
// ============================================================================

// Prompt 提示词实例
//
// Prompt 是填充后的提示词，可以是：
//   - 从 PromptTemplate 实例化（填充变量后）
//   - 直接创建（不基于模板）
//
// Prompt 与 Task 的关系：
//   - Task 包含一个 Prompt
//   - Prompt 描述任务的目标和要求
//
// Prompt 与 PromptTemplate 的关系：
//   - PromptTemplate 是可复用的模板
//   - Prompt 是具体的实例
type Prompt struct {
	// TemplateID 模板来源（可选，直接创建时为空）
	TemplateID *string `json:"template_id,omitempty"`

	// Content 填充后的提示词内容
	Content string `json:"content"`

	// Description 描述/说明（解释这个 Prompt 的目的）
	Description string `json:"description,omitempty"`

	// Variables 变量值（模板实例化时使用的变量值）
	Variables map[string]interface{} `json:"variables,omitempty"`

	// ContextStrategy 上下文增强策略
	ContextStrategy *ContextStrategy `json:"context_strategy,omitempty"`
}

// ============================================================================
// 辅助方法
// ============================================================================

// IsFromTemplate 判断 Prompt 是否来自模板
func (p *Prompt) IsFromTemplate() bool {
	return p.TemplateID != nil && *p.TemplateID != ""
}

// HasContextStrategy 判断是否有上下文策略
func (p *Prompt) HasContextStrategy() bool {
	return p.ContextStrategy != nil
}

// DefaultContextStrategy 返回默认的上下文策略
func DefaultContextStrategy() *ContextStrategy {
	return &ContextStrategy{
		IncludeWorkspace:           true,
		IncludeParentResult:        true,
		IncludeConversationHistory: true,
		MaxContextTokens:           8000,
		WorkspaceDepth:             3,
		FilePatterns:               []string{"*.go", "*.md", "*.yaml", "*.json"},
		ExcludePatterns:            []string{"vendor/*", "node_modules/*", ".git/*"},
	}
}

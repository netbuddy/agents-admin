// Package model 定义核心数据模型
//
// operation.go 包含系统操作相关的数据模型定义：
//   - Operation：系统操作定义（认证、运行时管理等）
//   - OperationType：操作类型枚举
//   - OperationStatus：操作状态枚举
package model

import (
	"encoding/json"
	"time"
)

// ============================================================================
// OperationType - 操作类型
// ============================================================================

// OperationType 表示系统操作的类型
//
// Operation 是系统操作的定义，不同于用户任务（Task）：
//   - 认证操作：oauth, api_key, device_code
//   - 运行时操作：runtime_create, runtime_start, runtime_stop, runtime_destroy
//
// 为什么不用 Task/Run？
//  1. 语义不同：Task 是用户知识工作，Operation 是系统管理操作
//  2. 执行者不同：Task 由 Runner 执行，Operation 由节点管理器 Handler 直接执行
//  3. 产出不同：Task 产出 Artifact，Operation 产出 Account/Runtime
type OperationType string

const (
	// 认证操作
	OperationTypeOAuth      OperationType = "oauth"       // OAuth 浏览器授权
	OperationTypeAPIKey     OperationType = "api_key"     // API Key 直接验证
	OperationTypeDeviceCode OperationType = "device_code" // Device Code 认证

	// 运行时操作
	OperationTypeRuntimeCreate  OperationType = "runtime_create"  // 创建运行时
	OperationTypeRuntimeStart   OperationType = "runtime_start"   // 启动运行时
	OperationTypeRuntimeStop    OperationType = "runtime_stop"    // 停止运行时
	OperationTypeRuntimeDestroy OperationType = "runtime_destroy" // 销毁运行时
)

// ============================================================================
// OperationStatus - 操作状态
// ============================================================================

// OperationStatus 表示 Operation 的整体状态
type OperationStatus string

const (
	// OperationStatusPending 待处理：Operation 刚创建
	OperationStatusPending OperationStatus = "pending"

	// OperationStatusInProgress 进行中：有 Action 正在执行
	OperationStatusInProgress OperationStatus = "in_progress"

	// OperationStatusCompleted 已完成：操作成功完成
	OperationStatusCompleted OperationStatus = "completed"

	// OperationStatusFailed 已失败：操作失败
	OperationStatusFailed OperationStatus = "failed"

	// OperationStatusCancelled 已取消：操作被取消
	OperationStatusCancelled OperationStatus = "cancelled"
)

// ============================================================================
// Operation - 系统操作定义
// ============================================================================

// Operation 表示一个系统操作的定义
//
// Operation 是系统操作的"定义"，类似于 Task 之于 Run：
//   - Operation 定义操作类型和配置
//   - Action 是 Operation 的执行实例
//
// 设计要点：
//  1. Operation 可以有多个 Action（重试机制）
//  2. Operation 的状态由其 Action 的状态决定
//  3. Operation 的配置存储在 Config 字段中（JSON）
//
// 数据库表：operations
type Operation struct {
	// 基本字段
	ID     string          `json:"id" bson:"_id" db:"id"`         // 唯一标识，格式：op-{random}
	Type   OperationType   `json:"type" bson:"type" db:"type"`     // 操作类型
	Config json.RawMessage `json:"config" bson:"config" db:"config"` // 操作配置（JSON）
	Status OperationStatus `json:"status" bson:"status" db:"status"` // 操作状态

	// 关联字段
	NodeID string `json:"node_id,omitempty" bson:"node_id,omitempty" db:"node_id"` // 目标节点 ID

	// 时间字段
	CreatedAt  time.Time  `json:"created_at" bson:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" bson:"updated_at" db:"updated_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty" bson:"finished_at,omitempty" db:"finished_at"`

	// 关联数据（非数据库字段）
	Actions []*Action `json:"actions,omitempty" bson:"actions,omitempty" db:"-"` // 执行实例列表
}

// ============================================================================
// OAuthConfig - OAuth 认证配置
// ============================================================================

// OAuthConfig 是 OAuth 类型 Operation 的配置
type OAuthConfig struct {
	Name      string `json:"name"`                 // 账号名称
	AgentType string `json:"agent_type"`           // Agent 类型
	ProxyID   string `json:"proxy_id,omitempty"`   // 代理 ID（可选）
}

// ============================================================================
// APIKeyConfig - API Key 认证配置
// ============================================================================

// APIKeyConfig 是 API Key 类型 Operation 的配置
type APIKeyConfig struct {
	Name      string `json:"name"`                 // 账号名称
	AgentType string `json:"agent_type"`           // Agent 类型
	APIKey    string `json:"api_key"`              // API Key 值
	ProxyID   string `json:"proxy_id,omitempty"`   // 代理 ID（可选）
}

// ============================================================================
// DeviceCodeConfig - Device Code 认证配置
// ============================================================================

// DeviceCodeConfig 是 Device Code 类型 Operation 的配置
type DeviceCodeConfig struct {
	Name      string `json:"name"`                 // 账号名称
	AgentType string `json:"agent_type"`           // Agent 类型
	ProxyID   string `json:"proxy_id,omitempty"`   // 代理 ID（可选）
}

// ============================================================================
// RuntimeConfig - 运行时操作配置
// ============================================================================

// RuntimeConfig 是运行时操作的配置
type RuntimeConfig struct {
	RuntimeID   string          `json:"runtime_id,omitempty"`   // 运行时 ID（start/stop/destroy 需要）
	RuntimeType string          `json:"runtime_type,omitempty"` // 运行时类型（create 需要）
	AgentID     string          `json:"agent_id,omitempty"`     // Agent ID（可选）
	Config      json.RawMessage `json:"config,omitempty"`       // 运行时配置（镜像、资源等）
}

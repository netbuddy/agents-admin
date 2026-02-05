// Package model 定义核心数据模型
//
// action.go 包含操作执行实例相关的数据模型定义：
//   - Action：Operation 的执行实例
//   - ActionStatus：生命周期状态（小而固定，驱动调度/重试）
//   - ActionPhase：语义阶段（按操作类型细分，描述"当前在做什么"）
//
// 状态设计参考 Kubernetes Pod Lifecycle 的三层模型：
//   - Status (= Pod Phase)：高层生命周期摘要
//   - Phase (= Container State + Reason)：语义化的执行阶段
//   - Message (= Condition Message)：人类可读的状态描述
package model

import (
	"encoding/json"
	"time"
)

// ============================================================================
// ActionStatus - 生命周期状态（小而固定）
// ============================================================================

// ActionStatus 表示 Action 的生命周期状态
//
// 这是一个小而固定的枚举，用于驱动调度、重试和 UI 高层展示。
// 语义化的执行细节由 ActionPhase 承载。
//
// 状态机：
//
//	assigned → running → success
//	             ↕
//	          waiting  → success
//	             ↘
//	         failed / timeout / cancelled
type ActionStatus string

const (
	ActionStatusAssigned  ActionStatus = "assigned"  // 已分配，等待节点执行
	ActionStatusRunning   ActionStatus = "running"   // 执行中
	ActionStatusWaiting   ActionStatus = "waiting"   // 等待外部输入
	ActionStatusSuccess   ActionStatus = "success"   // 执行成功（终态）
	ActionStatusFailed    ActionStatus = "failed"    // 执行失败（终态）
	ActionStatusTimeout   ActionStatus = "timeout"   // 执行超时（终态）
	ActionStatusCancelled ActionStatus = "cancelled" // 已取消（终态）
)

// IsTerminal 判断状态是否为终态
func (s ActionStatus) IsTerminal() bool {
	switch s {
	case ActionStatusSuccess, ActionStatusFailed, ActionStatusTimeout, ActionStatusCancelled:
		return true
	default:
		return false
	}
}

// ============================================================================
// ActionPhase - 语义阶段（按操作类型细分）
// ============================================================================

// ActionPhase 表示 Action 执行过程中的语义阶段
//
// 设计原则（参考 Kubernetes Conditions + Reason 模式）：
//  1. Phase 是"当前正在做什么"的机器可读标识
//  2. 同一 Status 下可以有多个不同的 Phase（如 running 下可以是 launching_container 或 authenticating）
//  3. Phase 按操作领域分组，每个领域有独立的阶段序列
//  4. Phase 变更不一定触发 Status 变更（细粒度进度追踪）
type ActionPhase string

// --- 通用阶段（所有操作类型共享）---
const (
	PhaseInitializing ActionPhase = "initializing" // 初始化执行环境
	PhaseFinalizing   ActionPhase = "finalizing"   // 清理和收尾
)

// --- 认证操作阶段（oauth / device_code / api_key）---
//
// OAuth 典型流程：
//
//	initializing → launching_container → authenticating →
//	waiting_oauth → verifying_credentials → extracting_token →
//	saving_credentials → finalizing
//
// DeviceCode 典型流程：
//
//	initializing → launching_container → requesting_device_code →
//	waiting_device_code → polling_token → extracting_token →
//	saving_credentials → finalizing
//
// API Key 典型流程（同步）：
//
//	initializing → validating_api_key → saving_credentials → finalizing
const (
	PhaseLaunchingContainer   ActionPhase = "launching_container"    // 启动认证容器
	PhaseAuthenticating       ActionPhase = "authenticating"         // 执行认证流程
	PhaseWaitingOAuth         ActionPhase = "waiting_oauth"          // 等待用户完成 OAuth 授权
	PhaseRequestingDeviceCode ActionPhase = "requesting_device_code" // 请求 Device Code
	PhaseWaitingDeviceCode    ActionPhase = "waiting_device_code"    // 等待用户输入 Device Code
	PhaseWaitingInput         ActionPhase = "waiting_input"          // 等待其他用户输入
	PhasePollingToken         ActionPhase = "polling_token"          // 轮询 Token 结果
	PhaseVerifyingCredentials ActionPhase = "verifying_credentials"  // 验证凭据有效性
	PhaseExtractingToken      ActionPhase = "extracting_token"       // 从容器提取 Token/Cookie
	PhaseSavingCredentials    ActionPhase = "saving_credentials"     // 保存凭据到 Volume
	PhaseValidatingAPIKey     ActionPhase = "validating_api_key"     // 验证 API Key 有效性
)

// --- 运行时操作阶段（runtime_create / start / stop / destroy）---
//
// Create 典型流程：
//
//	initializing → pulling_image → creating_container →
//	configuring_runtime → health_checking → finalizing
//
// Start 典型流程：
//
//	initializing → starting_runtime → health_checking → finalizing
//
// Stop 典型流程：
//
//	initializing → stopping_runtime → finalizing
//
// Destroy 典型流程：
//
//	initializing → stopping_runtime → removing_container →
//	cleaning_volumes → finalizing
const (
	PhasePullingImage       ActionPhase = "pulling_image"       // 拉取容器镜像
	PhaseCreatingContainer  ActionPhase = "creating_container"  // 创建容器
	PhaseConfiguringRuntime ActionPhase = "configuring_runtime" // 配置运行时环境
	PhaseStartingRuntime    ActionPhase = "starting_runtime"    // 启动运行时
	PhaseHealthChecking     ActionPhase = "health_checking"     // 健康检查
	PhaseStoppingRuntime    ActionPhase = "stopping_runtime"    // 停止运行时
	PhaseRemovingContainer  ActionPhase = "removing_container"  // 移除容器
	PhaseCleaningVolumes    ActionPhase = "cleaning_volumes"    // 清理存储卷
)

// ============================================================================
// Action - 操作执行实例
// ============================================================================

// Action 表示 Operation 的一次执行尝试
//
// 三层状态模型（参考 Kubernetes Pod Lifecycle）：
//   - Status：生命周期（assigned/running/waiting/success/failed/...）→ 驱动调度
//   - Phase：语义阶段（launching_container/waiting_oauth/...）→ 细粒度追踪
//   - Message：人类可读描述 → UI 展示
//
// 为什么需要 Action？
//  1. 支持重试：Operation 失败后可创建新 Action 重试
//  2. 保留历史：每次执行都有独立记录
//  3. 状态追踪：追踪每次执行的进度和结果
//
// 数据库表：actions
type Action struct {
	// 基本字段
	ID          string       `json:"id" db:"id"`                     // 唯一标识，格式：act-{random}
	OperationID string       `json:"operation_id" db:"operation_id"` // 关联的 Operation ID
	Status      ActionStatus `json:"status" db:"status"`             // 生命周期状态

	// 语义状态（Kubernetes Phase + Reason + Message 模式）
	Phase   ActionPhase `json:"phase,omitempty" db:"phase"`     // 当前语义阶段
	Message string      `json:"message,omitempty" db:"message"` // 人类可读状态描述

	// 执行信息
	Progress int             `json:"progress" db:"progress"`       // 执行进度 (0-100)
	Result   json.RawMessage `json:"result,omitempty" db:"result"` // 执行结果（JSON）
	Error    string          `json:"error,omitempty" db:"error"`   // 错误信息

	// 时间字段
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	StartedAt  *time.Time `json:"started_at,omitempty" db:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty" db:"finished_at"`

	// 关联数据（非数据库字段）
	Operation *Operation `json:"operation,omitempty" db:"-"` // 关联的 Operation
}

// ============================================================================
// AuthActionResult - 认证 Action 的结果
// ============================================================================

// AuthActionResult 是认证操作成功后的结果
type AuthActionResult struct {
	VolumeName    string `json:"volume_name"`              // Volume 名称
	VerifyURL     string `json:"verify_url,omitempty"`     // OAuth 验证 URL
	DeviceCode    string `json:"device_code,omitempty"`    // Device Code
	ContainerName string `json:"container_name,omitempty"` // 容器名称
}

// ============================================================================
// RuntimeActionResult - 运行时 Action 的结果
// ============================================================================

// RuntimeActionResult 是运行时操作的结果
type RuntimeActionResult struct {
	RuntimeID   string `json:"runtime_id,omitempty"`   // 运行时 ID
	ContainerID string `json:"container_id,omitempty"` // 容器 ID
	Status      string `json:"status,omitempty"`       // 运行时状态
}

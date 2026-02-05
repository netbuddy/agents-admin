// Package model 定义核心数据模型
//
// hitl.go 包含人在环路（Human-in-the-Loop）相关的数据模型定义：
//   - ApprovalRequest：审批请求（危险操作需要人工审批）
//   - ApprovalDecision：审批决定（approve/reject）
//   - HumanFeedback：人工反馈（guidance/correction/clarification）
//   - Intervention：干预（pause/resume/cancel/modify）
//   - Confirmation：确认请求（关键决策点需要用户确认）
package model

import (
	"encoding/json"
	"time"
)

// ============================================================================
// ApprovalType - 审批类型枚举
// ============================================================================

// ApprovalType 审批类型
type ApprovalType string

const (
	// ApprovalTypeDangerousOp 危险操作
	// 例如：删除文件、执行 shell 命令、修改系统配置
	ApprovalTypeDangerousOp ApprovalType = "dangerous_operation"

	// ApprovalTypeSensitiveAccess 敏感访问
	// 例如：访问密钥、读取敏感数据
	ApprovalTypeSensitiveAccess ApprovalType = "sensitive_access"

	// ApprovalTypeResourceExceeded 资源超限
	// 例如：Token 消耗超过阈值、执行时间过长
	ApprovalTypeResourceExceeded ApprovalType = "resource_exceeded"

	// ApprovalTypeExternalAccess 外部访问
	// 例如：网络外联、访问第三方 API
	ApprovalTypeExternalAccess ApprovalType = "external_access"
)

// ============================================================================
// ApprovalStatus - 审批状态枚举
// ============================================================================

// ApprovalStatus 审批状态
type ApprovalStatus string

const (
	// ApprovalStatusPending 待处理
	ApprovalStatusPending ApprovalStatus = "pending"

	// ApprovalStatusApproved 已批准
	ApprovalStatusApproved ApprovalStatus = "approved"

	// ApprovalStatusRejected 已拒绝
	ApprovalStatusRejected ApprovalStatus = "rejected"

	// ApprovalStatusExpired 已过期
	ApprovalStatusExpired ApprovalStatus = "expired"
)

// ============================================================================
// ApprovalRequest - 审批请求
// ============================================================================

// ApprovalRequest 审批请求
//
// 当 Agent 执行危险操作时，会创建审批请求并暂停执行，
// 等待用户审批后再继续或终止。
//
// 触发条件：
//   - 危险操作：删除文件、执行 shell 命令、修改系统配置
//   - 敏感访问：访问密钥、读取敏感数据、网络外联
//   - 资源超限：Token 消耗超过阈值、执行时间过长
//
// 状态流转：pending → approved / rejected / expired
type ApprovalRequest struct {
	// ID 唯一标识
	ID string `json:"id" db:"id"`

	// RunID 关联的 Run
	RunID string `json:"run_id" db:"run_id"`

	// Type 审批类型
	Type ApprovalType `json:"type" db:"type"`

	// Status 状态
	Status ApprovalStatus `json:"status" db:"status"`

	// Operation 请求的操作描述
	Operation string `json:"operation" db:"operation"`

	// Reason 请求原因
	Reason string `json:"reason" db:"reason"`

	// Context 操作上下文（详细信息，如命令、文件路径等）
	Context json.RawMessage `json:"context,omitempty" db:"context"`

	// ExpiresAt 过期时间（可选）
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// ResolvedAt 处理时间
	ResolvedAt *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
}

// IsPending 判断是否待处理
func (r *ApprovalRequest) IsPending() bool {
	return r.Status == ApprovalStatusPending
}

// IsExpired 判断是否已过期
func (r *ApprovalRequest) IsExpired() bool {
	if r.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*r.ExpiresAt)
}

// ============================================================================
// ApprovalDecision - 审批决定
// ============================================================================

// ApprovalDecision 审批决定
//
// 用户对 ApprovalRequest 的响应，包含决策结果和可选的附加指令。
type ApprovalDecision struct {
	// ID 唯一标识
	ID string `json:"id" db:"id"`

	// RequestID 关联的请求
	RequestID string `json:"request_id" db:"request_id"`

	// Decision 决策：approve 或 reject
	Decision string `json:"decision" db:"decision"`

	// DecidedBy 决策者 UserID
	DecidedBy string `json:"decided_by" db:"decided_by"`

	// Comment 审批意见（可选）
	Comment string `json:"comment,omitempty" db:"comment"`

	// Instructions 附加指令（批准时可选提供额外指导）
	Instructions string `json:"instructions,omitempty" db:"instructions"`

	// CreatedAt 决策时间
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// IsApproved 判断是否批准
func (d *ApprovalDecision) IsApproved() bool {
	return d.Decision == "approve"
}

// ============================================================================
// FeedbackType - 反馈类型枚举
// ============================================================================

// FeedbackType 反馈类型
type FeedbackType string

const (
	// FeedbackTypeGuidance 指导
	// 例如："请优先考虑性能"
	FeedbackTypeGuidance FeedbackType = "guidance"

	// FeedbackTypeCorrection 纠正
	// 例如："这个方向不对，应该..."
	FeedbackTypeCorrection FeedbackType = "correction"

	// FeedbackTypeClarification 澄清
	// 例如："我的意思是..."
	FeedbackTypeClarification FeedbackType = "clarification"
)

// ============================================================================
// HumanFeedback - 人工反馈
// ============================================================================

// HumanFeedback 人工反馈
//
// 用户在 Run 执行过程中随时提供的反馈，
// Agent 会在下一轮对话中考虑这些反馈。
//
// 类型：
//   - guidance：指导（"请优先考虑性能"）
//   - correction：纠正（"这个方向不对，应该..."）
//   - clarification：澄清（"我的意思是..."）
type HumanFeedback struct {
	// ID 唯一标识
	ID string `json:"id" db:"id"`

	// RunID 关联的 Run
	RunID string `json:"run_id" db:"run_id"`

	// Type 反馈类型
	Type FeedbackType `json:"type" db:"type"`

	// Content 反馈内容
	Content string `json:"content" db:"content"`

	// CreatedBy 创建者 UserID
	CreatedBy string `json:"created_by" db:"created_by"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// ProcessedAt 处理时间（Agent 确认收到反馈的时间）
	ProcessedAt *time.Time `json:"processed_at,omitempty" db:"processed_at"`
}

// IsProcessed 判断是否已处理
func (f *HumanFeedback) IsProcessed() bool {
	return f.ProcessedAt != nil
}

// ============================================================================
// InterventionAction - 干预动作枚举
// ============================================================================

// InterventionAction 干预动作
type InterventionAction string

const (
	// InterventionActionPause 暂停执行（保持状态，可恢复）
	InterventionActionPause InterventionAction = "pause"

	// InterventionActionResume 恢复执行
	InterventionActionResume InterventionAction = "resume"

	// InterventionActionCancel 取消执行（终止任务）
	InterventionActionCancel InterventionAction = "cancel"

	// InterventionActionModify 修改任务参数（如调整 Prompt、更换 Agent）
	InterventionActionModify InterventionAction = "modify"
)

// ============================================================================
// Intervention - 干预
// ============================================================================

// Intervention 干预
//
// 用户对正在执行的 Run 进行的主动干预。
//
// 场景：
//   - 用户发现问题需要立即干预
//   - 需要调整执行策略
//   - 紧急终止任务
type Intervention struct {
	// ID 唯一标识
	ID string `json:"id" db:"id"`

	// RunID 关联的 Run
	RunID string `json:"run_id" db:"run_id"`

	// Action 干预动作
	Action InterventionAction `json:"action" db:"action"`

	// Reason 干预原因（可选）
	Reason string `json:"reason,omitempty" db:"reason"`

	// Parameters 参数（modify 时使用，如新的 Prompt、新的 AgentID 等）
	Parameters json.RawMessage `json:"parameters,omitempty" db:"parameters"`

	// CreatedBy 创建者 UserID
	CreatedBy string `json:"created_by" db:"created_by"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// ExecutedAt 执行时间
	ExecutedAt *time.Time `json:"executed_at,omitempty" db:"executed_at"`
}

// IsExecuted 判断是否已执行
func (i *Intervention) IsExecuted() bool {
	return i.ExecutedAt != nil
}

// ============================================================================
// ConfirmationType - 确认类型枚举
// ============================================================================

// ConfirmationType 确认类型
type ConfirmationType string

const (
	// ConfirmationTypeDeployment 部署确认
	// 例如：部署到生产环境前确认
	ConfirmationTypeDeployment ConfirmationType = "deployment"

	// ConfirmationTypeDeletion 删除确认
	// 例如：删除重要数据前确认
	ConfirmationTypeDeletion ConfirmationType = "deletion"

	// ConfirmationTypePayment 付费确认
	// 例如：涉及付费操作前确认
	ConfirmationTypePayment ConfirmationType = "payment"

	// ConfirmationTypeIrreversible 不可逆操作确认
	// 例如：执行不可逆操作前确认
	ConfirmationTypeIrreversible ConfirmationType = "irreversible"
)

// ============================================================================
// ConfirmStatus - 确认状态枚举
// ============================================================================

// ConfirmStatus 确认状态
type ConfirmStatus string

const (
	// ConfirmStatusPending 待确认
	ConfirmStatusPending ConfirmStatus = "pending"

	// ConfirmStatusConfirmed 已确认
	ConfirmStatusConfirmed ConfirmStatus = "confirmed"

	// ConfirmStatusCancelled 已取消
	ConfirmStatusCancelled ConfirmStatus = "cancelled"
)

// ============================================================================
// Confirmation - 确认请求
// ============================================================================

// Confirmation 确认请求
//
// 关键决策点需要用户确认才能继续。
//
// 与 Approval 的区别：
//   - Approval 是安全检查（Agent 想做某事，需要批准）
//   - Confirmation 是决策确认（Agent 建议做某事，需要用户确认）
//
// 场景：
//   - deployment：部署到生产环境前确认
//   - deletion：删除重要数据前确认
//   - payment：涉及付费操作前确认
//   - irreversible：不可逆操作前确认
type Confirmation struct {
	// ID 唯一标识
	ID string `json:"id" db:"id"`

	// RunID 关联的 Run
	RunID string `json:"run_id" db:"run_id"`

	// Type 确认类型
	Type ConfirmationType `json:"type" db:"type"`

	// Message 确认消息（展示给用户）
	Message string `json:"message" db:"message"`

	// Status 状态
	Status ConfirmStatus `json:"status" db:"status"`

	// Options 可选项（如 ["继续", "取消", "稍后"]）
	Options []string `json:"options,omitempty" db:"options"`

	// SelectedOption 用户选择的选项
	SelectedOption *string `json:"selected_option,omitempty" db:"selected_option"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// ResolvedAt 处理时间
	ResolvedAt *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
}

// IsPending 判断是否待确认
func (c *Confirmation) IsPending() bool {
	return c.Status == ConfirmStatusPending
}

// IsConfirmed 判断是否已确认
func (c *Confirmation) IsConfirmed() bool {
	return c.Status == ConfirmStatusConfirmed
}

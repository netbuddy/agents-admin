// Package model 定义核心数据模型
//
// security.go 包含安全相关的数据模型定义：
//   - SecurityPolicyEntity：安全策略实体（数据库存储）
//   - ToolPermission：工具权限配置
//   - SandboxPolicy：沙箱策略配置
//   - Sandbox：沙箱实例
//   - SandboxType：沙箱类型枚举
//   - SandboxStatus：沙箱状态枚举
//
// 设计理念：
//   - SecurityPolicy 定义 Agent 的权限边界
//   - Sandbox 是 Agent 临时创建的隔离环境（执行危险操作时创建）
//   - ToolPermission 控制工具的访问权限（允许/拒绝/需审批）
//
// 注意：SecurityConfig、NetworkPolicy、ResourceLimits 定义在 task.go 中
// SecurityPolicyEntity 是可复用的安全策略实体，包含更完整的配置
package model

import (
	"time"
)

// ============================================================================
// ToolPermissionLevel - 工具权限级别
// ============================================================================

// ToolPermissionLevel 工具权限级别
type ToolPermissionLevel string

const (
	// ToolPermissionAllowed 允许使用
	ToolPermissionAllowed ToolPermissionLevel = "allowed"

	// ToolPermissionDenied 禁止使用
	ToolPermissionDenied ToolPermissionLevel = "denied"

	// ToolPermissionApprovalRequired 需要审批
	ToolPermissionApprovalRequired ToolPermissionLevel = "approval_required"
)

// ============================================================================
// ToolPermission - 工具权限配置
// ============================================================================

// ToolPermission 工具权限配置
//
// 定义单个工具或工具模式的访问权限：
//   - Tool：工具名称或模式（支持 * 通配符）
//   - Permission：权限级别
//   - Scopes：限定范围（如目录、域名）
type ToolPermission struct {
	// Tool 工具名称或模式
	// 支持通配符：file_* 匹配所有 file_ 开头的工具
	Tool string `json:"tool"`

	// Permission 权限级别
	Permission ToolPermissionLevel `json:"permission"`

	// Scopes 限定范围
	// 例如：file_write 工具只允许写入 /workspace 目录
	Scopes []string `json:"scopes,omitempty"`

	// ApprovalNote 审批说明（Permission=approval_required 时显示）
	ApprovalNote string `json:"approval_note,omitempty"`

	// Reason 权限设置原因
	Reason string `json:"reason,omitempty"`
}

// ============================================================================
// SandboxPolicy - 沙箱策略配置
// ============================================================================

// SandboxPolicy 沙箱策略配置
//
// 定义何时需要使用沙箱执行操作：
//   - RequiredForTools：这些工具必须在沙箱中执行
//   - DefaultType：默认沙箱类型
//   - AutoDestroy：操作完成后自动销毁
type SandboxPolicy struct {
	// RequiredForTools 需要沙箱的工具列表
	// 这些工具调用会自动创建沙箱
	RequiredForTools []string `json:"required_for_tools,omitempty"`

	// DefaultType 默认沙箱类型
	DefaultType SandboxType `json:"default_type,omitempty"`

	// AutoDestroy 操作完成后自动销毁
	AutoDestroy bool `json:"auto_destroy"`

	// MaxLifetime 最大存活时间（如 "1h", "30m"）
	MaxLifetime string `json:"max_lifetime,omitempty"`

	// ResourceLimits 沙箱资源限制
	ResourceLimits *ResourceLimits `json:"resource_limits,omitempty"`
}

// ============================================================================
// SecurityPolicyEntity - 安全策略实体
// ============================================================================

// SecurityPolicyEntity 安全策略实体
//
// SecurityPolicyEntity 是可复用的安全策略定义：
//   - 可以关联到 AgentTemplate 作为默认策略
//   - 可以关联到 Agent 覆盖模板策略
//   - 包含工具权限、资源限制、网络策略、沙箱策略
//
// 与 SecurityConfig 的区别：
//   - SecurityConfig 是简化版，用于 Task 级别的安全配置
//   - SecurityPolicyEntity 是完整版，用于 Agent 级别的策略管理
type SecurityPolicyEntity struct {
	// === 基础字段 ===

	// ID 唯一标识
	ID string `json:"id" db:"id"`

	// Name 策略名称
	Name string `json:"name" db:"name"`

	// Description 策略描述
	Description string `json:"description,omitempty" db:"description"`

	// === 权限配置 ===

	// ToolPermissions 工具权限列表
	ToolPermissions []ToolPermission `json:"tool_permissions,omitempty" db:"tool_permissions"`

	// === 资源限制 ===

	// ResourceLimits 资源限制配置
	ResourceLimits *ResourceLimits `json:"resource_limits,omitempty" db:"resource_limits"`

	// === 网络策略 ===

	// NetworkPolicy 网络访问策略
	NetworkPolicy *NetworkPolicy `json:"network_policy,omitempty" db:"network_policy"`

	// === 沙箱策略 ===

	// SandboxPolicy 沙箱策略
	SandboxPolicy *SandboxPolicy `json:"sandbox_policy,omitempty" db:"sandbox_policy"`

	// === 元数据 ===

	// IsBuiltin 是否内置策略
	IsBuiltin bool `json:"is_builtin" db:"is_builtin"`

	// Category 分类（如 development, production, testing）
	Category string `json:"category,omitempty" db:"category"`

	// === 时间戳 ===

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ============================================================================
// SecurityPolicyEntity 辅助方法
// ============================================================================

// HasToolPermission 检查是否有指定工具的权限配置
func (s *SecurityPolicyEntity) HasToolPermission(tool string) bool {
	for _, tp := range s.ToolPermissions {
		if tp.Tool == tool || tp.Tool == "*" {
			return true
		}
	}
	return false
}

// GetToolPermission 获取指定工具的权限
func (s *SecurityPolicyEntity) GetToolPermission(tool string) *ToolPermission {
	var wildcardMatch *ToolPermission
	for i := range s.ToolPermissions {
		tp := &s.ToolPermissions[i]
		if tp.Tool == tool {
			return tp
		}
		if tp.Tool == "*" {
			wildcardMatch = tp
		}
	}
	return wildcardMatch
}

// IsToolAllowed 检查工具是否被允许
func (s *SecurityPolicyEntity) IsToolAllowed(tool string) bool {
	tp := s.GetToolPermission(tool)
	if tp == nil {
		return true // 默认允许
	}
	return tp.Permission == ToolPermissionAllowed
}

// RequiresSandbox 检查工具是否需要沙箱
func (s *SecurityPolicyEntity) RequiresSandbox(tool string) bool {
	if s.SandboxPolicy == nil {
		return false
	}
	for _, t := range s.SandboxPolicy.RequiredForTools {
		if t == tool || t == "*" {
			return true
		}
	}
	return false
}

// ============================================================================
// SandboxType - 沙箱类型枚举
// ============================================================================

// SandboxType 沙箱类型
type SandboxType string

const (
	// SandboxTypeMicroVM MicroVM（如 Firecracker）
	// 最高隔离级别，独立内核
	SandboxTypeMicroVM SandboxType = "microvm"

	// SandboxTypeContainer 容器隔离（如 Docker）
	// 中等隔离级别，共享内核
	SandboxTypeContainer SandboxType = "container"

	// SandboxTypeGVisor gVisor 内核隔离
	// 高隔离级别，用户态内核
	SandboxTypeGVisor SandboxType = "gvisor"

	// SandboxTypeOSLevel OS 级别隔离（seccomp/namespaces）
	// 轻量级隔离
	SandboxTypeOSLevel SandboxType = "os"

	// SandboxTypeNone 无沙箱
	SandboxTypeNone SandboxType = "none"
)

// ============================================================================
// SandboxStatus - 沙箱状态枚举
// ============================================================================

// SandboxStatus 沙箱状态
type SandboxStatus string

const (
	// SandboxStatusCreating 创建中
	SandboxStatusCreating SandboxStatus = "creating"

	// SandboxStatusReady 就绪
	SandboxStatusReady SandboxStatus = "ready"

	// SandboxStatusRunning 运行中
	SandboxStatusRunning SandboxStatus = "running"

	// SandboxStatusStopping 停止中
	SandboxStatusStopping SandboxStatus = "stopping"

	// SandboxStatusStopped 已停止
	SandboxStatusStopped SandboxStatus = "stopped"

	// SandboxStatusDestroyed 已销毁
	SandboxStatusDestroyed SandboxStatus = "destroyed"

	// SandboxStatusError 错误
	SandboxStatusError SandboxStatus = "error"
)

// ============================================================================
// Sandbox - 沙箱实例
// ============================================================================

// Sandbox 沙箱实例
//
// Sandbox 是 Agent 临时创建的隔离环境：
//   - 执行危险操作时自动创建
//   - 操作完成后可自动销毁
//   - 提供文件系统、网络隔离
//
// 生命周期：
//
//	创建 → creating → ready → running → stopping → stopped → destroyed
//	                                       ↓
//	                                     error
type Sandbox struct {
	// === 基础字段 ===

	// ID 唯一标识
	ID string `json:"id" db:"id"`

	// AgentID 关联的 Agent ID
	AgentID string `json:"agent_id" db:"agent_id"`

	// Type 沙箱类型
	Type SandboxType `json:"type" db:"type"`

	// Status 沙箱状态
	Status SandboxStatus `json:"status" db:"status"`

	// === 隔离配置 ===

	// Isolation 隔离级别描述
	Isolation string `json:"isolation,omitempty" db:"isolation"`

	// FSRoot 文件系统根目录
	FSRoot string `json:"fs_root,omitempty" db:"fs_root"`

	// NetNamespace 网络命名空间
	NetNamespace string `json:"net_ns,omitempty" db:"net_ns"`

	// === 资源配置 ===

	// ResourceLimits 资源限制
	ResourceLimits *ResourceLimits `json:"resource_limits,omitempty" db:"resource_limits"`

	// === 关联 ===

	// RuntimeID 关联的 Runtime ID（如果在容器中创建）
	RuntimeID *string `json:"runtime_id,omitempty" db:"runtime_id"`

	// NodeID 所在节点 ID
	NodeID string `json:"node_id" db:"node_id"`

	// === 生命周期 ===

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// StartedAt 启动时间
	StartedAt *time.Time `json:"started_at,omitempty" db:"started_at"`

	// ExpiresAt 过期时间
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`

	// DestroyedAt 销毁时间
	DestroyedAt *time.Time `json:"destroyed_at,omitempty" db:"destroyed_at"`
}

// ============================================================================
// Sandbox 辅助方法
// ============================================================================

// IsActive 判断沙箱是否活跃
func (s *Sandbox) IsActive() bool {
	return s.Status == SandboxStatusReady || s.Status == SandboxStatusRunning
}

// IsTerminated 判断沙箱是否已终止
func (s *Sandbox) IsTerminated() bool {
	return s.Status == SandboxStatusStopped || s.Status == SandboxStatusDestroyed || s.Status == SandboxStatusError
}

// IsExpired 判断沙箱是否已过期
func (s *Sandbox) IsExpired() bool {
	if s.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*s.ExpiresAt)
}

// CanDestroy 判断沙箱是否可以销毁
func (s *Sandbox) CanDestroy() bool {
	return s.Status != SandboxStatusDestroyed && s.Status != SandboxStatusCreating
}

// ============================================================================
// 内置安全策略
// ============================================================================

// BuiltinSecurityPolicies 内置安全策略
var BuiltinSecurityPolicies = []SecurityPolicyEntity{
	{
		ID:          "builtin-strict",
		Name:        "严格策略",
		Description: "最小权限原则，禁止危险操作",
		ToolPermissions: []ToolPermission{
			{Tool: "file_read", Permission: ToolPermissionAllowed},
			{Tool: "file_write", Permission: ToolPermissionApprovalRequired, ApprovalNote: "文件写入需要审批"},
			{Tool: "file_delete", Permission: ToolPermissionDenied, Reason: "严格模式禁止删除文件"},
			{Tool: "command_execute", Permission: ToolPermissionDenied, Reason: "严格模式禁止执行命令"},
			{Tool: "network_*", Permission: ToolPermissionDenied, Reason: "严格模式禁止网络访问"},
		},
		ResourceLimits: &ResourceLimits{
			MaxCPU:       "1.0",
			MaxMemory:    "2Gi",
			MaxDisk:      "5Gi",
			MaxProcesses: 50,
			MaxOpenFiles: 256,
		},
		NetworkPolicy: &NetworkPolicy{
			AllowInternet: false,
		},
		IsBuiltin: true,
		Category:  "security",
	},
	{
		ID:          "builtin-standard",
		Name:        "标准策略",
		Description: "平衡安全与便利，适用于开发环境",
		ToolPermissions: []ToolPermission{
			{Tool: "file_*", Permission: ToolPermissionAllowed},
			{Tool: "command_execute", Permission: ToolPermissionApprovalRequired, ApprovalNote: "命令执行需要确认"},
			{Tool: "network_outbound", Permission: ToolPermissionAllowed, Scopes: []string{"github.com", "*.npmjs.org"}},
		},
		ResourceLimits: &ResourceLimits{
			MaxCPU:       "2.0",
			MaxMemory:    "4Gi",
			MaxDisk:      "20Gi",
			MaxProcesses: 100,
			MaxOpenFiles: 1024,
		},
		NetworkPolicy: &NetworkPolicy{
			AllowInternet:  true,
			AllowedDomains: []string{"github.com", "*.npmjs.org", "*.pypi.org", "*.golang.org"},
		},
		IsBuiltin: true,
		Category:  "development",
	},
	{
		ID:          "builtin-permissive",
		Name:        "宽松策略",
		Description: "较少限制，适用于受信环境",
		ToolPermissions: []ToolPermission{
			{Tool: "*", Permission: ToolPermissionAllowed},
		},
		ResourceLimits: &ResourceLimits{
			MaxCPU:       "4.0",
			MaxMemory:    "8Gi",
			MaxDisk:      "50Gi",
			MaxProcesses: 500,
			MaxOpenFiles: 4096,
		},
		NetworkPolicy: &NetworkPolicy{
			AllowInternet: true,
		},
		IsBuiltin: true,
		Category:  "trusted",
	},
}

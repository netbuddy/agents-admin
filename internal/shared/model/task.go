// Package model 定义核心数据模型
//
// task.go 包含任务相关的数据模型定义：
//   - TaskTemplate：任务模板（静态定义，可复用）
//   - Task：任务实例（动态实例，包含运行时状态）
//   - TaskType：任务类型枚举
//   - TaskStatus：任务状态枚举
//   - WorkspaceConfig：工作空间配置
//   - SecurityConfig：安全配置
//
// 注意：TaskContext/ContextItem/Message 已移至 context.go
package model

import (
	"time"
)

// ============================================================================
// TaskTemplate - 任务模板（静态定义）
// ============================================================================

// TaskTemplate 任务模板
//
// TaskTemplate 是可复用的任务定义模板，包含：
//   - 基本信息：名称、描述、类型
//   - 提示词模板：支持变量插值
//   - 默认配置：工作空间、安全策略等
//
// TaskTemplate 与 Task 的关系：
//   - TaskTemplate 是静态定义（可以保存、复用、分享）
//   - Task 是动态实例（包含运行时状态、关联的 Agent 等）
//   - 类似于 AgentTemplate + Agent 的设计模式
//
// 使用场景：
//   - 系统预置的常用任务模板（如"代码审查"、"Bug修复"、"重构"）
//   - 用户自定义的模板
//   - 团队共享的标准化任务流程
type TaskTemplate struct {
	// === 基础字段 ===

	// ID 唯一标识
	ID string `json:"id" db:"id"`

	// Name 模板名称
	Name string `json:"name" db:"name"`

	// Description 模板描述
	Description string `json:"description,omitempty" db:"description"`

	// Type 任务类型
	Type TaskType `json:"type" db:"type"`

	// === 提示词配置 ===

	// PromptTemplate 提示词模板（支持变量插值）
	// 可以直接内嵌内容，或引用 PromptTemplate ID
	PromptTemplate *PromptTemplate `json:"prompt_template,omitempty" db:"prompt_template"`

	// PromptTemplateID 引用的提示词模板 ID（与 PromptTemplate 二选一）
	PromptTemplateID *string `json:"prompt_template_id,omitempty" db:"prompt_template_id"`

	// === 默认配置 ===

	// DefaultWorkspace 默认工作空间配置
	DefaultWorkspace *WorkspaceConfig `json:"default_workspace,omitempty" db:"default_workspace"`

	// DefaultSecurity 默认安全配置
	DefaultSecurity *SecurityConfig `json:"default_security,omitempty" db:"default_security"`

	// DefaultLabels 默认标签
	DefaultLabels map[string]string `json:"default_labels,omitempty" db:"default_labels"`

	// === 变量定义 ===

	// Variables 模板变量定义（用于 PromptTemplate 中的变量）
	Variables []TemplateVariable `json:"variables,omitempty" db:"variables"`

	// === 元数据 ===

	// IsBuiltin 是否内置模板
	IsBuiltin bool `json:"is_builtin" db:"is_builtin"`

	// Category 分类（如 development, testing, documentation）
	Category string `json:"category,omitempty" db:"category"`

	// Tags 标签
	Tags []string `json:"tags,omitempty" db:"tags"`

	// Source 来源（builtin/custom/shared）
	Source string `json:"source,omitempty" db:"source"`

	// === 时间戳 ===

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ============================================================================
// TaskTemplate 辅助方法
// ============================================================================

// HasPromptTemplate 判断是否有提示词模板
func (t *TaskTemplate) HasPromptTemplate() bool {
	return t.PromptTemplate != nil || (t.PromptTemplateID != nil && *t.PromptTemplateID != "")
}

// CreateTask 从模板创建任务实例
//
// 参数：
//   - name: 任务名称
//   - variables: 变量值（用于填充提示词模板）
//
// 返回：
//   - 新创建的 Task 实例（未设置 ID 和时间戳，调用者需要设置）
func (t *TaskTemplate) CreateTask(name string, variables map[string]interface{}) *Task {
	task := &Task{
		Name:        name,
		Description: t.Description,
		Type:        t.Type, // 从模板复制 Type（冗余存储，用于查询优化）
		Workspace:   t.DefaultWorkspace,
		Security:    t.DefaultSecurity,
		Labels:      t.DefaultLabels,
		TemplateID:  &t.ID,
	}

	// 如果有内嵌的 PromptTemplate，创建 Prompt 实例
	if t.PromptTemplate != nil {
		task.Prompt = &Prompt{
			TemplateID:  &t.PromptTemplate.ID,
			Content:     t.PromptTemplate.Content, // TODO: 变量插值
			Description: t.PromptTemplate.Description,
			Variables:   variables,
		}
	}

	return task
}

// ============================================================================
// TaskType - 任务类型枚举
// ============================================================================

// TaskType 任务类型，影响默认配置和处理策略
type TaskType string

const (
	// TaskTypeGeneral 通用任务：默认类型，无特殊配置
	TaskTypeGeneral TaskType = "general"

	// TaskTypeDevelopment 开发任务：需要代码仓库和文件读写权限
	TaskTypeDevelopment TaskType = "development"

	// TaskTypeResearch 研究任务：纯对话，无工作空间
	TaskTypeResearch TaskType = "research"

	// TaskTypeOperation 运维任务：需要系统权限，SSH 访问
	TaskTypeOperation TaskType = "operation"

	// TaskTypeAutomation 自动化任务：后台执行，有资源限制
	TaskTypeAutomation TaskType = "automation"
)

// ============================================================================
// TaskStatus - 任务状态
// ============================================================================

// TaskStatus 表示任务（Task）的整体状态
//
// Task 是任务定义，TaskStatus 反映任务的整体进展：
//   - pending：任务已创建，尚未开始执行
//   - in_progress：任务正在处理中（有活跃的执行）
//   - completed：任务目标已达成
//   - failed：任务执行失败（所有重试均失败）
//   - cancelled：任务被用户取消
//
// 注意：TaskStatus 与 RunStatus 是不同层次的状态：
//   - TaskStatus 描述任务的整体目标是否达成（pending/in_progress/completed/failed/cancelled）
//   - RunStatus 描述单次执行的进展（queued/assigned/running/done/failed/cancelled/timeout）
//
// 术语规范：
//   - Task = 任务（定义"做什么"）
//   - Run = 执行（记录"一次尝试"）
type TaskStatus string

const (
	// TaskStatusPending 待处理：任务已创建，等待首次执行
	TaskStatusPending TaskStatus = "pending"

	// TaskStatusInProgress 处理中：至少有一个执行（Run）正在进行
	TaskStatusInProgress TaskStatus = "in_progress"

	// TaskStatusCompleted 已完成：任务目标已达成
	TaskStatusCompleted TaskStatus = "completed"

	// TaskStatusFailed 已失败：任务无法完成（可能需要人工介入）
	TaskStatusFailed TaskStatus = "failed"

	// TaskStatusCancelled 已取消：用户主动取消任务
	TaskStatusCancelled TaskStatus = "cancelled"
)

// ============================================================================
// WorkspaceType - 工作空间类型枚举
// ============================================================================

// WorkspaceType 工作空间类型
type WorkspaceType string

const (
	// WorkspaceTypeGit Git 仓库工作空间
	WorkspaceTypeGit WorkspaceType = "git"

	// WorkspaceTypeLocal 本地目录工作空间
	WorkspaceTypeLocal WorkspaceType = "local"

	// WorkspaceTypeRemote 远程系统工作空间
	WorkspaceTypeRemote WorkspaceType = "remote"

	// WorkspaceTypeVolume 持久化卷工作空间
	WorkspaceTypeVolume WorkspaceType = "volume"
)

// ============================================================================
// SecurityPolicy - 安全策略枚举
// ============================================================================

// SecurityPolicy 安全策略等级
type SecurityPolicy string

const (
	// SecurityPolicyStrict 严格策略：最小权限，需要审批
	SecurityPolicyStrict SecurityPolicy = "strict"

	// SecurityPolicyStandard 标准策略：平衡安全与便利
	SecurityPolicyStandard SecurityPolicy = "standard"

	// SecurityPolicyPermissive 宽松策略：较少限制，适用于受信环境
	SecurityPolicyPermissive SecurityPolicy = "permissive"
)

// ============================================================================
// WorkspaceConfig - 工作空间配置
// ============================================================================

// WorkspaceConfig 工作空间配置
//
// 工作空间定义任务执行的"场所"：
//   - Git：代码仓库，支持分支/提交指定
//   - Local：本地目录，直接挂载
//   - Remote：远程系统，SSH/容器访问
//   - Volume：持久化卷，跨任务共享
type WorkspaceConfig struct {
	// Type 工作空间类型
	Type WorkspaceType `json:"type"`

	// Git 配置（Type=git 时使用）
	Git *GitConfig `json:"git,omitempty"`

	// Local 配置（Type=local 时使用）
	Local *LocalConfig `json:"local,omitempty"`

	// Remote 配置（Type=remote 时使用）
	Remote *RemoteConfig `json:"remote,omitempty"`

	// Volume 配置（Type=volume 时使用）
	Volume *VolumeConfig `json:"volume,omitempty"`
}

// GitConfig Git 仓库配置
type GitConfig struct {
	// URL 仓库地址（HTTPS 或 SSH）
	URL string `json:"url"`

	// Branch 分支名称
	Branch string `json:"branch,omitempty"`

	// Commit 指定的提交 SHA（可选）
	Commit string `json:"commit,omitempty"`

	// Depth 克隆深度（shallow clone），0 表示完整克隆
	Depth int `json:"depth,omitempty"`
}

// LocalConfig 本地目录配置
type LocalConfig struct {
	// Path 本地目录路径
	Path string `json:"path"`

	// ReadOnly 是否只读挂载
	ReadOnly bool `json:"read_only,omitempty"`
}

// RemoteConfig 远程系统配置
type RemoteConfig struct {
	// Host 远程主机地址
	Host string `json:"host"`

	// Port SSH 端口，默认 22
	Port int `json:"port,omitempty"`

	// User 用户名
	User string `json:"user,omitempty"`

	// CredentialRef 凭据引用（从密钥管理系统获取）
	CredentialRef string `json:"credential_ref,omitempty"`
}

// VolumeConfig 持久化卷配置
type VolumeConfig struct {
	// Name 卷名称
	Name string `json:"name"`

	// SubPath 卷内子路径
	SubPath string `json:"sub_path,omitempty"`
}

// ============================================================================
// SecurityConfig - 安全配置
// ============================================================================

// SecurityConfig 定义任务执行的安全约束
//
// SecurityConfig 回答"允许做什么"的问题：
//   - Policy：预定义的安全策略等级
//   - Permissions：细粒度权限控制
//   - Limits：资源使用限制
//   - Network：网络访问控制
type SecurityConfig struct {
	// Policy 安全策略等级
	Policy SecurityPolicy `json:"policy"`

	// Permissions 权限列表（在 Policy 基础上调整）
	// 例如：["file_read", "file_write", "network_outbound"]
	Permissions []string `json:"permissions,omitempty"`

	// DeniedPermissions 明确禁止的权限
	// 优先级高于 Permissions
	DeniedPermissions []string `json:"denied_permissions,omitempty"`

	// Network 网络配置
	Network *NetworkPolicy `json:"network,omitempty"`

	// Limits 资源限制
	Limits *ResourceLimits `json:"limits,omitempty"`

	// RequireApproval 需要人工审批的操作列表
	// 例如：["file_delete", "command_execute"]
	RequireApproval []string `json:"require_approval,omitempty"`
}

// NetworkPolicy 网络访问策略
type NetworkPolicy struct {
	// AllowInternet 是否允许访问互联网
	AllowInternet bool `json:"allow_internet"`

	// AllowedDomains 允许访问的域名列表（白名单）
	AllowedDomains []string `json:"allowed_domains,omitempty"`

	// DeniedDomains 禁止访问的域名列表（黑名单）
	DeniedDomains []string `json:"denied_domains,omitempty"`

	// AllowedPorts 允许的端口列表
	AllowedPorts []int `json:"allowed_ports,omitempty"`
}

// ResourceLimits 资源限制配置
type ResourceLimits struct {
	// MaxCPU 最大 CPU 核数（如 "2.0"）
	MaxCPU string `json:"max_cpu,omitempty"`

	// MaxMemory 最大内存（如 "4Gi"）
	MaxMemory string `json:"max_memory,omitempty"`

	// MaxDisk 最大磁盘使用（如 "10Gi"）
	MaxDisk string `json:"max_disk,omitempty"`

	// MaxNetwork 最大网络带宽（如 "100Mbps"）
	MaxNetwork string `json:"max_network,omitempty"`

	// MaxProcesses 最大进程数
	MaxProcesses int `json:"max_processes,omitempty"`

	// MaxOpenFiles 最大打开文件数
	MaxOpenFiles int `json:"max_open_files,omitempty"`
}

// ============================================================================
// Task - 扁平化的任务结构（合并原 TaskSpec）
// ============================================================================

// Task 表示一个任务实例（动态概念）
//
// Task 是任务的"运行时实例"，包含执行状态和运行时信息。
// 与 TaskTemplate（静态模板）形成 Template + Instance 的关系，
// 与 AgentTemplate + Agent 的设计模式一致。
//
// 设计原则：
//   - TaskTemplate 定义"做什么类型的任务"（Type、默认配置）
//   - Task 定义"这次具体做什么"（Prompt、运行时状态）
//   - 配置继承：Workspace/Security/Labels 可从模板继承，也可覆盖
//
// 字段归属原则：
//   - TaskTemplate（静态）：Type、DefaultWorkspace、DefaultSecurity、DefaultLabels
//   - Task（动态）：Status、Prompt、Context、AgentID、ParentID
//   - 可继承可覆盖：Name、Description、Workspace、Security、Labels
//
// 字段分组：
//  1. 基础字段：ID, Name, Description, Status
//  2. 核心内容：Prompt, Context
//  3. 配置字段（可继承）：Workspace, Security, Labels
//  4. 关联字段：TemplateID, AgentID, ParentID
//  5. 时间戳：CreatedAt, UpdatedAt
type Task struct {
	// === 基础字段 ===

	// ID 任务唯一标识
	ID string `json:"id" db:"id"`

	// Name 任务名称（可从模板继承）
	Name string `json:"name" db:"name"`

	// Description 任务描述（可从模板继承）
	Description string `json:"description,omitempty" db:"description"`

	// Status 任务状态（纯运行时字段）
	Status TaskStatus `json:"status" db:"status"`

	// Type 任务类型（冗余字段，从 TaskTemplate 派生）
	// 注意：Type 的主定义在 TaskTemplate 中，此处仅用于查询优化和向后兼容
	// 创建 Task 时，如果有 TemplateID，应从模板复制 Type
	Type TaskType `json:"type" db:"type"`

	// === 核心内容（纯运行时字段）===

	// Prompt 任务提示词（结构化类型，从 PromptTemplate 实例化）
	// 包含：Content（内容）、Description（说明）、Variables（变量）、ContextStrategy（上下文策略）
	Prompt *Prompt `json:"prompt,omitempty" db:"prompt"`

	// Context 任务上下文（纯运行时字段）
	Context *TaskContext `json:"context,omitempty" db:"context"`

	// === 配置字段（可从模板继承，也可覆盖）===

	// Workspace 工作空间配置（未设置时使用模板的 DefaultWorkspace）
	Workspace *WorkspaceConfig `json:"workspace,omitempty" db:"workspace"`

	// Security 安全约束配置（未设置时使用模板的 DefaultSecurity）
	Security *SecurityConfig `json:"security,omitempty" db:"security"`

	// Labels 任务标签（与模板的 DefaultLabels 合并）
	Labels map[string]string `json:"labels,omitempty" db:"labels"`

	// === 关联字段 ===

	// TemplateID 关联的任务模板 ID（通过模板获取 Type 和默认配置）
	TemplateID *string `json:"template_id,omitempty" db:"template_id"`

	// AgentID 执行 Agent ID
	AgentID *string `json:"agent_id,omitempty" db:"agent_id"`

	// ParentID 父任务 ID（顶层任务为空）
	ParentID *string `json:"parent_id,omitempty" db:"parent_id"`

	// === 时间戳 ===

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ============================================================================
// 辅助方法
// ============================================================================

// GetPromptContent 获取提示词内容（便捷方法）
func (t *Task) GetPromptContent() string {
	if t.Prompt == nil {
		return ""
	}
	return t.Prompt.Content
}

// HasPrompt 判断是否有提示词
func (t *Task) HasPrompt() bool {
	return t.Prompt != nil && t.Prompt.Content != ""
}

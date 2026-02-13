// Package model 定义核心数据模型
//
// agent.go 包含智能体相关的数据模型定义：
//   - AgentTemplate：智能体模板/配置蓝图
//   - Agent：智能体实体/运行态
//   - AgentStatus：智能体状态枚举
//   - AgentModelType：智能体模型类型枚举
//   - Runtime：运行时环境
//   - RuntimeType：运行时类型枚举
//   - RuntimeStatus：运行时状态枚举
//
// 设计理念：
//   - AgentTemplate 是配置蓝图，定义能力、性格、技能
//   - Agent 是运行实例，基于模板创建，有状态、有记忆
//   - Runtime 是运行环境，Agent 运行于 Runtime 中（容器/VM/物理机）
package model

import (
	"encoding/json"
	"time"
)

// ============================================================================
// AgentModelType - 智能体模型类型枚举
// ============================================================================

// AgentModelType Agent 模型类型枚举
type AgentModelType string

const (
	// AgentModelTypeClaude Claude 模型
	AgentModelTypeClaude AgentModelType = "claude"

	// AgentModelTypeGemini Gemini 模型
	AgentModelTypeGemini AgentModelType = "gemini"

	// AgentModelTypeQwen Qwen 模型
	AgentModelTypeQwen AgentModelType = "qwen"

	// AgentModelTypeCodex OpenAI Codex 模型
	AgentModelTypeCodex AgentModelType = "codex"

	// AgentModelTypeCustom 自定义模型
	AgentModelTypeCustom AgentModelType = "custom"
)

// ============================================================================
// AgentStatus - 智能体状态枚举
// ============================================================================

// AgentStatus Agent 状态
type AgentStatus string

const (
	// AgentStatusPending 待创建：等待资源分配
	AgentStatusPending AgentStatus = "pending"

	// AgentStatusStarting 启动中：正在初始化
	AgentStatusStarting AgentStatus = "starting"

	// AgentStatusRunning 运行中：正常运行，可接受任务
	AgentStatusRunning AgentStatus = "running"

	// AgentStatusIdle 空闲：运行中但无任务
	AgentStatusIdle AgentStatus = "idle"

	// AgentStatusBusy 繁忙：正在执行任务
	AgentStatusBusy AgentStatus = "busy"

	// AgentStatusStopping 停止中：正在关闭
	AgentStatusStopping AgentStatus = "stopping"

	// AgentStatusStopped 已停止：已关闭
	AgentStatusStopped AgentStatus = "stopped"

	// AgentStatusError 错误：发生错误
	AgentStatusError AgentStatus = "error"
)

// ============================================================================
// AgentTemplate - 智能体模板
// ============================================================================

// AgentTemplate 智能体模板
//
// AgentTemplate 是智能体的配置蓝图，定义：
//   - 基本信息：名称、类型、角色、描述
//   - 身份与性格：性格特征、系统提示词
//   - 能力配置：技能、工具、MCP 服务
//   - 运行参数：模型、温度、上下文限制
//   - 安全配置：默认安全策略
//
// AgentTemplate 与 Agent 的关系：
//   - AgentTemplate 是静态配置（可保存、复用、分享）
//   - Agent 是运行实例（有状态、有记忆）
//   - 类似于 TaskTemplate + Task 的设计模式
type AgentTemplate struct {
	// === 基础字段 ===

	// ID 唯一标识
	ID string `json:"id" bson:"_id" db:"id"`

	// Name 模板名称
	Name string `json:"name" bson:"name" db:"name"`

	// Type 模型类型（claude/gemini/qwen/codex/custom）
	Type AgentModelType `json:"type" bson:"type" db:"type"`

	// Role 角色定位（如"代码助手"、"运维专家"）
	Role string `json:"role,omitempty" bson:"role,omitempty" db:"role"`

	// Description 模板描述
	Description string `json:"description,omitempty" bson:"description,omitempty" db:"description"`

	// === 身份与性格 ===

	// Personality 性格特征列表
	Personality []string `json:"personality,omitempty" bson:"personality,omitempty" db:"personality"`

	// SystemPrompt 系统提示词
	SystemPrompt string `json:"system_prompt,omitempty" bson:"system_prompt,omitempty" db:"system_prompt"`

	// === 能力配置 ===

	// Skills 技能 ID 列表
	Skills []string `json:"skills,omitempty" bson:"skills,omitempty" db:"skills"`

	// Tools 工具配置（允许/拒绝/需审批的工具列表）
	Tools json.RawMessage `json:"tools,omitempty" bson:"tools,omitempty" db:"tools"`

	// MCPServers MCP 服务器配置
	MCPServers json.RawMessage `json:"mcp_servers,omitempty" bson:"mcp_servers,omitempty" db:"mcp_servers"`

	// Documents 参考文档
	Documents json.RawMessage `json:"documents,omitempty" bson:"documents,omitempty" db:"documents"`

	// === 自动化配置 ===

	// Gambits IF-THEN 规则（自动响应规则）
	Gambits json.RawMessage `json:"gambits,omitempty" bson:"gambits,omitempty" db:"gambits"`

	// Hooks 事件钩子
	Hooks json.RawMessage `json:"hooks,omitempty" bson:"hooks,omitempty" db:"hooks"`

	// === 运行参数 ===

	// Model 具体模型名称（如 "claude-3-opus", "gemini-pro"）
	Model string `json:"model,omitempty" bson:"model,omitempty" db:"model"`

	// Temperature 温度参数（0-1）
	Temperature float64 `json:"temperature,omitempty" bson:"temperature,omitempty" db:"temperature"`

	// MaxContext 最大上下文 Token 数
	MaxContext int `json:"max_context,omitempty" bson:"max_context,omitempty" db:"max_context"`

	// === 安全配置 ===

	// DefaultSecurityPolicy 默认安全策略 ID
	DefaultSecurityPolicyID *string `json:"default_security_policy_id,omitempty" bson:"default_security_policy_id,omitempty" db:"default_security_policy_id"`

	// === 元数据 ===

	// IsBuiltin 是否内置模板
	IsBuiltin bool `json:"is_builtin" bson:"is_builtin" db:"is_builtin"`

	// Category 分类
	Category string `json:"category,omitempty" bson:"category,omitempty" db:"category"`

	// Tags 标签
	Tags []string `json:"tags,omitempty" bson:"tags,omitempty" db:"tags"`

	// === 时间戳 ===

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at" bson:"created_at" db:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at" db:"updated_at"`
}

// ============================================================================
// AgentTemplate 辅助方法
// ============================================================================

// CreateAgent 从模板创建 Agent 实例
//
// 参数：
//   - name: Agent 名称
//   - accountID: 绑定的账号 ID
//
// 返回：
//   - 新创建的 Agent 实例（未设置 ID 和时间戳）
func (t *AgentTemplate) CreateAgent(name string, accountID string) *Agent {
	return &Agent{
		Name:       name,
		TemplateID: &t.ID,
		AccountID:  accountID,
		Status:     AgentStatusPending,
		Type:       t.Type, // 从模板复制类型
	}
}

// HasSkills 判断是否有技能配置
func (t *AgentTemplate) HasSkills() bool {
	return len(t.Skills) > 0
}

// ============================================================================
// Agent - 智能体实体
// ============================================================================

// Agent 智能体实体
//
// Agent 是智能体的运行实例：
//   - 基于 AgentTemplate 创建
//   - 绑定到特定账号（认证凭据）
//   - 可运行于 Runtime（容器/VM）
//   - 有状态、可配置覆盖
//
// Agent 生命周期：
//
//	pending → starting → running/idle ⇄ busy
//	                          ↓
//	                      stopping → stopped
//	                          ↓
//	                        error
type Agent struct {
	// === 基础字段 ===

	// ID 唯一标识
	ID string `json:"id" bson:"_id" db:"id"`

	// Name Agent 名称
	Name string `json:"name" bson:"name" db:"name"`

	// TemplateID 关联的模板 ID
	TemplateID *string `json:"template_id,omitempty" bson:"template_id,omitempty" db:"template_id"`

	// Type 模型类型（从模板继承）
	Type AgentModelType `json:"type" bson:"type" db:"type"`

	// === 绑定关系 ===

	// AccountID 绑定的账号 ID
	AccountID string `json:"account_id" bson:"account_id" db:"account_id"`

	// RuntimeID 运行时 ID（运行中时填充）
	RuntimeID *string `json:"runtime_id,omitempty" bson:"runtime_id,omitempty" db:"runtime_id"`

	// NodeID 所在节点 ID
	NodeID *string `json:"node_id,omitempty" bson:"node_id,omitempty" db:"node_id"`

	// === 状态 ===

	// Status 当前状态
	Status AgentStatus `json:"status" bson:"status" db:"status"`

	// CurrentTaskID 当前执行的任务 ID
	CurrentTaskID *string `json:"current_task_id,omitempty" bson:"current_task_id,omitempty" db:"current_task_id"`

	// === 配置覆盖 ===

	// ConfigOverrides 配置覆盖（覆盖模板中的配置）
	ConfigOverrides json.RawMessage `json:"config_overrides,omitempty" bson:"config_overrides,omitempty" db:"config_overrides"`

	// SecurityPolicyID 安全策略 ID（覆盖模板的默认策略）
	SecurityPolicyID *string `json:"security_policy_id,omitempty" bson:"security_policy_id,omitempty" db:"security_policy_id"`

	// === 记忆与能力 ===

	// MemoryEnabled 是否启用记忆
	MemoryEnabled bool `json:"memory_enabled" bson:"memory_enabled" db:"memory_enabled"`

	// === 时间戳 ===

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at" bson:"created_at" db:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at" db:"updated_at"`

	// LastActiveAt 最后活跃时间
	LastActiveAt *time.Time `json:"last_active_at,omitempty" bson:"last_active_at,omitempty" db:"last_active_at"`
}

// ============================================================================
// Agent 辅助方法
// ============================================================================

// IsRunning 判断 Agent 是否正在运行
func (a *Agent) IsRunning() bool {
	return a.Status == AgentStatusRunning || a.Status == AgentStatusIdle || a.Status == AgentStatusBusy
}

// CanAcceptTask 判断 Agent 是否可以接受新任务
func (a *Agent) CanAcceptTask() bool {
	return a.Status == AgentStatusRunning || a.Status == AgentStatusIdle
}

// IsBusy 判断 Agent 是否繁忙
func (a *Agent) IsBusy() bool {
	return a.Status == AgentStatusBusy
}

// CanStart 判断 Agent 是否可以启动
func (a *Agent) CanStart() bool {
	return a.Status == AgentStatusPending || a.Status == AgentStatusStopped || a.Status == AgentStatusError
}

// CanStop 判断 Agent 是否可以停止
func (a *Agent) CanStop() bool {
	return a.IsRunning()
}

// ============================================================================
// RuntimeType - 运行时类型枚举
// ============================================================================

// RuntimeType 运行时类型
type RuntimeType string

const (
	// RuntimeTypeContainer 容器运行时（Docker）
	RuntimeTypeContainer RuntimeType = "container"

	// RuntimeTypeVM 虚拟机运行时
	RuntimeTypeVM RuntimeType = "vm"

	// RuntimeTypeHost 主机运行时（直接在主机上运行）
	RuntimeTypeHost RuntimeType = "host"

	// RuntimeTypeMicroVM MicroVM 运行时（如 Firecracker）
	RuntimeTypeMicroVM RuntimeType = "microvm"
)

// ============================================================================
// RuntimeStatus - 运行时状态枚举
// ============================================================================

// RuntimeStatus 运行时状态
type RuntimeStatus string

const (
	// RuntimeStatusCreating 创建中
	RuntimeStatusCreating RuntimeStatus = "creating"

	// RuntimeStatusReady 就绪：已创建，等待使用
	RuntimeStatusReady RuntimeStatus = "ready"

	// RuntimeStatusRunning 运行中
	RuntimeStatusRunning RuntimeStatus = "running"

	// RuntimeStatusStopping 停止中
	RuntimeStatusStopping RuntimeStatus = "stopping"

	// RuntimeStatusStopped 已停止
	RuntimeStatusStopped RuntimeStatus = "stopped"

	// RuntimeStatusError 错误
	RuntimeStatusError RuntimeStatus = "error"

	// RuntimeStatusDestroyed 已销毁
	RuntimeStatusDestroyed RuntimeStatus = "destroyed"
)

// ============================================================================
// Runtime - 运行时环境
// ============================================================================

// Runtime 运行时环境
//
// Runtime 是 Agent 的运行环境，可以是：
//   - 容器（Docker）
//   - 虚拟机
//   - MicroVM（如 Firecracker）
//   - 主机进程
//
// Runtime 运行在 Node 上，与 Agent 是 1:1 关系。
type Runtime struct {
	// === 基础字段 ===

	// ID 唯一标识
	ID string `json:"id" bson:"_id" db:"id"`

	// Type 运行时类型
	Type RuntimeType `json:"type" bson:"type" db:"type"`

	// Status 运行时状态
	Status RuntimeStatus `json:"status" bson:"status" db:"status"`

	// === 关联 ===

	// AgentID 关联的 Agent ID
	AgentID string `json:"agent_id" bson:"agent_id" db:"agent_id"`

	// NodeID 所在节点 ID
	NodeID string `json:"node_id" bson:"node_id" db:"node_id"`

	// === 容器特定字段 ===

	// ContainerID Docker 容器 ID
	ContainerID *string `json:"container_id,omitempty" bson:"container_id,omitempty" db:"container_id"`

	// ContainerName Docker 容器名称
	ContainerName *string `json:"container_name,omitempty" bson:"container_name,omitempty" db:"container_name"`

	// Image 容器镜像
	Image *string `json:"image,omitempty" bson:"image,omitempty" db:"image"`

	// === 工作空间 ===

	// WorkspacePath 工作空间路径
	WorkspacePath string `json:"workspace_path,omitempty" bson:"workspace_path,omitempty" db:"workspace_path"`

	// === 网络 ===

	// IPAddress IP 地址
	IPAddress *string `json:"ip_address,omitempty" bson:"ip_address,omitempty" db:"ip_address"`

	// Ports 端口映射
	Ports json.RawMessage `json:"ports,omitempty" bson:"ports,omitempty" db:"ports"`

	// === 资源 ===

	// Resources 资源配置
	Resources json.RawMessage `json:"resources,omitempty" bson:"resources,omitempty" db:"resources"`

	// === 时间戳 ===

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at" bson:"created_at" db:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at" db:"updated_at"`

	// StartedAt 启动时间
	StartedAt *time.Time `json:"started_at,omitempty" bson:"started_at,omitempty" db:"started_at"`

	// StoppedAt 停止时间
	StoppedAt *time.Time `json:"stopped_at,omitempty" bson:"stopped_at,omitempty" db:"stopped_at"`
}

// ============================================================================
// Runtime 辅助方法
// ============================================================================

// IsRunning 判断 Runtime 是否正在运行
func (r *Runtime) IsRunning() bool {
	return r.Status == RuntimeStatusRunning
}

// IsReady 判断 Runtime 是否就绪
func (r *Runtime) IsReady() bool {
	return r.Status == RuntimeStatusReady || r.Status == RuntimeStatusRunning
}

// IsTerminated 判断 Runtime 是否已终止
func (r *Runtime) IsTerminated() bool {
	return r.Status == RuntimeStatusStopped || r.Status == RuntimeStatusDestroyed || r.Status == RuntimeStatusError
}

// ============================================================================
// 内置 AgentTemplate
// ============================================================================

// BuiltinAgentTemplates 内置 Agent 模板
var BuiltinAgentTemplates = []AgentTemplate{
	{
		ID:          "builtin-claude-dev",
		Name:        "Claude 开发助手",
		Type:        AgentModelTypeClaude,
		Role:        "代码开发助手",
		Description: "基于 Claude 的代码开发智能体，擅长代码编写、审查和重构",
		Personality: []string{"专业", "严谨", "乐于助人"},
		Model:       "claude-3-opus",
		Temperature: 0.7,
		MaxContext:  128000,
		IsBuiltin:   true,
		Category:    "development",
	},
	{
		ID:          "builtin-gemini-research",
		Name:        "Gemini 研究助手",
		Type:        AgentModelTypeGemini,
		Role:        "技术研究助手",
		Description: "基于 Gemini 的研究智能体，擅长技术调研和方案设计",
		Personality: []string{"博学", "分析性强", "客观"},
		Model:       "gemini-pro",
		Temperature: 0.5,
		MaxContext:  100000,
		IsBuiltin:   true,
		Category:    "research",
	},
	{
		ID:          "builtin-qwen-code",
		Name:        "Qwen 编程助手",
		Type:        AgentModelTypeQwen,
		Role:        "编程助手",
		Description: "基于 Qwen 的编程智能体，支持多语言代码开发",
		Personality: []string{"高效", "精确", "实用"},
		Model:       "qwen-coder",
		Temperature: 0.6,
		MaxContext:  64000,
		IsBuiltin:   true,
		Category:    "development",
	},
}

// ============================================================================
// AgentTypeConfig - 预定义 Agent 类型配置（运行时配置）
// ============================================================================

// AgentTypeConfig 定义 Agent 类型的运行时配置
//
// 与 AgentTemplate 的区别：
//   - AgentTypeConfig 是运行时配置（Docker 镜像、认证目录等）
//   - AgentTemplate 是业务配置（角色、技能、提示词等）
//
// 用途：
//   - 定义 Docker 镜像和启动命令
//   - 定义认证文件位置
//   - 定义支持的登录方式
type AgentTypeConfig struct {
	ID           string   `json:"id"`            // 类型标识，如 qwen-code, openai-codex
	Name         string   `json:"name"`          // 显示名称
	Image        string   `json:"image"`         // Docker 镜像
	AuthDir      string   `json:"auth_dir"`      // 容器内认证目录
	AuthFile     string   `json:"auth_file"`     // 认证文件名
	LoginCmd     string   `json:"login_cmd"`     // 登录命令
	LoginMethods []string `json:"login_methods"` // 支持的登录方式
	Description  string   `json:"description"`   // 类型描述
}

// PredefinedAgentTypeConfigs 预定义的 Agent 类型配置
var PredefinedAgentTypeConfigs = []AgentTypeConfig{
	{
		ID:           "qwen-code",
		Name:         "Qwen-Code",
		Image:        "runners/qwencode:latest",
		AuthDir:      "/home/node/.qwen",
		AuthFile:     "auth.json",
		LoginCmd:     "qwen",
		LoginMethods: []string{"oauth", "api_key"},
		Description:  "基于 Qwen 大模型的 AI 编程助手",
	},
	{
		ID:           "openai-codex",
		Name:         "OpenAI Codex",
		Image:        "runners/codex:latest",
		AuthDir:      "/home/codex/.codex",
		AuthFile:     "auth.json",
		LoginCmd:     "codex login",
		LoginMethods: []string{"device_code", "oauth", "api_key"},
		Description:  "OpenAI 官方 AI 编程智能体",
	},
}

// ============================================================================
// Instance - Agent 实例（兼容旧代码）
// ============================================================================

// InstanceStatus 实例状态
//
// Deprecated: 使用 AgentStatus 替代
type InstanceStatus string

const (
	// InstanceStatusPending 待创建（等待 Executor 处理）
	InstanceStatusPending InstanceStatus = "pending"

	// InstanceStatusCreating 创建中
	InstanceStatusCreating InstanceStatus = "creating"

	// InstanceStatusRunning 运行中
	InstanceStatusRunning InstanceStatus = "running"

	// InstanceStatusStopping 停止中（等待 Executor 处理）
	InstanceStatusStopping InstanceStatus = "stopping"

	// InstanceStatusStopped 已停止
	InstanceStatusStopped InstanceStatus = "stopped"

	// InstanceStatusError 错误
	InstanceStatusError InstanceStatus = "error"
)

// Instance 表示运行中的 Agent 实例
//
// Deprecated: 使用 Agent 替代。Instance 保留用于向后兼容。
type Instance struct {
	ID            string         `json:"id" bson:"_id" db:"id"`
	Name          string         `json:"name" bson:"name" db:"name"`                                          // 显示名称
	AccountID     string         `json:"account_id" bson:"account_id" db:"account_id"`                        // 使用的账号 ID
	AgentTypeID   string         `json:"agent_type_id" bson:"agent_type_id" db:"agent_type_id"`               // Agent 类型 ID
	TemplateID    *string        `json:"template_id,omitempty" bson:"template_id,omitempty" db:"template_id"` // 关联的模板 ID（可选）
	ContainerName *string        `json:"container_name" bson:"container_name" db:"container_name"`            // Docker 容器名（Executor 回填）
	NodeID        *string        `json:"node_id" bson:"node_id" db:"node_id"`                                 // 所在节点 ID
	Status        InstanceStatus `json:"status" bson:"status" db:"status"`                                    // 实例状态
	CreatedAt     time.Time      `json:"created_at" bson:"created_at" db:"created_at"`                        // 创建时间
	UpdatedAt     time.Time      `json:"updated_at" bson:"updated_at" db:"updated_at"`                        // 更新时间
}

// IsRunning 判断实例是否正在运行
func (i *Instance) IsRunning() bool {
	return i.Status == InstanceStatusRunning
}

// CanStart 判断实例是否可以启动
func (i *Instance) CanStart() bool {
	return i.Status == InstanceStatusStopped || i.Status == InstanceStatusError
}

// ============================================================================
// 类型别名（兼容旧代码）
// ============================================================================

// AgentType 是 AgentTypeConfig 的别名，用于向后兼容
//
// Deprecated: 使用 AgentTypeConfig 替代
type AgentType = AgentTypeConfig

// PredefinedAgentTypes 是 PredefinedAgentTypeConfigs 的别名
//
// Deprecated: 使用 PredefinedAgentTypeConfigs 替代
var PredefinedAgentTypes = PredefinedAgentTypeConfigs

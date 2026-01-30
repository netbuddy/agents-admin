package driver

import "time"

// ============================================================================
// TaskSpec - 任务规格（用户定义）
// ============================================================================

// TaskSpec 定义一个任务的完整规格
//
// TaskSpec 回答"做什么"的问题：
//   - 任务目标（Prompt）
//   - 工作环境（Workspace）
//   - 安全约束（SecurityConfig）
//
// 设计说明：
//   - Workspace 是可选的，并非所有任务都需要工作目录
//   - 不同类型的任务有不同的默认配置
//   - Labels 用于调度匹配和分类筛选
//   - Agent 配置已移至 Run 级别，实现任务与执行者解耦
//
// 典型场景：
//  1. 代码开发：Workspace=Git，需要文件读写权限
//  2. 技术调研：Workspace=nil，纯对话任务
//  3. 运维操作：Workspace=Remote，需要 SSH/系统权限
//  4. 数据处理：Workspace=Local，读取/生成文件
type TaskSpec struct {
	// ID 任务唯一标识（通常从 model.Task 继承）
	ID string `json:"id"`

	// Name 任务名称，用户可读的描述
	Name string `json:"name"`

	// Type 任务类型，影响默认配置和处理策略
	// 可选，默认为 TaskTypeGeneral
	Type TaskType `json:"type,omitempty"`

	// Prompt 任务提示词，Agent 执行的目标描述
	// 这是 Agent 的核心输入
	Prompt string `json:"prompt"`

	// Workspace 工作空间配置（可选）
	// nil 表示无工作空间（纯对话任务）
	Workspace *WorkspaceConfig `json:"workspace,omitempty"`

	// Security 安全约束配置
	Security SecurityConfig `json:"security"`

	// Labels 任务标签，用于调度匹配和分类
	// 例如：{"priority": "high", "team": "platform"}
	Labels map[string]string `json:"labels,omitempty"`

	// Context 上下文信息（可选）
	// 用于传递额外的背景信息给 Agent
	Context *TaskContext `json:"context,omitempty"`
}

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
//
// 不同任务类型的典型 Workspace：
//   - development → Git
//   - operation → Remote
//   - automation → Local 或 Volume
//   - research → nil（无 Workspace）
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
//
// 安全分层：
//  1. Policy 提供基线（strict/standard/permissive）
//  2. Permissions 在 Policy 基础上微调
//  3. Limits 限制资源使用防止滥用
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
	Network *NetworkConfig `json:"network,omitempty"`

	// Limits 资源限制
	Limits *ResourceLimits `json:"limits,omitempty"`

	// RequireApproval 需要人工审批的操作列表
	// 例如：["file_delete", "command_execute"]
	RequireApproval []string `json:"require_approval,omitempty"`
}

// NetworkConfig 网络访问配置
type NetworkConfig struct {
	// Enabled 是否允许网络访问
	Enabled bool `json:"enabled"`

	// AllowedHosts 允许访问的主机列表（白名单）
	// 为空表示允许所有（如果 Enabled=true）
	AllowedHosts []string `json:"allowed_hosts,omitempty"`

	// DeniedHosts 禁止访问的主机列表（黑名单）
	DeniedHosts []string `json:"denied_hosts,omitempty"`
}

// ResourceLimits 资源限制配置
type ResourceLimits struct {
	// Timeout 执行超时时间
	Timeout time.Duration `json:"timeout"`

	// MaxTokens 最大 Token 数（API 调用限制）
	MaxTokens int `json:"max_tokens,omitempty"`

	// MaxFileSize 单个文件最大大小（字节）
	MaxFileSize int64 `json:"max_file_size,omitempty"`

	// MaxFiles 最大文件数量
	MaxFiles int `json:"max_files,omitempty"`

	// MemoryLimit 内存限制（字节）
	MemoryLimit int64 `json:"memory_limit,omitempty"`

	// CPULimit CPU 限制（核数，支持小数）
	CPULimit float64 `json:"cpu_limit,omitempty"`
}

// ============================================================================
// TaskContext - 任务上下文
// ============================================================================

// TaskContext 提供任务执行的额外上下文
//
// 用于传递：
//   - 相关文档和参考资料
//   - 前序任务的输出
//   - 用户偏好和约定
type TaskContext struct {
	// Documents 相关文档列表
	Documents []ContextDocument `json:"documents,omitempty"`

	// PreviousResults 前序任务的结果
	PreviousResults map[string]interface{} `json:"previous_results,omitempty"`

	// Instructions 额外指令
	Instructions string `json:"instructions,omitempty"`
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

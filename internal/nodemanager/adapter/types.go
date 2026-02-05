// Package adapter 定义 Agent CLI 适配器接口和核心数据结构
package adapter

// ============================================================================
// TaskType - 任务类型
// ============================================================================

// TaskType 定义任务的类型，不同类型有不同的处理策略和默认配置
//
// 任务类型决定：
//   - 默认的 Workspace 类型
//   - 默认的安全策略
//   - 调度优先级和资源分配
//   - UI 展示方式
//
// 注意：任务类型是可选的，默认为 TaskTypeGeneral
type TaskType string

const (
	// TaskTypeGeneral 通用任务：无特殊要求，适用于大多数场景
	TaskTypeGeneral TaskType = "general"

	// TaskTypeDevelopment 开发任务：代码开发、Bug 修复、功能实现
	// 通常需要 Git Workspace，可能修改文件
	TaskTypeDevelopment TaskType = "development"

	// TaskTypeOperation 运维任务：部署、监控、故障排查
	// 可能需要提权、SSH 访问、执行系统命令
	TaskTypeOperation TaskType = "operation"

	// TaskTypeResearch 研究任务：技术调研、方案设计、知识整理
	// 通常不需要 Workspace，以对话为主
	TaskTypeResearch TaskType = "research"

	// TaskTypeAutomation 自动化任务：数据处理、报告生成、定时任务
	// 可调度、可重复执行、通常有固定输入输出
	TaskTypeAutomation TaskType = "automation"

	// TaskTypeReview 审核任务：代码审查、内容审核、合规检查
	// 需要人工介入，有明确的审批流程
	TaskTypeReview TaskType = "review"
)

// ============================================================================
// WorkspaceType - 工作空间类型
// ============================================================================

// WorkspaceType 工作空间类型
type WorkspaceType string

const (
	// WorkspaceTypeGit Git 仓库工作空间
	// 用于代码开发任务，自动 clone/checkout
	WorkspaceTypeGit WorkspaceType = "git"

	// WorkspaceTypeLocal 本地目录工作空间
	// 用于文件处理任务，挂载本地目录
	WorkspaceTypeLocal WorkspaceType = "local"

	// WorkspaceTypeRemote 远程工作空间
	// 用于运维任务，通过 SSH/容器访问远程系统
	WorkspaceTypeRemote WorkspaceType = "remote"

	// WorkspaceTypeVolume 持久化卷工作空间
	// 用于需要持久化存储的任务
	WorkspaceTypeVolume WorkspaceType = "volume"
)

// ============================================================================
// SecurityPolicy - 安全策略等级
// ============================================================================

// SecurityPolicy 安全策略等级
type SecurityPolicy string

const (
	// SecurityPolicyStrict 严格模式：最小权限，禁止网络，只读文件系统
	SecurityPolicyStrict SecurityPolicy = "strict"

	// SecurityPolicyStandard 标准模式：允许文件读写，受限网络
	SecurityPolicyStandard SecurityPolicy = "standard"

	// SecurityPolicyPermissive 宽松模式：允许大部分操作，仍有基本限制
	SecurityPolicyPermissive SecurityPolicy = "permissive"

	// SecurityPolicyCustom 自定义模式：完全由 Permissions 定义
	SecurityPolicyCustom SecurityPolicy = "custom"
)

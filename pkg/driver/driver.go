// Package driver 定义 Agent Driver 接口和核心数据结构
//
// Driver 是 Agent CLI 的适配层，负责：
//   - 将平台统一的 TaskSpec 转换为具体 CLI 的启动命令
//   - 将各种 CLI 的输出解析为统一的 CanonicalEvent
//   - 收集执行产物（Artifacts）
//
// 设计原则：
//   - 每种 Agent CLI（Claude、Gemini、Codex 等）实现一个 Driver
//   - TaskSpec 定义"做什么"，RunConfig 定义"怎么执行"
//   - CanonicalEvent 是统一的事件格式，屏蔽 CLI 差异
//
// 架构关系：
//
//	用户定义 TaskSpec + AgentConfig
//	       │
//	       ▼  Driver.BuildCommand()
//	  生成 RunConfig（容器启动配置）
//	       │
//	       ▼  Node Agent 执行
//	  CLI 输出 stdout/stderr
//	       │
//	       ▼  Driver.ParseEvent()
//	  转换为 CanonicalEvent
//	       │
//	       ▼
//	  存储/推送/展示
//
// 文件组织：
//   - types.go: 基础类型定义（TaskType, WorkspaceType, SecurityPolicy）
//   - task.go: 任务规格相关（TaskSpec, WorkspaceConfig, SecurityConfig）
//   - agent.go: Agent 配置相关（AgentConfig, MCPServerConfig）
//   - run.go: 运行时配置相关（RunConfig, MountConfig）
//   - event.go: 事件和产物相关（CanonicalEvent, EventType, Artifacts）
//   - driver.go: Driver 接口和注册表
package driver

import "context"

// ============================================================================
// Driver 接口
// ============================================================================

// Driver 是 Agent CLI 的适配接口
//
// 每种 Agent CLI（Claude、Gemini、Codex 等）实现一个 Driver，负责：
//  1. 验证 Agent 配置是否支持（Validate）
//  2. 将 TaskSpec + AgentConfig 转换为 CLI 启动命令（BuildCommand）
//  3. 解析 CLI 输出为统一事件（ParseEvent）
//  4. 收集执行产物（CollectArtifacts）
//
// # Driver 是无状态的，所有状态通过参数传递
//
// 设计说明：
//   - TaskSpec 与 AgentConfig 分离，支持同一任务使用不同 Agent 执行
//   - 这使得 A/B 测试、失败重试切换 Agent 成为可能
//   - AgentConfig 属于 Run 级别，而非 Task 级别
//
// 实现注意事项：
//   - Name() 应返回唯一标识，如 "claude-v1", "gemini-v1"
//   - Validate() 应检查 AgentConfig.Type 是否匹配
//   - ParseEvent() 对无效行应返回 (nil, nil)，而非错误
//   - CollectArtifacts() 在 workspaceDir 中查找产物
//   - ctx 参数用于超时控制和链路追踪，当前简单实现可忽略
type Driver interface {
	// Name 返回驱动名称
	// 用于 Registry 查找和 AgentConfig.Type 匹配
	Name() string

	// Validate 验证 AgentConfig 是否适用于此 Driver
	// 检查项：Agent 类型、必要参数
	Validate(agent *AgentConfig) error

	// BuildCommand 根据 TaskSpec 和 AgentConfig 构建运行配置
	// 将平台抽象转换为具体的容器启动配置
	// ctx 用于超时控制（如需要获取远程凭据时）
	BuildCommand(ctx context.Context, spec *TaskSpec, agent *AgentConfig) (*RunConfig, error)

	// ParseEvent 解析 CLI 输出行，返回统一事件
	// 将 Agent 特定的输出格式转换为 CanonicalEvent
	// 如果该行不是有效事件，返回 (nil, nil)
	ParseEvent(line string) (*CanonicalEvent, error)

	// CollectArtifacts 收集运行产物
	// 在 workspaceDir 中查找事件日志、diff 等产物
	CollectArtifacts(ctx context.Context, workspaceDir string) (*Artifacts, error)
}

// Registry Driver 注册表
type Registry struct {
	drivers map[string]Driver
}

// NewRegistry 创建注册表
func NewRegistry() *Registry {
	return &Registry{
		drivers: make(map[string]Driver),
	}
}

// Register 注册 Driver
func (r *Registry) Register(d Driver) {
	r.drivers[d.Name()] = d
}

// Get 获取 Driver
func (r *Registry) Get(name string) (Driver, bool) {
	d, ok := r.drivers[name]
	return d, ok
}

// List 列出所有 Driver
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.drivers))
	for name := range r.drivers {
		names = append(names, name)
	}
	return names
}

// Package runtime 定义 Agent 运行时环境接口
//
// Runtime 是 Agent 执行任务的环境抽象，支持多种运行时类型：
//   - Docker 容器（当前实现）
//   - 虚拟机（未来扩展）
//   - 物理机（未来扩展）
//
// 与 auth/container.go 的区别：
//   - runtime/ 用于运行 Agent 任务（长期运行）
//   - auth/container.go 用于认证流程（临时容器）
package runtime

import (
	"context"
	"io"
)

// Runtime Agent 运行时环境接口
//
// 每种运行时环境（Docker、VM、物理机）实现此接口
type Runtime interface {
	// Name 返回运行时名称
	Name() string

	// Create 创建运行时实例
	Create(ctx context.Context, config *InstanceConfig) (*Instance, error)

	// Start 启动运行时实例
	Start(ctx context.Context, instanceID string) error

	// Stop 停止运行时实例
	Stop(ctx context.Context, instanceID string) error

	// Remove 删除运行时实例
	Remove(ctx context.Context, instanceID string, force bool) error

	// Exec 在运行时实例中执行命令
	Exec(ctx context.Context, instanceID string, cmd []string) (*ExecResult, error)

	// Attach 附加到运行时实例获取 IO
	Attach(ctx context.Context, instanceID string) (*AttachResult, error)

	// Status 获取运行时实例状态
	Status(ctx context.Context, instanceID string) (*InstanceStatus, error)

	// Logs 获取运行时实例日志
	Logs(ctx context.Context, instanceID string, tail int) (io.ReadCloser, error)
}

// InstanceConfig 运行时实例配置
type InstanceConfig struct {
	Name       string            // 实例名称
	Image      string            // 镜像名称（Docker）或模板（VM）
	Command    []string          // 启动命令
	Args       []string          // 命令参数
	Env        map[string]string // 环境变量
	WorkingDir string            // 工作目录
	Mounts     []Mount           // 挂载配置
	Resources  *ResourceConfig   // 资源限制
	Network    *NetworkConfig    // 网络配置
	TTY        bool              // 是否分配 TTY
	Stdin      bool              // 是否打开标准输入
}

// Mount 挂载配置
type Mount struct {
	Source   string // 宿主机路径或卷名
	Target   string // 容器内路径
	ReadOnly bool   // 是否只读
}

// ResourceConfig 资源限制配置
type ResourceConfig struct {
	CPULimit    float64 // CPU 限制（核数）
	MemoryLimit int64   // 内存限制（字节）
	DiskLimit   int64   // 磁盘限制（字节）
}

// NetworkConfig 网络配置
type NetworkConfig struct {
	NetworkMode string         // 网络模式
	PortMap     map[int]int    // 端口映射 host:container
	DNS         []string       // DNS 服务器
	ExtraHosts  []string       // 额外的 hosts 配置
}

// Instance 运行时实例
type Instance struct {
	ID        string          // 实例 ID
	Name      string          // 实例名称
	Runtime   string          // 运行时类型
	Status    InstanceStatus  // 实例状态
	Config    *InstanceConfig // 实例配置
}

// InstanceStatus 运行时实例状态
type InstanceStatus struct {
	State      InstanceState // 状态
	ExitCode   int           // 退出码
	StartedAt  string        // 启动时间
	FinishedAt string        // 结束时间
	Error      string        // 错误信息
}

// InstanceState 实例状态枚举
type InstanceState string

const (
	StateCreated  InstanceState = "created"  // 已创建
	StateRunning  InstanceState = "running"  // 运行中
	StatePaused   InstanceState = "paused"   // 已暂停
	StateStopped  InstanceState = "stopped"  // 已停止
	StateExited   InstanceState = "exited"   // 已退出
	StateRemoving InstanceState = "removing" // 正在删除
	StateUnknown  InstanceState = "unknown"  // 未知
)

// ExecResult 命令执行结果
type ExecResult struct {
	ExitCode int    // 退出码
	Stdout   string // 标准输出
	Stderr   string // 标准错误
}

// AttachResult 附加结果
type AttachResult struct {
	Conn   io.ReadWriteCloser // 连接
	Reader io.Reader          // 输出读取
	Writer io.Writer          // 输入写入
}

// Registry 运行时注册表
type Registry struct {
	runtimes map[string]Runtime
}

// NewRegistry 创建注册表
func NewRegistry() *Registry {
	return &Registry{
		runtimes: make(map[string]Runtime),
	}
}

// Register 注册运行时
func (r *Registry) Register(rt Runtime) {
	r.runtimes[rt.Name()] = rt
}

// Get 获取运行时
func (r *Registry) Get(name string) (Runtime, bool) {
	rt, ok := r.runtimes[name]
	return rt, ok
}

// List 列出所有运行时
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.runtimes))
	for name := range r.runtimes {
		names = append(names, name)
	}
	return names
}

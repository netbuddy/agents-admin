package driver

// ============================================================================
// RunConfig - 运行时配置（Driver 生成）
// ============================================================================

// RunConfig 定义任务的具体执行配置
//
// RunConfig 回答"怎么执行"的问题：
//   - 容器镜像和启动命令
//   - 环境变量和挂载点
//   - 资源限制
//
// 与 TaskSpec 的关系：
//   - TaskSpec 是用户/系统定义的"what"
//   - RunConfig 是 Driver 生成的"how"
//   - Driver.BuildCommand() 完成转换
//
// RunConfig 是 Node Agent 执行任务的直接输入
type RunConfig struct {
	// Image 容器镜像
	Image string `json:"image"`

	// Command 启动命令
	Command []string `json:"command"`

	// Args 命令参数
	Args []string `json:"args"`

	// Env 环境变量
	Env map[string]string `json:"env"`

	// WorkingDir 工作目录
	WorkingDir string `json:"working_dir"`

	// Mounts 挂载点配置
	Mounts []MountConfig `json:"mounts,omitempty"`

	// Resources 资源配置
	Resources *ContainerResources `json:"resources,omitempty"`

	// SecurityOpts 安全选项
	SecurityOpts []string `json:"security_opts,omitempty"`
}

// MountConfig 挂载配置
type MountConfig struct {
	// Source 源路径（主机/卷）
	Source string `json:"source"`

	// Target 目标路径（容器内）
	Target string `json:"target"`

	// ReadOnly 是否只读
	ReadOnly bool `json:"read_only,omitempty"`

	// Type 挂载类型（bind/volume/tmpfs）
	Type string `json:"type,omitempty"`
}

// ContainerResources 容器资源配置
type ContainerResources struct {
	// MemoryLimit 内存限制
	MemoryLimit string `json:"memory_limit,omitempty"`

	// CPULimit CPU 限制
	CPULimit string `json:"cpu_limit,omitempty"`

	// GPUCount GPU 数量
	GPUCount int `json:"gpu_count,omitempty"`
}

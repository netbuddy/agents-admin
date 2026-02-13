// Package model 定义核心数据模型
//
// run.go 包含执行相关的数据模型定义：
//   - Run：任务的单次执行实例
//   - RunStatus：执行状态枚举
//   - RunConfig：运行配置
//   - MountConfig：挂载配置
package model

import (
	"encoding/json"
	"time"
)

// ============================================================================
// RunStatus - 执行状态
// ============================================================================

// RunStatus 表示单次执行（Run）的状态
//
// Run 是 Task 的执行实例，RunStatus 反映这一次执行的进展：
//   - queued：等待调度（Task 无此状态，因为 Task 不参与调度）
//   - running：正在执行
//   - done：执行完成（不代表成功，只表示执行结束）
//   - failed：执行失败
//   - cancelled：执行被取消
//   - timeout：执行超时（Task 无此状态，超时是执行层面的概念）
//
// 为什么不合并到 TaskStatus？
//  1. 语义不同：Task "completed" = 目标达成；Run "done" = 执行结束
//  2. 独有状态：queued/timeout 只对 Run 有意义
//  3. 一对多关系：一个 Task 可有多个 Run，状态需独立追踪
type RunStatus string

const (
	// RunStatusQueued 排队中：等待调度器分配节点
	RunStatusQueued RunStatus = "queued"

	// RunStatusAssigned 已分配：调度器已分配节点，等待 Executor 领取执行
	RunStatusAssigned RunStatus = "assigned"

	// RunStatusRunning 执行中：Executor 已开始执行（上报了事件）
	RunStatusRunning RunStatus = "running"

	// RunStatusPaused 已暂停：用户干预暂停执行（可恢复）
	RunStatusPaused RunStatus = "paused"

	// RunStatusDone 已结束：执行正常结束（检查产物判断是否成功）
	RunStatusDone RunStatus = "done"

	// RunStatusFailed 已失败：执行过程出错
	RunStatusFailed RunStatus = "failed"

	// RunStatusCancelled 已取消：用户或系统取消了此次执行
	RunStatusCancelled RunStatus = "cancelled"

	// RunStatusTimeout 已超时：执行时间超过限制
	RunStatusTimeout RunStatus = "timeout"
)

// ============================================================================
// Run - 执行实例
// ============================================================================

// Run 表示 Task 的一次执行尝试
//
// Run 是任务的"执行记录"，记录一次具体的执行过程：
//   - 每个 Run 绑定到一个 Task
//   - 每个 Run 被调度到一个 Node 执行
//   - Run 产生事件流（Events）和产物（Artifacts）
//   - Run 是不可变的：一旦结束，状态不再改变
//
// 为什么需要 Run？
//  1. 支持重试：Task 失败后可创建新 Run 重试
//  2. 保留历史：每次执行都有独立记录
//  3. 资源隔离：每个 Run 有独立的事件流和产物
//  4. 状态独立：不同 Run 可以有不同结果
//
// 典型生命周期：
//
//	创建 → queued → assigned → running → done/failed/cancelled/timeout
//
// 状态说明：
//   - queued: 等待调度器分配节点
//   - assigned: 调度器已分配节点，等待 Executor 领取
//   - running: Executor 已开始执行（上报了第一个事件）
//
// 字段说明：
//   - ID：唯一标识符，格式如 "run-abc123"
//   - TaskID：所属任务 ID
//   - Status：执行状态
//   - NodeID：执行节点 ID（调度后填充）
//   - StartedAt：实际开始执行时间
//   - FinishedAt：执行结束时间
//   - Snapshot：执行时的任务快照（用于审计）
//   - Error：错误信息（失败时填充）
type Run struct {
	ID         string          `json:"id" bson:"_id" db:"id"`                             // 执行唯一标识
	TaskID     string          `json:"task_id" bson:"task_id" db:"task_id"`                   // 所属任务 ID
	Status     RunStatus       `json:"status" bson:"status" db:"status"`                     // 执行状态
	NodeID     *string         `json:"node_id,omitempty" bson:"node_id,omitempty" db:"node_id"`         // 执行节点 ID
	StartedAt  *time.Time      `json:"started_at,omitempty" bson:"started_at,omitempty" db:"started_at"`   // 开始时间
	FinishedAt *time.Time      `json:"finished_at,omitempty" bson:"finished_at,omitempty" db:"finished_at"` // 结束时间
	Snapshot   json.RawMessage `json:"snapshot,omitempty" bson:"snapshot,omitempty" db:"snapshot"`       // 任务快照
	Error      *string         `json:"error,omitempty" bson:"error,omitempty" db:"error"`             // 错误信息
	CreatedAt  time.Time       `json:"created_at" bson:"created_at" db:"created_at"`             // 创建时间
	UpdatedAt  time.Time       `json:"updated_at" bson:"updated_at" db:"updated_at"`             // 更新时间
}

// ============================================================================
// RunConfig - 运行配置（从 pkg/driver 迁入）
// ============================================================================

// RunConfig 定义 Run 的运行配置
//
// RunConfig 回答"在哪里运行、如何运行"的问题：
//   - Container：容器镜像和资源配置
//   - Mounts：文件系统挂载配置
//   - Network：网络配置
//   - Agent：Agent 类型和参数
type RunConfig struct {
	// Image 容器镜像
	Image string `json:"image"`

	// ContainerName 容器名称（可选）
	ContainerName string `json:"container_name,omitempty"`

	// Mounts 挂载配置
	Mounts []MountConfig `json:"mounts,omitempty"`

	// Environment 环境变量
	Environment map[string]string `json:"environment,omitempty"`

	// WorkingDir 工作目录
	WorkingDir string `json:"working_dir,omitempty"`

	// Entrypoint 容器入口点
	Entrypoint []string `json:"entrypoint,omitempty"`

	// Command 容器命令
	Command []string `json:"command,omitempty"`

	// AgentType Agent 类型（claude/gemini/qwen 等）
	AgentType string `json:"agent_type,omitempty"`

	// AgentModel Agent 模型名称
	AgentModel string `json:"agent_model,omitempty"`
}

// MountConfig 挂载配置
type MountConfig struct {
	// Type 挂载类型（bind/volume）
	Type string `json:"type"`

	// Source 源路径（bind）或卷名（volume）
	Source string `json:"source"`

	// Target 容器内目标路径
	Target string `json:"target"`

	// ReadOnly 是否只读
	ReadOnly bool `json:"read_only,omitempty"`
}

// ============================================================================
// 辅助方法
// ============================================================================

// IsTerminal 判断 Run 是否处于终止状态
func (r *Run) IsTerminal() bool {
	switch r.Status {
	case RunStatusDone, RunStatusFailed, RunStatusCancelled, RunStatusTimeout:
		return true
	default:
		return false
	}
}

// IsRunning 判断 Run 是否正在运行
func (r *Run) IsRunning() bool {
	return r.Status == RunStatusRunning
}

// CanRetry 判断 Run 是否可以重试
func (r *Run) CanRetry() bool {
	return r.Status == RunStatusFailed || r.Status == RunStatusTimeout
}

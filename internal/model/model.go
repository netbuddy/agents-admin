// Package model 定义核心数据模型
//
// 本包定义了 Agent Kanban 系统的核心领域模型，包括：
//   - Task（任务）：任务定义，描述"做什么"
//   - Run（执行）：任务的单次执行实例，记录"怎么做的"
//   - Event（事件）：执行过程中产生的事件流
//   - Node（节点）：执行任务的计算节点
//   - Artifact（产物）：执行产生的文件产物
//
// Task 与 Run 的关系：
//   - Task 是任务模板，定义目标和参数，可长期存在
//   - Run 是 Task 的执行实例，一个 Task 可多次执行产生多个 Run
//   - 这种设计支持：重试、历史追溯、结果比较
package model

import (
	"encoding/json"
	"time"
)

// ============================================================================
// TaskStatus - 任务状态
// ============================================================================

// TaskStatus 表示任务（Task）的整体状态
//
// Task 是任务定义，TaskStatus 反映任务的整体进展：
//   - pending：任务已创建，尚未开始执行
//   - running：任务正在执行中（有活跃的 Run）
//   - completed：任务目标已达成
//   - failed：任务执行失败（所有重试均失败）
//   - cancelled：任务被用户取消
//
// 注意：TaskStatus 与 RunStatus 是不同层次的状态：
//   - TaskStatus 描述任务的整体目标是否达成
//   - RunStatus 描述单次执行的状态
type TaskStatus string

const (
	// TaskStatusPending 待处理：任务已创建，等待首次执行
	TaskStatusPending TaskStatus = "pending"

	// TaskStatusRunning 执行中：至少有一个 Run 正在进行
	TaskStatusRunning TaskStatus = "running"

	// TaskStatusCompleted 已完成：任务目标已达成
	TaskStatusCompleted TaskStatus = "completed"

	// TaskStatusFailed 已失败：任务无法完成（可能需要人工介入）
	TaskStatusFailed TaskStatus = "failed"

	// TaskStatusCancelled 已取消：用户主动取消任务
	TaskStatusCancelled TaskStatus = "cancelled"
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
	// RunStatusQueued 排队中：等待节点领取执行
	RunStatusQueued RunStatus = "queued"

	// RunStatusRunning 执行中：节点正在执行此 Run
	RunStatusRunning RunStatus = "running"

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
// NodeStatus - 节点状态
// ============================================================================

// NodeStatus 表示计算节点的状态
//
// 节点生命周期：
//
//	starting → online ⇄ unhealthy
//	              ↓
//	          draining → offline → terminated
//	              ↓
//	         maintenance
//
// 状态说明：
//   - starting：节点启动中，正在初始化
//   - online：节点在线，可接受新任务
//   - unhealthy：节点不健康，心跳正常但健康检查失败
//   - draining：节点排空中，不接受新任务，等待现有任务完成
//   - maintenance：维护模式，管理员手动标记
//   - offline：节点离线，心跳超时或主动下线
//   - terminated：节点已终止，永久下线
//   - unknown：状态未知，无法确定节点状态
type NodeStatus string

const (
	// NodeStatusStarting 启动中：节点正在初始化
	NodeStatusStarting NodeStatus = "starting"

	// NodeStatusOnline 在线：节点正常运行，可接受任务
	NodeStatusOnline NodeStatus = "online"

	// NodeStatusUnhealthy 不健康：心跳正常但健康检查失败（如磁盘满、内存不足）
	NodeStatusUnhealthy NodeStatus = "unhealthy"

	// NodeStatusDraining 排空中：不再接受新任务，等待现有任务完成后下线
	NodeStatusDraining NodeStatus = "draining"

	// NodeStatusMaintenance 维护中：管理员手动标记，暂停调度
	NodeStatusMaintenance NodeStatus = "maintenance"

	// NodeStatusOffline 离线：节点已断开连接
	NodeStatusOffline NodeStatus = "offline"

	// NodeStatusTerminated 已终止：节点永久移除，不会再上线
	NodeStatusTerminated NodeStatus = "terminated"

	// NodeStatusUnknown 未知：无法确定节点状态（心跳超时但未确认下线）
	NodeStatusUnknown NodeStatus = "unknown"
)

// ============================================================================
// Task - 任务定义
// ============================================================================

// Task 表示一个任务卡片（看板中的卡片）
//
// Task 是任务的"定义"，描述需要完成的目标：
//   - 包含任务名称、提示词（Prompt）、Agent 配置等
//   - 可以被多次执行（产生多个 Run）
//   - 状态反映任务整体进展，而非单次执行结果
//   - 支持层级结构：ParentID 指向父任务，形成任务树
//
// 典型生命周期：
//
//	创建 → pending → running → completed/failed/cancelled
//
// 字段说明：
//   - ID：唯一标识符，格式如 "task-abc123"
//   - ParentID：父任务 ID（顶层任务为空）
//   - Name：任务名称，用户可读的描述
//   - Status：任务当前状态
//   - Spec：任务规格（JSON），包含 prompt、agent 配置等
//   - Context：任务上下文（累积的上下文信息）
//   - InstanceID：执行实例 ID
//   - CreatedAt/UpdatedAt：创建和更新时间戳
type Task struct {
	ID         string          `json:"id" db:"id"`                             // 任务唯一标识
	ParentID   *string         `json:"parent_id,omitempty" db:"parent_id"`     // 父任务 ID
	Name       string          `json:"name" db:"name"`                         // 任务名称
	Status     TaskStatus      `json:"status" db:"status"`                     // 任务状态
	Spec       json.RawMessage `json:"spec" db:"spec"`                         // 任务规格（TaskSpec JSON）
	Context    json.RawMessage `json:"context,omitempty" db:"context"`         // 任务上下文
	InstanceID *string         `json:"instance_id,omitempty" db:"instance_id"` // 执行实例 ID
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`             // 创建时间
	UpdatedAt  time.Time       `json:"updated_at" db:"updated_at"`             // 更新时间
}

// TaskContext 任务上下文结构
type TaskContext struct {
	// InheritedContext 继承的上下文（来自父任务和兄弟任务）
	InheritedContext []ContextItem `json:"inherited_context,omitempty"`
	// ProducedContext 本任务产出的上下文（供子任务使用）
	ProducedContext []ContextItem `json:"produced_context,omitempty"`
	// ConversationHistory 对话历史
	ConversationHistory []Message `json:"conversation_history,omitempty"`
}

// ContextItem 上下文项
type ContextItem struct {
	Type    string `json:"type"`              // file, summary, reference
	Name    string `json:"name"`              // 名称
	Content string `json:"content,omitempty"` // 内容
	Source  string `json:"source,omitempty"`  // 来源任务 ID
}

// Message 对话消息
type Message struct {
	Role      string    `json:"role"`      // user, assistant, system
	Content   string    `json:"content"`   // 消息内容
	Timestamp time.Time `json:"timestamp"` // 时间戳
}

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
//	创建 → queued → running → done/failed/cancelled/timeout
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
	ID         string          `json:"id" db:"id"`                             // 执行唯一标识
	TaskID     string          `json:"task_id" db:"task_id"`                   // 所属任务 ID
	Status     RunStatus       `json:"status" db:"status"`                     // 执行状态
	NodeID     *string         `json:"node_id,omitempty" db:"node_id"`         // 执行节点 ID
	StartedAt  *time.Time      `json:"started_at,omitempty" db:"started_at"`   // 开始时间
	FinishedAt *time.Time      `json:"finished_at,omitempty" db:"finished_at"` // 结束时间
	Snapshot   json.RawMessage `json:"snapshot,omitempty" db:"snapshot"`       // 任务快照
	Error      *string         `json:"error,omitempty" db:"error"`             // 错误信息
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`             // 创建时间
	UpdatedAt  time.Time       `json:"updated_at" db:"updated_at"`             // 更新时间
}

// ============================================================================
// Event - 执行事件
// ============================================================================

// Event 表示 Run 执行过程中产生的事件
//
// 事件是 Agent 执行过程的实时记录，用于：
//   - 实时监控：通过 WebSocket 推送到前端
//   - 审计追溯：记录 Agent 的每一步操作
//   - 调试分析：定位问题和优化性能
//
// 事件类型（Type）包括：
//   - run_started：执行开始
//   - message：Agent 输出的文本消息
//   - tool_use_start：开始使用工具
//   - tool_result：工具执行结果
//   - command：执行命令
//   - command_output：命令输出
//   - file_read/file_write：文件操作
//   - run_completed：执行完成
//   - error：错误事件
//
// 字段说明：
//   - ID：自增主键
//   - RunID：所属 Run ID
//   - Seq：事件序号（Run 内递增）
//   - Type：事件类型
//   - Timestamp：事件发生时间
//   - Payload：事件数据（JSON）
//   - Raw：原始输出（可选，用于调试）
type Event struct {
	ID        int64           `json:"id" db:"id"`                     // 事件 ID
	RunID     string          `json:"run_id" db:"run_id"`             // 所属 Run ID
	Seq       int             `json:"seq" db:"seq"`                   // 事件序号
	Type      string          `json:"type" db:"type"`                 // 事件类型
	Timestamp time.Time       `json:"timestamp" db:"timestamp"`       // 事件时间
	Payload   json.RawMessage `json:"payload,omitempty" db:"payload"` // 事件数据
	Raw       *string         `json:"raw,omitempty" db:"raw"`         // 原始输出
}

// ============================================================================
// Node - 计算节点
// ============================================================================

// Node 表示执行任务的计算节点
//
// Node 是 Node Agent 在 Control Plane 的注册信息：
//   - Node Agent 启动后向 API Server 注册
//   - 定期发送心跳保持在线状态
//   - 调度器根据 Node 状态和容量分配任务
//
// 字段说明：
//   - ID：节点唯一标识（通常是主机名或 UUID）
//   - Status：节点当前状态
//   - Labels：节点标签（用于调度匹配，如 os=linux, gpu=true）
//   - Capacity：节点容量（如 max_concurrent=4）
//   - LastHeartbeat：最后心跳时间（用于判断节点是否在线）
type Node struct {
	ID            string          `json:"id" db:"id"`                                   // 节点 ID
	Status        NodeStatus      `json:"status" db:"status"`                           // 节点状态
	Labels        json.RawMessage `json:"labels" db:"labels"`                           // 节点标签
	Capacity      json.RawMessage `json:"capacity" db:"capacity"`                       // 节点容量
	LastHeartbeat *time.Time      `json:"last_heartbeat,omitempty" db:"last_heartbeat"` // 最后心跳
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`                   // 创建时间
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`                   // 更新时间
}

// ============================================================================
// Artifact - 执行产物
// ============================================================================

// Artifact 表示 Run 产生的文件产物
//
// 产物是 Agent 执行过程中生成的文件：
//   - 代码变更（diff 文件）
//   - 事件日志（JSONL 文件）
//   - 生成的文件（如图片、文档等）
//
// 产物存储在对象存储（如 MinIO）中，Artifact 记录元数据。
//
// 字段说明：
//   - ID：自增主键
//   - RunID：所属 Run ID
//   - Name：产物名称（如 "events.jsonl"）
//   - Path：存储路径（对象存储 Key）
//   - Size：文件大小（字节）
//   - ContentType：MIME 类型
type Artifact struct {
	ID          int64     `json:"id" db:"id"`                               // 产物 ID
	RunID       string    `json:"run_id" db:"run_id"`                       // 所属 Run ID
	Name        string    `json:"name" db:"name"`                           // 产物名称
	Path        string    `json:"path" db:"path"`                           // 存储路径
	Size        *int64    `json:"size,omitempty" db:"size"`                 // 文件大小
	ContentType *string   `json:"content_type,omitempty" db:"content_type"` // MIME 类型
	CreatedAt   time.Time `json:"created_at" db:"created_at"`               // 创建时间
}

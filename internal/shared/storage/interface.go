// Package storage 定义持久化存储层抽象接口
//
// 设计原则：依赖倒置 (DIP)
//   - 调用方只依赖接口，不知道具体实现
//   - 具体实现在子包中：postgres/, etcd/
//   - 初始化时通过依赖注入传入实现
//
// 注意：缓存、事件总线、队列已迁移至独立包：
//   - cache/：缓存接口
//   - eventbus/：事件总线接口
//   - queue/：消息队列接口
package storage

import (
	"context"
	"encoding/json"
	"time"

	"agents-admin/internal/shared/model"
	"agents-admin/internal/shared/storagetypes"

	// 导入新包用于类型重导出（向后兼容）
	"agents-admin/internal/shared/cache"
	"agents-admin/internal/shared/eventbus"
	"agents-admin/internal/shared/queue"
)

// ============================================================================
// 类型重导出（向后兼容）
// ============================================================================

// 从 cache 包重导出
type (
	// Deprecated: 使用 cache.AuthSession
	AuthSession = cache.AuthSession
	// Deprecated: 使用 cache.WorkflowState
	RedisWorkflowState = cache.WorkflowState
	// Deprecated: 使用 cache.NodeStatus
	NodeStatus = cache.NodeStatus
)

// 从 eventbus 包重导出
type (
	// Deprecated: 使用 eventbus.WorkflowEvent
	RedisWorkflowEvent = eventbus.WorkflowEvent
	// Deprecated: 使用 eventbus.RunEvent
	RunEvent = eventbus.RunEvent
)

// 从 queue 包重导出
type (
	// Deprecated: 使用 queue.SchedulerMessage
	SchedulerMessage = queue.SchedulerMessage
	// Deprecated: 使用 queue.NodeTaskMessage
	NodeTaskMessage = queue.NodeTaskMessage
)

// 从 storagetypes 包重导出（etcd 相关）
type (
	EtcdHeartbeat     = storagetypes.EtcdHeartbeat
	WorkflowState     = storagetypes.WorkflowState
	WorkflowEvent     = storagetypes.WorkflowEvent
	WorkflowStateData = storagetypes.WorkflowStateData
)

// ============================================================================
// 常量重导出（向后兼容）
// ============================================================================

// Deprecated: 使用 cache.KeyAuthSession
const KeyAuthSession = cache.KeyAuthSession

// Deprecated: 使用 cache.KeyAuthSessionByAccount
const KeyAuthSessionByAccount = cache.KeyAuthSessionByAccount

// Deprecated: 使用 cache.KeyWorkflowState
const KeyWorkflowState = cache.KeyWorkflowState

// Deprecated: 使用 cache.KeyNodeHeartbeat
const KeyNodeHeartbeat = cache.KeyNodeHeartbeat

// Deprecated: 使用 cache.KeyOnlineNodes
const KeyOnlineNodes = cache.KeyOnlineNodes

// Deprecated: 使用 eventbus.KeyWorkflowEvents
const KeyWorkflowEvents = eventbus.KeyWorkflowEvents

// Deprecated: 使用 eventbus.KeyRunEvents
const KeyRunEvents = eventbus.KeyRunEvents

// Deprecated: 使用 queue.KeyTasksPending
const KeyTasksPending = queue.KeyTasksPending

// Deprecated: 使用 queue.KeyNodeTasks
const KeyNodeTasks = queue.KeyNodeTasks

// Deprecated: 使用 queue.KeyNodeTasksSuffix
const KeyNodeTasksSuffix = queue.KeyNodeTasksSuffix

// Deprecated: 使用 eventbus.MaxStreamLength
const MaxStreamLength = eventbus.MaxStreamLength

// Deprecated: 使用 queue.SchedulerConsumerGroup
const SchedulerConsumerGroup = queue.SchedulerConsumerGroup

// Deprecated: 使用 queue.NodeManagerConsumerGroup
const NodeManagerConsumerGroup = queue.NodeManagerConsumerGroup

// TTL 常量
var (
	TTLAuthSession   = cache.TTLAuthSession
	TTLWorkflowState = cache.TTLWorkflowState
	TTLNodeHeartbeat = cache.TTLNodeHeartbeat
)

// 工作流状态常量
const (
	WorkflowStatePending   = storagetypes.WorkflowStatePending
	WorkflowStateRunning   = storagetypes.WorkflowStateRunning
	WorkflowStateWaiting   = storagetypes.WorkflowStateWaiting
	WorkflowStateCompleted = storagetypes.WorkflowStateCompleted
	WorkflowStateFailed    = storagetypes.WorkflowStateFailed
	WorkflowStateCancelled = storagetypes.WorkflowStateCancelled
)

// ============================================================================
// 持久化存储接口（由 postgres.Store 实现）
// ============================================================================

// TaskFilter 任务查询过滤条件（类型重导出，避免循环导入）
type TaskFilter = storagetypes.TaskFilter

// TaskStore 任务存储接口
type TaskStore interface {
	CreateTask(ctx context.Context, task *model.Task) error
	GetTask(ctx context.Context, id string) (*model.Task, error)
	ListTasks(ctx context.Context, status string, limit, offset int) ([]*model.Task, error)
	ListTasksWithFilter(ctx context.Context, filter TaskFilter) ([]*model.Task, int, error)
	UpdateTaskStatus(ctx context.Context, id string, status model.TaskStatus) error
	DeleteTask(ctx context.Context, id string) error
	UpdateTaskContext(ctx context.Context, id string, taskContext json.RawMessage) error
	ListSubTasks(ctx context.Context, parentID string) ([]*model.Task, error)
	GetTaskTree(ctx context.Context, rootID string) ([]*model.Task, error)
}

// RunStore Run 存储接口
type RunStore interface {
	CreateRun(ctx context.Context, run *model.Run) error
	GetRun(ctx context.Context, id string) (*model.Run, error)
	ListRunsByTask(ctx context.Context, taskID string) ([]*model.Run, error)
	ListRunsByNode(ctx context.Context, nodeID string) ([]*model.Run, error)
	ListRunningRuns(ctx context.Context, limit int) ([]*model.Run, error)
	ListQueuedRuns(ctx context.Context, limit int) ([]*model.Run, error)
	ListStaleQueuedRuns(ctx context.Context, threshold time.Duration) ([]*model.Run, error)
	ResetRunToQueued(ctx context.Context, id string) error
	UpdateRunStatus(ctx context.Context, id string, status model.RunStatus, nodeID *string) error
	UpdateRunError(ctx context.Context, id string, errMsg string) error
	DeleteRun(ctx context.Context, id string) error
}

// EventStore 事件存储接口（归档）
type EventStore interface {
	CreateEvents(ctx context.Context, events []*model.Event) error
	CountEventsByRun(ctx context.Context, runID string) (int, error)
	GetEventsByRun(ctx context.Context, runID string, fromSeq int, limit int) ([]*model.Event, error)
}

// NodeStore 节点存储接口
type NodeStore interface {
	UpsertNode(ctx context.Context, node *model.Node) error
	UpsertNodeHeartbeat(ctx context.Context, node *model.Node) error // 心跳专用，不覆盖管理员设置的 status
	GetNode(ctx context.Context, id string) (*model.Node, error)
	ListAllNodes(ctx context.Context) ([]*model.Node, error)
	ListOnlineNodes(ctx context.Context) ([]*model.Node, error)
	DeactivateStaleNodes(ctx context.Context, activeNodeID string, hostname string) error
	DeleteNode(ctx context.Context, id string) error
	CreateNodeProvision(ctx context.Context, p *model.NodeProvision) error
	UpdateNodeProvision(ctx context.Context, p *model.NodeProvision) error
	GetNodeProvision(ctx context.Context, id string) (*model.NodeProvision, error)
	ListNodeProvisions(ctx context.Context) ([]*model.NodeProvision, error)
}

// AccountStore 账号存储接口
type AccountStore interface {
	CreateAccount(ctx context.Context, account *model.Account) error
	GetAccount(ctx context.Context, id string) (*model.Account, error)
	ListAccounts(ctx context.Context) ([]*model.Account, error)
	UpdateAccountStatus(ctx context.Context, id string, status model.AccountStatus) error
	UpdateAccountVolume(ctx context.Context, id string, volumeName string) error
	UpdateAccountVolumeArchive(ctx context.Context, id string, archiveKey string) error
	DeleteAccount(ctx context.Context, id string) error
}

// AuthTaskStore 认证任务存储接口
type AuthTaskStore interface {
	CreateAuthTask(ctx context.Context, task *model.AuthTask) error
	GetAuthTask(ctx context.Context, id string) (*model.AuthTask, error)
	GetAuthTaskByAccountID(ctx context.Context, accountID string) (*model.AuthTask, error)
	ListRecentAuthTasks(ctx context.Context, limit int) ([]*model.AuthTask, error)
	ListPendingAuthTasks(ctx context.Context, limit int) ([]*model.AuthTask, error)
	ListAuthTasksByNode(ctx context.Context, nodeID string) ([]*model.AuthTask, error)
	UpdateAuthTaskAssignment(ctx context.Context, id string, nodeID string) error
	UpdateAuthTaskStatus(ctx context.Context, id string, status model.AuthTaskStatus, terminalPort *int, terminalURL *string, containerName *string, message *string) error
}

// OperationStore Operation 存储接口
type OperationStore interface {
	CreateOperation(ctx context.Context, op *model.Operation) error
	GetOperation(ctx context.Context, id string) (*model.Operation, error)
	ListOperations(ctx context.Context, opType string, status string, limit, offset int) ([]*model.Operation, error)
	UpdateOperationStatus(ctx context.Context, id string, status model.OperationStatus) error
}

// ActionStore Action 存储接口
type ActionStore interface {
	CreateAction(ctx context.Context, action *model.Action) error
	GetAction(ctx context.Context, id string) (*model.Action, error)
	GetActionWithOperation(ctx context.Context, id string) (*model.Action, error)
	ListActionsByOperation(ctx context.Context, operationID string) ([]*model.Action, error)
	ListActionsByNode(ctx context.Context, nodeID string, status string) ([]*model.Action, error)
	UpdateActionStatus(ctx context.Context, id string, status model.ActionStatus, phase model.ActionPhase, message string, progress int, result json.RawMessage, errMsg string) error
}

// ProxyStore 代理存储接口
type ProxyStore interface {
	CreateProxy(ctx context.Context, proxy *model.Proxy) error
	GetProxy(ctx context.Context, id string) (*model.Proxy, error)
	ListProxies(ctx context.Context) ([]*model.Proxy, error)
	GetDefaultProxy(ctx context.Context) (*model.Proxy, error)
	UpdateProxy(ctx context.Context, proxy *model.Proxy) error
	SetDefaultProxy(ctx context.Context, id string) error
	ClearDefaultProxy(ctx context.Context) error
	DeleteProxy(ctx context.Context, id string) error
}

// AgentInstanceStore Agent 实例存储接口（原 InstanceStore，已重命名对齐领域模型）
type AgentInstanceStore interface {
	CreateAgentInstance(ctx context.Context, instance *model.Instance) error
	GetAgentInstance(ctx context.Context, id string) (*model.Instance, error)
	ListAgentInstances(ctx context.Context) ([]*model.Instance, error)
	ListAgentInstancesByNode(ctx context.Context, nodeID string) ([]*model.Instance, error)
	ListPendingAgentInstances(ctx context.Context, nodeID string) ([]*model.Instance, error)
	UpdateAgentInstance(ctx context.Context, id string, status model.InstanceStatus, containerName *string) error
	DeleteAgentInstance(ctx context.Context, id string) error
}

// InstanceStore 向后兼容别名
// Deprecated: 使用 AgentInstanceStore
type InstanceStore = AgentInstanceStore

// TerminalSessionStore 终端会话存储接口
type TerminalSessionStore interface {
	CreateTerminalSession(ctx context.Context, session *model.TerminalSession) error
	GetTerminalSession(ctx context.Context, id string) (*model.TerminalSession, error)
	ListTerminalSessions(ctx context.Context) ([]*model.TerminalSession, error)
	ListTerminalSessionsByNode(ctx context.Context, nodeID string) ([]*model.TerminalSession, error)
	ListPendingTerminalSessions(ctx context.Context, nodeID string) ([]*model.TerminalSession, error)
	UpdateTerminalSession(ctx context.Context, id string, status model.TerminalSessionStatus, port *int, url *string) error
	DeleteTerminalSession(ctx context.Context, id string) error
	CleanupExpiredTerminalSessions(ctx context.Context) (int64, error)
}

// HITLStore Human-in-the-Loop 存储接口
type HITLStore interface {
	CreateApprovalRequest(ctx context.Context, req *model.ApprovalRequest) error
	GetApprovalRequest(ctx context.Context, id string) (*model.ApprovalRequest, error)
	ListApprovalRequests(ctx context.Context, runID string, status string) ([]*model.ApprovalRequest, error)
	UpdateApprovalRequestStatus(ctx context.Context, id string, status model.ApprovalStatus) error
	CreateApprovalDecision(ctx context.Context, decision *model.ApprovalDecision) error
	CreateFeedback(ctx context.Context, feedback *model.HumanFeedback) error
	ListFeedbacks(ctx context.Context, runID string) ([]*model.HumanFeedback, error)
	MarkFeedbackProcessed(ctx context.Context, id string) error
	CreateIntervention(ctx context.Context, intervention *model.Intervention) error
	ListInterventions(ctx context.Context, runID string) ([]*model.Intervention, error)
	UpdateInterventionExecuted(ctx context.Context, id string) error
	CreateConfirmation(ctx context.Context, confirmation *model.Confirmation) error
	GetConfirmation(ctx context.Context, id string) (*model.Confirmation, error)
	ListConfirmations(ctx context.Context, runID string, status string) ([]*model.Confirmation, error)
	UpdateConfirmationStatus(ctx context.Context, id string, status model.ConfirmStatus, selectedOption *string) error
}

// TemplateStore 模板存储接口
type TemplateStore interface {
	CreateTaskTemplate(ctx context.Context, tmpl *model.TaskTemplate) error
	GetTaskTemplate(ctx context.Context, id string) (*model.TaskTemplate, error)
	ListTaskTemplates(ctx context.Context, category string) ([]*model.TaskTemplate, error)
	DeleteTaskTemplate(ctx context.Context, id string) error
	CreateAgentTemplate(ctx context.Context, tmpl *model.AgentTemplate) error
	GetAgentTemplate(ctx context.Context, id string) (*model.AgentTemplate, error)
	ListAgentTemplates(ctx context.Context, category string) ([]*model.AgentTemplate, error)
	UpdateAgentTemplate(ctx context.Context, tmpl *model.AgentTemplate) error
	DeleteAgentTemplate(ctx context.Context, id string) error
}

// SkillStore 技能存储接口
type SkillStore interface {
	CreateSkill(ctx context.Context, skill *model.Skill) error
	GetSkill(ctx context.Context, id string) (*model.Skill, error)
	ListSkills(ctx context.Context, category string) ([]*model.Skill, error)
	DeleteSkill(ctx context.Context, id string) error
}

// MCPServerStore MCP Server 存储接口
type MCPServerStore interface {
	CreateMCPServer(ctx context.Context, server *model.MCPServer) error
	GetMCPServer(ctx context.Context, id string) (*model.MCPServer, error)
	ListMCPServers(ctx context.Context, source string) ([]*model.MCPServer, error)
	DeleteMCPServer(ctx context.Context, id string) error
}

// SecurityPolicyStore 安全策略存储接口
type SecurityPolicyStore interface {
	CreateSecurityPolicy(ctx context.Context, policy *model.SecurityPolicyEntity) error
	GetSecurityPolicy(ctx context.Context, id string) (*model.SecurityPolicyEntity, error)
	ListSecurityPolicies(ctx context.Context, category string) ([]*model.SecurityPolicyEntity, error)
	DeleteSecurityPolicy(ctx context.Context, id string) error
}

// ============================================================================
// etcd 心跳接口（由 etcd.Store 实现）
// ============================================================================

// EtcdNodeHeartbeat etcd 节点心跳接口
type EtcdNodeHeartbeat interface {
	UpdateNodeHeartbeat(ctx context.Context, hb *EtcdHeartbeat) error
	GetNodeHeartbeat(ctx context.Context, nodeID string) (*EtcdHeartbeat, error)
	ListNodeHeartbeats(ctx context.Context) ([]*EtcdHeartbeat, error)
	IsNodeOnline(ctx context.Context, nodeID string) bool
}

// ============================================================================
// 组合接口
// ============================================================================

// UserStore 用户存储接口
type UserStore interface {
	CreateUser(ctx context.Context, user *model.User) error
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	GetUserByID(ctx context.Context, id string) (*model.User, error)
	UpdateUserPassword(ctx context.Context, id, passwordHash string) error
	ListUsers(ctx context.Context) ([]*model.User, error)
}

// PersistentStore 持久化存储组合接口
type PersistentStore interface {
	TaskStore
	RunStore
	EventStore
	NodeStore
	AccountStore
	AuthTaskStore
	OperationStore
	ActionStore
	ProxyStore
	InstanceStore
	TerminalSessionStore
	HITLStore
	TemplateStore
	SkillStore
	MCPServerStore
	SecurityPolicyStore
	UserStore
	Close() error
}

// ============================================================================
// 废弃接口（向后兼容，请使用新包）
// ============================================================================

// AuthSessionCache 认证会话缓存接口
// Deprecated: 使用 cache.AuthSessionCache
type AuthSessionCache = cache.AuthSessionCache

// WorkflowStateCache 工作流状态缓存接口
// Deprecated: 使用 cache.WorkflowStateCache
type WorkflowStateCache = cache.WorkflowStateCache

// NodeHeartbeatCache 节点心跳缓存接口
// Deprecated: 使用 cache.NodeHeartbeatCache
type NodeHeartbeatCache = cache.NodeHeartbeatCache

// WorkflowEventBus 工作流事件总线接口
// Deprecated: 使用 eventbus.WorkflowEventBus
type WorkflowEventBus = eventbus.WorkflowEventBus

// RunEventBus Run 事件总线接口
// Deprecated: 使用 eventbus.RunEventBus
type RunEventBus = eventbus.RunEventBus

// SchedulerQueue 调度队列接口
// Deprecated: 使用 queue.SchedulerQueue
type SchedulerQueue = queue.SchedulerQueue

// NodeTaskQueue 节点任务队列接口
// Deprecated: 使用 queue.NodeTaskQueue
type NodeTaskQueue = queue.NodeTaskQueue

// CacheStore 缓存存储组合接口
// Deprecated: 使用 cache.Cache + eventbus.EventBus + queue.Queue
type CacheStore interface {
	cache.Cache
	eventbus.EventBus
	queue.Queue
}

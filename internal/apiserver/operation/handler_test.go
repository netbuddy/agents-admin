package operation

import (
	"context"
	"encoding/json"
	"time"

	"agents-admin/internal/shared/model"
	"agents-admin/internal/shared/storage"
)

// ============================================================================
// Mock Store — 实现 storage.PersistentStore 接口中 operation 包用到的子集
// ============================================================================

type mockStore struct {
	// Operation/Action 存储
	operations map[string]*model.Operation
	actions    map[string]*model.Action

	// Account 存储
	accounts map[string]*model.Account

	// Node 存储
	nodes map[string]*model.Node
}

func newMockStore() *mockStore {
	return &mockStore{
		operations: make(map[string]*model.Operation),
		actions:    make(map[string]*model.Action),
		accounts:   make(map[string]*model.Account),
		nodes:      make(map[string]*model.Node),
	}
}

// --- OperationStore ---

func (m *mockStore) CreateOperation(_ context.Context, op *model.Operation) error {
	m.operations[op.ID] = op
	return nil
}

func (m *mockStore) GetOperation(_ context.Context, id string) (*model.Operation, error) {
	return m.operations[id], nil
}

func (m *mockStore) ListOperations(_ context.Context, opType string, status string, limit, offset int) ([]*model.Operation, error) {
	var result []*model.Operation
	for _, op := range m.operations {
		if opType != "" && string(op.Type) != opType {
			continue
		}
		if status != "" && string(op.Status) != status {
			continue
		}
		result = append(result, op)
	}
	return result, nil
}

func (m *mockStore) UpdateOperationStatus(_ context.Context, id string, status model.OperationStatus) error {
	if op, ok := m.operations[id]; ok {
		op.Status = status
		if status == model.OperationStatusCompleted || status == model.OperationStatusFailed {
			now := time.Now()
			op.FinishedAt = &now
		}
	}
	return nil
}

// --- ActionStore ---

func (m *mockStore) CreateAction(_ context.Context, action *model.Action) error {
	m.actions[action.ID] = action
	return nil
}

func (m *mockStore) GetAction(_ context.Context, id string) (*model.Action, error) {
	return m.actions[id], nil
}

func (m *mockStore) GetActionWithOperation(_ context.Context, id string) (*model.Action, error) {
	action := m.actions[id]
	if action == nil {
		return nil, nil
	}
	if op, ok := m.operations[action.OperationID]; ok {
		action.Operation = op
	}
	return action, nil
}

func (m *mockStore) ListActionsByOperation(_ context.Context, operationID string) ([]*model.Action, error) {
	var result []*model.Action
	for _, a := range m.actions {
		if a.OperationID == operationID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockStore) ListActionsByNode(_ context.Context, nodeID string, status string) ([]*model.Action, error) {
	var result []*model.Action
	for _, a := range m.actions {
		op := m.operations[a.OperationID]
		if op == nil || op.NodeID != nodeID {
			continue
		}
		if status != "" && string(a.Status) != status {
			continue
		}
		a.Operation = op
		result = append(result, a)
	}
	return result, nil
}

func (m *mockStore) UpdateActionStatus(_ context.Context, id string, status model.ActionStatus, phase model.ActionPhase, message string, progress int, result json.RawMessage, errMsg string) error {
	if a, ok := m.actions[id]; ok {
		a.Status = status
		a.Phase = phase
		a.Message = message
		a.Progress = progress
		a.Result = result
		a.Error = errMsg
	}
	return nil
}

// --- AccountStore ---

func (m *mockStore) CreateAccount(_ context.Context, account *model.Account) error {
	m.accounts[account.ID] = account
	return nil
}

func (m *mockStore) GetAccount(_ context.Context, id string) (*model.Account, error) {
	return m.accounts[id], nil
}

func (m *mockStore) ListAccounts(_ context.Context) ([]*model.Account, error) {
	var result []*model.Account
	for _, a := range m.accounts {
		result = append(result, a)
	}
	return result, nil
}

func (m *mockStore) UpdateAccountStatus(_ context.Context, id string, status model.AccountStatus) error {
	if a, ok := m.accounts[id]; ok {
		a.Status = status
	}
	return nil
}

func (m *mockStore) UpdateAccountVolumeArchive(_ context.Context, _ string, _ string) error {
	return nil
}

func (m *mockStore) UpdateAccountVolume(_ context.Context, id string, volumeName string) error {
	if a, ok := m.accounts[id]; ok {
		a.VolumeName = &volumeName
	}
	return nil
}

func (m *mockStore) DeleteAccount(_ context.Context, id string) error {
	delete(m.accounts, id)
	return nil
}

// --- NodeStore ---

func (m *mockStore) UpsertNode(_ context.Context, node *model.Node) error {
	m.nodes[node.ID] = node
	return nil
}

func (m *mockStore) UpsertNodeHeartbeat(_ context.Context, node *model.Node) error {
	m.nodes[node.ID] = node
	return nil
}

func (m *mockStore) GetNode(_ context.Context, id string) (*model.Node, error) {
	return m.nodes[id], nil
}

func (m *mockStore) ListAllNodes(_ context.Context) ([]*model.Node, error) {
	return nil, nil
}

func (m *mockStore) ListOnlineNodes(_ context.Context) ([]*model.Node, error) {
	return nil, nil
}

func (m *mockStore) DeleteNode(_ context.Context, id string) error {
	return nil
}
func (m *mockStore) DeactivateStaleNodes(_ context.Context, _ string, _ string) error {
	return nil
}
func (m *mockStore) CreateNodeProvision(_ context.Context, _ *model.NodeProvision) error {
	return nil
}
func (m *mockStore) UpdateNodeProvision(_ context.Context, _ *model.NodeProvision) error {
	return nil
}
func (m *mockStore) GetNodeProvision(_ context.Context, _ string) (*model.NodeProvision, error) {
	return nil, nil
}
func (m *mockStore) ListNodeProvisions(_ context.Context) ([]*model.NodeProvision, error) {
	return nil, nil
}

// --- 以下为 PersistentStore 接口中其他 Store 的空实现（满足接口） ---

func (m *mockStore) Close() error { return nil }

// TaskStore
func (m *mockStore) CreateTask(_ context.Context, _ *model.Task) error        { return nil }
func (m *mockStore) GetTask(_ context.Context, _ string) (*model.Task, error) { return nil, nil }
func (m *mockStore) ListTasks(_ context.Context, _ string, _, _ int) ([]*model.Task, error) {
	return nil, nil
}
func (m *mockStore) ListTasksWithFilter(_ context.Context, _ storage.TaskFilter) ([]*model.Task, int, error) {
	return nil, 0, nil
}
func (m *mockStore) UpdateTaskStatus(_ context.Context, _ string, _ model.TaskStatus) error {
	return nil
}
func (m *mockStore) DeleteTask(_ context.Context, _ string) error { return nil }
func (m *mockStore) UpdateTaskContext(_ context.Context, _ string, _ json.RawMessage) error {
	return nil
}
func (m *mockStore) ListSubTasks(_ context.Context, _ string) ([]*model.Task, error) {
	return nil, nil
}
func (m *mockStore) GetTaskTree(_ context.Context, _ string) ([]*model.Task, error) {
	return nil, nil
}

// RunStore
func (m *mockStore) CreateRun(_ context.Context, _ *model.Run) error        { return nil }
func (m *mockStore) GetRun(_ context.Context, _ string) (*model.Run, error) { return nil, nil }
func (m *mockStore) ListRunsByTask(_ context.Context, _ string) ([]*model.Run, error) {
	return nil, nil
}
func (m *mockStore) ListRunsByNode(_ context.Context, _ string) ([]*model.Run, error) {
	return nil, nil
}
func (m *mockStore) ListRunningRuns(_ context.Context, _ int) ([]*model.Run, error) { return nil, nil }
func (m *mockStore) ListQueuedRuns(_ context.Context, _ int) ([]*model.Run, error)  { return nil, nil }
func (m *mockStore) ListStaleQueuedRuns(_ context.Context, _ time.Duration) ([]*model.Run, error) {
	return nil, nil
}
func (m *mockStore) ResetRunToQueued(_ context.Context, _ string) error { return nil }
func (m *mockStore) UpdateRunStatus(_ context.Context, _ string, _ model.RunStatus, _ *string) error {
	return nil
}
func (m *mockStore) UpdateRunError(_ context.Context, _ string, _ string) error { return nil }
func (m *mockStore) DeleteRun(_ context.Context, _ string) error                { return nil }

// EventStore
func (m *mockStore) CreateEvents(_ context.Context, _ []*model.Event) error { return nil }
func (m *mockStore) CountEventsByRun(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (m *mockStore) GetEventsByRun(_ context.Context, _ string, _ int, _ int) ([]*model.Event, error) {
	return nil, nil
}

// AuthTaskStore
func (m *mockStore) CreateAuthTask(_ context.Context, _ *model.AuthTask) error { return nil }
func (m *mockStore) GetAuthTask(_ context.Context, _ string) (*model.AuthTask, error) {
	return nil, nil
}
func (m *mockStore) GetAuthTaskByAccountID(_ context.Context, _ string) (*model.AuthTask, error) {
	return nil, nil
}
func (m *mockStore) ListRecentAuthTasks(_ context.Context, _ int) ([]*model.AuthTask, error) {
	return nil, nil
}
func (m *mockStore) ListPendingAuthTasks(_ context.Context, _ int) ([]*model.AuthTask, error) {
	return nil, nil
}
func (m *mockStore) ListAuthTasksByNode(_ context.Context, _ string) ([]*model.AuthTask, error) {
	return nil, nil
}
func (m *mockStore) UpdateAuthTaskAssignment(_ context.Context, _ string, _ string) error {
	return nil
}
func (m *mockStore) UpdateAuthTaskStatus(_ context.Context, _ string, _ model.AuthTaskStatus, _ *int, _ *string, _ *string, _ *string) error {
	return nil
}

// ProxyStore
func (m *mockStore) CreateProxy(_ context.Context, _ *model.Proxy) error        { return nil }
func (m *mockStore) GetProxy(_ context.Context, _ string) (*model.Proxy, error) { return nil, nil }
func (m *mockStore) ListProxies(_ context.Context) ([]*model.Proxy, error)      { return nil, nil }
func (m *mockStore) GetDefaultProxy(_ context.Context) (*model.Proxy, error)    { return nil, nil }
func (m *mockStore) UpdateProxy(_ context.Context, _ *model.Proxy) error        { return nil }
func (m *mockStore) SetDefaultProxy(_ context.Context, _ string) error          { return nil }
func (m *mockStore) ClearDefaultProxy(_ context.Context) error                  { return nil }
func (m *mockStore) DeleteProxy(_ context.Context, _ string) error              { return nil }

// AgentInstanceStore
func (m *mockStore) CreateAgentInstance(_ context.Context, _ *model.Instance) error { return nil }
func (m *mockStore) GetAgentInstance(_ context.Context, _ string) (*model.Instance, error) {
	return nil, nil
}
func (m *mockStore) ListAgentInstances(_ context.Context) ([]*model.Instance, error) { return nil, nil }
func (m *mockStore) ListAgentInstancesByNode(_ context.Context, _ string) ([]*model.Instance, error) {
	return nil, nil
}
func (m *mockStore) ListPendingAgentInstances(_ context.Context, _ string) ([]*model.Instance, error) {
	return nil, nil
}
func (m *mockStore) UpdateAgentInstance(_ context.Context, _ string, _ model.InstanceStatus, _ *string) error {
	return nil
}
func (m *mockStore) DeleteAgentInstance(_ context.Context, _ string) error { return nil }

// TerminalSessionStore
func (m *mockStore) CreateTerminalSession(_ context.Context, _ *model.TerminalSession) error {
	return nil
}
func (m *mockStore) GetTerminalSession(_ context.Context, _ string) (*model.TerminalSession, error) {
	return nil, nil
}
func (m *mockStore) ListTerminalSessions(_ context.Context) ([]*model.TerminalSession, error) {
	return nil, nil
}
func (m *mockStore) ListTerminalSessionsByNode(_ context.Context, _ string) ([]*model.TerminalSession, error) {
	return nil, nil
}
func (m *mockStore) ListPendingTerminalSessions(_ context.Context, _ string) ([]*model.TerminalSession, error) {
	return nil, nil
}
func (m *mockStore) UpdateTerminalSession(_ context.Context, _ string, _ model.TerminalSessionStatus, _ *int, _ *string) error {
	return nil
}
func (m *mockStore) DeleteTerminalSession(_ context.Context, _ string) error { return nil }
func (m *mockStore) CleanupExpiredTerminalSessions(_ context.Context) (int64, error) {
	return 0, nil
}

// HITLStore
func (m *mockStore) CreateApprovalRequest(_ context.Context, _ *model.ApprovalRequest) error {
	return nil
}
func (m *mockStore) GetApprovalRequest(_ context.Context, _ string) (*model.ApprovalRequest, error) {
	return nil, nil
}
func (m *mockStore) ListApprovalRequests(_ context.Context, _ string, _ string) ([]*model.ApprovalRequest, error) {
	return nil, nil
}
func (m *mockStore) UpdateApprovalRequestStatus(_ context.Context, _ string, _ model.ApprovalStatus) error {
	return nil
}
func (m *mockStore) CreateApprovalDecision(_ context.Context, _ *model.ApprovalDecision) error {
	return nil
}
func (m *mockStore) CreateFeedback(_ context.Context, _ *model.HumanFeedback) error { return nil }
func (m *mockStore) ListFeedbacks(_ context.Context, _ string) ([]*model.HumanFeedback, error) {
	return nil, nil
}
func (m *mockStore) MarkFeedbackProcessed(_ context.Context, _ string) error { return nil }
func (m *mockStore) CreateIntervention(_ context.Context, _ *model.Intervention) error {
	return nil
}
func (m *mockStore) ListInterventions(_ context.Context, _ string) ([]*model.Intervention, error) {
	return nil, nil
}
func (m *mockStore) UpdateInterventionExecuted(_ context.Context, _ string) error { return nil }
func (m *mockStore) CreateConfirmation(_ context.Context, _ *model.Confirmation) error {
	return nil
}
func (m *mockStore) GetConfirmation(_ context.Context, _ string) (*model.Confirmation, error) {
	return nil, nil
}
func (m *mockStore) ListConfirmations(_ context.Context, _ string, _ string) ([]*model.Confirmation, error) {
	return nil, nil
}
func (m *mockStore) UpdateConfirmationStatus(_ context.Context, _ string, _ model.ConfirmStatus, _ *string) error {
	return nil
}

// TemplateStore
func (m *mockStore) CreateTaskTemplate(_ context.Context, _ *model.TaskTemplate) error { return nil }
func (m *mockStore) GetTaskTemplate(_ context.Context, _ string) (*model.TaskTemplate, error) {
	return nil, nil
}
func (m *mockStore) ListTaskTemplates(_ context.Context, _ string) ([]*model.TaskTemplate, error) {
	return nil, nil
}
func (m *mockStore) DeleteTaskTemplate(_ context.Context, _ string) error { return nil }
func (m *mockStore) CreateAgentTemplate(_ context.Context, _ *model.AgentTemplate) error {
	return nil
}
func (m *mockStore) GetAgentTemplate(_ context.Context, _ string) (*model.AgentTemplate, error) {
	return nil, nil
}
func (m *mockStore) ListAgentTemplates(_ context.Context, _ string) ([]*model.AgentTemplate, error) {
	return nil, nil
}
func (m *mockStore) DeleteAgentTemplate(_ context.Context, _ string) error { return nil }

// SkillStore
func (m *mockStore) CreateSkill(_ context.Context, _ *model.Skill) error            { return nil }
func (m *mockStore) GetSkill(_ context.Context, _ string) (*model.Skill, error)     { return nil, nil }
func (m *mockStore) ListSkills(_ context.Context, _ string) ([]*model.Skill, error) { return nil, nil }
func (m *mockStore) DeleteSkill(_ context.Context, _ string) error                  { return nil }

// MCPServerStore
func (m *mockStore) CreateMCPServer(_ context.Context, _ *model.MCPServer) error { return nil }
func (m *mockStore) GetMCPServer(_ context.Context, _ string) (*model.MCPServer, error) {
	return nil, nil
}
func (m *mockStore) ListMCPServers(_ context.Context, _ string) ([]*model.MCPServer, error) {
	return nil, nil
}
func (m *mockStore) DeleteMCPServer(_ context.Context, _ string) error { return nil }

// SecurityPolicyStore
func (m *mockStore) CreateSecurityPolicy(_ context.Context, _ *model.SecurityPolicyEntity) error {
	return nil
}
func (m *mockStore) GetSecurityPolicy(_ context.Context, _ string) (*model.SecurityPolicyEntity, error) {
	return nil, nil
}
func (m *mockStore) ListSecurityPolicies(_ context.Context, _ string) ([]*model.SecurityPolicyEntity, error) {
	return nil, nil
}
func (m *mockStore) DeleteSecurityPolicy(_ context.Context, _ string) error { return nil }

// UserStore
func (m *mockStore) CreateUser(_ context.Context, _ *model.User) error { return nil }
func (m *mockStore) GetUserByEmail(_ context.Context, _ string) (*model.User, error) {
	return nil, nil
}
func (m *mockStore) GetUserByID(_ context.Context, _ string) (*model.User, error) { return nil, nil }
func (m *mockStore) UpdateUserPassword(_ context.Context, _, _ string) error      { return nil }
func (m *mockStore) ListUsers(_ context.Context) ([]*model.User, error)           { return nil, nil }

// UpdateAgentTemplate
func (m *mockStore) UpdateAgentTemplate(_ context.Context, _ *model.AgentTemplate) error { return nil }

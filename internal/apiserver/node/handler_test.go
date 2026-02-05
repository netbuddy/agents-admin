package node

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agents-admin/internal/shared/model"
)

// mockStore 模拟存储层
type mockStore struct {
	nodes map[string]*model.Node
	runs  map[string][]*model.Run
}

func newMockStore() *mockStore {
	return &mockStore{
		nodes: make(map[string]*model.Node),
		runs:  make(map[string][]*model.Run),
	}
}

func (m *mockStore) GetNode(ctx context.Context, id string) (*model.Node, error) {
	return m.nodes[id], nil
}

func (m *mockStore) ListAllNodes(ctx context.Context) ([]*model.Node, error) {
	nodes := make([]*model.Node, 0, len(m.nodes))
	for _, n := range m.nodes {
		nodes = append(nodes, n)
	}
	return nodes, nil
}

func (m *mockStore) ListOnlineNodes(ctx context.Context) ([]*model.Node, error) {
	return m.ListAllNodes(ctx)
}

func (m *mockStore) UpsertNode(ctx context.Context, node *model.Node) error {
	m.nodes[node.ID] = node
	return nil
}

func (m *mockStore) UpsertNodeHeartbeat(ctx context.Context, node *model.Node) error {
	m.nodes[node.ID] = node
	return nil
}

func (m *mockStore) DeleteNode(ctx context.Context, id string) error {
	delete(m.nodes, id)
	return nil
}

func (m *mockStore) ListRunsByNode(ctx context.Context, nodeID string) ([]*model.Run, error) {
	return m.runs[nodeID], nil
}

// 实现其他必需的接口方法（空实现）
func (m *mockStore) CreateTask(ctx context.Context, task *model.Task) error      { return nil }
func (m *mockStore) GetTask(ctx context.Context, id string) (*model.Task, error) { return nil, nil }
func (m *mockStore) ListTasks(ctx context.Context, status string, limit, offset int) ([]*model.Task, error) {
	return nil, nil
}
func (m *mockStore) DeleteTask(ctx context.Context, id string) error { return nil }
func (m *mockStore) ListSubTasks(ctx context.Context, parentID string) ([]*model.Task, error) {
	return nil, nil
}
func (m *mockStore) GetTaskTree(ctx context.Context, rootID string) ([]*model.Task, error) {
	return nil, nil
}
func (m *mockStore) UpdateTaskContext(ctx context.Context, id string, context json.RawMessage) error {
	return nil
}
func (m *mockStore) CreateRun(ctx context.Context, run *model.Run) error { return nil }
func (m *mockStore) GetRun(ctx context.Context, id string) (*model.Run, error) {
	return nil, nil
}
func (m *mockStore) ListRuns(ctx context.Context, taskID string, limit, offset int) ([]*model.Run, error) {
	return nil, nil
}
func (m *mockStore) ListQueuedRuns(ctx context.Context, limit int) ([]*model.Run, error) {
	return nil, nil
}
func (m *mockStore) ListRunningRuns(ctx context.Context, limit int) ([]*model.Run, error) {
	return nil, nil
}
func (m *mockStore) UpdateRunStatus(ctx context.Context, id string, status model.RunStatus, nodeID *string) error {
	return nil
}
func (m *mockStore) UpdateRunResult(ctx context.Context, id string, result json.RawMessage) error {
	return nil
}
func (m *mockStore) DeleteRun(ctx context.Context, id string) error                { return nil }
func (m *mockStore) ResetRunToQueued(ctx context.Context, id string) error         { return nil }
func (m *mockStore) CreateEvent(ctx context.Context, event *model.Event) error     { return nil }
func (m *mockStore) GetEvent(ctx context.Context, id string) (*model.Event, error) { return nil, nil }
func (m *mockStore) ListEventsByRun(ctx context.Context, runID string) ([]*model.Event, error) {
	return nil, nil
}
func (m *mockStore) CountEventsByRun(ctx context.Context, runID string) (int, error) { return 0, nil }
func (m *mockStore) DeleteEventsByRun(ctx context.Context, runID string) error       { return nil }
func (m *mockStore) ListAccounts(ctx context.Context) ([]*model.Account, error)      { return nil, nil }
func (m *mockStore) GetAccount(ctx context.Context, id string) (*model.Account, error) {
	return nil, nil
}
func (m *mockStore) CreateAccount(ctx context.Context, account *model.Account) error { return nil }
func (m *mockStore) UpdateAccount(ctx context.Context, account *model.Account) error { return nil }
func (m *mockStore) DeleteAccount(ctx context.Context, id string) error              { return nil }
func (m *mockStore) ListInstances(ctx context.Context) ([]*model.Instance, error)    { return nil, nil }
func (m *mockStore) GetInstance(ctx context.Context, id string) (*model.Instance, error) {
	return nil, nil
}
func (m *mockStore) CreateInstance(ctx context.Context, instance *model.Instance) error { return nil }
func (m *mockStore) UpdateInstance(ctx context.Context, instance *model.Instance) error { return nil }
func (m *mockStore) DeleteInstance(ctx context.Context, id string) error                { return nil }
func (m *mockStore) ListTaskTemplates(ctx context.Context, category string) ([]*model.TaskTemplate, error) {
	return nil, nil
}
func (m *mockStore) GetTaskTemplate(ctx context.Context, id string) (*model.TaskTemplate, error) {
	return nil, nil
}
func (m *mockStore) CreateTaskTemplate(ctx context.Context, tmpl *model.TaskTemplate) error {
	return nil
}
func (m *mockStore) DeleteTaskTemplate(ctx context.Context, id string) error { return nil }
func (m *mockStore) ListAgentTemplates(ctx context.Context, agentType string) ([]*model.AgentTemplate, error) {
	return nil, nil
}
func (m *mockStore) GetAgentTemplate(ctx context.Context, id string) (*model.AgentTemplate, error) {
	return nil, nil
}
func (m *mockStore) CreateAgentTemplate(ctx context.Context, tmpl *model.AgentTemplate) error {
	return nil
}
func (m *mockStore) DeleteAgentTemplate(ctx context.Context, id string) error { return nil }
func (m *mockStore) ListSkills(ctx context.Context, category string) ([]*model.Skill, error) {
	return nil, nil
}
func (m *mockStore) GetSkill(ctx context.Context, id string) (*model.Skill, error)  { return nil, nil }
func (m *mockStore) CreateSkill(ctx context.Context, skill *model.Skill) error      { return nil }
func (m *mockStore) DeleteSkill(ctx context.Context, id string) error               { return nil }
func (m *mockStore) ListMCPServers(ctx context.Context) ([]*model.MCPServer, error) { return nil, nil }
func (m *mockStore) GetMCPServer(ctx context.Context, id string) (*model.MCPServer, error) {
	return nil, nil
}
func (m *mockStore) CreateMCPServer(ctx context.Context, server *model.MCPServer) error { return nil }
func (m *mockStore) DeleteMCPServer(ctx context.Context, id string) error               { return nil }
func (m *mockStore) ListSecurityPolicies(ctx context.Context) ([]*model.SecurityPolicy, error) {
	return nil, nil
}
func (m *mockStore) GetSecurityPolicy(ctx context.Context, id string) (*model.SecurityPolicy, error) {
	return nil, nil
}
func (m *mockStore) CreateSecurityPolicy(ctx context.Context, policy *model.SecurityPolicy) error {
	return nil
}
func (m *mockStore) DeleteSecurityPolicy(ctx context.Context, id string) error         { return nil }
func (m *mockStore) ListProxies(ctx context.Context) ([]*model.Proxy, error)           { return nil, nil }
func (m *mockStore) GetProxy(ctx context.Context, id string) (*model.Proxy, error)     { return nil, nil }
func (m *mockStore) CreateProxy(ctx context.Context, proxy *model.Proxy) error         { return nil }
func (m *mockStore) UpdateProxy(ctx context.Context, proxy *model.Proxy) error         { return nil }
func (m *mockStore) DeleteProxy(ctx context.Context, id string) error                  { return nil }
func (m *mockStore) CleanupExpiredTerminalSessions(ctx context.Context) (int64, error) { return 0, nil }
func (m *mockStore) ListPendingInstances(ctx context.Context, nodeID string) ([]*model.Instance, error) {
	return nil, nil
}
func (m *mockStore) ListPendingAuthTasks(ctx context.Context, limit int) ([]*model.AuthTask, error) {
	return nil, nil
}
func (m *mockStore) UpdateAuthTaskAssignment(ctx context.Context, id, nodeID string) error {
	return nil
}
func (m *mockStore) UpdateAccountStatus(ctx context.Context, id string, status model.AccountStatus) error {
	return nil
}
func (m *mockStore) UpdateAccountVolume(ctx context.Context, id, volumeName string) error {
	return nil
}

func TestHandler_Heartbeat(t *testing.T) {
	store := newMockStore()
	h := NewHandler(store, nil)

	tests := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name:       "成功心跳",
			body:       map[string]interface{}{"node_id": "node-1", "status": "online"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "缺少 node_id",
			body:       map[string]interface{}{"status": "online"},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/api/v1/nodes/heartbeat", bytes.NewReader(body))
			w := httptest.NewRecorder()

			h.Heartbeat(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestHandler_List(t *testing.T) {
	store := newMockStore()
	now := time.Now()
	store.nodes["node-1"] = &model.Node{
		ID:        "node-1",
		Status:    model.NodeStatusOnline,
		Labels:    []byte(`{"env":"prod"}`),
		Capacity:  []byte(`{"max_concurrent":5}`),
		CreatedAt: now,
		UpdatedAt: now,
	}

	h := NewHandler(store, nil)

	req := httptest.NewRequest("GET", "/api/v1/nodes", nil)
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["count"].(float64) != 1 {
		t.Errorf("expected count 1, got %v", resp["count"])
	}
}

func TestHandler_Get(t *testing.T) {
	store := newMockStore()
	now := time.Now()
	store.nodes["node-1"] = &model.Node{
		ID:        "node-1",
		Status:    model.NodeStatusOnline,
		CreatedAt: now,
		UpdatedAt: now,
	}

	h := NewHandler(store, nil)

	tests := []struct {
		name       string
		nodeID     string
		wantStatus int
	}{
		{
			name:       "节点存在",
			nodeID:     "node-1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "节点不存在",
			nodeID:     "node-999",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/nodes/"+tt.nodeID, nil)
			req.SetPathValue("id", tt.nodeID)
			w := httptest.NewRecorder()

			h.Get(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestHandler_Delete(t *testing.T) {
	store := newMockStore()
	now := time.Now()
	store.nodes["node-1"] = &model.Node{
		ID:        "node-1",
		Status:    model.NodeStatusOnline,
		CreatedAt: now,
		UpdatedAt: now,
	}
	store.nodes["node-2"] = &model.Node{
		ID:        "node-2",
		Status:    model.NodeStatusOnline,
		CreatedAt: now,
		UpdatedAt: now,
	}
	store.runs["node-2"] = []*model.Run{{ID: "run-1"}}

	h := NewHandler(store, nil)

	tests := []struct {
		name       string
		nodeID     string
		wantStatus int
	}{
		{
			name:       "成功删除",
			nodeID:     "node-1",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "节点不存在",
			nodeID:     "node-999",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "节点有运行中任务",
			nodeID:     "node-2",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", "/api/v1/nodes/"+tt.nodeID, nil)
			req.SetPathValue("id", tt.nodeID)
			w := httptest.NewRecorder()

			h.Delete(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestHandler_Update(t *testing.T) {
	store := newMockStore()
	now := time.Now()
	store.nodes["node-1"] = &model.Node{
		ID:        "node-1",
		Status:    model.NodeStatusOnline,
		Labels:    []byte(`{}`),
		CreatedAt: now,
		UpdatedAt: now,
	}

	h := NewHandler(store, nil)

	tests := []struct {
		name       string
		nodeID     string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name:       "成功更新状态",
			nodeID:     "node-1",
			body:       map[string]interface{}{"status": "draining"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "节点不存在",
			nodeID:     "node-999",
			body:       map[string]interface{}{"status": "draining"},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("PATCH", "/api/v1/nodes/"+tt.nodeID, bytes.NewReader(body))
			req.SetPathValue("id", tt.nodeID)
			w := httptest.NewRecorder()

			h.Update(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

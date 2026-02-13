// Package node 节点管理领域
package node

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"agents-admin/internal/shared/model"
	"agents-admin/internal/shared/storage"
)

// Manager 节点管理器
//
// 负责管理节点的在线状态、容量信息和运行任务计数
type Manager struct {
	store       storage.PersistentStore
	nodeRunning map[string]int // 节点当前运行的任务数（内存缓存）
}

// NewManager 创建节点管理器
func NewManager(store storage.PersistentStore) *Manager {
	return &Manager{
		store:       store,
		nodeRunning: make(map[string]int),
	}
}

// ListOnlineNodes 获取在线节点列表
//
// 基于 MongoDB last_heartbeat 时间窗口过滤，排除行政状态节点
func (m *Manager) ListOnlineNodes(ctx context.Context) ([]*model.Node, error) {
	nodes, err := m.store.ListAllNodes(ctx)
	if err != nil {
		return nil, err
	}

	var online []*model.Node
	for _, n := range FilterNodesByFreshHeartbeat(nodes, HeartbeatFreshWindow) {
		if !n.IsAdminStatus() {
			online = append(online, n)
		}
	}
	return online, nil
}

// RefreshRunningCount 刷新节点运行任务计数
func (m *Manager) RefreshRunningCount(ctx context.Context, nodes []*model.Node) {
	m.nodeRunning = make(map[string]int)

	for _, node := range nodes {
		runs, err := m.store.ListRunsByNode(ctx, node.ID)
		if err != nil {
			log.Printf("[node.manager] list runs for node %s failed: %v", node.ID, err)
			continue
		}
		m.nodeRunning[node.ID] = len(runs)
	}
}

// GetNodeRunning 获取节点运行任务计数
func (m *Manager) GetNodeRunning() map[string]int {
	result := make(map[string]int, len(m.nodeRunning))
	for k, v := range m.nodeRunning {
		result[k] = v
	}
	return result
}

// IncrementRunning 增加节点运行任务计数
func (m *Manager) IncrementRunning(nodeID string) {
	m.nodeRunning[nodeID]++
}

// ResolvePreferredNodeID 解析优先节点 ID（用于亲和性调度）
func (m *Manager) ResolvePreferredNodeID(ctx context.Context, taskID string, snapshot json.RawMessage) string {
	instanceID, _ := ExtractAgentIDs(snapshot)

	// 兼容：如果 instance_id 没放在 snapshot，从 Task.AgentID 补齐
	if instanceID == "" && taskID != "" {
		task, err := m.store.GetTask(ctx, taskID)
		if err != nil {
			log.Printf("[node.manager] GetTask error: %v", err)
		} else if task != nil && task.AgentID != nil && *task.AgentID != "" {
			instanceID = *task.AgentID
		}
	}

	if instanceID != "" {
		inst, err := m.store.GetAgentInstance(ctx, instanceID)
		if err != nil {
			log.Printf("[node.manager] GetAgentInstance error: %v", err)
		} else if inst != nil && inst.NodeID != nil && *inst.NodeID != "" {
			return *inst.NodeID
		}
	}

	return ""
}

// RequeueRunsAssignedToOfflineNodes 将分配到离线节点的 Run 重新入队
func (m *Manager) RequeueRunsAssignedToOfflineNodes(ctx context.Context, onlineIDs map[string]struct{}, threshold time.Duration) {
	runs, err := m.store.ListRunningRuns(ctx, 200)
	if err != nil {
		log.Printf("[node.manager] ListRunningRuns error: %v", err)
		return
	}

	now := time.Now()
	for _, run := range runs {
		if run == nil || run.NodeID == nil || *run.NodeID == "" {
			continue
		}

		if _, ok := onlineIDs[*run.NodeID]; ok {
			continue
		}

		if run.StartedAt == nil || now.Sub(*run.StartedAt) < threshold {
			continue
		}

		cnt, err := m.store.CountEventsByRun(ctx, run.ID)
		if err != nil {
			log.Printf("[node.manager] CountEventsByRun error (run=%s): %v", run.ID, err)
			continue
		}
		if cnt > 0 {
			// 有事件说明 NodeManager 已经开始执行，不自动回退
			continue
		}

		if err := m.store.ResetRunToQueued(ctx, run.ID); err != nil {
			log.Printf("[node.manager] ResetRunToQueued error (run=%s): %v", run.ID, err)
			continue
		}
		log.Printf("[node.manager] requeued run %s (offline node %s)", run.ID, *run.NodeID)
	}
}

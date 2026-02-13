// Package node 节点管理工具函数
package node

import (
	"encoding/json"
	"time"

	"agents-admin/internal/shared/model"
)

// FilterNodesByFreshHeartbeat 按心跳新鲜度过滤节点
func FilterNodesByFreshHeartbeat(nodes []*model.Node, window time.Duration) []*model.Node {
	if len(nodes) == 0 {
		return nil
	}

	now := time.Now()
	var out []*model.Node
	for _, n := range nodes {
		if n.LastHeartbeat == nil {
			continue
		}
		if now.Sub(*n.LastHeartbeat) > window {
			continue
		}
		out = append(out, n)
	}
	return out
}

// HeartbeatFreshWindow last_heartbeat 判断在线的时间窗口
const HeartbeatFreshWindow = 45 * time.Second

// NodeRealtimeStatus 节点实时状态（基于 MongoDB last_heartbeat 时间戳判定）
type NodeRealtimeStatus struct {
	Online        bool                   // 是否在线（可接受任务）
	Status        string                 // 展示状态：online/offline/draining/maintenance/...
	Capacity      map[string]interface{} // 容量信息
	LastHeartbeat *time.Time             // 最后心跳时间
}

// ResolveNodeStatus 计算单个节点的实时状态
//
// 判断优先级：
//  1. DB 中为行政状态（draining/maintenance/terminated/starting/unknown/unhealthy）→ 直接使用
//  2. 按 MongoDB last_heartbeat 时间窗口判断 online/offline
func ResolveNodeStatus(node *model.Node) NodeRealtimeStatus {
	var capacity map[string]interface{}
	if node.Capacity != nil {
		json.Unmarshal(node.Capacity, &capacity)
	}

	// 行政状态优先：管理员手动设置的状态不受心跳影响
	if isAdminStatus(node.Status) {
		return NodeRealtimeStatus{
			Online:        false,
			Status:        string(node.Status),
			Capacity:      capacity,
			LastHeartbeat: node.LastHeartbeat,
		}
	}

	// 按 last_heartbeat 时间窗口判断
	online := IsHeartbeatFresh(node.LastHeartbeat, HeartbeatFreshWindow)
	status := "offline"
	if online {
		status = "online"
	}
	return NodeRealtimeStatus{
		Online:        online,
		Status:        status,
		Capacity:      capacity,
		LastHeartbeat: node.LastHeartbeat,
	}
}

// isAdminStatus 判断是否为管理员手动设置的行政状态
//
// 这些状态由管理员通过 PATCH /nodes/{id} 设置，不会被心跳覆盖
func isAdminStatus(status model.NodeStatus) bool {
	n := &model.Node{Status: status}
	return n.IsAdminStatus()
}

// IsHeartbeatFresh 判断 last_heartbeat 是否在时间窗口内
func IsHeartbeatFresh(lastHeartbeat *time.Time, window time.Duration) bool {
	if lastHeartbeat == nil {
		return false
	}
	return time.Since(*lastHeartbeat) <= window
}

// GetNodeMaxConcurrent 获取节点的最大并发数
func GetNodeMaxConcurrent(node *model.Node) int {
	if len(node.Capacity) == 0 {
		return 1 // 默认最大并发数为 1
	}

	var capacity map[string]interface{}
	if err := json.Unmarshal(node.Capacity, &capacity); err != nil {
		return 1
	}

	if maxConcurrent, ok := capacity["max_concurrent"]; ok {
		switch v := maxConcurrent.(type) {
		case float64:
			return int(v)
		case int:
			return v
		}
	}

	return 1
}

// ExtractAgentIDs 从 snapshot 中提取 agent ID
func ExtractAgentIDs(snapshot json.RawMessage) (instanceID string, accountID string) {
	if len(snapshot) == 0 {
		return "", ""
	}

	var spec map[string]interface{}
	if err := json.Unmarshal(snapshot, &spec); err != nil {
		return "", ""
	}

	agentRaw, ok := spec["agent"]
	if !ok {
		return "", ""
	}

	agent, ok := agentRaw.(map[string]interface{})
	if !ok {
		return "", ""
	}

	if v, ok := agent["instance_id"].(string); ok {
		instanceID = v
	}
	if v, ok := agent["account_id"].(string); ok {
		accountID = v
	}
	return instanceID, accountID
}

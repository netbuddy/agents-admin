// Package node 节点管理工具函数
package node

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"agents-admin/internal/shared/cache"
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

// MergeOnlineNodesFromCache 合并 PostgreSQL 节点和 Redis 缓存心跳信息
//
// 从缓存获取在线节点列表，用缓存中的实时容量覆盖 PostgreSQL 中的历史值
func MergeOnlineNodesFromCache(ctx context.Context, nodes []*model.Node, nodeCache cache.NodeHeartbeatCache) []*model.Node {
	if len(nodes) == 0 || nodeCache == nil {
		return nil
	}

	onlineIDs, err := nodeCache.ListOnlineNodes(ctx)
	if err != nil {
		log.Printf("[node.utils] ListOnlineNodes from cache failed: %v", err)
		return nil
	}

	onlineSet := make(map[string]struct{}, len(onlineIDs))
	for _, id := range onlineIDs {
		onlineSet[id] = struct{}{}
	}
	if len(onlineSet) == 0 {
		return []*model.Node{} // 空切片表示"缓存正常但无在线节点"，nil 表示"缓存异常"
	}

	online := make([]*model.Node, 0, len(onlineSet))
	for _, node := range nodes {
		if _, ok := onlineSet[node.ID]; !ok {
			continue
		}

		n := *node // shallow copy
		n.Status = model.NodeStatusOnline

		// 用缓存中的实时容量覆盖 PostgreSQL 中的历史值
		if hb, err := nodeCache.GetNodeHeartbeat(ctx, node.ID); err == nil && hb != nil {
			updatedAt := hb.UpdatedAt
			n.LastHeartbeat = &updatedAt
			if capJSON, err := json.Marshal(NodeStatusToCapacity(hb)); err == nil {
				n.Capacity = capJSON
			}
		}

		online = append(online, &n)
	}
	return online
}

// NodeStatusToCapacity 将 cache.NodeStatus 的 Capacity 转换为 map[string]interface{}
func NodeStatusToCapacity(status *cache.NodeStatus) map[string]interface{} {
	if status == nil || len(status.Capacity) == 0 {
		return nil
	}
	result := make(map[string]interface{}, len(status.Capacity))
	for k, v := range status.Capacity {
		result[k] = v
	}
	return result
}

// HeartbeatFreshWindow PostgreSQL last_heartbeat 回退判断的时间窗口
const HeartbeatFreshWindow = 45 * time.Second

// NodeRealtimeStatus 节点实时状态（合并 cache + PostgreSQL）
type NodeRealtimeStatus struct {
	Online        bool                   // 是否在线（可接受任务）
	Status        string                 // 展示状态：online/offline/draining/maintenance/...
	Capacity      map[string]interface{} // 实时容量
	LastHeartbeat *time.Time             // 最后心跳时间
}

// ResolveNodeStatus 计算单个节点的实时状态
//
// 判断优先级：
//  1. DB 中为行政状态（draining/maintenance/terminated/starting/unknown）→ 直接使用，不查缓存
//  2. 缓存可用且有心跳 → online，使用缓存中的实时容量
//  3. 缓存可用但无心跳 → offline，使用 PostgreSQL 中的历史值
//  4. 缓存不可用 → 按 PostgreSQL 的 last_heartbeat 时间窗口判断
func ResolveNodeStatus(ctx context.Context, node *model.Node, nodeCache cache.NodeHeartbeatCache) NodeRealtimeStatus {
	var pgCapacity map[string]interface{}
	if node.Capacity != nil {
		json.Unmarshal(node.Capacity, &pgCapacity)
	}

	// 行政状态优先：管理员手动设置的状态不受缓存心跳影响
	if isAdminStatus(node.Status) {
		return NodeRealtimeStatus{
			Online:        false,
			Status:        string(node.Status),
			Capacity:      pgCapacity,
			LastHeartbeat: node.LastHeartbeat,
		}
	}

	// 缓存可用时：以缓存心跳为准
	if nodeCache != nil {
		hb, err := nodeCache.GetNodeHeartbeat(ctx, node.ID)
		if err != nil {
			log.Printf("[node.utils] GetNodeHeartbeat(%s) error: %v, fallback to postgres", node.ID, err)
			// 缓存异常 → 回退到 PostgreSQL
			return fallbackByLastHeartbeat(node, pgCapacity)
		}
		if hb != nil {
			// 缓存有心跳 → online
			updatedAt := hb.UpdatedAt
			return NodeRealtimeStatus{
				Online:        true,
				Status:        "online",
				Capacity:      NodeStatusToCapacity(hb),
				LastHeartbeat: &updatedAt,
			}
		}
		// 缓存无心跳 → offline
		return NodeRealtimeStatus{
			Online:        false,
			Status:        "offline",
			Capacity:      pgCapacity,
			LastHeartbeat: node.LastHeartbeat,
		}
	}

	// 缓存不可用 → 按 PostgreSQL last_heartbeat 时间窗口回退
	return fallbackByLastHeartbeat(node, pgCapacity)
}

// isAdminStatus 判断是否为管理员手动设置的行政状态
//
// 这些状态由管理员通过 PATCH /nodes/{id} 设置，优先于缓存心跳判断
func isAdminStatus(status model.NodeStatus) bool {
	n := &model.Node{Status: status}
	return n.IsAdminStatus()
}

// fallbackByLastHeartbeat 根据 PostgreSQL 的 last_heartbeat 判断在线
func fallbackByLastHeartbeat(node *model.Node, capacity map[string]interface{}) NodeRealtimeStatus {
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

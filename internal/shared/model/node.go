// Package model 定义核心数据模型
//
// node.go 包含计算节点相关的数据模型定义：
//   - Node：执行任务的计算节点
//   - NodeStatus：节点状态枚举
package model

import (
	"encoding/json"
	"time"
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
	ID            string          `json:"id" bson:"_id" db:"id"`                                                        // 节点 ID
	DisplayName   string          `json:"display_name,omitempty" bson:"display_name,omitempty" db:"display_name"`       // 用户设置的显示名称
	Status        NodeStatus      `json:"status" bson:"status" db:"status"`                                             // 节点状态
	Hostname      string          `json:"hostname,omitempty" bson:"hostname,omitempty" db:"hostname"`                   // 主机名
	IPs           string          `json:"ips,omitempty" bson:"ips,omitempty" db:"ips"`                                  // IP 地址列表（逗号分隔）
	Labels        json.RawMessage `json:"labels" bson:"labels" db:"labels"`                                             // 节点标签
	Capacity      json.RawMessage `json:"capacity" bson:"capacity" db:"capacity"`                                       // 节点容量
	LastHeartbeat *time.Time      `json:"last_heartbeat,omitempty" bson:"last_heartbeat,omitempty" db:"last_heartbeat"` // 最后心跳
	CreatedAt     time.Time       `json:"created_at" bson:"created_at" db:"created_at"`                                 // 创建时间
	UpdatedAt     time.Time       `json:"updated_at" bson:"updated_at" db:"updated_at"`                                 // 更新时间
}

// ============================================================================
// 辅助方法
// ============================================================================

// IsOnline 判断节点是否在线（可接受任务）
func (n *Node) IsOnline() bool {
	return n.Status == NodeStatusOnline
}

// IsAvailable 判断节点是否可用（可调度任务）
func (n *Node) IsAvailable() bool {
	return n.Status == NodeStatusOnline || n.Status == NodeStatusUnhealthy
}

// CanAcceptTasks 判断节点是否可以接受新任务
func (n *Node) CanAcceptTasks() bool {
	return n.Status == NodeStatusOnline
}

// IsAdminStatus 判断是否为管理员手动设置的行政状态
//
// 行政状态优先于缓存心跳判断，不会被心跳覆盖
func (n *Node) IsAdminStatus() bool {
	switch n.Status {
	case NodeStatusDraining, NodeStatusMaintenance, NodeStatusTerminated,
		NodeStatusStarting, NodeStatusUnknown, NodeStatusUnhealthy:
		return true
	default:
		return false
	}
}

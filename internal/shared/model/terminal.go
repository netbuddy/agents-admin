// Package model 定义核心数据模型
//
// terminal.go 包含终端会话相关的数据模型定义：
//   - TerminalSession：终端会话
//   - TerminalSessionStatus：终端会话状态枚举
package model

import "time"

// ============================================================================
// TerminalSessionStatus - 终端会话状态
// ============================================================================

// TerminalSessionStatus 终端会话状态
type TerminalSessionStatus string

const (
	// TerminalStatusPending 待创建（等待 Executor 处理）
	TerminalStatusPending TerminalSessionStatus = "pending"

	// TerminalStatusStarting 启动中
	TerminalStatusStarting TerminalSessionStatus = "starting"

	// TerminalStatusRunning 运行中
	TerminalStatusRunning TerminalSessionStatus = "running"

	// TerminalStatusClosed 已关闭
	TerminalStatusClosed TerminalSessionStatus = "closed"

	// TerminalStatusError 错误
	TerminalStatusError TerminalSessionStatus = "error"
)

// ============================================================================
// TerminalSession - 终端会话
// ============================================================================

// TerminalSession 终端会话
//
// 终端会话用于提供 Agent 容器的交互式终端访问。
// 状态由 Executor 管理并上报。
type TerminalSession struct {
	ID            string                `json:"id" bson:"_id" db:"id"`
	InstanceID    *string               `json:"instance_id" bson:"instance_id" db:"instance_id"`       // 目标实例 ID（可选）
	ContainerName string                `json:"container_name" bson:"container_name" db:"container_name"` // 目标容器名
	NodeID        *string               `json:"node_id" bson:"node_id" db:"node_id"`               // 节点 ID
	Port          *int                  `json:"port" bson:"port" db:"port"`                     // ttyd 端口（Executor 回填）
	URL           *string               `json:"url" bson:"url" db:"url"`                       // 终端访问 URL（Executor 回填）
	Status        TerminalSessionStatus `json:"status" bson:"status" db:"status"`                 // 会话状态
	CreatedAt     time.Time             `json:"created_at" bson:"created_at" db:"created_at"`
	ExpiresAt     *time.Time            `json:"expires_at" bson:"expires_at" db:"expires_at"` // 过期时间（可选）
}

// ============================================================================
// 辅助方法
// ============================================================================

// IsActive 判断终端会话是否活跃
func (ts *TerminalSession) IsActive() bool {
	return ts.Status == TerminalStatusRunning || ts.Status == TerminalStatusStarting
}

// IsTerminated 判断终端会话是否已终止
func (ts *TerminalSession) IsTerminated() bool {
	return ts.Status == TerminalStatusClosed || ts.Status == TerminalStatusError
}

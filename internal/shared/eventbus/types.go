// Package eventbus 事件总线类型定义
package eventbus

import (
	"time"
)

// ============================================================================
// 事件类型
// ============================================================================

// WorkflowEvent 工作流事件
type WorkflowEvent struct {
	ID        string                 `json:"id"`
	Seq       int                    `json:"seq"`
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// RunEvent Run 执行事件
type RunEvent struct {
	ID        string                 `json:"id"`
	RunID     string                 `json:"run_id"`
	Seq       int                    `json:"seq"`
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
	Raw       string                 `json:"raw,omitempty"`
}

// ============================================================================
// Key 前缀和常量
// ============================================================================

const (
	// Key 前缀
	KeyWorkflowEvents = "workflow_events:"
	KeyRunEvents      = "run_events:"

	// Stream 最大长度
	MaxStreamLength = 1000
)

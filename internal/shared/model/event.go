// Package model 定义核心数据模型
//
// event.go 包含事件相关的数据模型定义：
//   - Event：执行事件（数据库存储）
//   - CanonicalEvent：统一事件格式（从 pkg/driver 迁入）
//   - EventType：事件类型枚举
package model

import (
	"encoding/json"
	"time"
)

// ============================================================================
// EventType - 事件类型
// ============================================================================

// EventType 定义事件的类型
//
// 事件分类：
//  1. 生命周期事件：run_started, run_completed, run_failed
//  2. 输出事件：message, thinking, progress
//  3. 工具事件：tool_use_start, tool_result
//  4. 文件事件：file_read, file_write, file_delete
//  5. 命令事件：command, command_output
//  6. 控制事件：approval_request, checkpoint, heartbeat
//  7. 错误事件：error, warning
type EventType string

const (
	// === 生命周期事件 ===

	// EventTypeRunStarted 执行开始
	EventTypeRunStarted EventType = "run_started"

	// EventTypeRunCompleted 执行完成（成功）
	EventTypeRunCompleted EventType = "run_completed"

	// EventTypeRunFailed 执行失败
	EventTypeRunFailed EventType = "run_failed"

	// === 输出事件 ===

	// EventTypeMessage Agent 输出的文本消息
	EventTypeMessage EventType = "message"

	// EventTypeThinking Agent 思考过程（推理链）
	EventTypeThinking EventType = "thinking"

	// EventTypeProgress 进度更新
	// Payload: {"progress": 0.5, "message": "处理中..."}
	EventTypeProgress EventType = "progress"

	// === 工具事件 ===

	// EventTypeToolUseStart 开始使用工具
	// Payload: {"tool": "read_file", "input": {"path": "..."}}
	EventTypeToolUseStart EventType = "tool_use_start"

	// EventTypeToolResult 工具执行结果
	// Payload: {"tool": "read_file", "output": "...", "success": true}
	EventTypeToolResult EventType = "tool_result"

	// === 文件事件 ===

	// EventTypeFileRead 读取文件
	EventTypeFileRead EventType = "file_read"

	// EventTypeFileWrite 写入文件
	EventTypeFileWrite EventType = "file_write"

	// EventTypeFileDelete 删除文件
	EventTypeFileDelete EventType = "file_delete"

	// === 命令事件 ===

	// EventTypeCommand 执行命令
	// Payload: {"command": "git", "args": ["status"]}
	EventTypeCommand EventType = "command"

	// EventTypeCommandOutput 命令输出
	// Payload: {"stdout": "...", "stderr": "...", "exit_code": 0}
	EventTypeCommandOutput EventType = "command_output"

	// === 控制事件 ===

	// EventTypeApprovalRequest 请求人工审批
	// Payload: {"action": "delete_file", "target": "...", "reason": "..."}
	EventTypeApprovalRequest EventType = "approval_request"

	// EventTypeApprovalResponse 人工审批响应
	// Payload: {"approved": true, "comment": "..."}
	EventTypeApprovalResponse EventType = "approval_response"

	// EventTypeCheckpoint 检查点（可恢复）
	// Payload: {"state": {...}, "resumable": true}
	EventTypeCheckpoint EventType = "checkpoint"

	// EventTypeHeartbeat 心跳事件（表示任务仍在运行）
	EventTypeHeartbeat EventType = "heartbeat"

	// === 系统事件 ===

	// EventTypeSystemInfo 系统信息（Agent 初始化、配置等）
	// Payload: {"version": "...", "model": "...", "tools": [...]}
	EventTypeSystemInfo EventType = "system_info"

	// EventTypeResult 执行结果（Agent 返回的最终结果，不触发状态变更）
	// Payload: {"result": "...", "usage": {...}}
	EventTypeResult EventType = "result"

	// === 错误事件 ===

	// EventTypeError 错误事件
	// Payload: {"message": "...", "code": "...", "recoverable": false}
	EventTypeError EventType = "error"

	// EventTypeWarning 警告事件
	// Payload: {"message": "...", "code": "..."}
	EventTypeWarning EventType = "warning"
)

// ============================================================================
// Event - 执行事件（数据库存储）
// ============================================================================

// Event 表示 Run 执行过程中产生的事件
//
// 事件是 Agent 执行过程的实时记录，用于：
//   - 实时监控：通过 WebSocket 推送到前端
//   - 审计追溯：记录 Agent 的每一步操作
//   - 调试分析：定位问题和优化性能
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
// CanonicalEvent - 统一事件格式（从 pkg/driver 迁入）
// ============================================================================

// CanonicalEvent 是统一的事件格式
//
// 不同 Agent CLI 输出格式各异：
//   - Claude: {"type": "assistant", "message": {...}}
//   - Gemini: {"action": "text", "content": "..."}
//   - Codex: 完全不同的格式
//
// CanonicalEvent 统一所有格式，使得：
//   - 存储层无需关心 Agent 类型
//   - 前端展示逻辑统一
//   - 事件回放和分析标准化
//
// 数据流：
//
//	CLI stdout → Driver.ParseEvent() → CanonicalEvent
//	                                        │
//	              ┌─────────────────────────┼─────────────────────────┐
//	              ▼                         ▼                         ▼
//	          存储到 DB              WebSocket 推送              前端展示
type CanonicalEvent struct {
	// Seq 事件序号，Run 内递增，用于排序和去重
	Seq int64 `json:"seq"`

	// Type 事件类型
	Type EventType `json:"type"`

	// Timestamp 事件发生时间
	Timestamp time.Time `json:"timestamp"`

	// RunID 所属 Run ID
	RunID string `json:"run_id"`

	// Payload 事件数据，不同类型有不同的字段
	// message: {"content": "..."}
	// tool_use: {"tool": "...", "input": {...}}
	// command: {"command": "...", "args": [...]}
	// error: {"message": "...", "code": "..."}
	Payload map[string]interface{} `json:"payload,omitempty"`

	// Raw 原始输出（可选，用于调试）
	Raw string `json:"raw,omitempty"`
}

// ToEvent 将 CanonicalEvent 转换为 Event（用于数据库存储）
func (ce *CanonicalEvent) ToEvent() (*Event, error) {
	payload, err := json.Marshal(ce.Payload)
	if err != nil {
		return nil, err
	}

	var raw *string
	if ce.Raw != "" {
		raw = &ce.Raw
	}

	return &Event{
		RunID:     ce.RunID,
		Seq:       int(ce.Seq),
		Type:      string(ce.Type),
		Timestamp: ce.Timestamp,
		Payload:   payload,
		Raw:       raw,
	}, nil
}

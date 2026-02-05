package adapter

import "time"

// ============================================================================
// CanonicalEvent - 统一事件格式
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

	// EventRunStarted 执行开始
	EventRunStarted EventType = "run_started"

	// EventRunCompleted 执行完成（成功）
	EventRunCompleted EventType = "run_completed"

	// EventRunFailed 执行失败
	EventRunFailed EventType = "run_failed"

	// === 输出事件 ===

	// EventMessage Agent 输出的文本消息
	EventMessage EventType = "message"

	// EventThinking Agent 思考过程（推理链）
	EventThinking EventType = "thinking"

	// EventProgress 进度更新
	// Payload: {"progress": 0.5, "message": "处理中..."}
	EventProgress EventType = "progress"

	// === 工具事件 ===

	// EventToolUseStart 开始使用工具
	// Payload: {"tool": "read_file", "input": {"path": "..."}}
	EventToolUseStart EventType = "tool_use_start"

	// EventToolResult 工具执行结果
	// Payload: {"tool": "read_file", "output": "...", "success": true}
	EventToolResult EventType = "tool_result"

	// === 文件事件 ===

	// EventFileRead 读取文件
	EventFileRead EventType = "file_read"

	// EventFileWrite 写入文件
	EventFileWrite EventType = "file_write"

	// EventFileDelete 删除文件
	EventFileDelete EventType = "file_delete"

	// === 命令事件 ===

	// EventCommand 执行命令
	// Payload: {"command": "git", "args": ["status"]}
	EventCommand EventType = "command"

	// EventCommandOutput 命令输出
	// Payload: {"stdout": "...", "stderr": "...", "exit_code": 0}
	EventCommandOutput EventType = "command_output"

	// === 控制事件 ===

	// EventApprovalRequest 请求人工审批
	// Payload: {"action": "delete_file", "target": "...", "reason": "..."}
	EventApprovalRequest EventType = "approval_request"

	// EventApprovalResponse 人工审批响应
	// Payload: {"approved": true, "comment": "..."}
	EventApprovalResponse EventType = "approval_response"

	// EventCheckpoint 检查点（可恢复）
	// Payload: {"state": {...}, "resumable": true}
	EventCheckpoint EventType = "checkpoint"

	// EventHeartbeat 心跳事件（表示任务仍在运行）
	EventHeartbeat EventType = "heartbeat"

	// === 系统事件 ===

	// EventSystemInfo 系统信息（Agent 初始化、配置等）
	// Payload: {"version": "...", "model": "...", "tools": [...]}
	EventSystemInfo EventType = "system_info"

	// EventResult 执行结果（Agent 返回的最终结果，不触发状态变更）
	// Payload: {"result": "...", "usage": {...}}
	EventResult EventType = "result"

	// === 错误事件 ===

	// EventError 错误事件
	// Payload: {"message": "...", "code": "...", "recoverable": false}
	EventError EventType = "error"

	// EventWarning 警告事件
	// Payload: {"message": "...", "code": "..."}
	EventWarning EventType = "warning"
)

// ============================================================================
// Artifacts - 执行产物
// ============================================================================

// Artifacts 定义 Run 执行产生的产物
//
// 产物是 Agent 执行完成后需要持久化保存的输出：
//   - EventsFile：完整事件流（JSONL 格式）
//   - DiffFile：代码变更（git diff）
//   - OutputFiles：生成的文件列表
//   - Summary：执行摘要
//
// 存储方式：
//   - 产物上传到对象存储（如 MinIO）
//   - Artifacts 结构记录元数据和存储路径
//   - model.Artifact 持久化到数据库
type Artifacts struct {
	// EventsFile 事件日志文件路径
	// 格式：JSONL（每行一个 CanonicalEvent）
	// 用于事件回放和审计
	EventsFile string `json:"events_file"`

	// DiffFile 代码变更文件路径（可选）
	// 格式：unified diff
	// 仅代码开发任务产生
	DiffFile string `json:"diff_file,omitempty"`

	// OutputFiles 其他输出文件列表
	// 包括生成的报告、图片、数据等
	OutputFiles []OutputFile `json:"output_files,omitempty"`

	// Summary 执行摘要
	Summary *ExecutionSummary `json:"summary,omitempty"`
}

// OutputFile 输出文件信息
type OutputFile struct {
	// Name 文件名
	Name string `json:"name"`

	// Path 存储路径
	Path string `json:"path"`

	// Size 文件大小（字节）
	Size int64 `json:"size"`

	// ContentType MIME 类型
	ContentType string `json:"content_type,omitempty"`

	// Description 文件描述
	Description string `json:"description,omitempty"`
}

// ExecutionSummary 执行摘要
type ExecutionSummary struct {
	// TotalEvents 总事件数
	TotalEvents int `json:"total_events"`

	// Duration 执行时长
	Duration time.Duration `json:"duration"`

	// TokensUsed 使用的 Token 数
	TokensUsed int `json:"tokens_used,omitempty"`

	// FilesModified 修改的文件数
	FilesModified int `json:"files_modified,omitempty"`

	// CommandsExecuted 执行的命令数
	CommandsExecuted int `json:"commands_executed,omitempty"`

	// Result 执行结果摘要（Agent 生成）
	Result string `json:"result,omitempty"`
}

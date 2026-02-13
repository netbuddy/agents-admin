// Package model 定义核心数据模型
//
// artifact.go 包含执行产物相关的数据模型定义：
//   - Artifact：执行产物（数据库存储）
//   - Artifacts：产物集合（运行时使用）
//   - ExecutionSummary：执行摘要
//   - OutputFile：输出文件信息
package model

import "time"

// ============================================================================
// Artifact - 执行产物（数据库存储）
// ============================================================================

// Artifact 表示 Run 产生的文件产物
//
// 产物是 Agent 执行过程中生成的文件：
//   - 代码变更（diff 文件）
//   - 事件日志（JSONL 文件）
//   - 生成的文件（如图片、文档等）
//
// 产物存储在对象存储（如 MinIO）中，Artifact 记录元数据。
//
// 字段说明：
//   - ID：自增主键
//   - RunID：所属 Run ID
//   - Name：产物名称（如 "events.jsonl"）
//   - Path：存储路径（对象存储 Key）
//   - Size：文件大小（字节）
//   - ContentType：MIME 类型
type Artifact struct {
	ID          int64     `json:"id" bson:"_id" db:"id"`                               // 产物 ID
	RunID       string    `json:"run_id" bson:"run_id" db:"run_id"`                       // 所属 Run ID
	Name        string    `json:"name" bson:"name" db:"name"`                           // 产物名称
	Path        string    `json:"path" bson:"path" db:"path"`                           // 存储路径
	Size        *int64    `json:"size,omitempty" bson:"size,omitempty" db:"size"`                 // 文件大小
	ContentType *string   `json:"content_type,omitempty" bson:"content_type,omitempty" db:"content_type"` // MIME 类型
	CreatedAt   time.Time `json:"created_at" bson:"created_at" db:"created_at"`               // 创建时间
}

// ============================================================================
// Artifacts - 产物集合（从 pkg/driver 迁入）
// ============================================================================

// Artifacts 定义 Run 执行产生的产物集合
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
//   - Artifact 持久化到数据库
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

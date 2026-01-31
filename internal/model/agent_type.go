package model

import "time"

// AgentType 定义支持的 AI Agent 类型
type AgentType struct {
	ID           string   `json:"id"`            // 类型标识，如 qwen-code, openai-codex
	Name         string   `json:"name"`          // 显示名称
	Image        string   `json:"image"`         // Docker 镜像
	AuthDir      string   `json:"auth_dir"`      // 容器内认证目录
	AuthFile     string   `json:"auth_file"`     // 认证文件名
	LoginCmd     string   `json:"login_cmd"`     // 登录命令
	LoginMethods []string `json:"login_methods"` // 支持的登录方式
	Description  string   `json:"description"`   // 类型描述
}

// AccountStatus 账号状态
type AccountStatus string

const (
	AccountStatusPending        AccountStatus = "pending"        // 待认证
	AccountStatusAuthenticating AccountStatus = "authenticating" // 认证中
	AccountStatusAuthenticated  AccountStatus = "authenticated"  // 已认证
	AccountStatusExpired        AccountStatus = "expired"        // 已过期
)

// Account 表示 Agent 的认证账号
// 当前阶段：账号绑定到特定节点（Volume 存储在节点本地）
// 未来演进：账号存储到共享存储（NFS/S3），无需绑定节点
type Account struct {
	ID          string        `json:"id" db:"id"`
	Name        string        `json:"name" db:"name"`                           // 显示名称（如邮箱）
	AgentTypeID string        `json:"agent_type" db:"agent_type_id"`            // 关联的 Agent 类型
	NodeID      string        `json:"node_id" db:"node_id"`                     // 账号所属节点（当前阶段必填，未来可选）
	VolumeName  *string       `json:"volume_name,omitempty" db:"volume_name"`   // Docker Volume 名称（由 Node Agent 创建后回填）
	Status      AccountStatus `json:"status" db:"status"`                       // 账号状态
	CreatedAt   time.Time     `json:"created_at" db:"created_at"`               // 创建时间
	UpdatedAt   time.Time     `json:"updated_at" db:"updated_at"`               // 更新时间
	LastUsedAt  *time.Time    `json:"last_used_at,omitempty" db:"last_used_at"` // 最后使用时间
}

// AuthTaskStatus 认证任务状态
type AuthTaskStatus string

const (
	AuthTaskStatusPending     AuthTaskStatus = "pending"      // 待调度
	AuthTaskStatusAssigned    AuthTaskStatus = "assigned"     // 已分配节点
	AuthTaskStatusRunning     AuthTaskStatus = "running"      // 执行中
	AuthTaskStatusWaitingUser AuthTaskStatus = "waiting_user" // 等待用户操作
	AuthTaskStatusSuccess     AuthTaskStatus = "success"      // 认证成功
	AuthTaskStatusFailed      AuthTaskStatus = "failed"       // 认证失败
	AuthTaskStatusTimeout     AuthTaskStatus = "timeout"      // 超时
)

// AuthTask 认证任务（控制面/数据面分离设计）
// API Server 只创建任务记录，Node Agent 执行实际操作并上报状态
type AuthTask struct {
	ID        string `json:"id" db:"id"`
	AccountID string `json:"account_id" db:"account_id"`

	// 期望状态（由 API Server 设置）
	Method  string  `json:"method" db:"method"`     // oauth, api_key
	ProxyID *string `json:"proxy_id" db:"proxy_id"` // 代理ID（可选）

	// 节点信息（由用户指定，不走 Scheduler）
	NodeID string `json:"node_id" db:"node_id"`

	// 当前状态（由 Node Agent 上报）
	Status        AuthTaskStatus `json:"status" db:"status"`
	TerminalPort  *int           `json:"terminal_port,omitempty" db:"terminal_port"`
	TerminalURL   *string        `json:"terminal_url,omitempty" db:"terminal_url"`
	ContainerName *string        `json:"container_name,omitempty" db:"container_name"`
	OAuthURL      *string        `json:"oauth_url,omitempty" db:"oauth_url"` // OAuth 验证 URL
	UserCode      *string        `json:"user_code,omitempty" db:"user_code"` // 用户验证码
	Message       *string        `json:"message,omitempty" db:"message"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
}

// InstanceStatus 实例状态
type InstanceStatus string

const (
	InstanceStatusPending  InstanceStatus = "pending"  // 待创建（等待 Executor 处理）
	InstanceStatusCreating InstanceStatus = "creating" // 创建中
	InstanceStatusRunning  InstanceStatus = "running"  // 运行中
	InstanceStatusStopping InstanceStatus = "stopping" // 停止中（等待 Executor 处理）
	InstanceStatusStopped  InstanceStatus = "stopped"  // 已停止
	InstanceStatusError    InstanceStatus = "error"    // 错误
)

// Instance 表示运行中的 Agent 实例
// P2-2 重构：状态持久化到 PostgreSQL
type Instance struct {
	ID            string         `json:"id" db:"id"`
	Name          string         `json:"name" db:"name"`                         // 显示名称
	AccountID     string         `json:"account_id" db:"account_id"`             // 使用的账号 ID
	AgentTypeID   string         `json:"agent_type_id" db:"agent_type_id"`       // Agent 类型 ID
	ContainerName *string        `json:"container_name" db:"container_name"`     // Docker 容器名（Executor 回填）
	NodeID        *string        `json:"node_id" db:"node_id"`                   // 所在节点 ID
	Status        InstanceStatus `json:"status" db:"status"`                     // 实例状态
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`             // 创建时间
	UpdatedAt     time.Time      `json:"updated_at" db:"updated_at"`             // 更新时间
}

// TerminalSessionStatus 终端会话状态
type TerminalSessionStatus string

const (
	TerminalStatusPending  TerminalSessionStatus = "pending"  // 待创建（等待 Executor 处理）
	TerminalStatusStarting TerminalSessionStatus = "starting" // 启动中
	TerminalStatusRunning  TerminalSessionStatus = "running"  // 运行中
	TerminalStatusClosed   TerminalSessionStatus = "closed"   // 已关闭
	TerminalStatusError    TerminalSessionStatus = "error"    // 错误
)

// TerminalSession 终端会话
// P2-2 重构：状态持久化到 PostgreSQL
type TerminalSession struct {
	ID            string                `json:"id" db:"id"`
	InstanceID    *string               `json:"instance_id" db:"instance_id"`       // 目标实例 ID（可选）
	ContainerName string                `json:"container_name" db:"container_name"` // 目标容器名
	NodeID        *string               `json:"node_id" db:"node_id"`               // 节点 ID
	Port          *int                  `json:"port" db:"port"`                     // ttyd 端口（Executor 回填）
	URL           *string               `json:"url" db:"url"`                       // 终端访问 URL（Executor 回填）
	Status        TerminalSessionStatus `json:"status" db:"status"`                 // 会话状态
	CreatedAt     time.Time             `json:"created_at" db:"created_at"`
	ExpiresAt     *time.Time            `json:"expires_at" db:"expires_at"` // 过期时间（可选）
}

// AuthSession 认证会话（兼容旧 API，实际映射到 AuthTask）
// Deprecated: 使用 AuthTask 替代
type AuthSession struct {
	ID           string    `json:"id"`
	AccountID    string    `json:"account_id"`
	DeviceCode   string    `json:"device_code,omitempty"`   // 设备码（Device Code 认证）
	VerifyURL    string    `json:"verify_url,omitempty"`    // 验证 URL
	CallbackPort int       `json:"callback_port,omitempty"` // OAuth 回调端口
	Status       string    `json:"status"`                  // pending, waiting, success, failed
	Message      string    `json:"message,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
}

// ToAuthSession 将 AuthTask 转换为 AuthSession（兼容旧 API）
func (t *AuthTask) ToAuthSession() *AuthSession {
	session := &AuthSession{
		ID:        t.ID,
		AccountID: t.AccountID,
		ExpiresAt: t.ExpiresAt,
	}
	// 状态映射
	switch t.Status {
	case AuthTaskStatusPending, AuthTaskStatusAssigned:
		session.Status = "pending"
	case AuthTaskStatusRunning, AuthTaskStatusWaitingUser:
		session.Status = "waiting"
	case AuthTaskStatusSuccess:
		session.Status = "success"
	case AuthTaskStatusFailed, AuthTaskStatusTimeout:
		session.Status = "failed"
	default:
		session.Status = string(t.Status)
	}
	if t.TerminalPort != nil {
		session.CallbackPort = *t.TerminalPort
	}
	if t.OAuthURL != nil {
		session.VerifyURL = *t.OAuthURL
	}
	if t.UserCode != nil {
		session.DeviceCode = *t.UserCode
	}
	if t.Message != nil {
		session.Message = *t.Message
	}
	return session
}

// 预定义的 Agent 类型
var PredefinedAgentTypes = []AgentType{
	{
		ID:           "qwen-code",
		Name:         "Qwen-Code",
		Image:        "runners/qwencode:latest",
		AuthDir:      "/home/node/.qwen",
		AuthFile:     "auth.json",
		LoginCmd:     "qwen",
		LoginMethods: []string{"oauth", "api_key"},
		Description:  "基于 Qwen 大模型的 AI 编程助手",
	},
	{
		ID:           "openai-codex",
		Name:         "OpenAI Codex",
		Image:        "runners/codex:latest",
		AuthDir:      "/home/codex/.codex",
		AuthFile:     "auth.json",
		LoginCmd:     "codex login",
		LoginMethods: []string{"device_code", "oauth", "api_key"},
		Description:  "OpenAI 官方 AI 编程智能体",
	},
}

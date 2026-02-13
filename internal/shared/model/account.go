// Package model 定义核心数据模型
//
// account.go 包含账号认证相关的数据模型定义：
//   - Account：Agent 的认证账号
//   - AccountStatus：账号状态枚举
//   - AuthTask：认证任务
//   - AuthTaskStatus：认证任务状态枚举
//   - AuthSession：认证会话（兼容旧 API）
package model

import "time"

// ============================================================================
// AccountStatus - 账号状态
// ============================================================================

// AccountStatus 账号状态
type AccountStatus string

const (
	// AccountStatusPending 待认证
	AccountStatusPending AccountStatus = "pending"

	// AccountStatusAuthenticating 认证中
	AccountStatusAuthenticating AccountStatus = "authenticating"

	// AccountStatusAuthenticated 已认证
	AccountStatusAuthenticated AccountStatus = "authenticated"

	// AccountStatusExpired 已过期
	AccountStatusExpired AccountStatus = "expired"
)

// ============================================================================
// Account - 认证账号
// ============================================================================

// Account 表示 Agent 的认证账号
//
// 账号与节点无关，Volume 归档存储在 MinIO 共享存储中，任意节点均可恢复
type Account struct {
	ID               string        `json:"id" bson:"_id" db:"id"`
	Name             string        `json:"name" bson:"name" db:"name"`                                                               // 显示名称（如邮箱）
	AgentTypeID      string        `json:"agent_type" bson:"agent_type" db:"agent_type_id"`                                          // 关联的 Agent 类型
	VolumeName       *string       `json:"volume_name,omitempty" bson:"volume_name,omitempty" db:"volume_name"`                      // Docker Volume 名称（由 Node Agent 创建后回填）
	VolumeArchiveKey *string       `json:"volume_archive_key,omitempty" bson:"volume_archive_key,omitempty" db:"volume_archive_key"` // MinIO 中的 Volume 归档 key
	Status           AccountStatus `json:"status" bson:"status" db:"status"`                                                         // 账号状态
	CreatedAt        time.Time     `json:"created_at" bson:"created_at" db:"created_at"`                                             // 创建时间
	UpdatedAt        time.Time     `json:"updated_at" bson:"updated_at" db:"updated_at"`                                             // 更新时间
	LastUsedAt       *time.Time    `json:"last_used_at,omitempty" bson:"last_used_at,omitempty" db:"last_used_at"`                   // 最后使用时间
}

// ============================================================================
// AuthTaskStatus - 认证任务状态
// ============================================================================

// AuthTaskStatus 认证任务状态
type AuthTaskStatus string

const (
	// AuthTaskStatusPending 待调度
	AuthTaskStatusPending AuthTaskStatus = "pending"

	// AuthTaskStatusAssigned 已分配节点
	AuthTaskStatusAssigned AuthTaskStatus = "assigned"

	// AuthTaskStatusRunning 执行中
	AuthTaskStatusRunning AuthTaskStatus = "running"

	// AuthTaskStatusWaitingUser 等待用户操作
	AuthTaskStatusWaitingUser AuthTaskStatus = "waiting_user"

	// AuthTaskStatusSuccess 认证成功
	AuthTaskStatusSuccess AuthTaskStatus = "success"

	// AuthTaskStatusFailed 认证失败
	AuthTaskStatusFailed AuthTaskStatus = "failed"

	// AuthTaskStatusTimeout 超时
	AuthTaskStatusTimeout AuthTaskStatus = "timeout"
)

// ============================================================================
// AuthTask - 认证任务
// ============================================================================

// AuthTask 认证任务（控制面/数据面分离设计）
//
// API Server 只创建任务记录，Node Agent 执行实际操作并上报状态
type AuthTask struct {
	ID        string `json:"id" bson:"_id" db:"id"`
	AccountID string `json:"account_id" bson:"account_id" db:"account_id"`

	// 期望状态（由 API Server 设置）
	Method  string  `json:"method" bson:"method" db:"method"`       // oauth, api_key
	ProxyID *string `json:"proxy_id" bson:"proxy_id" db:"proxy_id"` // 代理ID（可选）

	// 节点信息（由用户指定，不走 Scheduler）
	NodeID string `json:"node_id" bson:"node_id" db:"node_id"`

	// 当前状态（由 Node Agent 上报）
	Status        AuthTaskStatus `json:"status" bson:"status" db:"status"`
	TerminalPort  *int           `json:"terminal_port,omitempty" bson:"terminal_port,omitempty" db:"terminal_port"`
	TerminalURL   *string        `json:"terminal_url,omitempty" bson:"terminal_url,omitempty" db:"terminal_url"`
	ContainerName *string        `json:"container_name,omitempty" bson:"container_name,omitempty" db:"container_name"`
	OAuthURL      *string        `json:"oauth_url,omitempty" bson:"oauth_url,omitempty" db:"oauth_url"` // OAuth 验证 URL
	UserCode      *string        `json:"user_code,omitempty" bson:"user_code,omitempty" db:"user_code"` // 用户验证码
	Message       *string        `json:"message,omitempty" bson:"message,omitempty" db:"message"`

	CreatedAt time.Time `json:"created_at" bson:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at" db:"updated_at"`
	ExpiresAt time.Time `json:"expires_at" bson:"expires_at" db:"expires_at"`
}

// ============================================================================
// AuthSession - 认证会话（兼容旧 API）
// ============================================================================

// AuthSession 认证会话（兼容旧 API，实际映射到 AuthTask）
//
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

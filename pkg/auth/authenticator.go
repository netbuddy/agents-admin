// Package auth 定义Agent认证器接口
//
// 提供统一的认证抽象，支持不同Agent的OAuth/DeviceCode/APIKey等认证方式
package auth

import (
	"context"

	"agents-admin/pkg/docker"
)

// AuthState 认证状态
type AuthState string

const (
	AuthStatePending      AuthState = "pending"       // 等待开始
	AuthStateRunning      AuthState = "running"       // 正在运行
	AuthStateWaitingInput AuthState = "waiting_input" // 等待用户输入
	AuthStateWaitingOAuth AuthState = "waiting_oauth" // 等待OAuth回调
	AuthStateSuccess      AuthState = "success"       // 认证成功
	AuthStateFailed       AuthState = "failed"        // 认证失败
	AuthStateTimeout      AuthState = "timeout"       // 认证超时
)

// AuthStatus 认证状态详情
type AuthStatus struct {
	State      AuthState // 当前状态
	OAuthURL   string    // OAuth验证URL（如有）
	DeviceCode string    // 设备码（如有）
	UserCode   string    // 用户码（如有）
	Message    string    // 状态消息
	Error      error     // 错误信息
}

// AuthTask 认证任务配置
type AuthTask struct {
	ID         string            // 任务ID
	AccountID  string            // 账号ID
	AgentType  string            // Agent类型
	Method     string            // 认证方法 (oauth, device_code, api_key)
	Image      string            // Docker镜像
	AuthDir    string            // 认证目录
	AuthFile   string            // 认证文件
	LoginCmd   string            // 登录命令
	VolumeName string            // 数据卷名称
	Env        map[string]string // 环境变量
	ProxyEnvs  []string          // 代理环境变量（已格式化，如 HTTP_PROXY=http://...）
}

// Authenticator 认证器接口
//
// 每种Agent实现自己的认证器，处理特定的认证流程
type Authenticator interface {
	// AgentType 返回支持的Agent类型
	AgentType() string

	// SupportedMethods 返回支持的认证方法
	SupportedMethods() []string

	// Start 启动认证流程
	// 返回的channel用于接收状态更新
	Start(ctx context.Context, task *AuthTask, dockerClient *docker.Client) (<-chan *AuthStatus, error)

	// SendInput 发送用户输入到容器
	SendInput(input string) error

	// GetStatus 获取当前认证状态
	GetStatus() *AuthStatus

	// Stop 停止认证并清理资源
	Stop() error
}

// Registry 认证器注册表
type Registry struct {
	authenticators map[string]Authenticator
}

// NewRegistry 创建注册表
func NewRegistry() *Registry {
	return &Registry{
		authenticators: make(map[string]Authenticator),
	}
}

// Register 注册认证器
func (r *Registry) Register(auth Authenticator) {
	r.authenticators[auth.AgentType()] = auth
}

// Get 获取认证器
func (r *Registry) Get(agentType string) (Authenticator, bool) {
	auth, ok := r.authenticators[agentType]
	return auth, ok
}

// List 列出所有认证器
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.authenticators))
	for name := range r.authenticators {
		names = append(names, name)
	}
	return names
}

// CreateAuthenticator 创建认证器工厂函数类型
type CreateAuthenticator func() Authenticator

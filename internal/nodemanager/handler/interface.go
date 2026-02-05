// Package handler 定义 NodeManager 的 Handler 接口
//
// Handler 是 NodeManager 的插件单元，每个 Handler 负责一类操作。
// 设计目标：
//   - 核心精简：NodeManager 只负责调度 Handler
//   - 独立开发：Handler 可独立开发测试
//   - 易于扩展：新增功能只需添加新 Handler
package handler

import (
	"context"
)

// Handler 接口 - 所有操作处理器实现此接口
type Handler interface {
	// Name 返回 Handler 名称（唯一标识）
	Name() string

	// Start 启动 Handler
	// Handler 应该启动自己的工作循环，直到 ctx 被取消
	Start(ctx context.Context) error

	// Stop 停止 Handler
	// 用于优雅关闭，释放资源
	Stop() error
}

// LifecycleHandler 扩展接口 - 支持初始化和健康检查
type LifecycleHandler interface {
	Handler

	// Init 初始化 Handler（在 Start 之前调用）
	Init(ctx context.Context) error

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error
}

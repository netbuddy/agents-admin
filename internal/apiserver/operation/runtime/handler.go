// Package runtime 运行时操作领域 - HTTP 处理
//
// 处理运行时相关的系统操作：
//   - runtime_create: 创建运行时环境
//   - runtime_start: 启动运行时
//   - runtime_stop: 停止运行时
//   - runtime_destroy: 销毁运行时
//
// 当前为占位包，具体实现将在运行时管理功能开发时添加。
package runtime

import (
	"agents-admin/internal/shared/storage"
)

// Handler 运行时操作领域 HTTP 处理器
type Handler struct {
	store storage.PersistentStore
}

// NewHandler 创建运行时操作处理器
func NewHandler(store storage.PersistentStore) *Handler {
	return &Handler{store: store}
}

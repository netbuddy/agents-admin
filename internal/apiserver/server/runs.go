// Package handler 执行管理接口
//
// 注意：HTTP 处理函数已迁移到 internal/apiserver/run 包
// 本文件只保留与 Handler 结构体相关的方法（如 StartScheduler）
package server

import (
	"context"
)

// StartScheduler 启动任务调度器
//
// 调度器会定期扫描 queued 状态的 Run，并分配到可用的 Node 执行。
//
// 参数：
//   - ctx: 上下文，用于控制调度器生命周期
func (h *Handler) StartScheduler(ctx context.Context) {
	h.scheduler.Start(ctx)
}

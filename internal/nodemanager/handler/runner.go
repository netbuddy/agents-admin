// Package handler Runner Handler - 负责 Run 执行
//
// HTTP-Only 架构：任务分发由 manager.go 的 taskLoop 通过 HTTP 轮询完成，
// 此 Handler 保留接口实现，用于未来 Handler 插件框架扩展。
package handler

import (
	"context"
	"log"
)

// RunExecutor Run 执行器接口
type RunExecutor interface {
	ExecuteRun(ctx context.Context, runID string)
	IsRunning(runID string) bool
}

// RunnerConfig Runner Handler 配置
type RunnerConfig struct {
	NodeID       string
	APIServerURL string
}

// RunnerHandler Run 执行 Handler
type RunnerHandler struct {
	config   RunnerConfig
	executor RunExecutor
	stopCh   chan struct{}
}

// NewRunnerHandler 创建 Runner Handler
func NewRunnerHandler(cfg RunnerConfig, executor RunExecutor) *RunnerHandler {
	return &RunnerHandler{
		config:   cfg,
		executor: executor,
		stopCh:   make(chan struct{}),
	}
}

// Name 返回 Handler 名称
func (h *RunnerHandler) Name() string {
	return "runner"
}

// Start 启动 Run 处理循环
// HTTP-Only 架构下，任务分发由 manager.go taskLoop 统一处理
func (h *RunnerHandler) Start(ctx context.Context) error {
	log.Printf("[handler.runner] starting on node: %s (HTTP-Only mode)", h.config.NodeID)
	<-ctx.Done()
	return nil
}

// Stop 停止 Handler
func (h *RunnerHandler) Stop() error {
	close(h.stopCh)
	return nil
}

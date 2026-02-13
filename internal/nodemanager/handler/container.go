// Package handler Container Handler - 负责容器管理
//
// 包括：
//   - Agent 实例生命周期管理（创建/启动/停止/销毁）
//   - Terminal 会话管理
package handler

import (
	"context"
	"sync"
)

// AgentWorker Agent 实例工作器接口
type AgentWorker interface {
	Start(ctx context.Context)
}

// TerminalWorker Terminal 工作器接口
type TerminalWorker interface {
	Start(ctx context.Context)
}

// ContainerHandler 容器管理 Handler
type ContainerHandler struct {
	agentWorker    AgentWorker
	terminalWorker TerminalWorker
}

// NewContainerHandler 创建容器 Handler
func NewContainerHandler(agentWorker AgentWorker, terminalWorker TerminalWorker) *ContainerHandler {
	return &ContainerHandler{
		agentWorker:    agentWorker,
		terminalWorker: terminalWorker,
	}
}

// Name 返回 Handler 名称
func (h *ContainerHandler) Name() string {
	return "container"
}

// Start 启动容器管理
func (h *ContainerHandler) Start(ctx context.Context) error {
	var wg sync.WaitGroup

	// 启动 Agent 工作器
	if h.agentWorker != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.agentWorker.Start(ctx)
		}()
	}

	// 启动 Terminal 工作器
	if h.terminalWorker != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.terminalWorker.Start(ctx)
		}()
	}

	wg.Wait()
	return nil
}

// Stop 停止 Handler
func (h *ContainerHandler) Stop() error {
	// Worker 会通过 context 取消自动停止
	return nil
}

// Package handler Runner Handler - 负责 Run 执行
//
// 职责：
//   - 从队列接收分配的 Run
//   - 派发 Run 给执行器执行
//   - 状态上报
package handler

import (
	"context"
	"log"
	"sync"
	"time"

	"agents-admin/internal/shared/queue"
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
	config    RunnerConfig
	nodeQueue queue.NodeRunQueue
	executor  RunExecutor
	stopCh    chan struct{}
}

// NewRunnerHandler 创建 Runner Handler
func NewRunnerHandler(cfg RunnerConfig, nodeQueue queue.NodeRunQueue, executor RunExecutor) *RunnerHandler {
	return &RunnerHandler{
		config:    cfg,
		nodeQueue: nodeQueue,
		executor:  executor,
		stopCh:    make(chan struct{}),
	}
}

// Name 返回 Handler 名称
func (h *RunnerHandler) Name() string {
	return "runner"
}

// Start 启动 Run 处理循环
func (h *RunnerHandler) Start(ctx context.Context) error {
	log.Printf("[handler.runner] starting on node: %s", h.config.NodeID)

	var wg sync.WaitGroup

	// 主路径：队列消费
	if h.nodeQueue != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.consumeFromQueue(ctx)
		}()
	}

	// 保底轮询
	wg.Add(1)
	go func() {
		defer wg.Done()
		h.fallbackPolling(ctx)
	}()

	wg.Wait()
	return nil
}

// Stop 停止 Handler
func (h *RunnerHandler) Stop() error {
	close(h.stopCh)
	return nil
}

// consumeFromQueue 从队列消费 Run
func (h *RunnerHandler) consumeFromQueue(ctx context.Context) {
	// 创建消费者组
	if err := h.nodeQueue.CreateNodeConsumerGroup(ctx, h.config.NodeID); err != nil {
		log.Printf("[handler.runner] create consumer group failed: %v", err)
	}

	log.Printf("[handler.runner] queue consumer started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-h.stopCh:
			return
		default:
		}

		// 消费消息
		messages, err := h.nodeQueue.ConsumeNodeRuns(ctx, h.config.NodeID, h.config.NodeID, 10, 5*time.Second)
		if err != nil {
			log.Printf("[handler.runner] consume error: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		for _, msg := range messages {
			if h.executor != nil && !h.executor.IsRunning(msg.RunID) {
				go h.executor.ExecuteRun(ctx, msg.RunID)
			}

			// ACK
			if err := h.nodeQueue.AckNodeRun(ctx, h.config.NodeID, msg.ID); err != nil {
				log.Printf("[handler.runner] ack error: %v", err)
			}
		}
	}
}

// fallbackPolling 保底轮询
func (h *RunnerHandler) fallbackPolling(ctx context.Context) {
	// 根据是否有队列设置不同的轮询间隔
	interval := 60 * time.Second
	if h.nodeQueue == nil {
		interval = 5 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-h.stopCh:
			return
		case <-ticker.C:
			// TODO: 实现保底轮询逻辑
			// 从 API Server 获取分配给本节点的 Run
		}
	}
}

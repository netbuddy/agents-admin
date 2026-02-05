// Package handler Auth Handler - 负责认证操作
//
// 适配器模式：包装 AuthControllerV2
package handler

import (
	"context"

	"agents-admin/internal/shared/storage"
)

// AuthController 认证控制器接口
type AuthController interface {
	Start(ctx context.Context)
	Close() error
	SetEventBus(eventBus *storage.EtcdEventBus)
}

// AuthHandler 认证操作 Handler
type AuthHandler struct {
	controller AuthController
	eventBus   *storage.EtcdEventBus
}

// NewAuthHandler 创建认证 Handler
func NewAuthHandler(controller AuthController) *AuthHandler {
	return &AuthHandler{
		controller: controller,
	}
}

// Name 返回 Handler 名称
func (h *AuthHandler) Name() string {
	return "auth"
}

// SetEventBus 设置事件总线
func (h *AuthHandler) SetEventBus(eventBus *storage.EtcdEventBus) {
	h.eventBus = eventBus
	if h.controller != nil {
		h.controller.SetEventBus(eventBus)
	}
}

// Start 启动认证任务控制循环
func (h *AuthHandler) Start(ctx context.Context) error {
	if h.controller != nil {
		h.controller.Start(ctx)
	}
	return nil
}

// Stop 停止 Handler
func (h *AuthHandler) Stop() error {
	if h.controller != nil {
		return h.controller.Close()
	}
	return nil
}

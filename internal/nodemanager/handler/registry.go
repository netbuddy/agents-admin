// Package handler Handler 注册表
package handler

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// Registry 管理所有 Handler
type Registry struct {
	handlers map[string]Handler
	mu       sync.RWMutex
}

// NewRegistry 创建 Handler 注册表
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]Handler),
	}
}

// Register 注册 Handler
func (r *Registry) Register(h Handler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := h.Name()
	if _, exists := r.handlers[name]; exists {
		return fmt.Errorf("handler %s already registered", name)
	}

	r.handlers[name] = h
	log.Printf("[handler.registry] registered: %s", name)
	return nil
}

// Get 获取 Handler
func (r *Registry) Get(name string) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[name]
	return h, ok
}

// List 列出所有 Handler 名称
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		names = append(names, name)
	}
	return names
}

// StartAll 启动所有 Handler
// 每个 Handler 在独立 goroutine 中运行
func (r *Registry) StartAll(ctx context.Context, wg *sync.WaitGroup) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, h := range r.handlers {
		wg.Add(1)
		go func(name string, h Handler) {
			defer wg.Done()
			log.Printf("[handler.registry] starting: %s", name)

			if err := h.Start(ctx); err != nil {
				log.Printf("[handler.registry] %s error: %v", name, err)
			}

			log.Printf("[handler.registry] stopped: %s", name)
		}(name, h)
	}
}

// StopAll 停止所有 Handler
func (r *Registry) StopAll() {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, h := range r.handlers {
		log.Printf("[handler.registry] stopping: %s", name)
		if err := h.Stop(); err != nil {
			log.Printf("[handler.registry] %s stop error: %v", name, err)
		}
	}
}

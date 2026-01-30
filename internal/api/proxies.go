package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"agents-admin/internal/model"
)

// ListProxies 获取代理列表
func (h *Handler) ListProxies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	proxies, err := h.store.ListProxies(ctx)
	if err != nil {
		log.Printf("[API] ListProxies error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list proxies")
		return
	}

	// 清除密码字段（安全考虑）
	for _, p := range proxies {
		p.Password = nil
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"proxies": proxies,
	})
}

// CreateProxy 创建代理
func (h *Handler) CreateProxy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Name      string  `json:"name"`
		Type      string  `json:"type"`
		Host      string  `json:"host"`
		Port      int     `json:"port"`
		Username  *string `json:"username,omitempty"`
		Password  *string `json:"password,omitempty"`
		NoProxy   *string `json:"no_proxy,omitempty"`
		IsDefault bool    `json:"is_default"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// 验证必填字段
	if req.Name == "" || req.Host == "" || req.Port == 0 {
		writeError(w, http.StatusBadRequest, "name, host and port are required")
		return
	}

	// 默认类型
	if req.Type == "" {
		req.Type = string(model.ProxyTypeHTTP)
	}

	// 生成ID
	proxyID := fmt.Sprintf("proxy_%d", time.Now().UnixNano())

	now := time.Now()
	proxy := &model.Proxy{
		ID:        proxyID,
		Name:      req.Name,
		Type:      model.ProxyType(req.Type),
		Host:      req.Host,
		Port:      req.Port,
		Username:  req.Username,
		Password:  req.Password,
		NoProxy:   req.NoProxy,
		IsDefault: req.IsDefault,
		Status:    model.ProxyStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// 验证
	if err := proxy.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 如果设为默认，先清除其他默认
	if req.IsDefault {
		if err := h.store.ClearDefaultProxy(ctx); err != nil {
			log.Printf("[API] ClearDefaultProxy error: %v", err)
		}
	}

	// 创建代理
	if err := h.store.CreateProxy(ctx, proxy); err != nil {
		log.Printf("[API] CreateProxy error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create proxy")
		return
	}

	log.Printf("[API] Proxy created: %s (%s:%d)", proxyID, req.Host, req.Port)

	// 返回时清除密码
	proxy.Password = nil
	writeJSON(w, http.StatusCreated, proxy)
}

// GetProxy 获取代理详情
func (h *Handler) GetProxy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	proxyID := r.PathValue("id")

	proxy, err := h.store.GetProxy(ctx, proxyID)
	if err != nil {
		log.Printf("[API] GetProxy error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get proxy")
		return
	}

	if proxy == nil {
		writeError(w, http.StatusNotFound, "proxy not found")
		return
	}

	// 清除密码
	proxy.Password = nil
	writeJSON(w, http.StatusOK, proxy)
}

// UpdateProxy 更新代理
func (h *Handler) UpdateProxy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	proxyID := r.PathValue("id")

	// 检查代理是否存在
	existing, err := h.store.GetProxy(ctx, proxyID)
	if err != nil {
		log.Printf("[API] GetProxy error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get proxy")
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "proxy not found")
		return
	}

	var req struct {
		Name     *string `json:"name,omitempty"`
		Type     *string `json:"type,omitempty"`
		Host     *string `json:"host,omitempty"`
		Port     *int    `json:"port,omitempty"`
		Username *string `json:"username,omitempty"`
		Password *string `json:"password,omitempty"`
		NoProxy  *string `json:"no_proxy,omitempty"`
		Status   *string `json:"status,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// 更新字段
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Type != nil {
		existing.Type = model.ProxyType(*req.Type)
	}
	if req.Host != nil {
		existing.Host = *req.Host
	}
	if req.Port != nil {
		existing.Port = *req.Port
	}
	if req.Username != nil {
		existing.Username = req.Username
	}
	if req.Password != nil {
		existing.Password = req.Password
	}
	if req.NoProxy != nil {
		existing.NoProxy = req.NoProxy
	}
	if req.Status != nil {
		existing.Status = model.ProxyStatus(*req.Status)
	}

	// 验证
	if err := existing.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 更新
	if err := h.store.UpdateProxy(ctx, existing); err != nil {
		log.Printf("[API] UpdateProxy error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update proxy")
		return
	}

	log.Printf("[API] Proxy updated: %s", proxyID)

	// 清除密码
	existing.Password = nil
	writeJSON(w, http.StatusOK, existing)
}

// DeleteProxy 删除代理
func (h *Handler) DeleteProxy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	proxyID := r.PathValue("id")

	// 检查代理是否存在
	proxy, err := h.store.GetProxy(ctx, proxyID)
	if err != nil {
		log.Printf("[API] GetProxy error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get proxy")
		return
	}
	if proxy == nil {
		writeError(w, http.StatusNotFound, "proxy not found")
		return
	}

	// 删除代理
	if err := h.store.DeleteProxy(ctx, proxyID); err != nil {
		log.Printf("[API] DeleteProxy error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to delete proxy")
		return
	}

	log.Printf("[API] Proxy deleted: %s", proxyID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "proxy deleted",
	})
}

// SetDefaultProxy 设置默认代理
func (h *Handler) SetDefaultProxy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	proxyID := r.PathValue("id")

	// 检查代理是否存在
	proxy, err := h.store.GetProxy(ctx, proxyID)
	if err != nil {
		log.Printf("[API] GetProxy error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get proxy")
		return
	}
	if proxy == nil {
		writeError(w, http.StatusNotFound, "proxy not found")
		return
	}

	// 设置默认代理
	if err := h.store.SetDefaultProxy(ctx, proxyID); err != nil {
		log.Printf("[API] SetDefaultProxy error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to set default proxy")
		return
	}

	log.Printf("[API] Default proxy set: %s", proxyID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "default proxy set",
	})
}

// ClearDefaultProxy 清除默认代理
func (h *Handler) ClearDefaultProxy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := h.store.ClearDefaultProxy(ctx); err != nil {
		log.Printf("[API] ClearDefaultProxy error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to clear default proxy")
		return
	}

	log.Printf("[API] Default proxy cleared")
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "default proxy cleared",
	})
}

// TestProxy 测试代理连通性
func (h *Handler) TestProxy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	proxyID := r.PathValue("id")

	// 获取代理
	proxy, err := h.store.GetProxy(ctx, proxyID)
	if err != nil {
		log.Printf("[API] GetProxy error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get proxy")
		return
	}
	if proxy == nil {
		writeError(w, http.StatusNotFound, "proxy not found")
		return
	}

	// 测试连通性（简单TCP连接测试）
	addr := net.JoinHostPort(proxy.Host, fmt.Sprintf("%d", proxy.Port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		log.Printf("[API] Proxy test failed: %s - %v", proxyID, err)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("connection failed: %v", err),
		})
		return
	}
	conn.Close()

	log.Printf("[API] Proxy test success: %s", proxyID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "proxy is reachable",
	})
}

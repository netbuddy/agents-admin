// Package proxy 代理领域 - HTTP 处理
package proxy

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"agents-admin/internal/shared/model"
	"agents-admin/internal/shared/storage"
)

// Handler 代理领域 HTTP 处理器
type Handler struct {
	store storage.PersistentStore
}

// NewHandler 创建代理处理器
func NewHandler(store storage.PersistentStore) *Handler {
	return &Handler{store: store}
}

// RegisterRoutes 注册代理相关路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/proxies", h.List)
	mux.HandleFunc("POST /api/v1/proxies", h.Create)
	mux.HandleFunc("GET /api/v1/proxies/{id}", h.Get)
	mux.HandleFunc("PUT /api/v1/proxies/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/proxies/{id}", h.Delete)
	mux.HandleFunc("POST /api/v1/proxies/{id}/test", h.Test)
	mux.HandleFunc("POST /api/v1/proxies/{id}/set-default", h.SetDefault)
	mux.HandleFunc("POST /api/v1/proxies/clear-default", h.ClearDefault)
}

// List 获取代理列表
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	proxies, err := h.store.ListProxies(r.Context())
	if err != nil {
		log.Printf("[proxy] ListProxies error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list proxies")
		return
	}

	for _, p := range proxies {
		p.Password = nil
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"proxies": proxies})
}

// Create 创建代理
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
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

	if req.Name == "" || req.Type == "" || req.Host == "" || req.Port == 0 {
		writeError(w, http.StatusBadRequest, "name, type, host and port are required")
		return
	}

	now := time.Now()
	proxy := &model.Proxy{
		ID:        generateID("proxy"),
		Name:      req.Name,
		Type:      model.ProxyType(req.Type),
		Host:      req.Host,
		Port:      req.Port,
		Username:  req.Username,
		Password:  req.Password,
		NoProxy:   req.NoProxy,
		IsDefault: req.IsDefault,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.store.CreateProxy(r.Context(), proxy); err != nil {
		log.Printf("[proxy] CreateProxy error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create proxy")
		return
	}

	proxy.Password = nil
	log.Printf("[proxy] Proxy created: %s", proxy.ID)
	writeJSON(w, http.StatusCreated, proxy)
}

// Get 获取代理详情
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	proxy, err := h.store.GetProxy(r.Context(), id)
	if err != nil {
		log.Printf("[proxy] GetProxy error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get proxy")
		return
	}
	if proxy == nil {
		writeError(w, http.StatusNotFound, "proxy not found")
		return
	}

	proxy.Password = nil
	writeJSON(w, http.StatusOK, proxy)
}

// Update 更新代理
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	proxy, err := h.store.GetProxy(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get proxy")
		return
	}
	if proxy == nil {
		writeError(w, http.StatusNotFound, "proxy not found")
		return
	}

	var req struct {
		Name      *string `json:"name,omitempty"`
		Type      *string `json:"type,omitempty"`
		Host      *string `json:"host,omitempty"`
		Port      *int    `json:"port,omitempty"`
		Username  *string `json:"username,omitempty"`
		Password  *string `json:"password,omitempty"`
		NoProxy   *string `json:"no_proxy,omitempty"`
		IsDefault *bool   `json:"is_default,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != nil {
		proxy.Name = *req.Name
	}
	if req.Type != nil {
		proxy.Type = model.ProxyType(*req.Type)
	}
	if req.Host != nil {
		proxy.Host = *req.Host
	}
	if req.Port != nil {
		proxy.Port = *req.Port
	}
	if req.Username != nil {
		proxy.Username = req.Username
	}
	if req.Password != nil {
		proxy.Password = req.Password
	}
	if req.NoProxy != nil {
		proxy.NoProxy = req.NoProxy
	}
	if req.IsDefault != nil {
		proxy.IsDefault = *req.IsDefault
	}
	proxy.UpdatedAt = time.Now()

	if err := h.store.UpdateProxy(r.Context(), proxy); err != nil {
		log.Printf("[proxy] UpdateProxy error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update proxy")
		return
	}

	proxy.Password = nil
	log.Printf("[proxy] Proxy updated: %s", id)
	writeJSON(w, http.StatusOK, proxy)
}

// Delete 删除代理
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.store.DeleteProxy(r.Context(), id); err != nil {
		log.Printf("[proxy] DeleteProxy error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to delete proxy")
		return
	}

	log.Printf("[proxy] Proxy deleted: %s", id)
	writeJSON(w, http.StatusOK, map[string]interface{}{"message": "proxy deleted"})
}

// Test 测试代理连通性
func (h *Handler) Test(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	proxy, err := h.store.GetProxy(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get proxy")
		return
	}
	if proxy == nil {
		writeError(w, http.StatusNotFound, "proxy not found")
		return
	}

	addr := net.JoinHostPort(proxy.Host, fmt.Sprintf("%d", proxy.Port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		log.Printf("[proxy] Test failed for %s: %v", id, err)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("connection failed: %v", err),
		})
		return
	}
	conn.Close()

	log.Printf("[proxy] Test success for %s", id)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "proxy is reachable",
	})
}

// SetDefault 设置默认代理
func (h *Handler) SetDefault(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	proxy, err := h.store.GetProxy(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get proxy")
		return
	}
	if proxy == nil {
		writeError(w, http.StatusNotFound, "proxy not found")
		return
	}

	if err := h.store.SetDefaultProxy(r.Context(), id); err != nil {
		log.Printf("[proxy] SetDefault error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to set default proxy")
		return
	}

	log.Printf("[proxy] Default proxy set: %s", id)
	writeJSON(w, http.StatusOK, map[string]interface{}{"message": "default proxy set"})
}

// ClearDefault 清除默认代理
func (h *Handler) ClearDefault(w http.ResponseWriter, r *http.Request) {
	if err := h.store.ClearDefaultProxy(r.Context()); err != nil {
		log.Printf("[proxy] ClearDefaultProxy error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to clear default proxy")
		return
	}

	log.Printf("[proxy] Default proxy cleared")
	writeJSON(w, http.StatusOK, map[string]interface{}{"message": "default proxy cleared"})
}

func generateID(prefix string) string {
	b := make([]byte, 6)
	rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

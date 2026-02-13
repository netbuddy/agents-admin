// Package proxy 代理领域 - HTTP 处理
package proxy

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
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
	// 默认代理功能已移除
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
		Status:    model.ProxyStatusActive,
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
// 支持两级验证：
//  1. TCP 端口连通性（默认，无 target_url 时）
//  2. 端到端 HTTP 代理验证（提供 target_url 时，通过代理实际请求目标）
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

	// 解析可选的 target_url
	var req struct {
		TargetURL string `json:"target_url"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.TargetURL != "" {
		// 端到端代理验证：通过代理请求目标 URL
		h.testProxyEndToEnd(w, proxy, req.TargetURL)
		return
	}

	// 默认：TCP 连通性检查
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

// testProxyEndToEnd 通过代理实际请求目标 URL，验证代理转发能力
func (h *Handler) testProxyEndToEnd(w http.ResponseWriter, proxy *model.Proxy, targetURL string) {
	proxyScheme := "http"
	if proxy.Type == "socks5" {
		proxyScheme = "socks5"
	}

	proxyURLStr := fmt.Sprintf("%s://%s:%d", proxyScheme, proxy.Host, proxy.Port)
	if proxy.Username != nil && *proxy.Username != "" {
		pwd := ""
		if proxy.Password != nil {
			pwd = *proxy.Password
		}
		proxyURLStr = fmt.Sprintf("%s://%s:%s@%s:%d", proxyScheme, *proxy.Username, pwd, proxy.Host, proxy.Port)
	}

	proxyURL, err := url.Parse(proxyURLStr)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success":    false,
			"target_url": targetURL,
			"message":    fmt.Sprintf("invalid proxy URL: %v", err),
		})
		return
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyURL(proxyURL),
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	start := time.Now()
	resp, err := client.Get(targetURL)
	latencyMs := time.Since(start).Milliseconds()

	if err != nil {
		log.Printf("[proxy] E2E test failed for %s -> %s: %v", proxy.ID, targetURL, err)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success":    false,
			"target_url": targetURL,
			"latency_ms": latencyMs,
			"message":    fmt.Sprintf("request failed: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	log.Printf("[proxy] E2E test for %s -> %s: HTTP %d (%dms)", proxy.ID, targetURL, resp.StatusCode, latencyMs)

	// 读取响应体摘要作为验证证据（最多 512 字节）
	bodyBuf := make([]byte, 512)
	n, _ := io.ReadFull(resp.Body, bodyBuf)
	bodySnippet := string(bodyBuf[:n])
	// 提取 <title> 标签内容
	titleSnippet := ""
	if idx := strings.Index(strings.ToLower(bodySnippet), "<title>"); idx >= 0 {
		rest := bodySnippet[idx+7:]
		if end := strings.Index(strings.ToLower(rest), "</title>"); end >= 0 {
			titleSnippet = strings.TrimSpace(rest[:end])
		}
	}

	// 收集关键响应头
	headers := map[string]string{}
	for _, key := range []string{"Content-Type", "Server", "X-Frame-Options"} {
		if v := resp.Header.Get(key); v != "" {
			headers[key] = v
		}
	}

	success := resp.StatusCode >= 200 && resp.StatusCode < 400
	msg := fmt.Sprintf("HTTP %d (%dms)", resp.StatusCode, latencyMs)
	if success {
		msg = fmt.Sprintf("代理可用，目标响应正常 (%dms)", latencyMs)
	} else {
		msg = fmt.Sprintf("目标返回 HTTP %d (%dms)", resp.StatusCode, latencyMs)
	}

	result := map[string]interface{}{
		"success":     success,
		"target_url":  targetURL,
		"status_code": resp.StatusCode,
		"latency_ms":  latencyMs,
		"message":     msg,
		"headers":     headers,
	}
	if titleSnippet != "" {
		result["page_title"] = titleSnippet
	}
	writeJSON(w, http.StatusOK, result)
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

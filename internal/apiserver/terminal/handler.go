// Package terminal 终端会话领域 - HTTP 处理
//
// 本文件实现终端会话相关的 API 端点：
//   - 终端会话 CRUD
//   - WebSocket 反向代理
//   - NodeManager 回调接口
package terminal

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"agents-admin/internal/shared/model"
	"agents-admin/internal/shared/storage"
)

// Handler 终端会话领域 HTTP 处理器
type Handler struct {
	store storage.PersistentStore
}

// NewHandler 创建终端会话处理器
func NewHandler(store storage.PersistentStore) *Handler {
	return &Handler{store: store}
}

// RegisterRoutes 注册终端会话相关路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/terminal-sessions", h.List)
	mux.HandleFunc("POST /api/v1/terminal-sessions", h.Create)
	mux.HandleFunc("GET /api/v1/terminal-sessions/{id}", h.Get)
	mux.HandleFunc("PATCH /api/v1/terminal-sessions/{id}", h.UpdateStatus)
	mux.HandleFunc("DELETE /api/v1/terminal-sessions/{id}", h.Delete)
	mux.HandleFunc("/terminal/{id}/", h.Proxy)

	// 兼容旧路径（将废弃）
	mux.HandleFunc("POST /api/v1/terminal/session", h.Create)
	mux.HandleFunc("GET /api/v1/terminal/session/{id}", h.Get)
	mux.HandleFunc("DELETE /api/v1/terminal/session/{id}", h.Delete)
}

// RegisterNodeManagerRoutes 注册 NodeManager 相关路由
func (h *Handler) RegisterNodeManagerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/nodes/{node_id}/terminal-sessions", h.ListPending)
}

// ============================================================================
// 用户 API
// ============================================================================

// Create 创建终端会话（声明式：只创建数据库记录，NodeManager 负责实际终端启动）
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InstanceID    string `json:"instance_id"`
		ContainerName string `json:"container_name"`
		NodeID        string `json:"node_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	containerName := req.ContainerName
	nodeID := req.NodeID

	// 如果指定了实例 ID，获取容器名和节点信息
	if req.InstanceID != "" {
		instance, err := h.store.GetAgentInstance(r.Context(), req.InstanceID)
		if err != nil {
			log.Printf("[terminal] Failed to get instance %s: %v", req.InstanceID, err)
			writeError(w, http.StatusInternalServerError, "failed to get instance")
			return
		}
		if instance == nil {
			writeError(w, http.StatusNotFound, "instance not found")
			return
		}

		if instance.Status != model.InstanceStatusRunning {
			writeError(w, http.StatusBadRequest, "instance not running")
			return
		}

		if instance.ContainerName != nil {
			containerName = *instance.ContainerName
		}
		if instance.NodeID != nil && nodeID == "" {
			nodeID = *instance.NodeID
		}
	}

	if containerName == "" {
		writeError(w, http.StatusBadRequest, "container_name or instance_id is required")
		return
	}

	if nodeID == "" {
		writeError(w, http.StatusBadRequest, "node_id is required")
		return
	}

	sessionID := generateID("term")
	expiresAt := time.Now().Add(30 * time.Minute)

	now := time.Now()
	var instanceIDPtr *string
	if req.InstanceID != "" {
		instanceIDPtr = &req.InstanceID
	}

	session := &model.TerminalSession{
		ID:            sessionID,
		InstanceID:    instanceIDPtr,
		ContainerName: containerName,
		NodeID:        &nodeID,
		Status:        model.TerminalStatusPending,
		CreatedAt:     now,
		ExpiresAt:     &expiresAt,
	}

	if err := h.store.CreateTerminalSession(r.Context(), session); err != nil {
		log.Printf("[terminal] Failed to create session: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	log.Printf("[terminal] Created session %s (container=%s, node=%s, status=pending)",
		sessionID, containerName, nodeID)

	writeJSON(w, http.StatusCreated, session)
}

// Get 获取终端会话状态
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")

	session, err := h.store.GetTerminalSession(r.Context(), sessionID)
	if err != nil {
		log.Printf("[terminal] Failed to get session %s: %v", sessionID, err)
		writeError(w, http.StatusInternalServerError, "failed to get session")
		return
	}
	if session == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	writeJSON(w, http.StatusOK, session)
}

// Delete 关闭终端会话
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")

	session, err := h.store.GetTerminalSession(r.Context(), sessionID)
	if err != nil {
		log.Printf("[terminal] Failed to get session %s: %v", sessionID, err)
		writeError(w, http.StatusInternalServerError, "failed to get session")
		return
	}
	if session == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	if err := h.store.UpdateTerminalSession(r.Context(), sessionID, model.TerminalStatusClosed, nil, nil); err != nil {
		log.Printf("[terminal] Failed to update session %s: %v", sessionID, err)
		writeError(w, http.StatusInternalServerError, "failed to close session")
		return
	}

	log.Printf("[terminal] Session %s marked as closed", sessionID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "session closed",
	})
}

// List 列出终端会话
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.store.ListTerminalSessions(r.Context())
	if err != nil {
		log.Printf("[terminal] Failed to list sessions: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list sessions")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// Proxy 代理终端连接（支持 HTTP + WebSocket）
func (h *Handler) Proxy(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")

	session, err := h.store.GetTerminalSession(r.Context(), sessionID)
	if err != nil {
		log.Printf("[terminal] Failed to get session %s: %v", sessionID, err)
		writeError(w, http.StatusInternalServerError, "failed to get session")
		return
	}
	if session == nil || session.Status != model.TerminalStatusRunning {
		writeError(w, http.StatusNotFound, "session not found or not running")
		return
	}

	if session.Port == nil {
		writeError(w, http.StatusServiceUnavailable, "terminal not ready")
		return
	}

	backendAddr := fmt.Sprintf("localhost:%d", *session.Port)

	// 剥离前缀
	r.URL.Path = strings.TrimPrefix(r.URL.Path, fmt.Sprintf("/terminal/%s", sessionID))
	if r.URL.Path == "" {
		r.URL.Path = "/"
	}

	// WebSocket 请求：使用 TCP 双向转发
	if isWebSocketUpgrade(r) {
		h.proxyWebSocket(w, r, backendAddr)
		return
	}

	// 普通 HTTP 请求：标准反向代理
	target, _ := url.Parse(fmt.Sprintf("http://%s", backendAddr))
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ServeHTTP(w, r)
}

// isWebSocketUpgrade 检测是否为 WebSocket 升级请求
func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

// proxyWebSocket 使用 TCP hijack 双向代理 WebSocket
func (h *Handler) proxyWebSocket(w http.ResponseWriter, r *http.Request, backendAddr string) {
	// 连接后端
	backendConn, err := net.DialTimeout("tcp", backendAddr, 5*time.Second)
	if err != nil {
		log.Printf("[terminal] WebSocket backend dial failed: %v", err)
		writeError(w, http.StatusBadGateway, "backend unavailable")
		return
	}
	defer backendConn.Close()

	// Hijack 客户端连接
	hj, ok := w.(http.Hijacker)
	if !ok {
		writeError(w, http.StatusInternalServerError, "hijack not supported")
		return
	}
	clientConn, clientBuf, err := hj.Hijack()
	if err != nil {
		log.Printf("[terminal] Hijack failed: %v", err)
		return
	}
	defer clientConn.Close()

	// 将原始请求转发给后端
	if err := r.Write(backendConn); err != nil {
		log.Printf("[terminal] Failed to write request to backend: %v", err)
		return
	}

	// 双向复制数据
	done := make(chan struct{}, 2)

	// 后端 → 客户端
	go func() {
		io.Copy(clientConn, backendConn)
		done <- struct{}{}
	}()

	// 客户端 → 后端（先 flush buffered reader 中的残留数据）
	go func() {
		if clientBuf.Reader.Buffered() > 0 {
			io.CopyN(backendConn, clientBuf, int64(clientBuf.Reader.Buffered()))
		}
		io.Copy(backendConn, clientConn)
		done <- struct{}{}
	}()

	<-done
}

// ============================================================================
// NodeManager API
// ============================================================================

// ListPending 列出节点待处理的终端会话（NodeManager 轮询用）
// GET /api/v1/nodes/{node_id}/terminal-sessions
func (h *Handler) ListPending(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("node_id")
	if nodeID == "" {
		writeError(w, http.StatusBadRequest, "node_id is required")
		return
	}

	sessions, err := h.store.ListPendingTerminalSessions(r.Context(), nodeID)
	if err != nil {
		log.Printf("[terminal] Failed to list pending sessions for node %s: %v", nodeID, err)
		writeError(w, http.StatusInternalServerError, "failed to list sessions")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// UpdateStatus 更新终端会话状态（NodeManager 回调）
// PATCH /api/v1/terminal-sessions/{id}
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")

	var req struct {
		Status string  `json:"status"`
		Port   *int    `json:"port,omitempty"`
		URL    *string `json:"url,omitempty"`
		Error  *string `json:"error,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Status == "" {
		writeError(w, http.StatusBadRequest, "status is required")
		return
	}

	status := model.TerminalSessionStatus(req.Status)
	validStatuses := map[model.TerminalSessionStatus]bool{
		model.TerminalStatusPending:  true,
		model.TerminalStatusStarting: true,
		model.TerminalStatusRunning:  true,
		model.TerminalStatusClosed:   true,
		model.TerminalStatusError:    true,
	}
	if !validStatuses[status] {
		writeError(w, http.StatusBadRequest, "invalid status value")
		return
	}

	if err := h.store.UpdateTerminalSession(r.Context(), sessionID, status, req.Port, req.URL); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		log.Printf("[terminal] Failed to update session %s: %v", sessionID, err)
		writeError(w, http.StatusInternalServerError, "failed to update session")
		return
	}

	log.Printf("[terminal] Session %s updated (status=%s, port=%v, url=%v)",
		sessionID, req.Status, req.Port, req.URL)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "session updated",
	})
}

// ============================================================================
// 工具函数
// ============================================================================

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

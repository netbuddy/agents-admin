// Package api Terminal 管理 API
//
// P2-2 重构后的实现：
//   - 状态持久化到 PostgreSQL（替代全局 map）
//   - 终端操作声明式（API 只更新状态，Executor 执行实际操作）
//   - 支持多节点和 API Server 重启恢复
package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"agents-admin/internal/model"
)

// CreateTerminalSession 创建终端会话（声明式：只创建数据库记录，Executor 负责实际终端启动）
func (h *Handler) CreateTerminalSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InstanceID    string `json:"instance_id"`
		ContainerName string `json:"container_name"` // 直接指定容器名
		NodeID        string `json:"node_id"`        // 运行节点
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	containerName := req.ContainerName
	nodeID := req.NodeID

	// 如果指定了实例 ID，获取容器名和节点信息
	if req.InstanceID != "" {
		instance, err := h.store.GetInstance(r.Context(), req.InstanceID)
		if err != nil {
			log.Printf("[Terminal] Failed to get instance %s: %v", req.InstanceID, err)
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

	// 生成会话 ID
	sessionID := generateID("term")

	// 计算过期时间
	expiresAt := time.Now().Add(30 * time.Minute)

	// 创建会话记录（状态为 pending，等待 Executor 启动终端）
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
		Status:        model.TerminalStatusPending, // 声明式：pending 状态，Executor 负责启动
		CreatedAt:     now,
		ExpiresAt:     &expiresAt,
	}

	if err := h.store.CreateTerminalSession(r.Context(), session); err != nil {
		log.Printf("[Terminal] Failed to create session: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	log.Printf("[Terminal] Created session %s (container=%s, node=%s, status=pending)",
		sessionID, containerName, nodeID)

	writeJSON(w, http.StatusCreated, session)
}

// GetTerminalSession 获取终端会话状态
func (h *Handler) GetTerminalSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")

	session, err := h.store.GetTerminalSession(r.Context(), sessionID)
	if err != nil {
		log.Printf("[Terminal] Failed to get session %s: %v", sessionID, err)
		writeError(w, http.StatusInternalServerError, "failed to get session")
		return
	}
	if session == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	writeJSON(w, http.StatusOK, session)
}

// DeleteTerminalSession 关闭终端会话
func (h *Handler) DeleteTerminalSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")

	session, err := h.store.GetTerminalSession(r.Context(), sessionID)
	if err != nil {
		log.Printf("[Terminal] Failed to get session %s: %v", sessionID, err)
		writeError(w, http.StatusInternalServerError, "failed to get session")
		return
	}
	if session == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	// 更新状态为 closed
	if err := h.store.UpdateTerminalSession(r.Context(), sessionID, model.TerminalStatusClosed, nil, nil); err != nil {
		log.Printf("[Terminal] Failed to update session %s: %v", sessionID, err)
		writeError(w, http.StatusInternalServerError, "failed to close session")
		return
	}

	log.Printf("[Terminal] Session %s marked as closed", sessionID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "session closed",
	})
}

// ListTerminalSessions 列出终端会话
func (h *Handler) ListTerminalSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.store.ListTerminalSessions(r.Context())
	if err != nil {
		log.Printf("[Terminal] Failed to list sessions: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list sessions")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// ProxyTerminalSession 代理终端 WebSocket 连接
func (h *Handler) ProxyTerminalSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")

	session, err := h.store.GetTerminalSession(r.Context(), sessionID)
	if err != nil {
		log.Printf("[Terminal] Failed to get session %s: %v", sessionID, err)
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

	// 创建反向代理到 ttyd 端口
	// 注意：在多节点场景下，需要代理到正确的节点
	// 这里简化为本地代理，实际生产环境需要根据 node_id 路由
	target, _ := url.Parse(fmt.Sprintf("http://localhost:%d", *session.Port))
	proxy := httputil.NewSingleHostReverseProxy(target)

	// 移除路径前缀
	r.URL.Path = strings.TrimPrefix(r.URL.Path, fmt.Sprintf("/terminal/%s", sessionID))
	if r.URL.Path == "" {
		r.URL.Path = "/"
	}

	proxy.ServeHTTP(w, r)
}

// ============================================================================
// Executor API（供 Executor 调用）
// ============================================================================

// ListPendingTerminalSessions 列出节点待处理的终端会话（Executor 轮询用）
// GET /api/v1/nodes/{node_id}/terminal-sessions
func (h *Handler) ListPendingTerminalSessions(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("node_id")
	if nodeID == "" {
		writeError(w, http.StatusBadRequest, "node_id is required")
		return
	}

	sessions, err := h.store.ListPendingTerminalSessions(r.Context(), nodeID)
	if err != nil {
		log.Printf("[Terminal] Failed to list pending sessions for node %s: %v", nodeID, err)
		writeError(w, http.StatusInternalServerError, "failed to list sessions")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// UpdateTerminalSessionStatus 更新终端会话状态（Executor 回调）
// PATCH /api/v1/terminal-sessions/{id}
func (h *Handler) UpdateTerminalSessionStatus(w http.ResponseWriter, r *http.Request) {
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

	// 验证状态值
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
		log.Printf("[Terminal] Failed to update session %s: %v", sessionID, err)
		writeError(w, http.StatusInternalServerError, "failed to update session")
		return
	}

	log.Printf("[Terminal] Session %s updated (status=%s, port=%v, url=%v)",
		sessionID, req.Status, req.Port, req.URL)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "session updated",
	})
}

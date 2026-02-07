// Package node 节点领域 - HTTP 处理
package node

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	openapi "agents-admin/api/generated/go"
	"agents-admin/internal/shared/cache"
	"agents-admin/internal/shared/model"
)

// Handler 节点领域 HTTP 处理器
type Handler struct {
	store       NodePersistentStore
	nodeCache   cache.NodeHeartbeatCache
	provisioner *Provisioner
}

// NodePersistentStore 节点处理器所需的持久化存储接口
type NodePersistentStore interface {
	UpsertNode(ctx context.Context, node *model.Node) error
	UpsertNodeHeartbeat(ctx context.Context, node *model.Node) error // 心跳专用，不覆盖 status
	GetNode(ctx context.Context, id string) (*model.Node, error)
	ListAllNodes(ctx context.Context) ([]*model.Node, error)
	ListOnlineNodes(ctx context.Context) ([]*model.Node, error)
	DeleteNode(ctx context.Context, id string) error
	ListRunsByNode(ctx context.Context, nodeID string) ([]*model.Run, error)
	CreateNodeProvision(ctx context.Context, p *model.NodeProvision) error
	UpdateNodeProvision(ctx context.Context, p *model.NodeProvision) error
	GetNodeProvision(ctx context.Context, id string) (*model.NodeProvision, error)
	ListNodeProvisions(ctx context.Context) ([]*model.NodeProvision, error)
}

// NewHandler 创建节点处理器
func NewHandler(store NodePersistentStore, nodeCache cache.NodeHeartbeatCache) *Handler {
	h := &Handler{store: store, nodeCache: nodeCache}
	h.provisioner = NewProvisioner(store, store)
	return h
}

// RegisterRoutes 注册节点相关路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/nodes", h.List)
	mux.HandleFunc("GET /api/v1/nodes/{id}", h.Get)
	mux.HandleFunc("DELETE /api/v1/nodes/{id}", h.Delete)
	mux.HandleFunc("PATCH /api/v1/nodes/{id}", h.Update)
	mux.HandleFunc("POST /api/v1/nodes/heartbeat", h.Heartbeat)
	mux.HandleFunc("GET /api/v1/nodes/{id}/runs", h.GetRuns)
	mux.HandleFunc("GET /api/v1/nodes/{id}/env-config", h.GetEnvConfig)
	mux.HandleFunc("PUT /api/v1/nodes/{id}/env-config", h.UpdateEnvConfig)
	mux.HandleFunc("POST /api/v1/nodes/{id}/env-config/test-proxy", h.TestProxy)
	mux.HandleFunc("POST /api/v1/node-provisions", h.Provision)
	mux.HandleFunc("GET /api/v1/node-provisions", h.ListProvisions)
	mux.HandleFunc("GET /api/v1/node-provisions/{id}", h.GetProvision)
}

// ============================================================================
// 类型别名
// ============================================================================

// HeartbeatRequest 节点心跳请求体
type HeartbeatRequest = openapi.HeartbeatRequest

// UpdateRequest 更新节点的请求体
type UpdateRequest = openapi.UpdateNodeRequest

// Response 节点响应结构
type Response struct {
	ID            string                 `json:"id"`
	Status        string                 `json:"status"`
	Labels        map[string]string      `json:"labels,omitempty"`
	Capacity      map[string]interface{} `json:"capacity,omitempty"`
	LastHeartbeat *time.Time             `json:"last_heartbeat,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// ============================================================================
// HTTP 处理函数
// ============================================================================

// Heartbeat 处理节点心跳
// POST /api/v1/nodes/heartbeat
func (h *Handler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	var req HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[node.heartbeat] ERROR: invalid request body: %v", err)
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.NodeId == "" {
		log.Printf("[node.heartbeat] ERROR: node_id is required")
		writeError(w, http.StatusBadRequest, "node_id is required")
		return
	}

	now := time.Now()

	labels := []byte("{}")
	capacity := []byte("{}")
	if req.Labels != nil {
		labels, _ = json.Marshal(*req.Labels)
	}
	if req.Capacity != nil {
		capacity, _ = json.Marshal(*req.Capacity)
	}

	status := "online"
	if req.Status != nil {
		status = *req.Status
	}

	log.Printf("[node.heartbeat] Received from node=%s, status=%s", req.NodeId, status)

	// 1. 先写 PostgreSQL（持久化优先，使用心跳专用 upsert 不覆盖行政状态）
	node := &model.Node{
		ID:            req.NodeId,
		Status:        model.NodeStatus(status),
		Labels:        labels,
		Capacity:      capacity,
		LastHeartbeat: &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := h.store.UpsertNodeHeartbeat(r.Context(), node); err != nil {
		log.Printf("[node.heartbeat] ERROR: failed to update postgres: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update node")
		return
	}

	// 2. 成功后写 Redis 缓存（可重建，失败不阻塞响应）
	if h.nodeCache != nil {
		capMap := make(map[string]int)
		if req.Capacity != nil {
			for k, v := range *req.Capacity {
				if fv, ok := v.(float64); ok {
					capMap[k] = int(fv)
				}
			}
		}
		nodeStatus := &cache.NodeStatus{
			Status:   status,
			Capacity: capMap,
		}
		if err := h.nodeCache.UpdateNodeHeartbeat(r.Context(), req.NodeId, nodeStatus); err != nil {
			log.Printf("[node.heartbeat] ERROR: failed to update cache: %v", err)
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// List 列出所有节点
// GET /api/v1/nodes
//
// 状态判断优先级：
//  1. 缓存可用且有心跳 → online（使用缓存中的实时容量）
//  2. 缓存可用但无心跳 → offline（使用 PostgreSQL 中的历史值）
//  3. 缓存不可用 → 按 PostgreSQL 的 last_heartbeat 时间窗口（45s）判断
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	nodes, err := h.store.ListAllNodes(r.Context())
	if err != nil {
		log.Printf("[node] ERROR: failed to list nodes: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list nodes")
		return
	}

	result := make([]Response, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, h.buildNodeResponse(r.Context(), n))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"nodes": result, "count": len(result)})
}

// Get 获取单个节点
// GET /api/v1/nodes/{id}
//
// 返回与 List 一致的实时状态（通过缓存判断 online/offline）
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	n, err := h.store.GetNode(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get node")
		return
	}
	if n == nil {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}
	writeJSON(w, http.StatusOK, h.buildNodeResponse(r.Context(), n))
}

// GetRuns 获取分配给节点的 Runs
// GET /api/v1/nodes/{id}/runs
func (h *Handler) GetRuns(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	runs, err := h.store.ListRunsByNode(r.Context(), nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list runs")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"runs": runs, "count": len(runs)})
}

// Delete 删除节点
// DELETE /api/v1/nodes/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	node, err := h.store.GetNode(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get node")
		return
	}
	if node == nil {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}

	runs, err := h.store.ListRunsByNode(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check node runs")
		return
	}
	if len(runs) > 0 {
		writeError(w, http.StatusBadRequest, "node has running tasks, please drain first")
		return
	}

	if err := h.store.DeleteNode(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete node")
		return
	}

	// 清理缓存心跳，避免已删除节点被调度器引用
	if h.nodeCache != nil {
		if err := h.nodeCache.DeleteNodeHeartbeat(r.Context(), id); err != nil {
			log.Printf("[node.delete] WARNING: failed to clear cache for node %s: %v", id, err)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// Update 更新节点信息
// PATCH /api/v1/nodes/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	node, err := h.store.GetNode(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get node")
		return
	}
	if node == nil {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Status != nil {
		node.Status = model.NodeStatus(*req.Status)
	}
	if req.Labels != nil {
		labels, _ := json.Marshal(*req.Labels)
		node.Labels = labels
	}
	node.UpdatedAt = time.Now()

	if err := h.store.UpsertNode(r.Context(), node); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update node")
		return
	}

	// 行政状态变更时清理缓存心跳，避免缓存数据残留
	if req.Status != nil && isAdminStatus(node.Status) && h.nodeCache != nil {
		if err := h.nodeCache.DeleteNodeHeartbeat(r.Context(), id); err != nil {
			log.Printf("[node.update] WARNING: failed to clear cache for node %s: %v", id, err)
		}
	}
	writeJSON(w, http.StatusOK, h.buildNodeResponse(r.Context(), node))
}

// GetEnvConfig 获取节点环境配置
// GET /api/v1/nodes/{id}/env-config
func (h *Handler) GetEnvConfig(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	node, err := h.store.GetNode(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get node")
		return
	}
	if node == nil {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}

	var envConfig *model.EnvConfig
	if node.Capacity != nil {
		var capacity struct {
			EnvConfig *model.EnvConfig `json:"env_config"`
		}
		if err := json.Unmarshal(node.Capacity, &capacity); err == nil {
			envConfig = capacity.EnvConfig
		}
	}

	if envConfig == nil {
		envConfig = &model.EnvConfig{}
	}

	writeJSON(w, http.StatusOK, envConfig)
}

// UpdateEnvConfig 更新节点环境配置
// PUT /api/v1/nodes/{id}/env-config
func (h *Handler) UpdateEnvConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	node, err := h.store.GetNode(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get node")
		return
	}
	if node == nil {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}

	var envConfig model.EnvConfig
	if err := json.NewDecoder(r.Body).Decode(&envConfig); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := envConfig.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var capacity map[string]interface{}
	if node.Capacity != nil {
		json.Unmarshal(node.Capacity, &capacity)
	}
	if capacity == nil {
		capacity = make(map[string]interface{})
	}
	capacity["env_config"] = envConfig

	capacityBytes, _ := json.Marshal(capacity)
	node.Capacity = capacityBytes
	node.UpdatedAt = time.Now()

	if err := h.store.UpsertNode(ctx, node); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update node")
		return
	}

	log.Printf("[node] Node %s env config updated", id)
	writeJSON(w, http.StatusOK, envConfig)
}

// TestProxy 测试节点代理连通性
// POST /api/v1/nodes/{id}/env-config/test-proxy
func (h *Handler) TestProxy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	node, err := h.store.GetNode(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get node")
		return
	}
	if node == nil {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}

	var envConfig *model.EnvConfig
	if node.Capacity != nil {
		var capacity struct {
			EnvConfig *model.EnvConfig `json:"env_config"`
		}
		if err := json.Unmarshal(node.Capacity, &capacity); err == nil {
			envConfig = capacity.EnvConfig
		}
	}

	if envConfig == nil || envConfig.Proxy == nil || !envConfig.Proxy.Enabled {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"message": "proxy not configured or not enabled",
		})
		return
	}

	proxy := envConfig.Proxy
	addr := net.JoinHostPort(proxy.Host, fmt.Sprintf("%d", proxy.Port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		log.Printf("[node] Proxy test failed for node %s: %v", id, err)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("connection failed: %v", err),
		})
		return
	}
	conn.Close()

	log.Printf("[node] Proxy test success for node %s", id)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "proxy is reachable",
	})
}

// buildNodeResponse 构建节点响应，使用缓存判断实时在线状态
func (h *Handler) buildNodeResponse(ctx context.Context, n *model.Node) Response {
	var labels map[string]string
	json.Unmarshal(n.Labels, &labels)

	rs := ResolveNodeStatus(ctx, n, h.nodeCache)

	return Response{
		ID:            n.ID,
		Status:        rs.Status,
		Labels:        labels,
		Capacity:      rs.Capacity,
		LastHeartbeat: rs.LastHeartbeat,
		CreatedAt:     n.CreatedAt,
		UpdatedAt:     n.UpdatedAt,
	}
}

// Provision 创建节点部署任务
// POST /api/v1/nodes/provision
func (h *Handler) Provision(w http.ResponseWriter, r *http.Request) {
	var req ProvisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Host == "" || req.SSHUser == "" || req.Version == "" || req.APIServerURL == "" {
		writeError(w, http.StatusBadRequest, "host, ssh_user, version, api_server_url are required")
		return
	}
	if req.NodeID == "" {
		req.NodeID = fmt.Sprintf("node-%s", req.Host)
	}
	if req.AuthMethod == "" {
		req.AuthMethod = "password"
	}

	prov, err := h.provisioner.StartProvision(r.Context(), req)
	if err != nil {
		log.Printf("[node.provision] ERROR: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to start provision")
		return
	}

	writeJSON(w, http.StatusAccepted, prov)
}

// ListProvisions 列出所有部署记录
// GET /api/v1/nodes/provisions
func (h *Handler) ListProvisions(w http.ResponseWriter, r *http.Request) {
	provisions, err := h.store.ListNodeProvisions(r.Context())
	if err != nil {
		log.Printf("[node.provisions] ERROR: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list provisions")
		return
	}
	if provisions == nil {
		provisions = []*model.NodeProvision{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"provisions": provisions,
		"count":      len(provisions),
	})
}

// GetProvision 获取单个部署记录
// GET /api/v1/nodes/provisions/{id}
func (h *Handler) GetProvision(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	prov, err := h.store.GetNodeProvision(r.Context(), id)
	if err != nil {
		log.Printf("[node.provision] ERROR: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get provision")
		return
	}
	if prov == nil {
		writeError(w, http.StatusNotFound, "provision not found")
		return
	}
	writeJSON(w, http.StatusOK, prov)
}

// ============================================================================
// 工具函数
// ============================================================================

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

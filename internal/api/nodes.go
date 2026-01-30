// Package api 节点管理接口
package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"agents-admin/internal/model"
	"agents-admin/internal/storage"
)

// ============================================================================
// 请求/响应结构体
// ============================================================================

// HeartbeatRequest 节点心跳请求体
//
// Node Agent 定期发送心跳以维持在线状态。
//
// 字段说明：
//   - NodeID: 节点唯一标识
//   - Status: 节点状态（online、draining 等）
//   - Labels: 节点标签（用于调度匹配）
//   - Capacity: 节点容量信息
type HeartbeatRequest struct {
	NodeID   string                 `json:"node_id"`  // 节点 ID（必填）
	Status   string                 `json:"status"`   // 节点状态
	Labels   map[string]string      `json:"labels"`   // 节点标签
	Capacity map[string]interface{} `json:"capacity"` // 节点容量
}

// UpdateNodeRequest 更新节点的请求体
//
// 字段说明：
//   - Status: 节点状态
//   - Labels: 节点标签
type UpdateNodeRequest struct {
	Status *string            `json:"status,omitempty"` // 节点状态
	Labels *map[string]string `json:"labels,omitempty"` // 节点标签
}

// ============================================================================
// Node 接口处理函数
// ============================================================================

// NodeHeartbeat 处理节点心跳
//
// 路由: POST /api/v1/nodes/heartbeat
//
// 请求体:
//
//	{
//	  "node_id": "node-001",
//	  "status": "online",
//	  "labels": {"os": "linux", "gpu": "true"},
//	  "capacity": {"max_concurrent": 4, "available": 4}
//	}
//
// 响应:
//   - 200 OK: 返回 {"status": "ok"}
//   - 400 Bad Request: 请求体格式错误
//   - 500 Internal Server Error: 服务器内部错误
//
// 业务逻辑：
//   - 节点不存在时创建，存在时更新
//   - 更新最后心跳时间
//   - 调度器根据心跳判断节点是否在线
func (h *Handler) NodeHeartbeat(w http.ResponseWriter, r *http.Request) {
	var req HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[heartbeat] ERROR: invalid request body: %v", err)
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.NodeID == "" {
		log.Printf("[heartbeat] ERROR: node_id is required")
		writeError(w, http.StatusBadRequest, "node_id is required")
		return
	}

	now := time.Now()
	labels, _ := json.Marshal(req.Labels)
	capacity, _ := json.Marshal(req.Capacity)

	log.Printf("[heartbeat] Received from node=%s, status=%s, capacity=%v", req.NodeID, req.Status, req.Capacity)

	// 1. 写入 etcd（实时心跳状态，带 TTL）
	if h.etcdStore != nil {
		hb := &storage.NodeHeartbeat{
			NodeID:   req.NodeID,
			Status:   req.Status,
			Capacity: req.Capacity,
		}
		if err := h.etcdStore.UpdateNodeHeartbeat(r.Context(), hb); err != nil {
			log.Printf("[heartbeat] ERROR: failed to update etcd: %v", err)
			// 继续执行，不阻塞 PostgreSQL 写入
		} else {
			log.Printf("[heartbeat] etcd updated: node=%s", req.NodeID)
		}
	}

	// 2. 写入 PostgreSQL（持久化节点元数据）
	node := &model.Node{
		ID:            req.NodeID,
		Status:        model.NodeStatus(req.Status),
		Labels:        labels,
		Capacity:      capacity,
		LastHeartbeat: &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := h.store.UpsertNode(r.Context(), node); err != nil {
		log.Printf("[heartbeat] ERROR: failed to update postgres: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update node")
		return
	}

	log.Printf("[heartbeat] PostgreSQL updated: node=%s, last_heartbeat=%s", req.NodeID, now.Format(time.RFC3339))
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// NodeResponse 节点响应结构（合并 etcd 心跳状态和 PostgreSQL 元数据）
type NodeResponse struct {
	ID            string                 `json:"id"`
	Status        string                 `json:"status"`
	Labels        map[string]string      `json:"labels,omitempty"`
	Capacity      map[string]interface{} `json:"capacity,omitempty"`
	LastHeartbeat *time.Time             `json:"last_heartbeat,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// ListNodes 列出所有节点
//
// 路由: GET /api/v1/nodes
//
// 架构说明：
//   - 节点元数据从 PostgreSQL 读取
//   - 心跳状态从 etcd 读取（实时，带 TTL）
//   - etcd 中有心跳 = 在线，无心跳 = 离线
func (h *Handler) ListNodes(w http.ResponseWriter, r *http.Request) {
	// 1. 从 PostgreSQL 获取所有节点元数据
	nodes, err := h.store.ListAllNodes(r.Context())
	if err != nil {
		log.Printf("[nodes] ERROR: failed to list nodes from postgres: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list nodes")
		return
	}

	// 2. 从 etcd 获取所有心跳状态
	var heartbeats map[string]*storage.NodeHeartbeat
	if h.etcdStore != nil {
		hbList, err := h.etcdStore.ListNodeHeartbeats(r.Context())
		if err != nil {
			log.Printf("[nodes] WARNING: failed to get heartbeats from etcd: %v", err)
		} else {
			heartbeats = make(map[string]*storage.NodeHeartbeat)
			for _, hb := range hbList {
				heartbeats[hb.NodeID] = hb
			}
		}
	}

	// 3. 合并数据：etcd 有心跳 = online，无心跳 = offline
	result := make([]NodeResponse, 0, len(nodes))
	for _, node := range nodes {
		var labels map[string]string
		var capacity map[string]interface{}
		json.Unmarshal(node.Labels, &labels)
		json.Unmarshal(node.Capacity, &capacity)

		resp := NodeResponse{
			ID:        node.ID,
			Labels:    labels,
			CreatedAt: node.CreatedAt,
			UpdatedAt: node.UpdatedAt,
		}

		// 从 etcd 判断在线状态
		if hb, ok := heartbeats[node.ID]; ok {
			resp.Status = "online"
			resp.LastHeartbeat = &hb.LastHeartbeat
			resp.Capacity = hb.Capacity
			log.Printf("[nodes] %s: online (etcd heartbeat: %s)", node.ID, hb.LastHeartbeat.Format(time.RFC3339))
		} else {
			resp.Status = "offline"
			resp.Capacity = capacity
			// 使用 PostgreSQL 中的历史心跳时间
			resp.LastHeartbeat = node.LastHeartbeat
			log.Printf("[nodes] %s: offline (no etcd heartbeat)", node.ID)
		}

		result = append(result, resp)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"nodes": result, "count": len(result)})
}

// GetNode 获取单个节点详情
//
// 路由: GET /api/v1/nodes/{id}
//
// 路径参数:
//   - id: 节点 ID
//
// 响应:
//   - 200 OK: 返回节点对象
//   - 404 Not Found: 节点不存在
//   - 500 Internal Server Error: 服务器内部错误
func (h *Handler) GetNode(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, node)
}

// GetNodeRuns 获取分配给节点的 Runs
//
// 路由: GET /api/v1/nodes/{id}/runs
//
// 路径参数:
//   - id: 节点 ID
//
// 响应:
//
//	{
//	  "runs": [...],
//	  "count": 2
//	}
//
// 错误响应:
//   - 500 Internal Server Error: 服务器内部错误
//
// 业务说明：
//   - 返回分配给该节点且状态为 running 的 Run
//   - Node Agent 通过此接口获取需要执行的任务
func (h *Handler) GetNodeRuns(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	runs, err := h.store.ListRunsByNode(r.Context(), nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list runs")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"runs": runs, "count": len(runs)})
}

// DeleteNode 删除节点
//
// 路由: DELETE /api/v1/nodes/{id}
//
// 路径参数:
//   - id: 节点 ID
//
// 响应:
//   - 204 No Content: 删除成功
//   - 400 Bad Request: 节点有正在执行的任务
//   - 404 Not Found: 节点不存在
//   - 500 Internal Server Error: 服务器内部错误
//
// 业务规则：
//   - 如果节点有正在执行的 Run，不允许删除
//   - 建议先将节点状态设为 draining，等待任务完成后再删除
func (h *Handler) DeleteNode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// 检查节点是否存在
	node, err := h.store.GetNode(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get node")
		return
	}
	if node == nil {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}

	// 检查是否有正在执行的任务
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
	w.WriteHeader(http.StatusNoContent)
}

// UpdateNode 更新节点信息
//
// 路由: PATCH /api/v1/nodes/{id}
//
// 路径参数:
//   - id: 节点 ID
//
// 请求体:
//
//	{
//	  "status": "draining",
//	  "labels": {"gpu": "false"}
//	}
//
// 响应:
//   - 200 OK: 返回更新后的节点对象
//   - 400 Bad Request: 请求体格式错误
//   - 404 Not Found: 节点不存在
//   - 500 Internal Server Error: 服务器内部错误
//
// 使用场景：
//   - 管理员手动将节点设为 draining/maintenance 状态
//   - 更新节点标签
func (h *Handler) UpdateNode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// 检查节点是否存在
	node, err := h.store.GetNode(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get node")
		return
	}
	if node == nil {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}

	var req UpdateNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// 更新状态
	if req.Status != nil {
		node.Status = model.NodeStatus(*req.Status)
	}

	// 更新标签
	if req.Labels != nil {
		labels, _ := json.Marshal(*req.Labels)
		node.Labels = labels
	}

	node.UpdatedAt = time.Now()

	if err := h.store.UpsertNode(r.Context(), node); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update node")
		return
	}
	writeJSON(w, http.StatusOK, node)
}

// GetNodeEnvConfig 获取节点环境配置
//
// 路由: GET /api/v1/nodes/{id}/env-config
func (h *Handler) GetNodeEnvConfig(w http.ResponseWriter, r *http.Request) {
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

	// 从 capacity 中提取 env_config
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

// UpdateNodeEnvConfig 更新节点环境配置
//
// 路由: PUT /api/v1/nodes/{id}/env-config
func (h *Handler) UpdateNodeEnvConfig(w http.ResponseWriter, r *http.Request) {
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

	// 验证配置
	if err := envConfig.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 更新 capacity 中的 env_config
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

	log.Printf("[API] Node %s env config updated", id)
	writeJSON(w, http.StatusOK, envConfig)
}

// TestNodeProxy 测试节点代理连通性
//
// 路由: POST /api/v1/nodes/{id}/env-config/test-proxy
func (h *Handler) TestNodeProxy(w http.ResponseWriter, r *http.Request) {
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

	// 从 capacity 中提取 env_config
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

	// 测试代理连通性
	proxy := envConfig.Proxy
	addr := net.JoinHostPort(proxy.Host, fmt.Sprintf("%d", proxy.Port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		log.Printf("[API] Proxy test failed for node %s: %v", id, err)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("connection failed: %v", err),
		})
		return
	}
	conn.Close()

	log.Printf("[API] Proxy test success for node %s", id)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "proxy is reachable",
	})
}

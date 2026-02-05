// Package integration 节点心跳集成测试
//
// 测试范围：节点注册、心跳更新、在线/离线状态判断（通过 Redis 缓存）
//
// 测试用例（对应文档 TC-NODE-HEARTBEAT-001 ~ TC-NODE-HEARTBEAT-006）：
//   - TC-NODE-HEARTBEAT-001: 首次心跳（节点注册）
//   - TC-NODE-HEARTBEAT-002: 心跳续期
//   - TC-NODE-HEARTBEAT-003: 更新节点信息
//   - TC-NODE-HEARTBEAT-004: 心跳超时（节点离线）— 仅验证缓存过期逻辑
//   - TC-NODE-HEARTBEAT-005: 缺少 node_id
//   - TC-NODE-HEARTBEAT-006: 带容量信息心跳
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"agents-admin/internal/apiserver/node"
	"agents-admin/internal/apiserver/server"
	"agents-admin/internal/config"
	"agents-admin/internal/shared/cache"
	"agents-admin/internal/shared/infra"
	"agents-admin/internal/shared/model"
	"agents-admin/internal/shared/storage"
)

var (
	testStore   *storage.PostgresStore
	testRedis   *infra.RedisInfra
	testHandler *server.Handler
	testServer  *httptest.Server
	idSeq       uint32
)

func uniqueID(prefix string) string {
	seq := atomic.AddUint32(&idSeq, 1) % 1000
	return fmt.Sprintf("%s-%s-%03d", prefix, time.Now().Format("150405"), seq)
}

func TestMain(m *testing.M) {
	os.Setenv("APP_ENV", "test")
	cfg := config.Load()

	var err error
	testStore, err = storage.NewPostgresStore(cfg.DatabaseURL)
	if err != nil {
		// 无法连接数据库，跳过集成测试
		os.Exit(0)
	}

	testRedis, err = infra.NewRedisInfra(cfg.RedisURL)
	if err != nil {
		testRedis = nil
	}

	var cacheStore storage.CacheStore
	if testRedis != nil {
		cacheStore = testRedis
	} else {
		cacheStore = storage.NewNoOpCacheStore()
	}

	testHandler = server.NewHandler(testStore, cacheStore)
	testServer = httptest.NewServer(testHandler.Router())

	code := m.Run()

	testServer.Close()
	if testRedis != nil {
		testRedis.Close()
	}
	testStore.Close()
	os.Exit(code)
}

// sendHeartbeat 发送心跳请求到测试服务器
func sendHeartbeat(t *testing.T, body map[string]interface{}) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body failed: %v", err)
	}
	resp, err := http.Post(testServer.URL+"/api/v1/nodes/heartbeat", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("POST heartbeat failed: %v", err)
	}
	return resp
}

// listNodes 获取节点列表
func listNodes(t *testing.T) map[string]interface{} {
	t.Helper()
	resp, err := http.Get(testServer.URL + "/api/v1/nodes")
	if err != nil {
		t.Fatalf("GET nodes failed: %v", err)
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result
}

// cleanupNode 清理测试节点
func cleanupNode(t *testing.T, nodeID string) {
	t.Helper()
	ctx := context.Background()
	testStore.DeleteNode(ctx, nodeID)
	// 清理 Redis 缓存
	if testRedis != nil {
		testRedis.Client().Del(ctx, cache.KeyNodeHeartbeat+nodeID).Err()
		testRedis.Client().SRem(ctx, cache.KeyOnlineNodes, nodeID).Err()
	}
}

// getNode 获取单个节点详情
func getNode(t *testing.T, nodeID string) map[string]interface{} {
	t.Helper()
	resp, err := http.Get(testServer.URL + "/api/v1/nodes/" + nodeID)
	if err != nil {
		t.Fatalf("GET node failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result
}

// findNodeInList 在节点列表中查找指定节点
func findNodeInList(t *testing.T, nodeID string) map[string]interface{} {
	t.Helper()
	result := listNodes(t)
	nodesRaw, ok := result["nodes"].([]interface{})
	if !ok {
		return nil
	}
	for _, n := range nodesRaw {
		nm, ok := n.(map[string]interface{})
		if !ok {
			continue
		}
		if nm["id"] == nodeID {
			return nm
		}
	}
	return nil
}

// TC-NODE-HEARTBEAT-001: 首次心跳（节点注册）
func TestNodeHeartbeat_FirstHeartbeat(t *testing.T) {
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	nodeID := uniqueID("hb-first")
	defer cleanupNode(t, nodeID)

	// 发送首次心跳
	resp := sendHeartbeat(t, map[string]interface{}{
		"node_id": nodeID,
		"status":  "online",
		"labels":  map[string]string{"env": "test"},
		"capacity": map[string]interface{}{
			"max_concurrent": 4,
			"available":      4,
		},
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", body["status"])
	}

	// 验证 Redis 缓存
	ctx := context.Background()
	hb, err := testRedis.GetNodeHeartbeat(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNodeHeartbeat failed: %v", err)
	}
	if hb == nil {
		t.Fatal("expected heartbeat in Redis, got nil")
	}
	if hb.Status != "online" {
		t.Errorf("expected status=online in cache, got %s", hb.Status)
	}

	// 验证 Redis TTL
	ttl := testRedis.Client().TTL(ctx, cache.KeyNodeHeartbeat+nodeID).Val()
	if ttl <= 0 || ttl > cache.TTLNodeHeartbeat {
		t.Errorf("expected TTL in (0, %v], got %v", cache.TTLNodeHeartbeat, ttl)
	}

	// 验证 PostgreSQL
	dbNode, err := testStore.GetNode(ctx, nodeID)
	if err != nil || dbNode == nil {
		t.Fatalf("node not found in PostgreSQL: %v", err)
	}
	if string(dbNode.Status) != "online" {
		t.Errorf("expected DB status=online, got %s", dbNode.Status)
	}

	// 验证 List 接口中状态为 online
	found := findNodeInList(t, nodeID)
	if found == nil {
		t.Fatal("node not found in list")
	}
	if found["status"] != "online" {
		t.Errorf("expected list status=online, got %v", found["status"])
	}
}

// TC-NODE-HEARTBEAT-002: 心跳续期
func TestNodeHeartbeat_Renewal(t *testing.T) {
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	nodeID := uniqueID("hb-renew")
	defer cleanupNode(t, nodeID)

	ctx := context.Background()

	// 首次心跳
	resp := sendHeartbeat(t, map[string]interface{}{
		"node_id": nodeID,
		"status":  "online",
	})
	resp.Body.Close()

	// 记录首次心跳时间
	hb1, _ := testRedis.GetNodeHeartbeat(ctx, nodeID)
	if hb1 == nil {
		t.Fatal("first heartbeat not found")
	}
	firstUpdatedAt := hb1.UpdatedAt

	// 记录首次 DB 心跳时间
	dbNode1, _ := testStore.GetNode(ctx, nodeID)
	if dbNode1 == nil || dbNode1.LastHeartbeat == nil {
		t.Fatal("DB node or last_heartbeat is nil after first heartbeat")
	}
	firstDBHeartbeat := *dbNode1.LastHeartbeat

	// 等待一小段时间再发心跳
	time.Sleep(100 * time.Millisecond)

	// 续期心跳
	resp = sendHeartbeat(t, map[string]interface{}{
		"node_id": nodeID,
		"status":  "online",
	})
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for renewal, got %d", resp.StatusCode)
	}

	// 验证 TTL 已重置
	ttl := testRedis.Client().TTL(ctx, cache.KeyNodeHeartbeat+nodeID).Val()
	if ttl <= 25*time.Second {
		t.Errorf("expected TTL > 25s after renewal, got %v", ttl)
	}

	// 验证 updated_at 已更新
	hb2, _ := testRedis.GetNodeHeartbeat(ctx, nodeID)
	if hb2 == nil {
		t.Fatal("renewed heartbeat not found")
	}
	if !hb2.UpdatedAt.After(firstUpdatedAt) {
		t.Errorf("expected updated_at to advance, first=%v renewed=%v", firstUpdatedAt, hb2.UpdatedAt)
	}

	// 验证 DB last_heartbeat 实际更新了（而不仅仅是非 nil）
	dbNode2, _ := testStore.GetNode(ctx, nodeID)
	if dbNode2 == nil || dbNode2.LastHeartbeat == nil {
		t.Fatal("DB node or last_heartbeat is nil after renewal")
	}
	if !dbNode2.LastHeartbeat.After(firstDBHeartbeat) {
		t.Errorf("expected DB last_heartbeat to advance, first=%v renewed=%v", firstDBHeartbeat, *dbNode2.LastHeartbeat)
	}
}

// TC-NODE-HEARTBEAT-003: 更新节点信息
func TestNodeHeartbeat_UpdateLabels(t *testing.T) {
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	nodeID := uniqueID("hb-label")
	defer cleanupNode(t, nodeID)

	ctx := context.Background()

	// 首次心跳，labels = {"env": "test"}
	resp := sendHeartbeat(t, map[string]interface{}{
		"node_id": nodeID,
		"labels":  map[string]string{"env": "test"},
	})
	resp.Body.Close()

	dbNode, _ := testStore.GetNode(ctx, nodeID)
	if dbNode == nil {
		t.Fatal("node not created")
	}

	// 更新 labels
	resp = sendHeartbeat(t, map[string]interface{}{
		"node_id": nodeID,
		"labels":  map[string]string{"env": "prod", "gpu": "true"},
	})
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// 验证 DB labels 已更新
	dbNode, _ = testStore.GetNode(ctx, nodeID)
	if dbNode == nil {
		t.Fatal("node not found after update")
	}
	var labels map[string]string
	json.Unmarshal(dbNode.Labels, &labels)
	if labels["env"] != "prod" || labels["gpu"] != "true" {
		t.Errorf("expected labels={env:prod,gpu:true}, got %v", labels)
	}
}

// TC-NODE-HEARTBEAT-004: 心跳超时（节点离线）
// 通过删除 Redis key 模拟 TTL 过期，验证 List 和 Get API 都返回 offline
func TestNodeHeartbeat_OfflineDetection(t *testing.T) {
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	nodeID := uniqueID("hb-offline")
	defer cleanupNode(t, nodeID)

	ctx := context.Background()

	// 发送心跳
	resp := sendHeartbeat(t, map[string]interface{}{
		"node_id": nodeID,
		"status":  "online",
	})
	resp.Body.Close()

	// 验证节点在线（List API）
	found := findNodeInList(t, nodeID)
	if found == nil || found["status"] != "online" {
		t.Fatal("expected node to be online in list")
	}

	// 验证节点在线（Get API）
	getResp := getNode(t, nodeID)
	if getResp == nil || getResp["status"] != "online" {
		t.Fatalf("expected node to be online via Get API, got %v", getResp)
	}

	// 手动删除 Redis 缓存 key（模拟 TTL 过期）
	testRedis.Client().Del(ctx, cache.KeyNodeHeartbeat+nodeID)

	// 验证 List API 返回 offline
	found = findNodeInList(t, nodeID)
	if found == nil {
		t.Fatal("node not found in list")
	}
	if found["status"] != "offline" {
		t.Errorf("expected offline in List after cache expiry, got %v", found["status"])
	}

	// 验证 Get API 也返回 offline
	getResp = getNode(t, nodeID)
	if getResp == nil {
		t.Fatal("node not found via Get API")
	}
	if getResp["status"] != "offline" {
		t.Errorf("expected offline in Get API after cache expiry, got %v", getResp["status"])
	}

	// 验证 DB 记录仍存在，且 status 仍为 "online"（设计如此：DB 不自动同步缓存过期）
	dbNode, _ := testStore.GetNode(ctx, nodeID)
	if dbNode == nil {
		t.Fatal("expected DB record to persist after going offline")
	}
	if string(dbNode.Status) != "online" {
		t.Errorf("DB status should remain 'online' (not auto-synced), got %s", dbNode.Status)
	}
}

// TC-NODE-HEARTBEAT-005: 缺少 node_id
func TestNodeHeartbeat_MissingNodeID(t *testing.T) {
	resp := sendHeartbeat(t, map[string]interface{}{
		"status": "online",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] == "" {
		t.Error("expected error message in response")
	}
}

// TC-NODE-HEARTBEAT-006: 带容量信息心跳
func TestNodeHeartbeat_WithCapacity(t *testing.T) {
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	nodeID := uniqueID("hb-cap")
	defer cleanupNode(t, nodeID)

	ctx := context.Background()

	resp := sendHeartbeat(t, map[string]interface{}{
		"node_id": nodeID,
		"capacity": map[string]interface{}{
			"max_concurrent": 8,
			"available":      6,
		},
	})

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 验证 DB capacity
	dbNode, _ := testStore.GetNode(ctx, nodeID)
	if dbNode == nil {
		t.Fatal("node not found in DB")
	}
	var dbCap map[string]interface{}
	json.Unmarshal(dbNode.Capacity, &dbCap)
	if dbCap["max_concurrent"] != float64(8) || dbCap["available"] != float64(6) {
		t.Errorf("expected DB capacity={max_concurrent:8,available:6}, got %v", dbCap)
	}

	// 验证 Redis 缓存 capacity
	hb, _ := testRedis.GetNodeHeartbeat(ctx, nodeID)
	if hb == nil {
		t.Fatal("heartbeat not found in cache")
	}
	if hb.Capacity["max_concurrent"] != 8 || hb.Capacity["available"] != 6 {
		t.Errorf("expected cache capacity={max_concurrent:8,available:6}, got %v", hb.Capacity)
	}
}

// TC-NODE-HEARTBEAT-007: Manager.ListOnlineNodes 使用 Redis 缓存
func TestNodeManager_ListOnlineNodes(t *testing.T) {
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	nodeOnline := uniqueID("mgr-on")
	nodeOffline := uniqueID("mgr-off")
	defer cleanupNode(t, nodeOnline)
	defer cleanupNode(t, nodeOffline)

	ctx := context.Background()

	// 创建两个节点
	for _, nid := range []string{nodeOnline, nodeOffline} {
		resp := sendHeartbeat(t, map[string]interface{}{
			"node_id": nid,
			"status":  "online",
		})
		resp.Body.Close()
	}

	// 模拟 nodeOffline 离线（删除缓存）
	testRedis.Client().Del(ctx, cache.KeyNodeHeartbeat+nodeOffline)

	// 使用 Manager 获取在线节点
	mgr := node.NewManager(testStore)
	mgr.SetNodeCache(testRedis)

	onlineNodes, err := mgr.ListOnlineNodes(ctx)
	if err != nil {
		t.Fatalf("ListOnlineNodes failed: %v", err)
	}

	// 验证 nodeOnline 在线
	foundOnline := false
	foundOffline := false
	for _, n := range onlineNodes {
		if n.ID == nodeOnline {
			foundOnline = true
		}
		if n.ID == nodeOffline {
			foundOffline = true
		}
	}

	if !foundOnline {
		t.Errorf("expected %s to be in online list", nodeOnline)
	}
	if foundOffline {
		t.Errorf("expected %s to NOT be in online list", nodeOffline)
	}
}

// TC-NODE-HEARTBEAT-009: 删除节点时清理缓存
func TestNodeHeartbeat_DeleteClearsCache(t *testing.T) {
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	nodeID := uniqueID("hb-delcache")
	defer cleanupNode(t, nodeID)

	ctx := context.Background()

	// 发送心跳
	resp := sendHeartbeat(t, map[string]interface{}{
		"node_id": nodeID,
		"status":  "online",
	})
	resp.Body.Close()

	// 验证缓存存在
	hb, _ := testRedis.GetNodeHeartbeat(ctx, nodeID)
	if hb == nil {
		t.Fatal("expected cache entry before delete")
	}

	// 删除节点
	req, _ := http.NewRequest("DELETE", testServer.URL+"/api/v1/nodes/"+nodeID, nil)
	delResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE failed: %v", err)
	}
	delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", delResp.StatusCode)
	}

	// 验证缓存已被清理
	hb, _ = testRedis.GetNodeHeartbeat(ctx, nodeID)
	if hb != nil {
		t.Error("expected cache entry to be cleared after node deletion")
	}
}

// TC-NODE-HEARTBEAT-010: 心跳不覆盖管理员设置的行政状态
func TestNodeHeartbeat_PreservesAdminStatus(t *testing.T) {
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	nodeID := uniqueID("hb-admstat")
	defer cleanupNode(t, nodeID)

	ctx := context.Background()

	// 1. 首次心跳注册节点
	resp := sendHeartbeat(t, map[string]interface{}{
		"node_id": nodeID,
		"status":  "online",
	})
	resp.Body.Close()

	// 验证 DB 状态为 online
	dbNode, _ := testStore.GetNode(ctx, nodeID)
	if dbNode == nil || string(dbNode.Status) != "online" {
		t.Fatal("expected DB status=online after first heartbeat")
	}

	// 2. 管理员设置为 draining
	patchBody, _ := json.Marshal(map[string]interface{}{"status": "draining"})
	patchReq, _ := http.NewRequest("PATCH", testServer.URL+"/api/v1/nodes/"+nodeID, bytes.NewReader(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchResp, err := http.DefaultClient.Do(patchReq)
	if err != nil {
		t.Fatalf("PATCH failed: %v", err)
	}
	patchResp.Body.Close()
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for PATCH, got %d", patchResp.StatusCode)
	}

	// 验证 DB 状态为 draining
	dbNode, _ = testStore.GetNode(ctx, nodeID)
	if string(dbNode.Status) != "draining" {
		t.Fatalf("expected DB status=draining, got %s", dbNode.Status)
	}

	// 3. 节点继续发送心跳（status=online）
	resp = sendHeartbeat(t, map[string]interface{}{
		"node_id": nodeID,
		"status":  "online",
	})
	resp.Body.Close()

	// 验证 DB 状态仍然是 draining（心跳不应覆盖行政状态）
	dbNode, _ = testStore.GetNode(ctx, nodeID)
	if string(dbNode.Status) != "draining" {
		t.Errorf("heartbeat should NOT overwrite admin status, DB status=%s, want draining", dbNode.Status)
	}

	// 验证 API 返回 draining（行政状态优先）
	getResp := getNode(t, nodeID)
	if getResp == nil || getResp["status"] != "draining" {
		t.Errorf("API should return admin status 'draining', got %v", getResp)
	}
}

// TC-NODE-HEARTBEAT-011: 行政状态变更清理缓存
func TestNodeHeartbeat_AdminStatusClearsCache(t *testing.T) {
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	nodeID := uniqueID("hb-admclr")
	defer cleanupNode(t, nodeID)

	ctx := context.Background()

	// 发送心跳
	resp := sendHeartbeat(t, map[string]interface{}{
		"node_id": nodeID,
		"status":  "online",
	})
	resp.Body.Close()

	// 验证缓存存在
	hb, _ := testRedis.GetNodeHeartbeat(ctx, nodeID)
	if hb == nil {
		t.Fatal("expected cache entry before admin status change")
	}

	// 设置为 maintenance
	patchBody, _ := json.Marshal(map[string]interface{}{"status": "maintenance"})
	patchReq, _ := http.NewRequest("PATCH", testServer.URL+"/api/v1/nodes/"+nodeID, bytes.NewReader(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchResp, err := http.DefaultClient.Do(patchReq)
	if err != nil {
		t.Fatalf("PATCH failed: %v", err)
	}
	patchResp.Body.Close()

	// 验证缓存已被清理
	hb, _ = testRedis.GetNodeHeartbeat(ctx, nodeID)
	if hb != nil {
		t.Error("expected cache to be cleared after admin status change to maintenance")
	}

	// 验证 API 返回 maintenance
	getResp := getNode(t, nodeID)
	if getResp == nil || getResp["status"] != "maintenance" {
		t.Errorf("API should return 'maintenance', got %v", getResp)
	}
}

// TC-NODE-HEARTBEAT-008: Manager 缓存不可用时回退到 PostgreSQL
func TestNodeManager_FallbackToPostgres(t *testing.T) {
	nodeID := uniqueID("mgr-fb")
	defer cleanupNode(t, nodeID)

	ctx := context.Background()
	now := time.Now()

	// 直接在 DB 中创建节点（模拟无缓存场景）
	testStore.UpsertNode(ctx, &model.Node{
		ID:            nodeID,
		Status:        model.NodeStatusOnline,
		Labels:        json.RawMessage(`{}`),
		Capacity:      json.RawMessage(`{}`),
		LastHeartbeat: &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	})

	// 创建无缓存的 Manager
	mgr := node.NewManager(testStore)
	// 不设置 nodeCache -> 回退到 PostgreSQL

	onlineNodes, err := mgr.ListOnlineNodes(ctx)
	if err != nil {
		t.Fatalf("ListOnlineNodes failed: %v", err)
	}

	// 应该能从 PostgreSQL 回退获取到节点
	found := false
	for _, n := range onlineNodes {
		if n.ID == nodeID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected %s in fallback online list", nodeID)
	}
}

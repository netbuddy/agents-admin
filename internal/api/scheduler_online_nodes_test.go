package api

import (
	"encoding/json"
	"testing"
	"time"

	"agents-admin/internal/model"
	"agents-admin/internal/storage"
)

func TestMergeOnlineNodes_UsesEtcdHeartbeats(t *testing.T) {
	nodes := []*model.Node{
		{
			ID:       "dev-node-01",
			Status:   model.NodeStatusOnline,
			Labels:   mustJSON(t, map[string]string{"os": "linux"}),
			Capacity: mustJSON(t, map[string]interface{}{"max_concurrent": float64(999)}), // 应被 etcd 覆盖
		},
		{
			ID:       "test-node-001",
			Status:   model.NodeStatusOnline, // PostgreSQL 里可能残留为 online
			Labels:   mustJSON(t, map[string]string{"os": "linux", "gpu": "true"}),
			Capacity: mustJSON(t, map[string]interface{}{"max_concurrent": float64(4)}),
		},
	}

	hbList := []*storage.NodeHeartbeat{
		{
			NodeID:        "dev-node-01",
			Status:        "online",
			LastHeartbeat: time.Now(),
			Capacity:      map[string]interface{}{"max_concurrent": float64(2), "available": float64(2)},
		},
		// 注意：test-node-001 没有心跳 => 应视为离线
	}

	online, hbMap := mergeOnlineNodes(nodes, hbList)
	if len(online) != 1 {
		t.Fatalf("online nodes = %d, want 1", len(online))
	}
	if online[0].ID != "dev-node-01" {
		t.Fatalf("online[0].ID = %s, want dev-node-01", online[0].ID)
	}
	if _, ok := hbMap["dev-node-01"]; !ok {
		t.Fatalf("hbMap missing dev-node-01")
	}

	s := &Scheduler{nodeRunning: map[string]int{}}
	if got := s.getNodeMaxConcurrent(online[0]); got != 2 {
		t.Fatalf("max_concurrent = %d, want 2 (from etcd heartbeat capacity)", got)
	}
}

func TestExtractAgentIDs(t *testing.T) {
	snapshot := json.RawMessage(`{"agent":{"type":"qwen-code","account_id":"acc-1","instance_id":"inst-1"},"prompt":"hi"}`)
	inst, acc := extractAgentIDs(snapshot)
	if inst != "inst-1" {
		t.Fatalf("instanceID = %q, want %q", inst, "inst-1")
	}
	if acc != "acc-1" {
		t.Fatalf("accountID = %q, want %q", acc, "acc-1")
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return b
}

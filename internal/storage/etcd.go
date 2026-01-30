// Package storage etcd 存储实现
package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdStore etcd 存储客户端
type EtcdStore struct {
	client *clientv3.Client
	prefix string
}

// NodeHeartbeat 节点心跳数据（存储在 etcd）
type NodeHeartbeat struct {
	NodeID        string                 `json:"node_id"`
	Status        string                 `json:"status"`
	LastHeartbeat time.Time              `json:"last_heartbeat"`
	Capacity      map[string]interface{} `json:"capacity"`
}

// EtcdConfig etcd 配置
type EtcdConfig struct {
	Endpoints   []string
	DialTimeout time.Duration
	Prefix      string
}

// NewEtcdStore 创建 etcd 存储客户端
func NewEtcdStore(cfg EtcdConfig) (*EtcdStore, error) {
	if cfg.DialTimeout == 0 {
		cfg.DialTimeout = 5 * time.Second
	}
	if cfg.Prefix == "" {
		cfg.Prefix = "/agents"
	}

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: cfg.DialTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etcd: %w", err)
	}

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err = client.Status(ctx, cfg.Endpoints[0])
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("etcd health check failed: %w", err)
	}

	log.Printf("[etcd] Connected to %v", cfg.Endpoints)
	return &EtcdStore{
		client: client,
		prefix: cfg.Prefix,
	}, nil
}

// Close 关闭连接
func (s *EtcdStore) Close() error {
	return s.client.Close()
}

// UpdateNodeHeartbeat 更新节点心跳（写入 etcd）
func (s *EtcdStore) UpdateNodeHeartbeat(ctx context.Context, hb *NodeHeartbeat) error {
	key := fmt.Sprintf("%s/nodes/%s/heartbeat", s.prefix, hb.NodeID)
	hb.LastHeartbeat = time.Now()

	data, err := json.Marshal(hb)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat: %w", err)
	}

	// 使用 30 秒 TTL，心跳过期自动删除
	lease, err := s.client.Grant(ctx, 30)
	if err != nil {
		return fmt.Errorf("failed to create lease: %w", err)
	}

	_, err = s.client.Put(ctx, key, string(data), clientv3.WithLease(lease.ID))
	if err != nil {
		return fmt.Errorf("failed to put heartbeat: %w", err)
	}

	log.Printf("[etcd] Updated heartbeat: %s, status=%s, capacity=%v", hb.NodeID, hb.Status, hb.Capacity)
	return nil
}

// GetNodeHeartbeat 获取节点心跳
func (s *EtcdStore) GetNodeHeartbeat(ctx context.Context, nodeID string) (*NodeHeartbeat, error) {
	key := fmt.Sprintf("%s/nodes/%s/heartbeat", s.prefix, nodeID)

	resp, err := s.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get heartbeat: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, nil // 节点离线（无心跳）
	}

	var hb NodeHeartbeat
	if err := json.Unmarshal(resp.Kvs[0].Value, &hb); err != nil {
		return nil, fmt.Errorf("failed to unmarshal heartbeat: %w", err)
	}

	return &hb, nil
}

// ListNodeHeartbeats 列出所有节点心跳
func (s *EtcdStore) ListNodeHeartbeats(ctx context.Context) ([]*NodeHeartbeat, error) {
	prefix := fmt.Sprintf("%s/nodes/", s.prefix)

	resp, err := s.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to list heartbeats: %w", err)
	}

	var heartbeats []*NodeHeartbeat
	for _, kv := range resp.Kvs {
		var hb NodeHeartbeat
		if err := json.Unmarshal(kv.Value, &hb); err != nil {
			log.Printf("[etcd] Failed to unmarshal heartbeat at %s: %v", string(kv.Key), err)
			continue
		}
		heartbeats = append(heartbeats, &hb)
	}

	return heartbeats, nil
}

// WatchNodeHeartbeats 监听节点心跳变化
func (s *EtcdStore) WatchNodeHeartbeats(ctx context.Context) clientv3.WatchChan {
	prefix := fmt.Sprintf("%s/nodes/", s.prefix)
	return s.client.Watch(ctx, prefix, clientv3.WithPrefix())
}

// IsNodeOnline 检查节点是否在线（etcd 中有心跳记录）
func (s *EtcdStore) IsNodeOnline(ctx context.Context, nodeID string) bool {
	hb, err := s.GetNodeHeartbeat(ctx, nodeID)
	if err != nil {
		log.Printf("[etcd] Error checking node %s online status: %v", nodeID, err)
		return false
	}
	return hb != nil
}

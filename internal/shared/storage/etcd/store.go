// Package etcd etcd 存储实现
package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"agents-admin/internal/shared/storagetypes"
)

// Store etcd 存储客户端
type Store struct {
	client *clientv3.Client
	prefix string
}

// Config etcd 配置
type Config struct {
	Endpoints   []string
	DialTimeout time.Duration
	Prefix      string
}

// NewStore 创建 etcd 存储客户端
func NewStore(cfg Config) (*Store, error) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err = client.Status(ctx, cfg.Endpoints[0])
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("etcd health check failed: %w", err)
	}

	log.Printf("[etcd] Connected to %v", cfg.Endpoints)
	return &Store{
		client: client,
		prefix: cfg.Prefix,
	}, nil
}

// Close 关闭连接
func (s *Store) Close() error {
	return s.client.Close()
}

// Client 返回底层 etcd 客户端
func (s *Store) Client() *clientv3.Client {
	return s.client
}

// Prefix 返回 key 前缀
func (s *Store) Prefix() string {
	return s.prefix
}

// UpdateNodeHeartbeat 更新节点心跳
func (s *Store) UpdateNodeHeartbeat(ctx context.Context, hb *storagetypes.EtcdHeartbeat) error {
	key := fmt.Sprintf("%s/nodes/%s/heartbeat", s.prefix, hb.NodeID)
	hb.LastHeartbeat = time.Now()

	data, err := json.Marshal(hb)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat: %w", err)
	}

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
func (s *Store) GetNodeHeartbeat(ctx context.Context, nodeID string) (*storagetypes.EtcdHeartbeat, error) {
	key := fmt.Sprintf("%s/nodes/%s/heartbeat", s.prefix, nodeID)

	resp, err := s.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get heartbeat: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	var hb storagetypes.EtcdHeartbeat
	if err := json.Unmarshal(resp.Kvs[0].Value, &hb); err != nil {
		return nil, fmt.Errorf("failed to unmarshal heartbeat: %w", err)
	}

	return &hb, nil
}

// ListNodeHeartbeats 列出所有节点心跳
func (s *Store) ListNodeHeartbeats(ctx context.Context) ([]*storagetypes.EtcdHeartbeat, error) {
	prefix := fmt.Sprintf("%s/nodes/", s.prefix)

	resp, err := s.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to list heartbeats: %w", err)
	}

	var heartbeats []*storagetypes.EtcdHeartbeat
	for _, kv := range resp.Kvs {
		var hb storagetypes.EtcdHeartbeat
		if err := json.Unmarshal(kv.Value, &hb); err != nil {
			log.Printf("[etcd] Failed to unmarshal heartbeat at %s: %v", string(kv.Key), err)
			continue
		}
		heartbeats = append(heartbeats, &hb)
	}

	return heartbeats, nil
}

// WatchNodeHeartbeats 监听节点心跳变化
func (s *Store) WatchNodeHeartbeats(ctx context.Context) clientv3.WatchChan {
	prefix := fmt.Sprintf("%s/nodes/", s.prefix)
	return s.client.Watch(ctx, prefix, clientv3.WithPrefix())
}

// IsNodeOnline 检查节点是否在线
func (s *Store) IsNodeOnline(ctx context.Context, nodeID string) bool {
	hb, err := s.GetNodeHeartbeat(ctx, nodeID)
	if err != nil {
		log.Printf("[etcd] Error checking node %s online status: %v", nodeID, err)
		return false
	}
	return hb != nil
}

// 接口验证移到使用 storage 包的地方进行

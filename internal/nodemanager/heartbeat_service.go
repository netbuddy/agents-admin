// Package nodemanager 心跳服务
//
// 定期向 API Server 上报节点状态
package nodemanager

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// HeartbeatService 心跳服务
type HeartbeatService struct {
	config     Config
	httpClient *http.Client
	getRunning func() int // 获取当前运行任务数的回调
}

// NewHeartbeatService 创建心跳服务
func NewHeartbeatService(cfg Config, httpClient *http.Client, getRunning func() int) *HeartbeatService {
	return &HeartbeatService{
		config:     cfg,
		httpClient: httpClient,
		getRunning: getRunning,
	}
}

// Start 启动心跳循环
func (s *HeartbeatService) Start(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	s.send(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.send(ctx)
		}
	}
}

func (s *HeartbeatService) send(ctx context.Context) {
	runningCount := 0
	if s.getRunning != nil {
		runningCount = s.getRunning()
	}

	payload := map[string]interface{}{
		"node_id": s.config.NodeID,
		"status":  "online",
		"labels":  s.config.Labels,
		"capacity": map[string]interface{}{
			"max_concurrent": 2,
			"available":      2 - runningCount,
		},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST",
		s.config.APIServerURL+"/api/v1/nodes/heartbeat",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("[heartbeat] failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Printf("[heartbeat] error status=%d", resp.StatusCode)
	}
}

// Package nodemanager 节点管理器测试
package nodemanager

import (
	"context"
	"testing"
	"time"

	"agents-admin/internal/nodemanager/adapter"
)

// TestNewNodeManager 测试执行器创建
func TestNewNodeManager(t *testing.T) {
	cfg := Config{
		NodeID:       "test-node",
		APIServerURL: "http://localhost:8080",
		WorkspaceDir: "/tmp/test-workspace",
		Labels: map[string]string{
			"os": "linux",
		},
	}

	executor, err := NewNodeManager(cfg)
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}

	if executor == nil {
		t.Fatal("Expected non-nil executor")
	}
	if executor.config.NodeID != "test-node" {
		t.Errorf("NodeID = %v, want test-node", executor.config.NodeID)
	}
	if executor.adapters == nil {
		t.Error("Expected non-nil adapters registry")
	}
	if executor.running == nil {
		t.Error("Expected non-nil running map")
	}
}

// TestRegisterAdapter 测试 Adapter 注册
func TestRegisterAdapter(t *testing.T) {
	executor, err := NewNodeManager(Config{
		NodeID:       "test-node",
		APIServerURL: "http://localhost:8080",
		WorkspaceDir: "/tmp/test-workspace",
	})
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}

	mockAdapter := &mockAdapter{name: "mock-v1"}
	executor.RegisterAdapter(mockAdapter)

	a, ok := executor.adapters.Get("mock-v1")
	if !ok {
		t.Fatal("Expected adapter to be registered")
	}
	if a.Name() != "mock-v1" {
		t.Errorf("Adapter name = %v, want mock-v1", a.Name())
	}
}

// TestCancelRun 测试取消运行
func TestCancelRun(t *testing.T) {
	executor, err := NewNodeManager(Config{
		NodeID:       "test-node",
		APIServerURL: "http://localhost:8080",
		WorkspaceDir: "/tmp/test-workspace",
	})
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}

	// 模拟一个运行中的任务
	ctx, cancel := context.WithCancel(context.Background())
	executor.running["run-123"] = cancel

	// 取消任务
	executor.CancelRun("run-123")

	// 验证 context 已取消
	select {
	case <-ctx.Done():
		// 预期行为
	default:
		t.Error("Expected context to be cancelled")
	}
}

// TestCancelNonExistentRun 测试取消不存在的运行
func TestCancelNonExistentRun(t *testing.T) {
	executor, err := NewNodeManager(Config{
		NodeID:       "test-node",
		APIServerURL: "http://localhost:8080",
		WorkspaceDir: "/tmp/test-workspace",
	})
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}

	// 取消不存在的任务应该不会 panic
	executor.CancelRun("non-existent")
}

// mockAdapter 用于测试的 Mock Adapter
type mockAdapter struct {
	name string
}

func (m *mockAdapter) Name() string { return m.name }
func (m *mockAdapter) Validate(agent *adapter.AgentConfig) error {
	return nil
}
func (m *mockAdapter) BuildCommand(ctx context.Context, spec *adapter.TaskSpec, agent *adapter.AgentConfig) (*adapter.RunConfig, error) {
	return &adapter.RunConfig{
		Command: []string{"echo"},
		Args:    []string{"test"},
	}, nil
}
func (m *mockAdapter) ParseEvent(line string) (*adapter.CanonicalEvent, error) {
	return nil, nil
}
func (m *mockAdapter) CollectArtifacts(ctx context.Context, workDir string) (*adapter.Artifacts, error) {
	return &adapter.Artifacts{}, nil
}

// TestConfigFields 测试配置字段
func TestConfigFields(t *testing.T) {
	cfg := Config{
		NodeID:       "node-001",
		APIServerURL: "http://api.example.com:8080",
		WorkspaceDir: "/var/lib/agent/workspaces",
		Labels: map[string]string{
			"os":     "linux",
			"arch":   "amd64",
			"region": "us-west",
		},
	}

	if cfg.NodeID != "node-001" {
		t.Errorf("NodeID = %v, want node-001", cfg.NodeID)
	}
	if cfg.APIServerURL != "http://api.example.com:8080" {
		t.Errorf("APIServerURL = %v, want http://api.example.com:8080", cfg.APIServerURL)
	}
	if cfg.WorkspaceDir != "/var/lib/agent/workspaces" {
		t.Errorf("WorkspaceDir = %v, want /var/lib/agent/workspaces", cfg.WorkspaceDir)
	}
	if len(cfg.Labels) != 3 {
		t.Errorf("Labels count = %v, want 3", len(cfg.Labels))
	}
}

// TestExecutorHTTPClientTimeout 测试 HTTP 客户端超时设置
func TestExecutorHTTPClientTimeout(t *testing.T) {
	executor, err := NewNodeManager(Config{
		NodeID:       "test-node",
		APIServerURL: "http://localhost:8080",
		WorkspaceDir: "/tmp/test-workspace",
	})
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}

	if executor.httpClient.Timeout != 30*time.Second {
		t.Errorf("HTTP client timeout = %v, want 30s", executor.httpClient.Timeout)
	}
}

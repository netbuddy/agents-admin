package adapter

import (
	"context"
	"testing"
	"time"
)

// TestTaskTypeConstants 测试任务类型常量
func TestTaskTypeConstants(t *testing.T) {
	tests := []struct {
		taskType TaskType
		want     string
	}{
		{TaskTypeGeneral, "general"},
		{TaskTypeDevelopment, "development"},
		{TaskTypeOperation, "operation"},
		{TaskTypeResearch, "research"},
		{TaskTypeAutomation, "automation"},
		{TaskTypeReview, "review"},
	}

	for _, tt := range tests {
		if string(tt.taskType) != tt.want {
			t.Errorf("TaskType = %v, want %v", tt.taskType, tt.want)
		}
	}
}

// TestWorkspaceTypeConstants 测试工作空间类型常量
func TestWorkspaceTypeConstants(t *testing.T) {
	tests := []struct {
		wsType WorkspaceType
		want   string
	}{
		{WorkspaceTypeGit, "git"},
		{WorkspaceTypeLocal, "local"},
		{WorkspaceTypeRemote, "remote"},
		{WorkspaceTypeVolume, "volume"},
	}

	for _, tt := range tests {
		if string(tt.wsType) != tt.want {
			t.Errorf("WorkspaceType = %v, want %v", tt.wsType, tt.want)
		}
	}
}

// TestSecurityPolicyConstants 测试安全策略常量
func TestSecurityPolicyConstants(t *testing.T) {
	tests := []struct {
		policy SecurityPolicy
		want   string
	}{
		{SecurityPolicyStrict, "strict"},
		{SecurityPolicyStandard, "standard"},
		{SecurityPolicyPermissive, "permissive"},
		{SecurityPolicyCustom, "custom"},
	}

	for _, tt := range tests {
		if string(tt.policy) != tt.want {
			t.Errorf("SecurityPolicy = %v, want %v", tt.policy, tt.want)
		}
	}
}

// TestEventTypeConstants 测试事件类型常量
func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		eventType EventType
		want      string
	}{
		// 生命周期事件
		{EventRunStarted, "run_started"},
		{EventRunCompleted, "run_completed"},
		{EventRunFailed, "run_failed"},
		// 输出事件
		{EventMessage, "message"},
		{EventThinking, "thinking"},
		{EventProgress, "progress"},
		// 工具事件
		{EventToolUseStart, "tool_use_start"},
		{EventToolResult, "tool_result"},
		// 文件事件
		{EventFileRead, "file_read"},
		{EventFileWrite, "file_write"},
		{EventFileDelete, "file_delete"},
		// 命令事件
		{EventCommand, "command"},
		{EventCommandOutput, "command_output"},
		// 控制事件
		{EventApprovalRequest, "approval_request"},
		{EventApprovalResponse, "approval_response"},
		{EventCheckpoint, "checkpoint"},
		{EventHeartbeat, "heartbeat"},
		// 错误事件
		{EventError, "error"},
		{EventWarning, "warning"},
	}

	for _, tt := range tests {
		if string(tt.eventType) != tt.want {
			t.Errorf("EventType = %v, want %v", tt.eventType, tt.want)
		}
	}
}

// TestTaskSpecWithWorkspace 测试带 Workspace 的 TaskSpec
func TestTaskSpecWithWorkspace(t *testing.T) {
	// 测试 Git Workspace
	spec := &TaskSpec{
		ID:     "task-001",
		Name:   "Fix login bug",
		Type:   TaskTypeDevelopment,
		Prompt: "修复登录页面的验证问题",
		Workspace: &WorkspaceConfig{
			Type: WorkspaceTypeGit,
			Git: &GitConfig{
				URL:    "https://github.com/example/repo.git",
				Branch: "main",
			},
		},
		Security: SecurityConfig{
			Policy: SecurityPolicyStandard,
		},
	}

	if spec.Workspace == nil {
		t.Fatal("Expected workspace to be set")
	}
	if spec.Workspace.Type != WorkspaceTypeGit {
		t.Errorf("Workspace.Type = %v, want git", spec.Workspace.Type)
	}
	if spec.Workspace.Git.URL != "https://github.com/example/repo.git" {
		t.Errorf("Workspace.Git.URL = %v, want https://github.com/example/repo.git", spec.Workspace.Git.URL)
	}
}

// TestTaskSpecWithoutWorkspace 测试无 Workspace 的 TaskSpec（研究类任务）
func TestTaskSpecWithoutWorkspace(t *testing.T) {
	spec := &TaskSpec{
		ID:     "task-002",
		Name:   "技术调研",
		Type:   TaskTypeResearch,
		Prompt: "分析 Kubernetes 与 Docker Swarm 的优劣",
		Security: SecurityConfig{
			Policy: SecurityPolicyStrict,
		},
	}

	if spec.Workspace != nil {
		t.Error("Expected workspace to be nil for research task")
	}
	if spec.Type != TaskTypeResearch {
		t.Errorf("Type = %v, want research", spec.Type)
	}
}

// TestSecurityConfigWithLimits 测试带资源限制的安全配置
func TestSecurityConfigWithLimits(t *testing.T) {
	config := SecurityConfig{
		Policy:      SecurityPolicyStandard,
		Permissions: []string{"file_read", "file_write"},
		Network: &NetworkConfig{
			Enabled:      true,
			AllowedHosts: []string{"api.github.com", "pypi.org"},
		},
		Limits: &ResourceLimits{
			Timeout:     30 * time.Minute,
			MaxTokens:   100000,
			MaxFileSize: 10 * 1024 * 1024,       // 10MB
			MemoryLimit: 4 * 1024 * 1024 * 1024, // 4GB
		},
	}

	if config.Network == nil || !config.Network.Enabled {
		t.Error("Expected network to be enabled")
	}
	if len(config.Network.AllowedHosts) != 2 {
		t.Errorf("Expected 2 allowed hosts, got %d", len(config.Network.AllowedHosts))
	}
	if config.Limits.Timeout != 30*time.Minute {
		t.Errorf("Timeout = %v, want 30m", config.Limits.Timeout)
	}
}

func TestRegistryRegisterAndGet(t *testing.T) {
	registry := NewRegistry()

	mockAdapter := &mockAdapter{name: "test-adapter"}
	registry.Register(mockAdapter)

	got, ok := registry.Get("test-adapter")
	if !ok {
		t.Fatal("Expected adapter to be found")
	}

	if got.Name() != "test-adapter" {
		t.Errorf("Adapter name = %v, want test-adapter", got.Name())
	}
}

func TestRegistryGetNotFound(t *testing.T) {
	registry := NewRegistry()

	_, ok := registry.Get("nonexistent")
	if ok {
		t.Error("Expected adapter not to be found")
	}
}

func TestRegistryList(t *testing.T) {
	registry := NewRegistry()

	registry.Register(&mockAdapter{name: "adapter-a"})
	registry.Register(&mockAdapter{name: "adapter-b"})

	names := registry.List()
	if len(names) != 2 {
		t.Errorf("Expected 2 adapters, got %d", len(names))
	}

	found := make(map[string]bool)
	for _, name := range names {
		found[name] = true
	}

	if !found["adapter-a"] || !found["adapter-b"] {
		t.Errorf("Expected adapters adapter-a and adapter-b, got %v", names)
	}
}

type mockAdapter struct {
	name string
}

func (m *mockAdapter) Name() string { return m.name }
func (m *mockAdapter) Validate(agent *AgentConfig) error {
	return nil
}
func (m *mockAdapter) BuildCommand(ctx context.Context, spec *TaskSpec, agent *AgentConfig) (*RunConfig, error) {
	return &RunConfig{
		Command: []string{"echo"},
		Args:    []string{"test"},
	}, nil
}
func (m *mockAdapter) ParseEvent(line string) (*CanonicalEvent, error) {
	return nil, nil
}
func (m *mockAdapter) CollectArtifacts(ctx context.Context, workDir string) (*Artifacts, error) {
	return &Artifacts{}, nil
}

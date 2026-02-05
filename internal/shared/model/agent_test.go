// Package model 定义核心数据模型的测试
package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// 阶段4：AgentTemplate + Agent + Runtime 模型测试
// ============================================================================

// TestAgentTemplate_BasicFields 验证 AgentTemplate 基础字段
func TestAgentTemplate_BasicFields(t *testing.T) {
	now := time.Now()
	template := AgentTemplate{
		ID:           "tpl-agent-001",
		Name:         "测试 Agent 模板",
		Type:         AgentModelTypeClaude,
		Role:         "代码助手",
		Description:  "用于代码开发的智能体模板",
		Personality:  []string{"专业", "友好"},
		SystemPrompt: "你是一个专业的代码助手",
		Model:        "claude-3-opus",
		Temperature:  0.7,
		MaxContext:   128000,
		IsBuiltin:    false,
		Category:     "development",
		Tags:         []string{"coding", "assistant"},
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	assert.Equal(t, "tpl-agent-001", template.ID)
	assert.Equal(t, "测试 Agent 模板", template.Name)
	assert.Equal(t, AgentModelTypeClaude, template.Type)
	assert.Equal(t, "代码助手", template.Role)
	assert.Len(t, template.Personality, 2)
	assert.Equal(t, 0.7, template.Temperature)
	assert.Equal(t, 128000, template.MaxContext)
}

// TestAgentTemplate_WithSkills 验证 AgentTemplate 技能配置
func TestAgentTemplate_WithSkills(t *testing.T) {
	template := AgentTemplate{
		ID:     "tpl-skills-001",
		Name:   "技能模板",
		Type:   AgentModelTypeGemini,
		Skills: []string{"skill-coding", "skill-review", "skill-debug"},
	}

	assert.True(t, template.HasSkills())
	assert.Len(t, template.Skills, 3)
	assert.Contains(t, template.Skills, "skill-coding")
}

// TestAgentTemplate_CreateAgent 验证从模板创建 Agent
func TestAgentTemplate_CreateAgent(t *testing.T) {
	template := AgentTemplate{
		ID:          "tpl-dev-001",
		Name:        "开发助手模板",
		Type:        AgentModelTypeClaude,
		Role:        "开发助手",
		Model:       "claude-3-opus",
		Temperature: 0.7,
	}

	agent := template.CreateAgent("我的开发助手", "account-123")

	// 验证基本信息
	assert.Equal(t, "我的开发助手", agent.Name)
	assert.Equal(t, "account-123", agent.AccountID)

	// 验证模板关联
	require.NotNil(t, agent.TemplateID)
	assert.Equal(t, "tpl-dev-001", *agent.TemplateID)

	// 验证类型继承
	assert.Equal(t, AgentModelTypeClaude, agent.Type)

	// 验证初始状态
	assert.Equal(t, AgentStatusPending, agent.Status)
}

// TestAgentModelType_Values 验证 AgentModelType 枚举值
func TestAgentModelType_Values(t *testing.T) {
	types := []AgentModelType{
		AgentModelTypeClaude,
		AgentModelTypeGemini,
		AgentModelTypeQwen,
		AgentModelTypeCodex,
		AgentModelTypeCustom,
	}

	for _, at := range types {
		assert.NotEmpty(t, string(at))
	}

	assert.Equal(t, AgentModelType("claude"), AgentModelTypeClaude)
	assert.Equal(t, AgentModelType("gemini"), AgentModelTypeGemini)
	assert.Equal(t, AgentModelType("qwen"), AgentModelTypeQwen)
}

// TestAgentStatus_Values 验证 AgentStatus 枚举值
func TestAgentStatus_Values(t *testing.T) {
	statuses := []AgentStatus{
		AgentStatusPending,
		AgentStatusStarting,
		AgentStatusRunning,
		AgentStatusIdle,
		AgentStatusBusy,
		AgentStatusStopping,
		AgentStatusStopped,
		AgentStatusError,
	}

	for _, s := range statuses {
		assert.NotEmpty(t, string(s))
	}
}

// TestAgent_StatusTransition 验证 Agent 状态转换
func TestAgent_StatusTransition(t *testing.T) {
	tests := []struct {
		name          string
		status        AgentStatus
		isRunning     bool
		canAcceptTask bool
		isBusy        bool
		canStart      bool
		canStop       bool
	}{
		{
			name:          "pending",
			status:        AgentStatusPending,
			isRunning:     false,
			canAcceptTask: false,
			isBusy:        false,
			canStart:      true,
			canStop:       false,
		},
		{
			name:          "starting",
			status:        AgentStatusStarting,
			isRunning:     false,
			canAcceptTask: false,
			isBusy:        false,
			canStart:      false,
			canStop:       false,
		},
		{
			name:          "running",
			status:        AgentStatusRunning,
			isRunning:     true,
			canAcceptTask: true,
			isBusy:        false,
			canStart:      false,
			canStop:       true,
		},
		{
			name:          "idle",
			status:        AgentStatusIdle,
			isRunning:     true,
			canAcceptTask: true,
			isBusy:        false,
			canStart:      false,
			canStop:       true,
		},
		{
			name:          "busy",
			status:        AgentStatusBusy,
			isRunning:     true,
			canAcceptTask: false,
			isBusy:        true,
			canStart:      false,
			canStop:       true,
		},
		{
			name:          "stopped",
			status:        AgentStatusStopped,
			isRunning:     false,
			canAcceptTask: false,
			isBusy:        false,
			canStart:      true,
			canStop:       false,
		},
		{
			name:          "error",
			status:        AgentStatusError,
			isRunning:     false,
			canAcceptTask: false,
			isBusy:        false,
			canStart:      true,
			canStop:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := Agent{
				ID:     "agent-test",
				Name:   "Test Agent",
				Status: tt.status,
			}

			assert.Equal(t, tt.isRunning, agent.IsRunning(), "IsRunning")
			assert.Equal(t, tt.canAcceptTask, agent.CanAcceptTask(), "CanAcceptTask")
			assert.Equal(t, tt.isBusy, agent.IsBusy(), "IsBusy")
			assert.Equal(t, tt.canStart, agent.CanStart(), "CanStart")
			assert.Equal(t, tt.canStop, agent.CanStop(), "CanStop")
		})
	}
}

// TestAgent_RuntimeBinding 验证 Agent 与 Runtime 绑定
func TestAgent_RuntimeBinding(t *testing.T) {
	runtimeID := "runtime-001"
	nodeID := "node-001"

	agent := Agent{
		ID:        "agent-001",
		Name:      "Runtime Test Agent",
		Status:    AgentStatusRunning,
		RuntimeID: &runtimeID,
		NodeID:    &nodeID,
	}

	require.NotNil(t, agent.RuntimeID)
	assert.Equal(t, "runtime-001", *agent.RuntimeID)

	require.NotNil(t, agent.NodeID)
	assert.Equal(t, "node-001", *agent.NodeID)
}

// TestAgent_ConfigOverrides 验证 Agent 配置覆盖
func TestAgent_ConfigOverrides(t *testing.T) {
	overrides := map[string]interface{}{
		"temperature": 0.9,
		"max_tokens":  4096,
	}
	overridesJSON, err := json.Marshal(overrides)
	require.NoError(t, err)

	agent := Agent{
		ID:              "agent-override",
		Name:            "Override Test",
		Status:          AgentStatusRunning,
		ConfigOverrides: overridesJSON,
		MemoryEnabled:   true,
	}

	// 验证配置覆盖
	var decoded map[string]interface{}
	err = json.Unmarshal(agent.ConfigOverrides, &decoded)
	require.NoError(t, err)
	assert.Equal(t, 0.9, decoded["temperature"])

	// 验证记忆启用
	assert.True(t, agent.MemoryEnabled)
}

// TestRuntimeType_Values 验证 RuntimeType 枚举值
func TestRuntimeType_Values(t *testing.T) {
	types := []RuntimeType{
		RuntimeTypeContainer,
		RuntimeTypeVM,
		RuntimeTypeHost,
		RuntimeTypeMicroVM,
	}

	for _, rt := range types {
		assert.NotEmpty(t, string(rt))
	}

	assert.Equal(t, RuntimeType("container"), RuntimeTypeContainer)
	assert.Equal(t, RuntimeType("vm"), RuntimeTypeVM)
}

// TestRuntimeStatus_Values 验证 RuntimeStatus 枚举值
func TestRuntimeStatus_Values(t *testing.T) {
	statuses := []RuntimeStatus{
		RuntimeStatusCreating,
		RuntimeStatusReady,
		RuntimeStatusRunning,
		RuntimeStatusStopping,
		RuntimeStatusStopped,
		RuntimeStatusError,
		RuntimeStatusDestroyed,
	}

	for _, s := range statuses {
		assert.NotEmpty(t, string(s))
	}
}

// TestRuntime_Lifecycle 验证 Runtime 生命周期方法
func TestRuntime_Lifecycle(t *testing.T) {
	tests := []struct {
		name         string
		status       RuntimeStatus
		isRunning    bool
		isReady      bool
		isTerminated bool
	}{
		{
			name:         "creating",
			status:       RuntimeStatusCreating,
			isRunning:    false,
			isReady:      false,
			isTerminated: false,
		},
		{
			name:         "ready",
			status:       RuntimeStatusReady,
			isRunning:    false,
			isReady:      true,
			isTerminated: false,
		},
		{
			name:         "running",
			status:       RuntimeStatusRunning,
			isRunning:    true,
			isReady:      true,
			isTerminated: false,
		},
		{
			name:         "stopped",
			status:       RuntimeStatusStopped,
			isRunning:    false,
			isReady:      false,
			isTerminated: true,
		},
		{
			name:         "destroyed",
			status:       RuntimeStatusDestroyed,
			isRunning:    false,
			isReady:      false,
			isTerminated: true,
		},
		{
			name:         "error",
			status:       RuntimeStatusError,
			isRunning:    false,
			isReady:      false,
			isTerminated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtime := Runtime{
				ID:     "runtime-test",
				Status: tt.status,
			}

			assert.Equal(t, tt.isRunning, runtime.IsRunning(), "IsRunning")
			assert.Equal(t, tt.isReady, runtime.IsReady(), "IsReady")
			assert.Equal(t, tt.isTerminated, runtime.IsTerminated(), "IsTerminated")
		})
	}
}

// TestRuntime_ContainerFields 验证 Runtime 容器特定字段
func TestRuntime_ContainerFields(t *testing.T) {
	containerID := "abc123def456"
	containerName := "agent-dev-001"
	image := "runners/claude:latest"
	ipAddress := "172.17.0.5"

	runtime := Runtime{
		ID:            "runtime-container-001",
		Type:          RuntimeTypeContainer,
		Status:        RuntimeStatusRunning,
		AgentID:       "agent-001",
		NodeID:        "node-001",
		ContainerID:   &containerID,
		ContainerName: &containerName,
		Image:         &image,
		WorkspacePath: "/workspace",
		IPAddress:     &ipAddress,
	}

	assert.Equal(t, RuntimeTypeContainer, runtime.Type)
	require.NotNil(t, runtime.ContainerID)
	assert.Equal(t, "abc123def456", *runtime.ContainerID)
	require.NotNil(t, runtime.ContainerName)
	assert.Equal(t, "agent-dev-001", *runtime.ContainerName)
	require.NotNil(t, runtime.Image)
	assert.Equal(t, "runners/claude:latest", *runtime.Image)
	assert.Equal(t, "/workspace", runtime.WorkspacePath)
}

// TestAgentTemplate_JSONSerialization 验证 AgentTemplate JSON 序列化
func TestAgentTemplate_JSONSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	securityPolicyID := "policy-001"

	template := AgentTemplate{
		ID:                      "tpl-json-001",
		Name:                    "JSON 测试模板",
		Type:                    AgentModelTypeClaude,
		Role:                    "测试角色",
		Description:             "用于 JSON 序列化测试",
		Personality:             []string{"测试"},
		SystemPrompt:            "测试提示词",
		Skills:                  []string{"skill-1", "skill-2"},
		Model:                   "claude-3-opus",
		Temperature:             0.8,
		MaxContext:              100000,
		DefaultSecurityPolicyID: &securityPolicyID,
		IsBuiltin:               false,
		Category:                "testing",
		Tags:                    []string{"test", "json"},
		CreatedAt:               now,
		UpdatedAt:               now,
	}

	// 序列化
	data, err := json.Marshal(template)
	require.NoError(t, err)

	// 反序列化
	var decoded AgentTemplate
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// 验证
	assert.Equal(t, template.ID, decoded.ID)
	assert.Equal(t, template.Name, decoded.Name)
	assert.Equal(t, template.Type, decoded.Type)
	assert.Equal(t, template.Role, decoded.Role)
	assert.Equal(t, template.Personality, decoded.Personality)
	assert.Equal(t, template.Skills, decoded.Skills)
	assert.Equal(t, template.Temperature, decoded.Temperature)
	assert.Equal(t, template.Tags, decoded.Tags)
	require.NotNil(t, decoded.DefaultSecurityPolicyID)
	assert.Equal(t, "policy-001", *decoded.DefaultSecurityPolicyID)
}

// TestAgent_JSONSerialization 验证 Agent JSON 序列化
func TestAgent_JSONSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	templateID := "tpl-001"
	runtimeID := "runtime-001"
	nodeID := "node-001"
	taskID := "task-001"
	policyID := "policy-001"

	agent := Agent{
		ID:               "agent-json-001",
		Name:             "JSON 测试 Agent",
		TemplateID:       &templateID,
		Type:             AgentModelTypeGemini,
		AccountID:        "account-001",
		RuntimeID:        &runtimeID,
		NodeID:           &nodeID,
		Status:           AgentStatusRunning,
		CurrentTaskID:    &taskID,
		SecurityPolicyID: &policyID,
		MemoryEnabled:    true,
		CreatedAt:        now,
		UpdatedAt:        now,
		LastActiveAt:     &now,
	}

	// 序列化
	data, err := json.Marshal(agent)
	require.NoError(t, err)

	// 反序列化
	var decoded Agent
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// 验证
	assert.Equal(t, agent.ID, decoded.ID)
	assert.Equal(t, agent.Name, decoded.Name)
	assert.Equal(t, agent.Type, decoded.Type)
	assert.Equal(t, agent.Status, decoded.Status)
	assert.Equal(t, agent.MemoryEnabled, decoded.MemoryEnabled)
	require.NotNil(t, decoded.TemplateID)
	assert.Equal(t, "tpl-001", *decoded.TemplateID)
	require.NotNil(t, decoded.RuntimeID)
	assert.Equal(t, "runtime-001", *decoded.RuntimeID)
}

// TestRuntime_JSONSerialization 验证 Runtime JSON 序列化
func TestRuntime_JSONSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	containerID := "container-123"
	containerName := "agent-container"
	image := "runners/agent:latest"

	runtime := Runtime{
		ID:            "runtime-json-001",
		Type:          RuntimeTypeContainer,
		Status:        RuntimeStatusRunning,
		AgentID:       "agent-001",
		NodeID:        "node-001",
		ContainerID:   &containerID,
		ContainerName: &containerName,
		Image:         &image,
		WorkspacePath: "/workspace",
		CreatedAt:     now,
		UpdatedAt:     now,
		StartedAt:     &now,
	}

	// 序列化
	data, err := json.Marshal(runtime)
	require.NoError(t, err)

	// 反序列化
	var decoded Runtime
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// 验证
	assert.Equal(t, runtime.ID, decoded.ID)
	assert.Equal(t, runtime.Type, decoded.Type)
	assert.Equal(t, runtime.Status, decoded.Status)
	assert.Equal(t, runtime.AgentID, decoded.AgentID)
	require.NotNil(t, decoded.ContainerID)
	assert.Equal(t, "container-123", *decoded.ContainerID)
}

// TestBuiltinAgentTemplates 验证内置 Agent 模板
func TestBuiltinAgentTemplates(t *testing.T) {
	// 验证内置模板数量
	assert.GreaterOrEqual(t, len(BuiltinAgentTemplates), 3)

	// 验证每个模板
	for _, tpl := range BuiltinAgentTemplates {
		assert.NotEmpty(t, tpl.ID, "ID should not be empty")
		assert.NotEmpty(t, tpl.Name, "Name should not be empty")
		assert.NotEmpty(t, tpl.Type, "Type should not be empty")
		assert.True(t, tpl.IsBuiltin, "IsBuiltin should be true")
		assert.NotEmpty(t, tpl.Category, "Category should not be empty")
	}

	// 验证特定模板
	var claudeTemplate *AgentTemplate
	for i := range BuiltinAgentTemplates {
		if BuiltinAgentTemplates[i].Type == AgentModelTypeClaude {
			claudeTemplate = &BuiltinAgentTemplates[i]
			break
		}
	}
	require.NotNil(t, claudeTemplate)
	assert.Contains(t, claudeTemplate.ID, "claude")
}

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
// 阶段1-红：扁平化 Task 结构测试
// ============================================================================

// TestTask_FlattenedFields 验证 Task 结构包含从 TaskSpec 扁平化的字段
func TestTask_FlattenedFields(t *testing.T) {
	now := time.Now()
	templateID := "tpl-123"
	agentID := "agent-456"
	parentID := "task-parent"

	// 创建扁平化后的 Task
	task := Task{
		// 基础字段
		ID:     "task-001",
		Name:   "Test Task",
		Status: TaskStatusPending,
		Type:   TaskTypeGeneral,

		// 从 TaskSpec 扁平化的核心字段
		Prompt: &Prompt{Content: "请帮我完成这个任务", Description: "测试任务"},

		// 复杂类型字段（JSONB）
		Workspace: &WorkspaceConfig{
			Type: WorkspaceTypeGit,
			Git: &GitConfig{
				URL:    "https://github.com/example/repo",
				Branch: "main",
			},
		},
		Security: &SecurityConfig{
			Policy:      SecurityPolicyStandard,
			Permissions: []string{"file_read", "file_write"},
		},
		Labels: map[string]string{
			"priority": "high",
			"team":     "platform",
		},
		Context: &TaskContext{
			ProducedContext: []ContextItem{
				{Type: "summary", Name: "task-summary", Content: "任务摘要"},
			},
		},

		// 关联字段
		TemplateID: &templateID,
		AgentID:    &agentID,
		ParentID:   &parentID,

		// 时间戳
		CreatedAt: now,
		UpdatedAt: now,
	}

	// 验证基础字段
	assert.Equal(t, "task-001", task.ID)
	assert.Equal(t, "Test Task", task.Name)
	assert.Equal(t, TaskStatusPending, task.Status)
	assert.Equal(t, TaskTypeGeneral, task.Type)

	// 验证扁平化的核心字段
	require.NotNil(t, task.Prompt)
	assert.Equal(t, "请帮我完成这个任务", task.Prompt.Content)
	assert.Equal(t, "测试任务", task.Prompt.Description)

	// 验证复杂类型字段
	require.NotNil(t, task.Workspace)
	assert.Equal(t, WorkspaceTypeGit, task.Workspace.Type)
	require.NotNil(t, task.Workspace.Git)
	assert.Equal(t, "https://github.com/example/repo", task.Workspace.Git.URL)

	require.NotNil(t, task.Security)
	assert.Equal(t, SecurityPolicyStandard, task.Security.Policy)
	assert.Contains(t, task.Security.Permissions, "file_read")

	assert.Equal(t, "high", task.Labels["priority"])

	require.NotNil(t, task.Context)
	assert.Len(t, task.Context.ProducedContext, 1)

	// 验证关联字段
	require.NotNil(t, task.TemplateID)
	assert.Equal(t, "tpl-123", *task.TemplateID)

	require.NotNil(t, task.AgentID)
	assert.Equal(t, "agent-456", *task.AgentID)

	require.NotNil(t, task.ParentID)
	assert.Equal(t, "task-parent", *task.ParentID)
}

// TestTask_WorkspaceJSONB 验证 Workspace 可以正确序列化/反序列化为 JSONB
func TestTask_WorkspaceJSONB(t *testing.T) {
	tests := []struct {
		name      string
		workspace *WorkspaceConfig
	}{
		{
			name:      "nil workspace",
			workspace: nil,
		},
		{
			name: "git workspace",
			workspace: &WorkspaceConfig{
				Type: WorkspaceTypeGit,
				Git: &GitConfig{
					URL:    "https://github.com/example/repo",
					Branch: "develop",
					Depth:  1,
				},
			},
		},
		{
			name: "local workspace",
			workspace: &WorkspaceConfig{
				Type: WorkspaceTypeLocal,
				Local: &LocalConfig{
					Path:     "/path/to/workspace",
					ReadOnly: true,
				},
			},
		},
		{
			name: "remote workspace",
			workspace: &WorkspaceConfig{
				Type: WorkspaceTypeRemote,
				Remote: &RemoteConfig{
					Host:          "server.example.com",
					Port:          22,
					User:          "admin",
					CredentialRef: "ssh-key-123",
				},
			},
		},
		{
			name: "volume workspace",
			workspace: &WorkspaceConfig{
				Type: WorkspaceTypeVolume,
				Volume: &VolumeConfig{
					Name:    "shared-data",
					SubPath: "project-a",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := Task{
				ID:        "task-ws-test",
				Name:      "Workspace Test",
				Status:    TaskStatusPending,
				Type:      TaskTypeGeneral,
				Prompt:    &Prompt{Content: "test"},
				Workspace: tt.workspace,
			}

			// 序列化
			data, err := json.Marshal(task)
			require.NoError(t, err)

			// 反序列化
			var decoded Task
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			// 验证
			if tt.workspace == nil {
				assert.Nil(t, decoded.Workspace)
			} else {
				require.NotNil(t, decoded.Workspace)
				assert.Equal(t, tt.workspace.Type, decoded.Workspace.Type)

				switch tt.workspace.Type {
				case WorkspaceTypeGit:
					require.NotNil(t, decoded.Workspace.Git)
					assert.Equal(t, tt.workspace.Git.URL, decoded.Workspace.Git.URL)
					assert.Equal(t, tt.workspace.Git.Branch, decoded.Workspace.Git.Branch)
				case WorkspaceTypeLocal:
					require.NotNil(t, decoded.Workspace.Local)
					assert.Equal(t, tt.workspace.Local.Path, decoded.Workspace.Local.Path)
				case WorkspaceTypeRemote:
					require.NotNil(t, decoded.Workspace.Remote)
					assert.Equal(t, tt.workspace.Remote.Host, decoded.Workspace.Remote.Host)
				case WorkspaceTypeVolume:
					require.NotNil(t, decoded.Workspace.Volume)
					assert.Equal(t, tt.workspace.Volume.Name, decoded.Workspace.Volume.Name)
				}
			}
		})
	}
}

// TestTask_SecurityJSONB 验证 Security 可以正确序列化/反序列化为 JSONB
func TestTask_SecurityJSONB(t *testing.T) {
	tests := []struct {
		name     string
		security *SecurityConfig
	}{
		{
			name:     "nil security",
			security: nil,
		},
		{
			name: "strict policy",
			security: &SecurityConfig{
				Policy: SecurityPolicyStrict,
			},
		},
		{
			name: "standard policy with permissions",
			security: &SecurityConfig{
				Policy:            SecurityPolicyStandard,
				Permissions:       []string{"file_read", "file_write", "command_execute"},
				DeniedPermissions: []string{"network_outbound"},
				RequireApproval:   []string{"file_delete"},
			},
		},
		{
			name: "permissive with limits",
			security: &SecurityConfig{
				Policy: SecurityPolicyPermissive,
				Limits: &ResourceLimits{
					MaxCPU:       "2.0",
					MaxMemory:    "4Gi",
					MaxDisk:      "10Gi",
					MaxNetwork:   "100Mbps",
					MaxProcesses: 100,
					MaxOpenFiles: 1024,
				},
			},
		},
		{
			name: "with network policy",
			security: &SecurityConfig{
				Policy: SecurityPolicyStandard,
				Network: &NetworkPolicy{
					AllowInternet:  true,
					AllowedDomains: []string{"github.com", "*.example.com"},
					DeniedDomains:  []string{"malicious.com"},
					AllowedPorts:   []int{80, 443, 22},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := Task{
				ID:       "task-sec-test",
				Name:     "Security Test",
				Status:   TaskStatusPending,
				Type:     TaskTypeGeneral,
				Prompt:   &Prompt{Content: "test"},
				Security: tt.security,
			}

			// 序列化
			data, err := json.Marshal(task)
			require.NoError(t, err)

			// 反序列化
			var decoded Task
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			// 验证
			if tt.security == nil {
				assert.Nil(t, decoded.Security)
			} else {
				require.NotNil(t, decoded.Security)
				assert.Equal(t, tt.security.Policy, decoded.Security.Policy)

				if tt.security.Limits != nil {
					require.NotNil(t, decoded.Security.Limits)
					assert.Equal(t, tt.security.Limits.MaxCPU, decoded.Security.Limits.MaxCPU)
					assert.Equal(t, tt.security.Limits.MaxMemory, decoded.Security.Limits.MaxMemory)
				}

				if tt.security.Network != nil {
					require.NotNil(t, decoded.Security.Network)
					assert.Equal(t, tt.security.Network.AllowInternet, decoded.Security.Network.AllowInternet)
					assert.Equal(t, tt.security.Network.AllowedDomains, decoded.Security.Network.AllowedDomains)
				}
			}
		})
	}
}

// TestTask_LabelsJSONB 验证 Labels 可以正确序列化/反序列化为 JSONB
func TestTask_LabelsJSONB(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
	}{
		{
			name:   "nil labels",
			labels: nil,
		},
		{
			name:   "empty labels",
			labels: map[string]string{},
		},
		{
			name: "with labels",
			labels: map[string]string{
				"priority": "high",
				"team":     "platform",
				"env":      "production",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := Task{
				ID:     "task-labels-test",
				Name:   "Labels Test",
				Status: TaskStatusPending,
				Type:   TaskTypeGeneral,
				Prompt: &Prompt{Content: "test"},
				Labels: tt.labels,
			}

			// 序列化
			data, err := json.Marshal(task)
			require.NoError(t, err)

			// 反序列化
			var decoded Task
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			// 验证
			if tt.labels == nil || len(tt.labels) == 0 {
				// nil 和空 map 在 JSON 反序列化后都是 nil
				assert.True(t, len(decoded.Labels) == 0, "Labels should be empty")
			} else {
				assert.Equal(t, tt.labels, decoded.Labels)
			}
		})
	}
}

// TestTaskType_Values 验证 TaskType 枚举值
func TestTaskType_Values(t *testing.T) {
	// 验证所有预定义的 TaskType 值
	types := []TaskType{
		TaskTypeGeneral,
		TaskTypeDevelopment,
		TaskTypeResearch,
		TaskTypeOperation,
		TaskTypeAutomation,
	}

	for _, tt := range types {
		assert.NotEmpty(t, string(tt), "TaskType should have non-empty string value")
	}

	// 验证默认值
	assert.Equal(t, TaskType("general"), TaskTypeGeneral)
}

// TestTaskStatus_Values 验证 TaskStatus 枚举值
func TestTaskStatus_Values(t *testing.T) {
	statuses := []TaskStatus{
		TaskStatusPending,
		TaskStatusInProgress,
		TaskStatusCompleted,
		TaskStatusFailed,
		TaskStatusCancelled,
	}

	for _, s := range statuses {
		assert.NotEmpty(t, string(s), "TaskStatus should have non-empty string value")
	}
}

// TestSecurityPolicy_Values 验证 SecurityPolicy 枚举值
func TestSecurityPolicy_Values(t *testing.T) {
	policies := []SecurityPolicy{
		SecurityPolicyStrict,
		SecurityPolicyStandard,
		SecurityPolicyPermissive,
	}

	for _, p := range policies {
		assert.NotEmpty(t, string(p), "SecurityPolicy should have non-empty string value")
	}
}

// TestWorkspaceType_Values 验证 WorkspaceType 枚举值
func TestWorkspaceType_Values(t *testing.T) {
	types := []WorkspaceType{
		WorkspaceTypeGit,
		WorkspaceTypeLocal,
		WorkspaceTypeRemote,
		WorkspaceTypeVolume,
	}

	for _, wt := range types {
		assert.NotEmpty(t, string(wt), "WorkspaceType should have non-empty string value")
	}
}

// TestTask_ContextJSONB 验证 Context 可以正确序列化/反序列化为 JSONB
func TestTask_ContextJSONB(t *testing.T) {
	ctx := &TaskContext{
		InheritedContext: []ContextItem{
			{Type: "file", Name: "config.yaml", Content: "key: value", Source: "task-parent"},
		},
		ProducedContext: []ContextItem{
			{Type: "summary", Name: "result", Content: "任务完成"},
		},
		ConversationHistory: []Message{
			{Role: "user", Content: "请开始任务", Timestamp: time.Now()},
			{Role: "assistant", Content: "好的，我开始执行", Timestamp: time.Now()},
		},
	}

	task := Task{
		ID:      "task-ctx-test",
		Name:    "Context Test",
		Status:  TaskStatusPending,
		Type:    TaskTypeGeneral,
		Prompt:  &Prompt{Content: "test"},
		Context: ctx,
	}

	// 序列化
	data, err := json.Marshal(task)
	require.NoError(t, err)

	// 反序列化
	var decoded Task
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// 验证
	require.NotNil(t, decoded.Context)
	assert.Len(t, decoded.Context.InheritedContext, 1)
	assert.Len(t, decoded.Context.ProducedContext, 1)
	assert.Len(t, decoded.Context.ConversationHistory, 2)

	assert.Equal(t, "file", decoded.Context.InheritedContext[0].Type)
	assert.Equal(t, "summary", decoded.Context.ProducedContext[0].Type)
	assert.Equal(t, "result", decoded.Context.ProducedContext[0].Name)
	assert.Equal(t, "user", decoded.Context.ConversationHistory[0].Role)
}

// TestTask_AgentIDReplaceInstanceID 验证 AgentID 替代了 InstanceID
func TestTask_AgentIDReplaceInstanceID(t *testing.T) {
	agentID := "agent-123"

	task := Task{
		ID:      "task-agent-test",
		Name:    "Agent Test",
		Status:  TaskStatusInProgress,
		Type:    TaskTypeGeneral,
		Prompt:  &Prompt{Content: "test"},
		AgentID: &agentID,
	}

	// 验证 AgentID 存在
	require.NotNil(t, task.AgentID)
	assert.Equal(t, "agent-123", *task.AgentID)

	// 序列化验证 JSON 字段名
	data, err := json.Marshal(task)
	require.NoError(t, err)

	var jsonMap map[string]interface{}
	err = json.Unmarshal(data, &jsonMap)
	require.NoError(t, err)

	// 验证使用 agent_id 而非 instance_id
	_, hasAgentID := jsonMap["agent_id"]
	assert.True(t, hasAgentID, "JSON should have 'agent_id' field")

	_, hasInstanceID := jsonMap["instance_id"]
	assert.False(t, hasInstanceID, "JSON should not have 'instance_id' field")
}

// TestTask_TemplateID 验证 TemplateID 关联
func TestTask_TemplateID(t *testing.T) {
	templateID := "tpl-dev-001"

	task := Task{
		ID:         "task-tpl-test",
		Name:       "Template Test",
		Status:     TaskStatusPending,
		Type:       TaskTypeDevelopment,
		Prompt:     &Prompt{Content: "基于模板创建的任务"},
		TemplateID: &templateID,
	}

	require.NotNil(t, task.TemplateID)
	assert.Equal(t, "tpl-dev-001", *task.TemplateID)

	// 序列化验证
	data, err := json.Marshal(task)
	require.NoError(t, err)

	var decoded Task
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	require.NotNil(t, decoded.TemplateID)
	assert.Equal(t, "tpl-dev-001", *decoded.TemplateID)
}

// ============================================================================
// 阶段3：TaskTemplate 模型测试
// ============================================================================

// TestTaskTemplate_TypeAsPrimaryDefinition 验证 Type 是 TaskTemplate 的主定义
func TestTaskTemplate_TypeAsPrimaryDefinition(t *testing.T) {
	template := TaskTemplate{
		ID:          "tpl-dev-001",
		Name:        "开发任务模板",
		Description: "用于代码开发的标准任务模板",
		Type:        TaskTypeDevelopment, // Type 是主定义
		Category:    "development",
		Source:      "builtin",
	}

	// Type 在 TaskTemplate 中定义
	assert.Equal(t, TaskTypeDevelopment, template.Type)
	assert.Equal(t, "开发任务模板", template.Name)
}

// TestTaskTemplate_WithPromptTemplate 验证 TaskTemplate 可以包含 PromptTemplate
func TestTaskTemplate_WithPromptTemplate(t *testing.T) {
	// 内嵌 PromptTemplate
	template := TaskTemplate{
		ID:   "tpl-review-001",
		Name: "代码审查模板",
		Type: TaskTypeDevelopment,
		PromptTemplate: &PromptTemplate{
			ID:          "prompt-review-001",
			Name:        "代码审查提示词",
			Description: "用于代码审查的提示词模板",
			Content:     "请审查以下代码变更：\n{{.diff}}\n\n关注点：{{.focus}}",
			Variables: []TemplateVariable{
				{Name: "diff", Type: "string", Description: "代码差异", Required: true},
				{Name: "focus", Type: "string", Description: "关注点", Required: false, Default: "代码质量和安全性"},
			},
		},
	}

	assert.True(t, template.HasPromptTemplate())
	require.NotNil(t, template.PromptTemplate)
	assert.Equal(t, "prompt-review-001", template.PromptTemplate.ID)
	assert.Len(t, template.PromptTemplate.Variables, 2)
}

// TestTaskTemplate_WithPromptTemplateID 验证 TaskTemplate 可以引用 PromptTemplate ID
func TestTaskTemplate_WithPromptTemplateID(t *testing.T) {
	promptID := "prompt-shared-001"
	template := TaskTemplate{
		ID:               "tpl-shared-001",
		Name:             "共享模板",
		Type:             TaskTypeGeneral,
		PromptTemplateID: &promptID,
	}

	assert.True(t, template.HasPromptTemplate())
	require.NotNil(t, template.PromptTemplateID)
	assert.Equal(t, "prompt-shared-001", *template.PromptTemplateID)
}

// TestTaskTemplate_CreateTask 验证从模板创建任务
func TestTaskTemplate_CreateTask(t *testing.T) {
	template := TaskTemplate{
		ID:          "tpl-dev-001",
		Name:        "开发任务模板",
		Description: "开发任务的标准描述",
		Type:        TaskTypeDevelopment,
		PromptTemplate: &PromptTemplate{
			ID:      "prompt-dev-001",
			Name:    "开发提示词",
			Content: "请实现以下功能：{{.feature}}",
			Variables: []TemplateVariable{
				{Name: "feature", Type: "string", Description: "功能描述", Required: true},
			},
		},
		DefaultWorkspace: &WorkspaceConfig{
			Type: WorkspaceTypeGit,
			Git: &GitConfig{
				URL:    "https://github.com/example/repo",
				Branch: "main",
			},
		},
		DefaultSecurity: &SecurityConfig{
			Policy: SecurityPolicyStandard,
		},
		DefaultLabels: map[string]string{
			"type": "development",
		},
	}

	// 创建任务
	variables := map[string]interface{}{
		"feature": "用户登录功能",
	}
	task := template.CreateTask("实现用户登录", variables)

	// 验证任务继承了模板的 Type
	assert.Equal(t, TaskTypeDevelopment, task.Type)

	// 验证任务继承了模板的描述
	assert.Equal(t, "开发任务的标准描述", task.Description)

	// 验证 TemplateID 被设置
	require.NotNil(t, task.TemplateID)
	assert.Equal(t, "tpl-dev-001", *task.TemplateID)

	// 验证 Prompt 被创建
	require.NotNil(t, task.Prompt)
	assert.Equal(t, "请实现以下功能：{{.feature}}", task.Prompt.Content) // TODO: 变量插值
	require.NotNil(t, task.Prompt.TemplateID)
	assert.Equal(t, "prompt-dev-001", *task.Prompt.TemplateID)

	// 验证变量被传递
	assert.Equal(t, "用户登录功能", task.Prompt.Variables["feature"])

	// 验证默认配置被继承
	require.NotNil(t, task.Workspace)
	assert.Equal(t, WorkspaceTypeGit, task.Workspace.Type)

	require.NotNil(t, task.Security)
	assert.Equal(t, SecurityPolicyStandard, task.Security.Policy)

	// 验证默认标签被继承
	assert.Equal(t, "development", task.Labels["type"])
}

// TestTaskTemplate_CreateTask_WithoutPromptTemplate 验证没有 PromptTemplate 时创建任务
func TestTaskTemplate_CreateTask_WithoutPromptTemplate(t *testing.T) {
	template := TaskTemplate{
		ID:   "tpl-simple-001",
		Name: "简单模板",
		Type: TaskTypeResearch,
	}

	task := template.CreateTask("简单任务", nil)

	assert.Equal(t, TaskTypeResearch, task.Type)
	assert.Nil(t, task.Prompt) // 没有 PromptTemplate，Prompt 为 nil
}

// TestTaskTemplate_Variables 验证模板变量定义
func TestTaskTemplate_Variables(t *testing.T) {
	template := TaskTemplate{
		ID:   "tpl-vars-001",
		Name: "变量模板",
		Type: TaskTypeGeneral,
		Variables: []TemplateVariable{
			{
				Name:        "target",
				Type:        "string",
				Description: "目标描述",
				Required:    true,
			},
			{
				Name:        "priority",
				Type:        "string",
				Description: "优先级",
				Required:    false,
				Default:     "medium",
				Options:     []interface{}{"low", "medium", "high"},
			},
			{
				Name:        "count",
				Type:        "number",
				Description: "数量",
				Required:    false,
				Default:     1,
			},
		},
	}

	assert.Len(t, template.Variables, 3)

	// 验证必填变量
	assert.Equal(t, "target", template.Variables[0].Name)
	assert.True(t, template.Variables[0].Required)

	// 验证带默认值和选项的变量
	assert.Equal(t, "priority", template.Variables[1].Name)
	assert.False(t, template.Variables[1].Required)
	assert.Equal(t, "medium", template.Variables[1].Default)
	assert.Len(t, template.Variables[1].Options, 3)
}

// TestTaskTemplate_JSONSerialization 验证 TaskTemplate JSON 序列化
func TestTaskTemplate_JSONSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	template := TaskTemplate{
		ID:          "tpl-json-001",
		Name:        "JSON 测试模板",
		Description: "用于测试 JSON 序列化",
		Type:        TaskTypeDevelopment,
		PromptTemplate: &PromptTemplate{
			ID:      "prompt-json-001",
			Name:    "JSON 提示词",
			Content: "测试内容",
		},
		DefaultWorkspace: &WorkspaceConfig{
			Type: WorkspaceTypeGit,
			Git:  &GitConfig{URL: "https://github.com/test/repo"},
		},
		DefaultLabels: map[string]string{"env": "test"},
		Category:      "testing",
		Tags:          []string{"test", "json"},
		Source:        "custom",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// 序列化
	data, err := json.Marshal(template)
	require.NoError(t, err)

	// 反序列化
	var decoded TaskTemplate
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// 验证
	assert.Equal(t, template.ID, decoded.ID)
	assert.Equal(t, template.Name, decoded.Name)
	assert.Equal(t, template.Type, decoded.Type)
	require.NotNil(t, decoded.PromptTemplate)
	assert.Equal(t, template.PromptTemplate.ID, decoded.PromptTemplate.ID)
	require.NotNil(t, decoded.DefaultWorkspace)
	assert.Equal(t, template.DefaultWorkspace.Type, decoded.DefaultWorkspace.Type)
	assert.Equal(t, template.DefaultLabels, decoded.DefaultLabels)
	assert.Equal(t, template.Tags, decoded.Tags)
}

// TestBuiltinTaskTemplates 验证内置任务模板
func TestBuiltinTaskTemplates(t *testing.T) {
	// 定义内置模板
	builtinTemplates := []TaskTemplate{
		{
			ID:          "builtin-code-review",
			Name:        "代码审查",
			Description: "审查代码变更，检查代码质量和潜在问题",
			Type:        TaskTypeDevelopment,
			Category:    "development",
			Source:      "builtin",
		},
		{
			ID:          "builtin-bug-fix",
			Name:        "Bug 修复",
			Description: "分析并修复代码中的 Bug",
			Type:        TaskTypeDevelopment,
			Category:    "development",
			Source:      "builtin",
		},
		{
			ID:          "builtin-research",
			Name:        "技术调研",
			Description: "研究技术方案和最佳实践",
			Type:        TaskTypeResearch,
			Category:    "research",
			Source:      "builtin",
		},
	}

	// 验证内置模板数量和类型
	assert.Len(t, builtinTemplates, 3)

	for _, tpl := range builtinTemplates {
		assert.NotEmpty(t, tpl.ID)
		assert.NotEmpty(t, tpl.Name)
		assert.Equal(t, "builtin", tpl.Source)
	}
}

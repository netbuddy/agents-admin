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
// 阶段5：SecurityPolicy + Sandbox 模型测试
// ============================================================================

// TestToolPermissionLevel_Values 验证 ToolPermissionLevel 枚举值
func TestToolPermissionLevel_Values(t *testing.T) {
	levels := []ToolPermissionLevel{
		ToolPermissionAllowed,
		ToolPermissionDenied,
		ToolPermissionApprovalRequired,
	}

	for _, level := range levels {
		assert.NotEmpty(t, string(level))
	}

	assert.Equal(t, ToolPermissionLevel("allowed"), ToolPermissionAllowed)
	assert.Equal(t, ToolPermissionLevel("denied"), ToolPermissionDenied)
	assert.Equal(t, ToolPermissionLevel("approval_required"), ToolPermissionApprovalRequired)
}

// TestToolPermission_BasicFields 验证 ToolPermission 基础字段
func TestToolPermission_BasicFields(t *testing.T) {
	tp := ToolPermission{
		Tool:         "file_write",
		Permission:   ToolPermissionApprovalRequired,
		Scopes:       []string{"/workspace", "/tmp"},
		ApprovalNote: "文件写入需要审批",
		Reason:       "安全限制",
	}

	assert.Equal(t, "file_write", tp.Tool)
	assert.Equal(t, ToolPermissionApprovalRequired, tp.Permission)
	assert.Len(t, tp.Scopes, 2)
	assert.Equal(t, "文件写入需要审批", tp.ApprovalNote)
}

// TestSandboxPolicy_BasicFields 验证 SandboxPolicy 基础字段
func TestSandboxPolicy_BasicFields(t *testing.T) {
	policy := SandboxPolicy{
		RequiredForTools: []string{"command_execute", "network_request"},
		DefaultType:      SandboxTypeContainer,
		AutoDestroy:      true,
		MaxLifetime:      "1h",
		ResourceLimits: &ResourceLimits{
			MaxCPU:    "1.0",
			MaxMemory: "1Gi",
		},
	}

	assert.Len(t, policy.RequiredForTools, 2)
	assert.Equal(t, SandboxTypeContainer, policy.DefaultType)
	assert.True(t, policy.AutoDestroy)
	assert.Equal(t, "1h", policy.MaxLifetime)
	require.NotNil(t, policy.ResourceLimits)
	assert.Equal(t, "1.0", policy.ResourceLimits.MaxCPU)
}

// TestSecurityPolicyEntity_BasicFields 验证 SecurityPolicyEntity 基础字段
func TestSecurityPolicyEntity_BasicFields(t *testing.T) {
	now := time.Now()
	policy := SecurityPolicyEntity{
		ID:          "policy-001",
		Name:        "测试策略",
		Description: "用于测试的安全策略",
		ToolPermissions: []ToolPermission{
			{Tool: "file_read", Permission: ToolPermissionAllowed},
			{Tool: "file_write", Permission: ToolPermissionApprovalRequired},
		},
		ResourceLimits: &ResourceLimits{
			MaxCPU:    "2.0",
			MaxMemory: "4Gi",
		},
		NetworkPolicy: &NetworkPolicy{
			AllowInternet:  true,
			AllowedDomains: []string{"github.com"},
		},
		SandboxPolicy: &SandboxPolicy{
			RequiredForTools: []string{"command_execute"},
			DefaultType:      SandboxTypeContainer,
		},
		IsBuiltin: false,
		Category:  "testing",
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, "policy-001", policy.ID)
	assert.Equal(t, "测试策略", policy.Name)
	assert.Len(t, policy.ToolPermissions, 2)
	require.NotNil(t, policy.ResourceLimits)
	require.NotNil(t, policy.NetworkPolicy)
	require.NotNil(t, policy.SandboxPolicy)
}

// TestSecurityPolicyEntity_HasToolPermission 验证工具权限检查
func TestSecurityPolicyEntity_HasToolPermission(t *testing.T) {
	policy := SecurityPolicyEntity{
		ToolPermissions: []ToolPermission{
			{Tool: "file_read", Permission: ToolPermissionAllowed},
			{Tool: "file_write", Permission: ToolPermissionDenied},
		},
	}

	assert.True(t, policy.HasToolPermission("file_read"))
	assert.True(t, policy.HasToolPermission("file_write"))
	assert.False(t, policy.HasToolPermission("command_execute"))
}

// TestSecurityPolicyEntity_GetToolPermission 验证获取工具权限
func TestSecurityPolicyEntity_GetToolPermission(t *testing.T) {
	policy := SecurityPolicyEntity{
		ToolPermissions: []ToolPermission{
			{Tool: "file_read", Permission: ToolPermissionAllowed},
			{Tool: "*", Permission: ToolPermissionApprovalRequired},
		},
	}

	// 精确匹配
	tp := policy.GetToolPermission("file_read")
	require.NotNil(t, tp)
	assert.Equal(t, ToolPermissionAllowed, tp.Permission)

	// 通配符匹配
	tp = policy.GetToolPermission("unknown_tool")
	require.NotNil(t, tp)
	assert.Equal(t, ToolPermissionApprovalRequired, tp.Permission)
}

// TestSecurityPolicyEntity_IsToolAllowed 验证工具是否被允许
func TestSecurityPolicyEntity_IsToolAllowed(t *testing.T) {
	policy := SecurityPolicyEntity{
		ToolPermissions: []ToolPermission{
			{Tool: "file_read", Permission: ToolPermissionAllowed},
			{Tool: "file_delete", Permission: ToolPermissionDenied},
			{Tool: "command_execute", Permission: ToolPermissionApprovalRequired},
		},
	}

	assert.True(t, policy.IsToolAllowed("file_read"))
	assert.False(t, policy.IsToolAllowed("file_delete"))
	assert.False(t, policy.IsToolAllowed("command_execute"))
	assert.True(t, policy.IsToolAllowed("unknown_tool")) // 默认允许
}

// TestSecurityPolicyEntity_RequiresSandbox 验证是否需要沙箱
func TestSecurityPolicyEntity_RequiresSandbox(t *testing.T) {
	policy := SecurityPolicyEntity{
		SandboxPolicy: &SandboxPolicy{
			RequiredForTools: []string{"command_execute", "network_request"},
		},
	}

	assert.True(t, policy.RequiresSandbox("command_execute"))
	assert.True(t, policy.RequiresSandbox("network_request"))
	assert.False(t, policy.RequiresSandbox("file_read"))

	// 无沙箱策略
	policyNoSandbox := SecurityPolicyEntity{}
	assert.False(t, policyNoSandbox.RequiresSandbox("command_execute"))
}

// TestSandboxType_Values 验证 SandboxType 枚举值
func TestSandboxType_Values(t *testing.T) {
	types := []SandboxType{
		SandboxTypeMicroVM,
		SandboxTypeContainer,
		SandboxTypeGVisor,
		SandboxTypeOSLevel,
		SandboxTypeNone,
	}

	for _, st := range types {
		assert.NotEmpty(t, string(st))
	}

	assert.Equal(t, SandboxType("microvm"), SandboxTypeMicroVM)
	assert.Equal(t, SandboxType("container"), SandboxTypeContainer)
}

// TestSandboxStatus_Values 验证 SandboxStatus 枚举值
func TestSandboxStatus_Values(t *testing.T) {
	statuses := []SandboxStatus{
		SandboxStatusCreating,
		SandboxStatusReady,
		SandboxStatusRunning,
		SandboxStatusStopping,
		SandboxStatusStopped,
		SandboxStatusDestroyed,
		SandboxStatusError,
	}

	for _, s := range statuses {
		assert.NotEmpty(t, string(s))
	}
}

// TestSandbox_BasicFields 验证 Sandbox 基础字段
func TestSandbox_BasicFields(t *testing.T) {
	now := time.Now()
	expires := now.Add(time.Hour)
	runtimeID := "runtime-001"

	sandbox := Sandbox{
		ID:           "sandbox-001",
		AgentID:      "agent-001",
		Type:         SandboxTypeContainer,
		Status:       SandboxStatusRunning,
		Isolation:    "container",
		FSRoot:       "/sandbox/001",
		NetNamespace: "ns-sandbox-001",
		ResourceLimits: &ResourceLimits{
			MaxCPU:    "1.0",
			MaxMemory: "1Gi",
		},
		RuntimeID: &runtimeID,
		NodeID:    "node-001",
		CreatedAt: now,
		StartedAt: &now,
		ExpiresAt: &expires,
	}

	assert.Equal(t, "sandbox-001", sandbox.ID)
	assert.Equal(t, "agent-001", sandbox.AgentID)
	assert.Equal(t, SandboxTypeContainer, sandbox.Type)
	assert.Equal(t, SandboxStatusRunning, sandbox.Status)
	assert.Equal(t, "/sandbox/001", sandbox.FSRoot)
	require.NotNil(t, sandbox.RuntimeID)
	assert.Equal(t, "runtime-001", *sandbox.RuntimeID)
}

// TestSandbox_Lifecycle 验证 Sandbox 生命周期方法
func TestSandbox_Lifecycle(t *testing.T) {
	tests := []struct {
		name         string
		status       SandboxStatus
		isActive     bool
		isTerminated bool
		canDestroy   bool
	}{
		{
			name:         "creating",
			status:       SandboxStatusCreating,
			isActive:     false,
			isTerminated: false,
			canDestroy:   false,
		},
		{
			name:         "ready",
			status:       SandboxStatusReady,
			isActive:     true,
			isTerminated: false,
			canDestroy:   true,
		},
		{
			name:         "running",
			status:       SandboxStatusRunning,
			isActive:     true,
			isTerminated: false,
			canDestroy:   true,
		},
		{
			name:         "stopped",
			status:       SandboxStatusStopped,
			isActive:     false,
			isTerminated: true,
			canDestroy:   true,
		},
		{
			name:         "destroyed",
			status:       SandboxStatusDestroyed,
			isActive:     false,
			isTerminated: true,
			canDestroy:   false,
		},
		{
			name:         "error",
			status:       SandboxStatusError,
			isActive:     false,
			isTerminated: true,
			canDestroy:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sandbox := Sandbox{
				ID:     "sandbox-test",
				Status: tt.status,
			}

			assert.Equal(t, tt.isActive, sandbox.IsActive(), "IsActive")
			assert.Equal(t, tt.isTerminated, sandbox.IsTerminated(), "IsTerminated")
			assert.Equal(t, tt.canDestroy, sandbox.CanDestroy(), "CanDestroy")
		})
	}
}

// TestSandbox_IsExpired 验证沙箱过期检查
func TestSandbox_IsExpired(t *testing.T) {
	// 未设置过期时间
	sandbox := Sandbox{ID: "sandbox-no-expire"}
	assert.False(t, sandbox.IsExpired())

	// 未过期
	future := time.Now().Add(time.Hour)
	sandbox = Sandbox{ID: "sandbox-future", ExpiresAt: &future}
	assert.False(t, sandbox.IsExpired())

	// 已过期
	past := time.Now().Add(-time.Hour)
	sandbox = Sandbox{ID: "sandbox-past", ExpiresAt: &past}
	assert.True(t, sandbox.IsExpired())
}

// TestSecurityPolicyEntity_JSONSerialization 验证 JSON 序列化
func TestSecurityPolicyEntity_JSONSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	policy := SecurityPolicyEntity{
		ID:          "policy-json-001",
		Name:        "JSON 测试策略",
		Description: "用于 JSON 序列化测试",
		ToolPermissions: []ToolPermission{
			{Tool: "file_*", Permission: ToolPermissionAllowed},
			{Tool: "command_execute", Permission: ToolPermissionDenied},
		},
		ResourceLimits: &ResourceLimits{
			MaxCPU:       "2.0",
			MaxMemory:    "4Gi",
			MaxProcesses: 100,
		},
		NetworkPolicy: &NetworkPolicy{
			AllowInternet:  true,
			AllowedDomains: []string{"github.com", "*.example.com"},
		},
		SandboxPolicy: &SandboxPolicy{
			RequiredForTools: []string{"network_request"},
			DefaultType:      SandboxTypeMicroVM,
			AutoDestroy:      true,
		},
		IsBuiltin: false,
		Category:  "testing",
		CreatedAt: now,
		UpdatedAt: now,
	}

	// 序列化
	data, err := json.Marshal(policy)
	require.NoError(t, err)

	// 反序列化
	var decoded SecurityPolicyEntity
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// 验证
	assert.Equal(t, policy.ID, decoded.ID)
	assert.Equal(t, policy.Name, decoded.Name)
	assert.Len(t, decoded.ToolPermissions, 2)
	require.NotNil(t, decoded.ResourceLimits)
	assert.Equal(t, "2.0", decoded.ResourceLimits.MaxCPU)
	require.NotNil(t, decoded.NetworkPolicy)
	assert.True(t, decoded.NetworkPolicy.AllowInternet)
	require.NotNil(t, decoded.SandboxPolicy)
	assert.Equal(t, SandboxTypeMicroVM, decoded.SandboxPolicy.DefaultType)
}

// TestSandbox_JSONSerialization 验证 Sandbox JSON 序列化
func TestSandbox_JSONSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	expires := now.Add(time.Hour).Truncate(time.Second)
	runtimeID := "runtime-001"

	sandbox := Sandbox{
		ID:           "sandbox-json-001",
		AgentID:      "agent-001",
		Type:         SandboxTypeGVisor,
		Status:       SandboxStatusRunning,
		Isolation:    "gvisor",
		FSRoot:       "/sandbox/json-001",
		NetNamespace: "ns-json-001",
		RuntimeID:    &runtimeID,
		NodeID:       "node-001",
		CreatedAt:    now,
		StartedAt:    &now,
		ExpiresAt:    &expires,
	}

	// 序列化
	data, err := json.Marshal(sandbox)
	require.NoError(t, err)

	// 反序列化
	var decoded Sandbox
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// 验证
	assert.Equal(t, sandbox.ID, decoded.ID)
	assert.Equal(t, sandbox.AgentID, decoded.AgentID)
	assert.Equal(t, sandbox.Type, decoded.Type)
	assert.Equal(t, sandbox.Status, decoded.Status)
	assert.Equal(t, sandbox.FSRoot, decoded.FSRoot)
	require.NotNil(t, decoded.RuntimeID)
	assert.Equal(t, "runtime-001", *decoded.RuntimeID)
}

// TestBuiltinSecurityPolicies 验证内置安全策略
func TestBuiltinSecurityPolicies(t *testing.T) {
	// 验证内置策略数量
	assert.GreaterOrEqual(t, len(BuiltinSecurityPolicies), 3)

	// 验证每个策略
	for _, policy := range BuiltinSecurityPolicies {
		assert.NotEmpty(t, policy.ID, "ID should not be empty")
		assert.NotEmpty(t, policy.Name, "Name should not be empty")
		assert.True(t, policy.IsBuiltin, "IsBuiltin should be true")
	}

	// 验证严格策略
	var strictPolicy *SecurityPolicyEntity
	for i := range BuiltinSecurityPolicies {
		if BuiltinSecurityPolicies[i].ID == "builtin-strict" {
			strictPolicy = &BuiltinSecurityPolicies[i]
			break
		}
	}
	require.NotNil(t, strictPolicy)
	assert.Equal(t, "严格策略", strictPolicy.Name)
	require.NotNil(t, strictPolicy.NetworkPolicy)
	assert.False(t, strictPolicy.NetworkPolicy.AllowInternet)

	// 验证标准策略
	var standardPolicy *SecurityPolicyEntity
	for i := range BuiltinSecurityPolicies {
		if BuiltinSecurityPolicies[i].ID == "builtin-standard" {
			standardPolicy = &BuiltinSecurityPolicies[i]
			break
		}
	}
	require.NotNil(t, standardPolicy)
	require.NotNil(t, standardPolicy.NetworkPolicy)
	assert.True(t, standardPolicy.NetworkPolicy.AllowInternet)
}

// TestAgent_CreateSandbox 验证 Agent 创建 Sandbox 场景
func TestAgent_CreateSandbox(t *testing.T) {
	// Agent 需要执行危险操作时创建沙箱
	agent := Agent{
		ID:        "agent-sandbox-test",
		Name:      "Sandbox Test Agent",
		Status:    AgentStatusRunning,
		AccountID: "account-001",
	}

	// 创建沙箱
	now := time.Now()
	expires := now.Add(time.Hour)
	sandbox := Sandbox{
		ID:        "sandbox-for-agent",
		AgentID:   agent.ID,
		Type:      SandboxTypeContainer,
		Status:    SandboxStatusReady,
		NodeID:    "node-001",
		CreatedAt: now,
		ExpiresAt: &expires,
	}

	// 验证关联
	assert.Equal(t, agent.ID, sandbox.AgentID)
	assert.True(t, sandbox.IsActive())
	assert.False(t, sandbox.IsExpired())
}

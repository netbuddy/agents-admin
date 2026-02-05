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
// 阶段7：Skill 模型测试
// ============================================================================

// TestSkillCategory_Values 验证 SkillCategory 枚举值
func TestSkillCategory_Values(t *testing.T) {
	categories := []SkillCategory{
		SkillCategoryCoding,
		SkillCategoryWriting,
		SkillCategoryResearch,
		SkillCategoryAnalysis,
		SkillCategoryDesign,
		SkillCategoryData,
		SkillCategoryDevOps,
		SkillCategoryOther,
	}

	for _, c := range categories {
		assert.NotEmpty(t, string(c))
	}

	assert.Equal(t, SkillCategory("coding"), SkillCategoryCoding)
	assert.Equal(t, SkillCategory("research"), SkillCategoryResearch)
}

// TestSkillLevel_Values 验证 SkillLevel 枚举值
func TestSkillLevel_Values(t *testing.T) {
	levels := []SkillLevel{
		SkillLevelBasic,
		SkillLevelAdvanced,
		SkillLevelExpert,
	}

	for _, l := range levels {
		assert.NotEmpty(t, string(l))
	}

	assert.Equal(t, SkillLevel("basic"), SkillLevelBasic)
	assert.Equal(t, SkillLevel("expert"), SkillLevelExpert)
}

// TestSkillSource_Values 验证 SkillSource 枚举值
func TestSkillSource_Values(t *testing.T) {
	sources := []SkillSource{
		SkillSourceBuiltin,
		SkillSourceUser,
		SkillSourceCommunity,
	}

	for _, s := range sources {
		assert.NotEmpty(t, string(s))
	}

	assert.Equal(t, SkillSource("builtin"), SkillSourceBuiltin)
	assert.Equal(t, SkillSource("community"), SkillSourceCommunity)
}

// TestSkill_BasicFields 验证 Skill 基础字段
func TestSkill_BasicFields(t *testing.T) {
	now := time.Now()
	authorID := "user-001"

	skill := Skill{
		ID:           "skill-001",
		Name:         "Python 开发",
		Category:     SkillCategoryCoding,
		Level:        SkillLevelAdvanced,
		Description:  "Python 编程和最佳实践",
		Instructions: "作为 Python 专家...",
		Tools:        json.RawMessage(`["file_read", "file_write", "command_execute"]`),
		Examples:     json.RawMessage(`[{"input": "写个排序", "output": "..."}]`),
		Source:       SkillSourceUser,
		AuthorID:     &authorID,
		Version:      "1.0.0",
		IsBuiltin:    false,
		Tags:         []string{"python", "programming"},
		UseCount:     100,
		Rating:       4.5,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	assert.Equal(t, "skill-001", skill.ID)
	assert.Equal(t, "Python 开发", skill.Name)
	assert.Equal(t, SkillCategoryCoding, skill.Category)
	assert.Equal(t, SkillLevelAdvanced, skill.Level)
	assert.Equal(t, SkillSourceUser, skill.Source)
	require.NotNil(t, skill.AuthorID)
	assert.Equal(t, "user-001", *skill.AuthorID)
	assert.Equal(t, int64(100), skill.UseCount)
	assert.Equal(t, 4.5, skill.Rating)
}

// TestSkill_IsBuiltinSkill 验证内置技能判断
func TestSkill_IsBuiltinSkill(t *testing.T) {
	// 通过 IsBuiltin 字段
	builtinByFlag := Skill{IsBuiltin: true, Source: SkillSourceUser}
	assert.True(t, builtinByFlag.IsBuiltinSkill())

	// 通过 Source 字段
	builtinBySource := Skill{IsBuiltin: false, Source: SkillSourceBuiltin}
	assert.True(t, builtinBySource.IsBuiltinSkill())

	// 非内置
	userSkill := Skill{IsBuiltin: false, Source: SkillSourceUser}
	assert.False(t, userSkill.IsBuiltinSkill())
}

// TestSkill_GetTools 验证获取工具列表
func TestSkill_GetTools(t *testing.T) {
	// 有工具
	skill := Skill{
		Tools: json.RawMessage(`["file_read", "file_write", "command_execute"]`),
	}
	tools := skill.GetTools()
	assert.Len(t, tools, 3)
	assert.Contains(t, tools, "file_read")
	assert.Contains(t, tools, "command_execute")

	// 无工具
	noTools := Skill{}
	assert.Nil(t, noTools.GetTools())
}

// TestSkill_JSONSerialization 验证 Skill JSON 序列化
func TestSkill_JSONSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	authorID := "user-001"

	skill := Skill{
		ID:           "skill-json-001",
		Name:         "代码审查",
		Category:     SkillCategoryCoding,
		Level:        SkillLevelExpert,
		Description:  "专业代码审查",
		Instructions: "作为代码审查专家...",
		Tools:        json.RawMessage(`["file_read"]`),
		Source:       SkillSourceCommunity,
		AuthorID:     &authorID,
		Version:      "2.0.0",
		IsBuiltin:    false,
		Tags:         []string{"review", "quality"},
		UseCount:     500,
		Rating:       4.8,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// 序列化
	data, err := json.Marshal(skill)
	require.NoError(t, err)

	// 反序列化
	var decoded Skill
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// 验证
	assert.Equal(t, skill.ID, decoded.ID)
	assert.Equal(t, skill.Name, decoded.Name)
	assert.Equal(t, skill.Category, decoded.Category)
	assert.Equal(t, skill.Level, decoded.Level)
	assert.Equal(t, skill.Source, decoded.Source)
	assert.Equal(t, skill.UseCount, decoded.UseCount)
	assert.Equal(t, skill.Rating, decoded.Rating)
	require.NotNil(t, decoded.AuthorID)
	assert.Equal(t, "user-001", *decoded.AuthorID)
}

// TestSkillRegistry_BasicFields 验证 SkillRegistry 基础字段
func TestSkillRegistry_BasicFields(t *testing.T) {
	now := time.Now()
	ownerID := "org-001"

	registry := SkillRegistry{
		ID:          "registry-001",
		Name:        "社区技能市场",
		Description: "社区共享的技能集合",
		Type:        SkillSourceCommunity,
		OwnerID:     &ownerID,
		IsPublic:    true,
		SkillCount:  50,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, "registry-001", registry.ID)
	assert.Equal(t, "社区技能市场", registry.Name)
	assert.Equal(t, SkillSourceCommunity, registry.Type)
	assert.True(t, registry.IsPublic)
	assert.Equal(t, 50, registry.SkillCount)
}

// TestAgentSkill_BasicFields 验证 AgentSkill 关联
func TestAgentSkill_BasicFields(t *testing.T) {
	now := time.Now()

	agentSkill := AgentSkill{
		AgentID:   "agent-001",
		SkillID:   "skill-001",
		Enabled:   true,
		Config:    json.RawMessage(`{"max_tokens": 4000}`),
		Priority:  1,
		CreatedAt: now,
	}

	assert.Equal(t, "agent-001", agentSkill.AgentID)
	assert.Equal(t, "skill-001", agentSkill.SkillID)
	assert.True(t, agentSkill.Enabled)
	assert.Equal(t, 1, agentSkill.Priority)
}

// TestBuiltinSkills 验证内置技能
func TestBuiltinSkills(t *testing.T) {
	// 验证内置技能数量
	assert.GreaterOrEqual(t, len(BuiltinSkills), 4)

	// 验证每个内置技能
	for _, skill := range BuiltinSkills {
		assert.NotEmpty(t, skill.ID, "ID should not be empty")
		assert.NotEmpty(t, skill.Name, "Name should not be empty")
		assert.NotEmpty(t, skill.Description, "Description should not be empty")
		assert.NotEmpty(t, skill.Instructions, "Instructions should not be empty")
		assert.True(t, skill.IsBuiltin, "IsBuiltin should be true")
		assert.Equal(t, SkillSourceBuiltin, skill.Source, "Source should be builtin")
	}

	// 验证代码审查技能
	var codeReview *Skill
	for i := range BuiltinSkills {
		if BuiltinSkills[i].ID == "builtin-code-review" {
			codeReview = &BuiltinSkills[i]
			break
		}
	}
	require.NotNil(t, codeReview)
	assert.Equal(t, "代码审查", codeReview.Name)
	assert.Equal(t, SkillCategoryCoding, codeReview.Category)
	assert.Equal(t, SkillLevelAdvanced, codeReview.Level)
}

// TestSkill_AllCategories 验证所有技能分类
func TestSkill_AllCategories(t *testing.T) {
	tests := []struct {
		category SkillCategory
		expected string
	}{
		{SkillCategoryCoding, "coding"},
		{SkillCategoryWriting, "writing"},
		{SkillCategoryResearch, "research"},
		{SkillCategoryAnalysis, "analysis"},
		{SkillCategoryDesign, "design"},
		{SkillCategoryData, "data"},
		{SkillCategoryDevOps, "devops"},
		{SkillCategoryOther, "other"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.category))
		})
	}
}

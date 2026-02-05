// Package model 定义核心数据模型
//
// skill.go 包含技能相关的数据模型定义：
//   - Skill：技能定义
//   - SkillCategory：技能分类枚举
//   - SkillLevel：技能等级枚举
//   - SkillSource：技能来源枚举
//   - SkillRegistry：技能市场
//   - AgentSkill：Agent 与 Skill 的关联
//
// 设计理念：
//   - Skill 定义 Agent 可以执行的能力
//   - SkillRegistry 作为技能市场，管理可用技能
//   - AgentSkill 实现 Agent 与 Skill 的多对多关联
package model

import (
	"encoding/json"
	"time"
)

// ============================================================================
// SkillCategory - 技能分类枚举
// ============================================================================

// SkillCategory 技能分类
type SkillCategory string

const (
	// SkillCategoryCoding 编程
	SkillCategoryCoding SkillCategory = "coding"

	// SkillCategoryWriting 写作
	SkillCategoryWriting SkillCategory = "writing"

	// SkillCategoryResearch 研究
	SkillCategoryResearch SkillCategory = "research"

	// SkillCategoryAnalysis 分析
	SkillCategoryAnalysis SkillCategory = "analysis"

	// SkillCategoryDesign 设计
	SkillCategoryDesign SkillCategory = "design"

	// SkillCategoryData 数据处理
	SkillCategoryData SkillCategory = "data"

	// SkillCategoryDevOps DevOps
	SkillCategoryDevOps SkillCategory = "devops"

	// SkillCategoryOther 其他
	SkillCategoryOther SkillCategory = "other"
)

// ============================================================================
// SkillLevel - 技能等级枚举
// ============================================================================

// SkillLevel 技能等级
type SkillLevel string

const (
	// SkillLevelBasic 基础
	SkillLevelBasic SkillLevel = "basic"

	// SkillLevelAdvanced 进阶
	SkillLevelAdvanced SkillLevel = "advanced"

	// SkillLevelExpert 专家
	SkillLevelExpert SkillLevel = "expert"
)

// ============================================================================
// SkillSource - 技能来源枚举
// ============================================================================

// SkillSource 技能来源
type SkillSource string

const (
	// SkillSourceBuiltin 内置技能
	SkillSourceBuiltin SkillSource = "builtin"

	// SkillSourceUser 用户自定义
	SkillSourceUser SkillSource = "user"

	// SkillSourceCommunity 社区市场
	SkillSourceCommunity SkillSource = "community"
)

// ============================================================================
// Skill - 技能定义
// ============================================================================

// Skill 技能定义
//
// Skill 定义 Agent 可以执行的能力：
//   - 包含技能说明和使用指南
//   - 可以依赖特定工具
//   - 支持示例和文档
//
// 使用场景：
//   - 为 Agent 添加特定领域能力
//   - 复用经过验证的任务执行方式
//   - 从社区市场获取共享技能
type Skill struct {
	// === 基础字段 ===

	// ID 唯一标识
	ID string `json:"id" db:"id"`

	// Name 技能名称
	Name string `json:"name" db:"name"`

	// Category 技能分类
	Category SkillCategory `json:"category" db:"category"`

	// Level 技能等级
	Level SkillLevel `json:"level" db:"level"`

	// Description 技能描述
	Description string `json:"description" db:"description"`

	// === 技能定义 ===

	// Instructions 技能说明/提示词
	// 详细描述如何使用这个技能
	Instructions string `json:"instructions" db:"instructions"`

	// Tools 依赖的工具列表
	// JSON 数组，如 ["file_read", "file_write", "command_execute"]
	Tools json.RawMessage `json:"tools,omitempty" db:"tools"`

	// Examples 使用示例
	// JSON 数组，包含输入输出示例
	Examples json.RawMessage `json:"examples,omitempty" db:"examples"`

	// Parameters 技能参数定义
	// JSON Schema 格式，定义技能需要的配置参数
	Parameters json.RawMessage `json:"parameters,omitempty" db:"parameters"`

	// === 来源 ===

	// Source 技能来源
	Source SkillSource `json:"source" db:"source"`

	// AuthorID 作者 ID（用户自定义或社区贡献）
	AuthorID *string `json:"author_id,omitempty" db:"author_id"`

	// RegistryID 所属技能市场 ID
	RegistryID *string `json:"registry_id,omitempty" db:"registry_id"`

	// === 元数据 ===

	// Version 版本号（语义化版本）
	Version string `json:"version" db:"version"`

	// IsBuiltin 是否内置技能
	IsBuiltin bool `json:"is_builtin" db:"is_builtin"`

	// Tags 标签
	Tags []string `json:"tags,omitempty" db:"tags"`

	// === 统计 ===

	// UseCount 使用次数
	UseCount int64 `json:"use_count" db:"use_count"`

	// Rating 评分（0-5）
	Rating float64 `json:"rating" db:"rating"`

	// === 时间戳 ===

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ============================================================================
// Skill 辅助方法
// ============================================================================

// IsBuiltinSkill 判断是否为内置技能
func (s *Skill) IsBuiltinSkill() bool {
	return s.IsBuiltin || s.Source == SkillSourceBuiltin
}

// GetTools 获取依赖的工具列表
func (s *Skill) GetTools() []string {
	if len(s.Tools) == 0 {
		return nil
	}
	var tools []string
	json.Unmarshal(s.Tools, &tools)
	return tools
}

// ============================================================================
// SkillRegistry - 技能市场
// ============================================================================

// SkillRegistry 技能市场
//
// 管理一组技能的集合：
//   - builtin：内置技能集
//   - user：用户自定义技能集
//   - community：社区共享技能市场
type SkillRegistry struct {
	// ID 唯一标识
	ID string `json:"id" db:"id"`

	// Name 名称
	Name string `json:"name" db:"name"`

	// Description 描述
	Description string `json:"description" db:"description"`

	// Type 类型
	Type SkillSource `json:"type" db:"type"`

	// OwnerID 所有者（用户或组织）
	OwnerID *string `json:"owner_id,omitempty" db:"owner_id"`

	// IsPublic 是否公开
	IsPublic bool `json:"is_public" db:"is_public"`

	// SkillCount 技能数量
	SkillCount int `json:"skill_count" db:"skill_count"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ============================================================================
// AgentSkill - Agent 与 Skill 关联
// ============================================================================

// AgentSkill Agent 与 Skill 的关联（N:M）
//
// 记录 Agent 拥有的技能以及相关配置。
type AgentSkill struct {
	// AgentID Agent ID
	AgentID string `json:"agent_id" db:"agent_id"`

	// SkillID Skill ID
	SkillID string `json:"skill_id" db:"skill_id"`

	// Enabled 是否启用
	Enabled bool `json:"enabled" db:"enabled"`

	// Config 技能配置覆盖
	// 覆盖 Skill 默认参数
	Config json.RawMessage `json:"config,omitempty" db:"config"`

	// Priority 优先级（数值越小优先级越高）
	Priority int `json:"priority" db:"priority"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ============================================================================
// 内置技能
// ============================================================================

// BuiltinSkills 内置技能列表
var BuiltinSkills = []Skill{
	{
		ID:          "builtin-code-review",
		Name:        "代码审查",
		Category:    SkillCategoryCoding,
		Level:       SkillLevelAdvanced,
		Description: "审查代码质量、发现问题、提出改进建议",
		Instructions: `作为代码审查专家，你需要：
1. 检查代码风格和规范一致性
2. 识别潜在的 bug 和安全问题
3. 评估代码的可读性和可维护性
4. 提出具体的改进建议
5. 关注性能和资源使用`,
		Source:    SkillSourceBuiltin,
		Version:   "1.0.0",
		IsBuiltin: true,
		Tags:      []string{"code", "review", "quality"},
	},
	{
		ID:          "builtin-unit-test",
		Name:        "单元测试编写",
		Category:    SkillCategoryCoding,
		Level:       SkillLevelAdvanced,
		Description: "编写高质量的单元测试，确保代码覆盖率",
		Instructions: `作为测试专家，你需要：
1. 分析被测代码的功能和边界条件
2. 编写测试用例覆盖正常路径和异常路径
3. 使用合适的断言和 mock
4. 确保测试的独立性和可重复性
5. 追求高代码覆盖率`,
		Source:    SkillSourceBuiltin,
		Version:   "1.0.0",
		IsBuiltin: true,
		Tags:      []string{"test", "unit-test", "coverage"},
	},
	{
		ID:          "builtin-doc-writing",
		Name:        "技术文档编写",
		Category:    SkillCategoryWriting,
		Level:       SkillLevelAdvanced,
		Description: "编写清晰、准确的技术文档",
		Instructions: `作为技术文档专家，你需要：
1. 理解目标读者和使用场景
2. 组织清晰的文档结构
3. 使用准确的技术术语
4. 提供代码示例和图表
5. 保持文档的可维护性`,
		Source:    SkillSourceBuiltin,
		Version:   "1.0.0",
		IsBuiltin: true,
		Tags:      []string{"documentation", "writing", "technical"},
	},
	{
		ID:          "builtin-debugging",
		Name:        "问题调试",
		Category:    SkillCategoryCoding,
		Level:       SkillLevelExpert,
		Description: "系统性地定位和解决代码问题",
		Instructions: `作为调试专家，你需要：
1. 收集和分析错误信息
2. 复现问题并隔离变量
3. 使用日志和调试工具定位问题
4. 分析根本原因
5. 提出修复方案并验证`,
		Source:    SkillSourceBuiltin,
		Version:   "1.0.0",
		IsBuiltin: true,
		Tags:      []string{"debug", "troubleshoot", "problem-solving"},
	},
}

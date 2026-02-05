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
// 阶段7：Memory 模型测试
// ============================================================================

// TestMemoryType_Values 验证 MemoryType 枚举值
func TestMemoryType_Values(t *testing.T) {
	types := []MemoryType{
		MemoryTypeWorking,
		MemoryTypeEpisodic,
		MemoryTypeSemantic,
		MemoryTypeProcedural,
	}

	for _, mt := range types {
		assert.NotEmpty(t, string(mt))
	}

	assert.Equal(t, MemoryType("working"), MemoryTypeWorking)
	assert.Equal(t, MemoryType("episodic"), MemoryTypeEpisodic)
	assert.Equal(t, MemoryType("semantic"), MemoryTypeSemantic)
	assert.Equal(t, MemoryType("procedural"), MemoryTypeProcedural)
}

// TestMemory_BasicFields 验证 Memory 基础字段
func TestMemory_BasicFields(t *testing.T) {
	now := time.Now()
	sourceRun := "run-001"

	memory := Memory{
		ID:          "mem-001",
		AgentID:     "agent-001",
		Type:        MemoryTypeSemantic,
		Content:     "用户偏好：喜欢简洁的代码风格",
		Embedding:   []float32{0.1, 0.2, 0.3},
		Metadata:    json.RawMessage(`{"source": "conversation"}`),
		Importance:  0.8,
		Tags:        []string{"preference", "coding-style"},
		SourceRunID: &sourceRun,
		CreatedAt:   now,
		AccessedAt:  now,
	}

	assert.Equal(t, "mem-001", memory.ID)
	assert.Equal(t, "agent-001", memory.AgentID)
	assert.Equal(t, MemoryTypeSemantic, memory.Type)
	assert.Equal(t, "用户偏好：喜欢简洁的代码风格", memory.Content)
	assert.Len(t, memory.Embedding, 3)
	assert.Equal(t, 0.8, memory.Importance)
	assert.Len(t, memory.Tags, 2)
	require.NotNil(t, memory.SourceRunID)
	assert.Equal(t, "run-001", *memory.SourceRunID)
}

// TestMemory_IsExpired 验证记忆过期检查
func TestMemory_IsExpired(t *testing.T) {
	// 未设置过期时间
	m := Memory{ID: "mem-no-expire"}
	assert.False(t, m.IsExpired())

	// 未过期
	future := time.Now().Add(time.Hour)
	m = Memory{ID: "mem-future", ExpiresAt: &future}
	assert.False(t, m.IsExpired())

	// 已过期
	past := time.Now().Add(-time.Hour)
	m = Memory{ID: "mem-past", ExpiresAt: &past}
	assert.True(t, m.IsExpired())
}

// TestMemory_IsWorking 验证工作记忆判断
func TestMemory_IsWorking(t *testing.T) {
	working := Memory{Type: MemoryTypeWorking}
	assert.True(t, working.IsWorking())

	semantic := Memory{Type: MemoryTypeSemantic}
	assert.False(t, semantic.IsWorking())
}

// TestMemory_HasEmbedding 验证向量表示检查
func TestMemory_HasEmbedding(t *testing.T) {
	noEmbed := Memory{Content: "test"}
	assert.False(t, noEmbed.HasEmbedding())

	withEmbed := Memory{Content: "test", Embedding: []float32{0.1, 0.2}}
	assert.True(t, withEmbed.HasEmbedding())
}

// TestMemory_JSONSerialization 验证 Memory JSON 序列化
func TestMemory_JSONSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	expires := now.Add(24 * time.Hour).Truncate(time.Second)
	sourceRun := "run-001"

	memory := Memory{
		ID:          "mem-json-001",
		AgentID:     "agent-001",
		Type:        MemoryTypeEpisodic,
		Content:     "用户说：请帮我优化这段代码",
		Embedding:   []float32{0.1, 0.2, 0.3, 0.4},
		Metadata:    json.RawMessage(`{"turn": 5}`),
		Importance:  0.6,
		Tags:        []string{"conversation"},
		SourceRunID: &sourceRun,
		CreatedAt:   now,
		AccessedAt:  now,
		ExpiresAt:   &expires,
	}

	// 序列化
	data, err := json.Marshal(memory)
	require.NoError(t, err)

	// 反序列化
	var decoded Memory
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// 验证
	assert.Equal(t, memory.ID, decoded.ID)
	assert.Equal(t, memory.AgentID, decoded.AgentID)
	assert.Equal(t, memory.Type, decoded.Type)
	assert.Equal(t, memory.Content, decoded.Content)
	assert.Len(t, decoded.Embedding, 4)
	assert.Equal(t, memory.Importance, decoded.Importance)
	require.NotNil(t, decoded.SourceRunID)
	assert.Equal(t, "run-001", *decoded.SourceRunID)
}

// TestMemoryQuery_BasicFields 验证 MemoryQuery 基础字段
func TestMemoryQuery_BasicFields(t *testing.T) {
	query := MemoryQuery{
		AgentID:       "agent-001",
		Types:         []MemoryType{MemoryTypeSemantic, MemoryTypeEpisodic},
		Query:         "代码风格偏好",
		Tags:          []string{"preference"},
		MinImportance: 0.5,
		Limit:         10,
	}

	assert.Equal(t, "agent-001", query.AgentID)
	assert.Len(t, query.Types, 2)
	assert.Equal(t, "代码风格偏好", query.Query)
	assert.Equal(t, 0.5, query.MinImportance)
	assert.Equal(t, 10, query.Limit)
}

// TestMemoryStats_BasicFields 验证 MemoryStats 基础字段
func TestMemoryStats_BasicFields(t *testing.T) {
	stats := MemoryStats{
		AgentID:    "agent-001",
		TotalCount: 100,
		CountByType: map[MemoryType]int{
			MemoryTypeWorking:    10,
			MemoryTypeEpisodic:   50,
			MemoryTypeSemantic:   30,
			MemoryTypeProcedural: 10,
		},
		TotalSize:     102400,
		AvgImportance: 0.65,
	}

	assert.Equal(t, "agent-001", stats.AgentID)
	assert.Equal(t, 100, stats.TotalCount)
	assert.Equal(t, 50, stats.CountByType[MemoryTypeEpisodic])
	assert.Equal(t, int64(102400), stats.TotalSize)
}

// TestMemory_AllTypes 验证所有记忆类型
func TestMemory_AllTypes(t *testing.T) {
	tests := []struct {
		name      string
		memType   MemoryType
		isWorking bool
		expected  string
	}{
		{
			name:      "working memory",
			memType:   MemoryTypeWorking,
			isWorking: true,
			expected:  "working",
		},
		{
			name:      "episodic memory",
			memType:   MemoryTypeEpisodic,
			isWorking: false,
			expected:  "episodic",
		},
		{
			name:      "semantic memory",
			memType:   MemoryTypeSemantic,
			isWorking: false,
			expected:  "semantic",
		},
		{
			name:      "procedural memory",
			memType:   MemoryTypeProcedural,
			isWorking: false,
			expected:  "procedural",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Memory{Type: tt.memType}
			assert.Equal(t, tt.expected, string(m.Type))
			assert.Equal(t, tt.isWorking, m.IsWorking())
		})
	}
}

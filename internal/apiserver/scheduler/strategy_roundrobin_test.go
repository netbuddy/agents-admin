package scheduler

import (
	"context"
	"testing"

	"agents-admin/internal/shared/model"
)

func TestRoundRobinStrategy_SelectNode(t *testing.T) {
	ctx := context.Background()
	strategy := NewRoundRobinStrategy()

	nodes := []*model.Node{
		createTestNode("node-1", nil, 5),
		createTestNode("node-2", nil, 5),
		createTestNode("node-3", nil, 5),
	}

	req := &ScheduleRequest{
		CandidateNodes: nodes,
		NodeRunning:    map[string]int{},
	}

	// 连续调用应该轮询
	expected := []string{"node-1", "node-2", "node-3", "node-1", "node-2"}
	for i, want := range expected {
		node, _ := strategy.SelectNode(ctx, req)
		if node == nil {
			t.Fatalf("call %d: expected node %s, got nil", i, want)
		}
		if node.ID != want {
			t.Errorf("call %d: expected node %s, got %s", i, want, node.ID)
		}
	}
}

func TestRoundRobinStrategy_Reset(t *testing.T) {
	strategy := NewRoundRobinStrategy()
	ctx := context.Background()

	nodes := []*model.Node{
		createTestNode("node-1", nil, 5),
		createTestNode("node-2", nil, 5),
	}

	req := &ScheduleRequest{
		CandidateNodes: nodes,
		NodeRunning:    map[string]int{},
	}

	// 先调用一次
	strategy.SelectNode(ctx, req)

	// 重置后应该从头开始
	strategy.Reset()

	node, _ := strategy.SelectNode(ctx, req)
	if node == nil || node.ID != "node-1" {
		t.Errorf("after reset, expected node-1, got %v", node)
	}
}

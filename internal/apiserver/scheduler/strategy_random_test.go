package scheduler

import (
	"context"
	"testing"

	"agents-admin/internal/shared/model"
)

func TestRandomStrategy_SelectNode(t *testing.T) {
	ctx := context.Background()
	strategy := NewRandomStrategy()

	nodes := []*model.Node{
		createTestNode("node-1", nil, 5),
		createTestNode("node-2", nil, 5),
	}

	req := &ScheduleRequest{
		CandidateNodes: nodes,
		NodeRunning:    map[string]int{},
	}

	// 多次调用应该返回有效节点
	for i := 0; i < 10; i++ {
		node, _ := strategy.SelectNode(ctx, req)
		if node == nil {
			t.Errorf("call %d: expected non-nil node", i)
		}
	}
}

func TestRandomStrategy_AllNodesFull(t *testing.T) {
	ctx := context.Background()
	strategy := NewRandomStrategy()

	nodes := []*model.Node{
		createTestNode("node-1", nil, 5),
		createTestNode("node-2", nil, 5),
	}

	req := &ScheduleRequest{
		CandidateNodes: nodes,
		NodeRunning:    map[string]int{"node-1": 5, "node-2": 5},
	}

	// 所有节点已满时应返回 nil
	node, _ := strategy.SelectNode(ctx, req)
	if node != nil {
		t.Errorf("expected nil when all nodes full, got %s", node.ID)
	}
}

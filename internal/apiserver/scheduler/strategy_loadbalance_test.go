package scheduler

import (
	"context"
	"testing"

	"agents-admin/internal/shared/model"
)

func TestLoadBalanceStrategy_SelectNode(t *testing.T) {
	ctx := context.Background()
	strategy := NewLoadBalanceStrategy()

	tests := []struct {
		name        string
		nodes       []*model.Node
		nodeRunning map[string]int
		wantNode    string
	}{
		{
			name: "选择容量最大的节点",
			nodes: []*model.Node{
				createTestNode("node-1", nil, 5),
				createTestNode("node-2", nil, 10),
				createTestNode("node-3", nil, 8),
			},
			nodeRunning: map[string]int{"node-1": 2, "node-2": 2, "node-3": 2},
			wantNode:    "node-2",
		},
		{
			name: "所有节点已满",
			nodes: []*model.Node{
				createTestNode("node-1", nil, 2),
				createTestNode("node-2", nil, 3),
			},
			nodeRunning: map[string]int{"node-1": 2, "node-2": 3},
			wantNode:    "",
		},
		{
			name:        "空节点列表",
			nodes:       []*model.Node{},
			nodeRunning: map[string]int{},
			wantNode:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &ScheduleRequest{
				CandidateNodes: tt.nodes,
				NodeRunning:    tt.nodeRunning,
			}

			node, _ := strategy.SelectNode(ctx, req)

			if tt.wantNode == "" {
				if node != nil {
					t.Errorf("expected nil node, got %s", node.ID)
				}
			} else {
				if node == nil {
					t.Errorf("expected node %s, got nil", tt.wantNode)
				} else if node.ID != tt.wantNode {
					t.Errorf("expected node %s, got %s", tt.wantNode, node.ID)
				}
			}
		})
	}
}

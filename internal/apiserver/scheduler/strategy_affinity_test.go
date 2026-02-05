package scheduler

import (
	"context"
	"testing"

	"agents-admin/internal/shared/model"
)

func TestAffinityStrategy_SelectNode(t *testing.T) {
	ctx := context.Background()
	strategy := NewAffinityStrategy()

	tests := []struct {
		name          string
		preferredNode string
		nodes         []*model.Node
		nodeRunning   map[string]int
		wantNode      string
		wantReason    string
	}{
		{
			name:          "选择优先节点",
			preferredNode: "node-1",
			nodes: []*model.Node{
				createTestNode("node-1", nil, 5),
				createTestNode("node-2", nil, 5),
			},
			nodeRunning: map[string]int{"node-1": 2, "node-2": 0},
			wantNode:    "node-1",
			wantReason:  "affinity",
		},
		{
			name:          "优先节点无容量",
			preferredNode: "node-1",
			nodes: []*model.Node{
				createTestNode("node-1", nil, 2),
				createTestNode("node-2", nil, 5),
			},
			nodeRunning: map[string]int{"node-1": 2, "node-2": 0},
			wantNode:    "",
			wantReason:  "affinity_no_capacity",
		},
		{
			name:          "无优先节点",
			preferredNode: "",
			nodes: []*model.Node{
				createTestNode("node-1", nil, 5),
			},
			nodeRunning: map[string]int{},
			wantNode:    "",
			wantReason:  "",
		},
		{
			name:          "优先节点不在候选列表",
			preferredNode: "node-3",
			nodes: []*model.Node{
				createTestNode("node-1", nil, 5),
				createTestNode("node-2", nil, 5),
			},
			nodeRunning: map[string]int{},
			wantNode:    "",
			wantReason:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &ScheduleRequest{
				CandidateNodes: tt.nodes,
				NodeRunning:    tt.nodeRunning,
				PreferredNode:  tt.preferredNode,
			}

			node, reason := strategy.SelectNode(ctx, req)

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

			if reason != tt.wantReason {
				t.Errorf("expected reason %q, got %q", tt.wantReason, reason)
			}
		})
	}
}

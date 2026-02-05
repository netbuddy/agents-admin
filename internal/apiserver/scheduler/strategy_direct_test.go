package scheduler

import (
	"context"
	"encoding/json"
	"testing"

	"agents-admin/internal/shared/model"
)

func TestDirectStrategy_SelectNode(t *testing.T) {
	ctx := context.Background()
	strategy := NewDirectStrategy()

	tests := []struct {
		name        string
		snapshot    map[string]interface{}
		nodes       []*model.Node
		nodeRunning map[string]int
		wantNode    string
		wantReason  string
	}{
		{
			name:     "选择直接指定的节点",
			snapshot: map[string]interface{}{"node_id": "node-1"},
			nodes: []*model.Node{
				createTestNode("node-1", nil, 5),
				createTestNode("node-2", nil, 5),
			},
			nodeRunning: map[string]int{},
			wantNode:    "node-1",
			wantReason:  "direct",
		},
		{
			name:     "支持 target_node 字段",
			snapshot: map[string]interface{}{"target_node": "node-2"},
			nodes: []*model.Node{
				createTestNode("node-1", nil, 5),
				createTestNode("node-2", nil, 5),
			},
			nodeRunning: map[string]int{},
			wantNode:    "node-2",
			wantReason:  "direct",
		},
		{
			name:     "指定节点无容量",
			snapshot: map[string]interface{}{"node_id": "node-1"},
			nodes: []*model.Node{
				createTestNode("node-1", nil, 2),
				createTestNode("node-2", nil, 5),
			},
			nodeRunning: map[string]int{"node-1": 2},
			wantNode:    "",
			wantReason:  "direct_no_capacity",
		},
		{
			name:     "指定节点不存在",
			snapshot: map[string]interface{}{"node_id": "node-3"},
			nodes: []*model.Node{
				createTestNode("node-1", nil, 5),
				createTestNode("node-2", nil, 5),
			},
			nodeRunning: map[string]int{},
			wantNode:    "",
			wantReason:  "direct_node_unavailable",
		},
		{
			name:        "未指定节点",
			snapshot:    map[string]interface{}{},
			nodes:       []*model.Node{createTestNode("node-1", nil, 5)},
			nodeRunning: map[string]int{},
			wantNode:    "",
			wantReason:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshotBytes, _ := json.Marshal(tt.snapshot)
			run := &model.Run{ID: "run-1", Snapshot: snapshotBytes}

			req := &ScheduleRequest{
				Run:            run,
				CandidateNodes: tt.nodes,
				NodeRunning:    tt.nodeRunning,
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

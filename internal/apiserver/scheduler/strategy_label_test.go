package scheduler

import (
	"context"
	"testing"

	"agents-admin/internal/shared/model"
)

func TestLabelMatchStrategy_SelectNode(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		loadBalance bool
		taskLabels  map[string]string
		nodes       []*model.Node
		nodeRunning map[string]int
		wantNode    string
	}{
		{
			name:        "无标签要求_选择第一个有容量的节点",
			loadBalance: false,
			taskLabels:  nil,
			nodes: []*model.Node{
				createTestNode("node-1", map[string]string{"env": "prod"}, 5),
				createTestNode("node-2", map[string]string{"env": "staging"}, 5),
			},
			nodeRunning: map[string]int{},
			wantNode:    "node-1",
		},
		{
			name:        "标签匹配_单节点",
			loadBalance: false,
			taskLabels:  map[string]string{"env": "prod"},
			nodes: []*model.Node{
				createTestNode("node-1", map[string]string{"env": "prod"}, 5),
				createTestNode("node-2", map[string]string{"env": "staging"}, 5),
			},
			nodeRunning: map[string]int{},
			wantNode:    "node-1",
		},
		{
			name:        "标签匹配_多标签",
			loadBalance: false,
			taskLabels:  map[string]string{"env": "prod", "gpu": "true"},
			nodes: []*model.Node{
				createTestNode("node-1", map[string]string{"env": "prod"}, 5),
				createTestNode("node-2", map[string]string{"env": "prod", "gpu": "true"}, 5),
			},
			nodeRunning: map[string]int{},
			wantNode:    "node-2",
		},
		{
			name:        "标签不匹配",
			loadBalance: false,
			taskLabels:  map[string]string{"env": "prod"},
			nodes: []*model.Node{
				createTestNode("node-1", map[string]string{"env": "staging"}, 5),
				createTestNode("node-2", map[string]string{"env": "dev"}, 5),
			},
			nodeRunning: map[string]int{},
			wantNode:    "",
		},
		{
			name:        "启用负载均衡_选择容量最大",
			loadBalance: true,
			taskLabels:  map[string]string{"env": "prod"},
			nodes: []*model.Node{
				createTestNode("node-1", map[string]string{"env": "prod"}, 5),
				createTestNode("node-2", map[string]string{"env": "prod"}, 10),
			},
			nodeRunning: map[string]int{"node-1": 2, "node-2": 2},
			wantNode:    "node-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := NewLabelMatchStrategy(tt.loadBalance)
			req := &ScheduleRequest{
				Task:           createTestTask("task-1", tt.taskLabels),
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

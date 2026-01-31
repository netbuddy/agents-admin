package api

import (
	"encoding/json"
	"testing"

	"agents-admin/internal/model"
)

func TestScheduler_SelectNode(t *testing.T) {
	s := &Scheduler{
		nodeRunning: make(map[string]int),
	}

	// Test with empty nodes
	node := s.selectNode(nil)
	if node != nil {
		t.Error("Expected nil for empty nodes")
	}

	// Test with nodes - should return first one (simple strategy)
	// More complex tests would require mocking the storage layer
}

func TestNewScheduler(t *testing.T) {
	s := NewScheduler(nil, nil)
	if s == nil {
		t.Error("Expected non-nil scheduler")
	}

	if s.stopCh == nil {
		t.Error("Expected stopCh to be initialized")
	}

	if s.nodeRunning == nil {
		t.Error("Expected nodeRunning to be initialized")
	}
}

func TestScheduler_MatchLabels(t *testing.T) {
	s := &Scheduler{
		nodeRunning: make(map[string]int),
	}

	tests := []struct {
		name       string
		nodeLabels map[string]string
		taskLabels map[string]string
		want       bool
	}{
		{
			name:       "no task labels - should match any node",
			nodeLabels: map[string]string{"os": "linux"},
			taskLabels: nil,
			want:       true,
		},
		{
			name:       "empty task labels - should match any node",
			nodeLabels: map[string]string{"os": "linux"},
			taskLabels: map[string]string{},
			want:       true,
		},
		{
			name:       "exact match",
			nodeLabels: map[string]string{"os": "linux", "gpu": "true"},
			taskLabels: map[string]string{"os": "linux"},
			want:       true,
		},
		{
			name:       "multiple labels match",
			nodeLabels: map[string]string{"os": "linux", "gpu": "true", "env": "prod"},
			taskLabels: map[string]string{"os": "linux", "gpu": "true"},
			want:       true,
		},
		{
			name:       "label value mismatch",
			nodeLabels: map[string]string{"os": "linux"},
			taskLabels: map[string]string{"os": "windows"},
			want:       false,
		},
		{
			name:       "label key not found",
			nodeLabels: map[string]string{"os": "linux"},
			taskLabels: map[string]string{"gpu": "true"},
			want:       false,
		},
		{
			name:       "node has no labels",
			nodeLabels: nil,
			taskLabels: map[string]string{"os": "linux"},
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &model.Node{ID: "test-node"}
			if tt.nodeLabels != nil {
				labelsJSON, _ := json.Marshal(tt.nodeLabels)
				node.Labels = labelsJSON
			}

			got := s.matchLabels(node, tt.taskLabels)
			if got != tt.want {
				t.Errorf("matchLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScheduler_GetNodeMaxConcurrent(t *testing.T) {
	s := &Scheduler{
		nodeRunning: make(map[string]int),
	}

	tests := []struct {
		name     string
		capacity map[string]interface{}
		want     int
	}{
		{
			name:     "no capacity - default to 1",
			capacity: nil,
			want:     1,
		},
		{
			name:     "with max_concurrent",
			capacity: map[string]interface{}{"max_concurrent": float64(4)},
			want:     4,
		},
		{
			name:     "with max_concurrent as int",
			capacity: map[string]interface{}{"max_concurrent": 8},
			want:     8,
		},
		{
			name:     "without max_concurrent key",
			capacity: map[string]interface{}{"other": "value"},
			want:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &model.Node{ID: "test-node"}
			if tt.capacity != nil {
				capacityJSON, _ := json.Marshal(tt.capacity)
				node.Capacity = capacityJSON
			}

			got := s.getNodeMaxConcurrent(node)
			if got != tt.want {
				t.Errorf("getNodeMaxConcurrent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScheduler_SelectNodeWithLabels(t *testing.T) {
	s := &Scheduler{
		nodeRunning: make(map[string]int),
	}

	// 创建测试节点
	nodeLinux := &model.Node{ID: "node-linux"}
	nodeLinux.Labels, _ = json.Marshal(map[string]string{"os": "linux"})
	nodeLinux.Capacity, _ = json.Marshal(map[string]interface{}{"max_concurrent": 4})

	nodeWindows := &model.Node{ID: "node-windows"}
	nodeWindows.Labels, _ = json.Marshal(map[string]string{"os": "windows"})
	nodeWindows.Capacity, _ = json.Marshal(map[string]interface{}{"max_concurrent": 2})

	nodeGPU := &model.Node{ID: "node-gpu"}
	nodeGPU.Labels, _ = json.Marshal(map[string]string{"os": "linux", "gpu": "true"})
	nodeGPU.Capacity, _ = json.Marshal(map[string]interface{}{"max_concurrent": 2})

	nodes := []*model.Node{nodeLinux, nodeWindows, nodeGPU}

	tests := []struct {
		name        string
		taskLabels  map[string]string
		nodeRunning map[string]int
		wantNodeID  string
	}{
		{
			name:        "no labels - select node with most capacity",
			taskLabels:  nil,
			nodeRunning: map[string]int{},
			wantNodeID:  "node-linux", // 4 > 2 > 2
		},
		{
			name:        "match linux",
			taskLabels:  map[string]string{"os": "linux"},
			nodeRunning: map[string]int{},
			wantNodeID:  "node-linux", // 4 > 2
		},
		{
			name:        "match windows",
			taskLabels:  map[string]string{"os": "windows"},
			nodeRunning: map[string]int{},
			wantNodeID:  "node-windows",
		},
		{
			name:        "match gpu",
			taskLabels:  map[string]string{"gpu": "true"},
			nodeRunning: map[string]int{},
			wantNodeID:  "node-gpu",
		},
		{
			name:        "no matching labels",
			taskLabels:  map[string]string{"os": "macos"},
			nodeRunning: map[string]int{},
			wantNodeID:  "", // no match
		},
		{
			name:        "capacity exhausted",
			taskLabels:  map[string]string{"os": "windows"},
			nodeRunning: map[string]int{"node-windows": 2}, // 已满
			wantNodeID:  "",                                // no available
		},
		{
			name:        "load balancing - select less loaded node",
			taskLabels:  map[string]string{"os": "linux"},
			nodeRunning: map[string]int{"node-linux": 3, "node-gpu": 0}, // node-linux 只剩 1，node-gpu 还有 2
			wantNodeID:  "node-gpu",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s.nodeRunning = tt.nodeRunning

			got := s.selectNodeWithLabels(nodes, tt.taskLabels)

			if tt.wantNodeID == "" {
				if got != nil {
					t.Errorf("selectNodeWithLabels() = %v, want nil", got.ID)
				}
			} else {
				if got == nil {
					t.Errorf("selectNodeWithLabels() = nil, want %v", tt.wantNodeID)
				} else if got.ID != tt.wantNodeID {
					t.Errorf("selectNodeWithLabels() = %v, want %v", got.ID, tt.wantNodeID)
				}
			}
		})
	}
}

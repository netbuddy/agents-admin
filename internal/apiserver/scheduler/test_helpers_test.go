package scheduler

import (
	"encoding/json"

	"agents-admin/internal/shared/model"
)

// createTestNode 创建测试节点
func createTestNode(id string, labels map[string]string, maxConcurrent int) *model.Node {
	labelsJSON, _ := json.Marshal(labels)
	capacityJSON, _ := json.Marshal(map[string]interface{}{"max_concurrent": maxConcurrent})
	return &model.Node{
		ID:       id,
		Status:   model.NodeStatusOnline,
		Labels:   labelsJSON,
		Capacity: capacityJSON,
	}
}

// createTestTask 创建测试任务
func createTestTask(id string, labels map[string]string) *model.Task {
	return &model.Task{
		ID:     id,
		Labels: labels,
	}
}

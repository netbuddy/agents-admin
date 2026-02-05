// Package redis WorkflowState 缓存操作
package redis

import (
	"context"
	"fmt"
	"strconv"

	"agents-admin/internal/shared/cache"
)

// SetWorkflowState 设置工作流状态
func (s *Store) SetWorkflowState(ctx context.Context, wfType, wfID string, state *cache.WorkflowState) error {
	key := fmt.Sprintf("%s%s:%s", cache.KeyWorkflowState, wfType, wfID)

	data := map[string]interface{}{
		"state":        state.State,
		"progress":     state.Progress,
		"current_step": state.CurrentStep,
		"error":        state.Error,
	}

	pipe := s.client.Pipeline()
	pipe.HSet(ctx, key, data)
	pipe.Expire(ctx, key, cache.TTLWorkflowState)
	_, err := pipe.Exec(ctx)

	return err
}

// GetWorkflowState 获取工作流状态
func (s *Store) GetWorkflowState(ctx context.Context, wfType, wfID string) (*cache.WorkflowState, error) {
	key := fmt.Sprintf("%s%s:%s", cache.KeyWorkflowState, wfType, wfID)

	result, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	state := &cache.WorkflowState{
		State:       result["state"],
		CurrentStep: result["current_step"],
		Error:       result["error"],
	}

	if progress, err := strconv.Atoi(result["progress"]); err == nil {
		state.Progress = progress
	}

	return state, nil
}

// DeleteWorkflowState 删除工作流状态
func (s *Store) DeleteWorkflowState(ctx context.Context, wfType, wfID string) error {
	key := fmt.Sprintf("%s%s:%s", cache.KeyWorkflowState, wfType, wfID)
	return s.client.Del(ctx, key).Err()
}

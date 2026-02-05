// Package repository Operation 和 Action 相关的存储操作
package repository

import (
	"agents-admin/internal/shared/model"
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"time"
)

// === Operation 操作 ===

// CreateOperation 创建 Operation
func (s *Store) CreateOperation(ctx context.Context, op *model.Operation) error {
	query := s.rebind(`
		INSERT INTO operations (id, type, config, status, node_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`)
	_, err := s.db.ExecContext(ctx, query,
		op.ID, op.Type, op.Config, op.Status, op.NodeID,
		op.CreatedAt, op.UpdatedAt)
	return err
}

// GetOperation 获取 Operation
func (s *Store) GetOperation(ctx context.Context, id string) (*model.Operation, error) {
	query := s.rebind(`SELECT id, type, config, status, node_id, created_at, updated_at, finished_at
			  FROM operations WHERE id = $1`)
	op := &model.Operation{}
	var config *[]byte
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&op.ID, &op.Type, &config, &op.Status, &op.NodeID,
		&op.CreatedAt, &op.UpdatedAt, &op.FinishedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if config != nil {
		op.Config = *config
	}
	return op, nil
}

// ListOperations 列出 Operation（支持类型和状态过滤）
func (s *Store) ListOperations(ctx context.Context, opType string, status string, limit, offset int) ([]*model.Operation, error) {
	query := `SELECT id, type, config, status, node_id, created_at, updated_at, finished_at
			  FROM operations WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if opType != "" {
		query += ` AND type = $` + strconv.Itoa(argIdx)
		args = append(args, opType)
		argIdx++
	}
	if status != "" {
		query += ` AND status = $` + strconv.Itoa(argIdx)
		args = append(args, status)
		argIdx++
	}

	query += ` ORDER BY created_at DESC`

	if limit > 0 {
		query += ` LIMIT $` + strconv.Itoa(argIdx)
		args = append(args, limit)
		argIdx++
	}
	if offset > 0 {
		query += ` OFFSET $` + strconv.Itoa(argIdx)
		args = append(args, offset)
	}

	rows, err := s.db.QueryContext(ctx, s.rebind(query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ops []*model.Operation
	for rows.Next() {
		op := &model.Operation{}
		var config *[]byte
		if err := rows.Scan(&op.ID, &op.Type, &config, &op.Status, &op.NodeID,
			&op.CreatedAt, &op.UpdatedAt, &op.FinishedAt); err != nil {
			return nil, err
		}
		if config != nil {
			op.Config = *config
		}
		ops = append(ops, op)
	}
	return ops, rows.Err()
}

// UpdateOperationStatus 更新 Operation 状态
func (s *Store) UpdateOperationStatus(ctx context.Context, id string, status model.OperationStatus) error {
	var query string
	if status == model.OperationStatusCompleted || status == model.OperationStatusFailed || status == model.OperationStatusCancelled {
		now := time.Now()
		query = s.rebind(`UPDATE operations SET status = $1, finished_at = $2 WHERE id = $3`)
		_, err := s.db.ExecContext(ctx, query, status, now, id)
		return err
	}
	query = s.rebind(`UPDATE operations SET status = $1 WHERE id = $2`)
	_, err := s.db.ExecContext(ctx, query, status, id)
	return err
}

// === Action 操作 ===

// CreateAction 创建 Action
func (s *Store) CreateAction(ctx context.Context, action *model.Action) error {
	query := s.rebind(`
		INSERT INTO actions (id, operation_id, status, phase, message, progress, result, error, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`)
	_, err := s.db.ExecContext(ctx, query,
		action.ID, action.OperationID, action.Status, action.Phase, action.Message,
		action.Progress, action.Result, action.Error, action.CreatedAt)
	return err
}

// GetAction 获取 Action
func (s *Store) GetAction(ctx context.Context, id string) (*model.Action, error) {
	query := s.rebind(`SELECT id, operation_id, status, phase, message, progress, result, error, created_at, started_at, finished_at
			  FROM actions WHERE id = $1`)
	action := &model.Action{}
	var result *[]byte
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&action.ID, &action.OperationID, &action.Status, &action.Phase, &action.Message,
		&action.Progress, &result, &action.Error, &action.CreatedAt, &action.StartedAt, &action.FinishedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if result != nil {
		action.Result = *result
	}
	return action, nil
}

// GetActionWithOperation 获取 Action 并关联 Operation
func (s *Store) GetActionWithOperation(ctx context.Context, id string) (*model.Action, error) {
	query := s.rebind(`
		SELECT a.id, a.operation_id, a.status, a.phase, a.message, a.progress, a.result, a.error, a.created_at, a.started_at, a.finished_at,
		       o.id, o.type, o.config, o.status, o.node_id, o.created_at, o.updated_at, o.finished_at
		FROM actions a
		JOIN operations o ON a.operation_id = o.id
		WHERE a.id = $1
	`)
	action := &model.Action{}
	op := &model.Operation{}
	var actionResult, opConfig *[]byte
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&action.ID, &action.OperationID, &action.Status, &action.Phase, &action.Message,
		&action.Progress, &actionResult, &action.Error, &action.CreatedAt, &action.StartedAt, &action.FinishedAt,
		&op.ID, &op.Type, &opConfig, &op.Status, &op.NodeID,
		&op.CreatedAt, &op.UpdatedAt, &op.FinishedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if actionResult != nil {
		action.Result = *actionResult
	}
	if opConfig != nil {
		op.Config = *opConfig
	}
	action.Operation = op
	return action, nil
}

// ListActionsByOperation 列出 Operation 的所有 Action
func (s *Store) ListActionsByOperation(ctx context.Context, operationID string) ([]*model.Action, error) {
	query := s.rebind(`SELECT id, operation_id, status, phase, message, progress, result, error, created_at, started_at, finished_at
			  FROM actions WHERE operation_id = $1 ORDER BY created_at DESC`)
	rows, err := s.db.QueryContext(ctx, query, operationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []*model.Action
	for rows.Next() {
		action := &model.Action{}
		var result *[]byte
		if err := rows.Scan(&action.ID, &action.OperationID, &action.Status, &action.Phase, &action.Message,
			&action.Progress, &result, &action.Error, &action.CreatedAt, &action.StartedAt, &action.FinishedAt); err != nil {
			return nil, err
		}
		if result != nil {
			action.Result = *result
		}
		actions = append(actions, action)
	}
	return actions, rows.Err()
}

// ListActionsByNode 列出分配给节点的 Action
func (s *Store) ListActionsByNode(ctx context.Context, nodeID string, status string) ([]*model.Action, error) {
	query := `
		SELECT a.id, a.operation_id, a.status, a.phase, a.message, a.progress, a.result, a.error, a.created_at, a.started_at, a.finished_at,
		       o.id, o.type, o.config, o.status, o.node_id, o.created_at, o.updated_at, o.finished_at
		FROM actions a
		JOIN operations o ON a.operation_id = o.id
		WHERE o.node_id = $1
	`
	args := []interface{}{nodeID}
	argIdx := 2

	if status != "" {
		query += ` AND a.status = $` + strconv.Itoa(argIdx)
		args = append(args, status)
	}

	query += ` ORDER BY a.created_at ASC`

	rows, err := s.db.QueryContext(ctx, s.rebind(query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []*model.Action
	for rows.Next() {
		action := &model.Action{}
		op := &model.Operation{}
		var actionResult, opConfig *[]byte
		if err := rows.Scan(
			&action.ID, &action.OperationID, &action.Status, &action.Phase, &action.Message,
			&action.Progress, &actionResult, &action.Error, &action.CreatedAt, &action.StartedAt, &action.FinishedAt,
			&op.ID, &op.Type, &opConfig, &op.Status, &op.NodeID,
			&op.CreatedAt, &op.UpdatedAt, &op.FinishedAt); err != nil {
			return nil, err
		}
		if actionResult != nil {
			action.Result = *actionResult
		}
		if opConfig != nil {
			op.Config = *opConfig
		}
		action.Operation = op
		actions = append(actions, action)
	}
	return actions, rows.Err()
}

// UpdateActionStatus 更新 Action 状态
func (s *Store) UpdateActionStatus(ctx context.Context, id string, status model.ActionStatus, phase model.ActionPhase, message string, progress int, result json.RawMessage, errMsg string) error {
	now := time.Now()
	var query string

	if status == model.ActionStatusRunning {
		query = s.rebind(`UPDATE actions SET status = $1, phase = $2, message = $3, progress = $4, result = $5, error = $6, started_at = $7 WHERE id = $8`)
		_, err := s.db.ExecContext(ctx, query, status, phase, message, progress, result, errMsg, now, id)
		return err
	}
	if status.IsTerminal() {
		query = s.rebind(`UPDATE actions SET status = $1, phase = $2, message = $3, progress = $4, result = $5, error = $6, finished_at = $7 WHERE id = $8`)
		_, err := s.db.ExecContext(ctx, query, status, phase, message, progress, result, errMsg, now, id)
		return err
	}

	query = s.rebind(`UPDATE actions SET status = $1, phase = $2, message = $3, progress = $4, result = $5, error = $6 WHERE id = $7`)
	_, err := s.db.ExecContext(ctx, query, status, phase, message, progress, result, errMsg, id)
	return err
}

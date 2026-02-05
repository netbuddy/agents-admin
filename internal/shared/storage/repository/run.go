// Package repository Run 相关的存储操作
package repository

import (
	"context"
	"database/sql"
	"time"

	"agents-admin/internal/shared/model"
)

// CreateRun 创建 Run
func (s *Store) CreateRun(ctx context.Context, run *model.Run) error {
	query := s.rebind(`
		INSERT INTO runs (id, task_id, status, node_id, started_at, finished_at, snapshot, error, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`)
	_, err := s.db.ExecContext(ctx, query,
		run.ID, run.TaskID, run.Status, run.NodeID, run.StartedAt, run.FinishedAt,
		run.Snapshot, run.Error, run.CreatedAt, run.UpdatedAt)
	return err
}

// GetRun 获取 Run
func (s *Store) GetRun(ctx context.Context, id string) (*model.Run, error) {
	query := s.rebind(`SELECT id, task_id, status, node_id, started_at, finished_at, snapshot, error, created_at, updated_at 
			  FROM runs WHERE id = $1`)
	row := s.db.QueryRowContext(ctx, query, id)
	run, err := scanRun(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return run, err
}

// scanRun 辅助函数
func scanRun(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.Run, error) {
	run := &model.Run{}
	var snapshot *[]byte
	err := scanner.Scan(
		&run.ID, &run.TaskID, &run.Status, &run.NodeID, &run.StartedAt,
		&run.FinishedAt, &snapshot, &run.Error, &run.CreatedAt, &run.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if snapshot != nil {
		run.Snapshot = *snapshot
	}
	return run, nil
}

// scanRuns 批量扫描
func scanRuns(rows *sql.Rows) ([]*model.Run, error) {
	var runs []*model.Run
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

// ListRunsByTask 列出任务的所有 Run
func (s *Store) ListRunsByTask(ctx context.Context, taskID string) ([]*model.Run, error) {
	query := s.rebind(`SELECT id, task_id, status, node_id, started_at, finished_at, snapshot, error, created_at, updated_at 
			  FROM runs WHERE task_id = $1 ORDER BY created_at DESC`)
	rows, err := s.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRuns(rows)
}

// ListRunsByNode 列出分配给节点的活跃 Run
func (s *Store) ListRunsByNode(ctx context.Context, nodeID string) ([]*model.Run, error) {
	query := s.rebind(`SELECT id, task_id, status, node_id, started_at, finished_at, snapshot, error, created_at, updated_at 
			  FROM runs WHERE node_id = $1 AND status IN ('assigned', 'running') ORDER BY created_at ASC`)
	rows, err := s.db.QueryContext(ctx, query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRuns(rows)
}

// ListRunningRuns 列出所有活跃的 Run
func (s *Store) ListRunningRuns(ctx context.Context, limit int) ([]*model.Run, error) {
	if limit <= 0 {
		limit = 100
	}
	var query string
	if s.dialect.SupportsNullsLast() {
		query = s.rebind(`SELECT id, task_id, status, node_id, started_at, finished_at, snapshot, error, created_at, updated_at
			  FROM runs WHERE status IN ('assigned', 'running') ORDER BY started_at ASC ` + s.dialect.NullsLastClause() + `, created_at ASC LIMIT $1`)
	} else {
		// SQLite/MySQL: 用 CASE 模拟 NULLS LAST
		query = s.rebind(`SELECT id, task_id, status, node_id, started_at, finished_at, snapshot, error, created_at, updated_at
			  FROM runs WHERE status IN ('assigned', 'running') ORDER BY CASE WHEN started_at IS NULL THEN 1 ELSE 0 END, started_at ASC, created_at ASC LIMIT $1`)
	}
	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRuns(rows)
}

// ListQueuedRuns 列出待执行的 Run
func (s *Store) ListQueuedRuns(ctx context.Context, limit int) ([]*model.Run, error) {
	query := s.rebind(`SELECT id, task_id, status, node_id, started_at, finished_at, snapshot, error, created_at, updated_at 
			  FROM runs WHERE status = 'queued' ORDER BY created_at ASC LIMIT $1`)
	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRuns(rows)
}

// ListStaleQueuedRuns 列出"过期"的 queued 状态 Run
func (s *Store) ListStaleQueuedRuns(ctx context.Context, threshold time.Duration) ([]*model.Run, error) {
	cutoff := time.Now().Add(-threshold)
	query := s.rebind(`SELECT id, task_id, status, node_id, started_at, finished_at, snapshot, error, created_at, updated_at 
			  FROM runs 
			  WHERE status = 'queued' AND created_at < $1 
			  ORDER BY created_at ASC 
			  LIMIT 100`)
	rows, err := s.db.QueryContext(ctx, query, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRuns(rows)
}

// ResetRunToQueued 将已分配的 Run 重置为 queued
func (s *Store) ResetRunToQueued(ctx context.Context, id string) error {
	query := s.rebind(`UPDATE runs 
			  SET status = 'queued', node_id = NULL, started_at = NULL, error = NULL, updated_at = $2
			  WHERE id = $1 AND status IN ('assigned', 'running')`)
	_, err := s.db.ExecContext(ctx, query, id, time.Now())
	return err
}

// UpdateRunStatus 更新 Run 状态
func (s *Store) UpdateRunStatus(ctx context.Context, id string, status model.RunStatus, nodeID *string) error {
	var query string
	var args []interface{}
	switch status {
	case model.RunStatusAssigned:
		query = s.rebind(`UPDATE runs SET status = $1, node_id = $2, updated_at = $3 WHERE id = $4`)
		args = []interface{}{status, nodeID, time.Now(), id}
	case model.RunStatusRunning:
		now := time.Now()
		query = s.rebind(`UPDATE runs SET status = $1, started_at = $2, updated_at = $3 WHERE id = $4`)
		args = []interface{}{status, now, now, id}
	case model.RunStatusDone, model.RunStatusFailed, model.RunStatusCancelled, model.RunStatusTimeout:
		now := time.Now()
		query = s.rebind(`UPDATE runs SET status = $1, finished_at = $2, updated_at = $3 WHERE id = $4`)
		args = []interface{}{status, now, now, id}
	default:
		query = s.rebind(`UPDATE runs SET status = $1, updated_at = $2 WHERE id = $3`)
		args = []interface{}{status, time.Now(), id}
	}

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	// 当 Run 开始执行时，更新 Task 状态为 in_progress
	if status == model.RunStatusRunning {
		var taskID string
		err = s.db.QueryRowContext(ctx, s.rebind(`SELECT task_id FROM runs WHERE id = $1`), id).Scan(&taskID)
		if err == nil {
			_, _ = s.db.ExecContext(ctx,
				s.rebind(`UPDATE tasks SET status = $1, updated_at = $2 WHERE id = $3 AND status = 'pending'`),
				model.TaskStatusInProgress, time.Now(), taskID)
		}
	}

	// 当 Run 完成或失败时，更新关联的 Task 状态
	if status == model.RunStatusDone || status == model.RunStatusFailed ||
		status == model.RunStatusCancelled || status == model.RunStatusTimeout {
		var taskID string
		err = s.db.QueryRowContext(ctx, s.rebind(`SELECT task_id FROM runs WHERE id = $1`), id).Scan(&taskID)
		if err != nil {
			return nil
		}

		var taskStatus model.TaskStatus
		switch status {
		case model.RunStatusDone:
			taskStatus = model.TaskStatusCompleted
		case model.RunStatusFailed:
			taskStatus = model.TaskStatusFailed
		case model.RunStatusCancelled:
			taskStatus = model.TaskStatusCancelled
		case model.RunStatusTimeout:
			taskStatus = model.TaskStatusFailed
		}

		_, _ = s.db.ExecContext(ctx,
			s.rebind(`UPDATE tasks SET status = $1, updated_at = $2 WHERE id = $3`),
			taskStatus, time.Now(), taskID)
	}

	return nil
}

// UpdateRunError 更新 Run 错误信息
func (s *Store) UpdateRunError(ctx context.Context, id string, errMsg string) error {
	query := s.rebind(`UPDATE runs SET error = $1, status = 'failed', finished_at = $2 WHERE id = $3`)
	_, err := s.db.ExecContext(ctx, query, errMsg, time.Now(), id)
	return err
}

// DeleteRun 删除 Run
func (s *Store) DeleteRun(ctx context.Context, id string) error {
	query := s.rebind(`DELETE FROM runs WHERE id = $1`)
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

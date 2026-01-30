// Package storage 提供数据存储层
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"agents-admin/internal/model"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresStore PostgreSQL 存储
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore 创建 PostgreSQL 存储
func NewPostgresStore(databaseURL string) (*PostgresStore, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresStore{db: db}, nil
}

// Close 关闭连接
func (s *PostgresStore) Close() error {
	return s.db.Close()
}

// === Task 操作 ===

// CreateTask 创建任务
func (s *PostgresStore) CreateTask(ctx context.Context, task *model.Task) error {
	query := `
		INSERT INTO tasks (id, name, status, spec, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := s.db.ExecContext(ctx, query,
		task.ID, task.Name, task.Status, task.Spec, task.CreatedAt, task.UpdatedAt)
	return err
}

// GetTask 获取任务
func (s *PostgresStore) GetTask(ctx context.Context, id string) (*model.Task, error) {
	query := `SELECT id, name, status, spec, created_at, updated_at FROM tasks WHERE id = $1`
	task := &model.Task{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID, &task.Name, &task.Status, &task.Spec, &task.CreatedAt, &task.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return task, err
}

// ListTasks 列出任务
func (s *PostgresStore) ListTasks(ctx context.Context, status string, limit, offset int) ([]*model.Task, error) {
	var query string
	var args []interface{}

	if status != "" {
		query = `SELECT id, name, status, spec, created_at, updated_at 
				 FROM tasks WHERE status = $1 
				 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		args = []interface{}{status, limit, offset}
	} else {
		query = `SELECT id, name, status, spec, created_at, updated_at 
				 FROM tasks ORDER BY created_at DESC LIMIT $1 OFFSET $2`
		args = []interface{}{limit, offset}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*model.Task
	for rows.Next() {
		task := &model.Task{}
		if err := rows.Scan(&task.ID, &task.Name, &task.Status, &task.Spec,
			&task.CreatedAt, &task.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

// UpdateTaskStatus 更新任务状态
func (s *PostgresStore) UpdateTaskStatus(ctx context.Context, id string, status model.TaskStatus) error {
	query := `UPDATE tasks SET status = $1 WHERE id = $2`
	_, err := s.db.ExecContext(ctx, query, status, id)
	return err
}

// DeleteTask 删除任务（级联删除关联的 runs 和 events）
func (s *PostgresStore) DeleteTask(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 先删除关联的 events（通过 run_id）
	_, err = tx.ExecContext(ctx, `DELETE FROM events WHERE run_id IN (SELECT id FROM runs WHERE task_id = $1)`, id)
	if err != nil {
		return err
	}

	// 删除关联的 artifacts
	_, err = tx.ExecContext(ctx, `DELETE FROM artifacts WHERE run_id IN (SELECT id FROM runs WHERE task_id = $1)`, id)
	if err != nil {
		return err
	}

	// 删除关联的 runs
	_, err = tx.ExecContext(ctx, `DELETE FROM runs WHERE task_id = $1`, id)
	if err != nil {
		return err
	}

	// 最后删除 task
	_, err = tx.ExecContext(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// === Run 操作 ===

// CreateRun 创建 Run
func (s *PostgresStore) CreateRun(ctx context.Context, run *model.Run) error {
	query := `
		INSERT INTO runs (id, task_id, status, node_id, started_at, finished_at, snapshot, error, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := s.db.ExecContext(ctx, query,
		run.ID, run.TaskID, run.Status, run.NodeID, run.StartedAt, run.FinishedAt,
		run.Snapshot, run.Error, run.CreatedAt, run.UpdatedAt)
	return err
}

// GetRun 获取 Run
func (s *PostgresStore) GetRun(ctx context.Context, id string) (*model.Run, error) {
	query := `SELECT id, task_id, status, node_id, started_at, finished_at, snapshot, error, created_at, updated_at 
			  FROM runs WHERE id = $1`
	run := &model.Run{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&run.ID, &run.TaskID, &run.Status, &run.NodeID, &run.StartedAt, &run.FinishedAt,
		&run.Snapshot, &run.Error, &run.CreatedAt, &run.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return run, err
}

// ListRunsByTask 列出任务的所有 Run
func (s *PostgresStore) ListRunsByTask(ctx context.Context, taskID string) ([]*model.Run, error) {
	query := `SELECT id, task_id, status, node_id, started_at, finished_at, snapshot, error, created_at, updated_at 
			  FROM runs WHERE task_id = $1 ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*model.Run
	for rows.Next() {
		run := &model.Run{}
		if err := rows.Scan(&run.ID, &run.TaskID, &run.Status, &run.NodeID, &run.StartedAt,
			&run.FinishedAt, &run.Snapshot, &run.Error, &run.CreatedAt, &run.UpdatedAt); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

// ListRunsByNode 列出分配给节点的 Run（running 状态）
func (s *PostgresStore) ListRunsByNode(ctx context.Context, nodeID string) ([]*model.Run, error) {
	query := `SELECT id, task_id, status, node_id, started_at, finished_at, snapshot, error, created_at, updated_at 
			  FROM runs WHERE node_id = $1 AND status = 'running' ORDER BY created_at ASC`
	rows, err := s.db.QueryContext(ctx, query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*model.Run
	for rows.Next() {
		run := &model.Run{}
		if err := rows.Scan(&run.ID, &run.TaskID, &run.Status, &run.NodeID, &run.StartedAt,
			&run.FinishedAt, &run.Snapshot, &run.Error, &run.CreatedAt, &run.UpdatedAt); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

// ListQueuedRuns 列出待执行的 Run
func (s *PostgresStore) ListQueuedRuns(ctx context.Context, limit int) ([]*model.Run, error) {
	query := `SELECT id, task_id, status, node_id, started_at, finished_at, snapshot, error, created_at, updated_at 
			  FROM runs WHERE status = 'queued' ORDER BY created_at ASC LIMIT $1`
	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*model.Run
	for rows.Next() {
		run := &model.Run{}
		if err := rows.Scan(&run.ID, &run.TaskID, &run.Status, &run.NodeID, &run.StartedAt,
			&run.FinishedAt, &run.Snapshot, &run.Error, &run.CreatedAt, &run.UpdatedAt); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

// UpdateRunStatus 更新 Run 状态
// 当 Run 完成或失败时，同时更新关联的 Task 状态
func (s *PostgresStore) UpdateRunStatus(ctx context.Context, id string, status model.RunStatus, nodeID *string) error {
	var query string
	var args []interface{}

	if status == model.RunStatusRunning {
		now := time.Now()
		query = `UPDATE runs SET status = $1, node_id = $2, started_at = $3 WHERE id = $4`
		args = []interface{}{status, nodeID, now, id}
	} else if status == model.RunStatusDone || status == model.RunStatusFailed ||
		status == model.RunStatusCancelled || status == model.RunStatusTimeout {
		now := time.Now()
		query = `UPDATE runs SET status = $1, finished_at = $2 WHERE id = $3`
		args = []interface{}{status, now, id}
	} else {
		query = `UPDATE runs SET status = $1 WHERE id = $2`
		args = []interface{}{status, id}
	}

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	// 当 Run 完成或失败时，更新关联的 Task 状态
	if status == model.RunStatusDone || status == model.RunStatusFailed ||
		status == model.RunStatusCancelled || status == model.RunStatusTimeout {
		// 获取 Run 关联的 task_id
		var taskID string
		err = s.db.QueryRowContext(ctx, `SELECT task_id FROM runs WHERE id = $1`, id).Scan(&taskID)
		if err != nil {
			return nil // 不影响 Run 状态更新
		}

		// 根据 Run 状态映射 Task 状态
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

		// 更新 Task 状态
		_, err = s.db.ExecContext(ctx,
			`UPDATE tasks SET status = $1, updated_at = $2 WHERE id = $3`,
			taskStatus, time.Now(), taskID)
		// 忽略 Task 更新错误，Run 状态已更新成功
	}

	return nil
}

// UpdateRunError 更新 Run 错误信息
func (s *PostgresStore) UpdateRunError(ctx context.Context, id string, errMsg string) error {
	query := `UPDATE runs SET error = $1, status = 'failed', finished_at = $2 WHERE id = $3`
	_, err := s.db.ExecContext(ctx, query, errMsg, time.Now(), id)
	return err
}

// === Event 操作 ===

// CreateEvents 批量创建事件
func (s *PostgresStore) CreateEvents(ctx context.Context, events []*model.Event) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO events (run_id, seq, type, timestamp, payload, raw) VALUES ($1, $2, $3, $4, $5, $6)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range events {
		_, err := stmt.ExecContext(ctx, e.RunID, e.Seq, e.Type, e.Timestamp, e.Payload, e.Raw)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetEventsByRun 获取 Run 的事件
func (s *PostgresStore) GetEventsByRun(ctx context.Context, runID string, fromSeq int, limit int) ([]*model.Event, error) {
	query := `SELECT id, run_id, seq, type, timestamp, payload, raw 
			  FROM events WHERE run_id = $1 AND seq > $2 ORDER BY seq ASC LIMIT $3`
	rows, err := s.db.QueryContext(ctx, query, runID, fromSeq, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*model.Event
	for rows.Next() {
		e := &model.Event{}
		if err := rows.Scan(&e.ID, &e.RunID, &e.Seq, &e.Type, &e.Timestamp, &e.Payload, &e.Raw); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// === Node 操作 ===

// UpsertNode 更新或插入节点
func (s *PostgresStore) UpsertNode(ctx context.Context, node *model.Node) error {
	query := `
		INSERT INTO nodes (id, status, labels, capacity, last_heartbeat, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			labels = EXCLUDED.labels,
			capacity = EXCLUDED.capacity,
			last_heartbeat = EXCLUDED.last_heartbeat
	`
	_, err := s.db.ExecContext(ctx, query,
		node.ID, node.Status, node.Labels, node.Capacity,
		node.LastHeartbeat, node.CreatedAt, node.UpdatedAt)
	return err
}

// GetNode 获取节点
func (s *PostgresStore) GetNode(ctx context.Context, id string) (*model.Node, error) {
	query := `SELECT id, status, labels, capacity, last_heartbeat, created_at, updated_at FROM nodes WHERE id = $1`
	node := &model.Node{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&node.ID, &node.Status, &node.Labels, &node.Capacity,
		&node.LastHeartbeat, &node.CreatedAt, &node.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return node, err
}

// ListAllNodes 列出所有节点（不过滤状态）
func (s *PostgresStore) ListAllNodes(ctx context.Context) ([]*model.Node, error) {
	query := `SELECT id, status, labels, capacity, last_heartbeat, created_at, updated_at 
			  FROM nodes ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*model.Node
	for rows.Next() {
		node := &model.Node{}
		if err := rows.Scan(&node.ID, &node.Status, &node.Labels, &node.Capacity,
			&node.LastHeartbeat, &node.CreatedAt, &node.UpdatedAt); err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, rows.Err()
}

// ListOnlineNodes 列出在线节点（已废弃，使用 ListAllNodes + etcd 心跳判断）
func (s *PostgresStore) ListOnlineNodes(ctx context.Context) ([]*model.Node, error) {
	query := `SELECT id, status, labels, capacity, last_heartbeat, created_at, updated_at 
			  FROM nodes WHERE status = 'online' ORDER BY last_heartbeat DESC`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*model.Node
	for rows.Next() {
		node := &model.Node{}
		if err := rows.Scan(&node.ID, &node.Status, &node.Labels, &node.Capacity,
			&node.LastHeartbeat, &node.CreatedAt, &node.UpdatedAt); err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, rows.Err()
}

// DeleteNode 删除节点
func (s *PostgresStore) DeleteNode(ctx context.Context, id string) error {
	query := `DELETE FROM nodes WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

func parseTaskSpec(spec json.RawMessage) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(spec, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// === Account 操作 ===

// CreateAccount 创建账号
func (s *PostgresStore) CreateAccount(ctx context.Context, account *model.Account) error {
	query := `
		INSERT INTO accounts (id, name, agent_type_id, node_id, volume_name, status, created_at, updated_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := s.db.ExecContext(ctx, query,
		account.ID, account.Name, account.AgentTypeID, account.NodeID, account.VolumeName,
		account.Status, account.CreatedAt, account.UpdatedAt, account.LastUsedAt)
	return err
}

// GetAccount 获取账号
func (s *PostgresStore) GetAccount(ctx context.Context, id string) (*model.Account, error) {
	query := `SELECT id, name, agent_type_id, node_id, volume_name, status, created_at, updated_at, last_used_at 
			  FROM accounts WHERE id = $1`
	account := &model.Account{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&account.ID, &account.Name, &account.AgentTypeID, &account.NodeID, &account.VolumeName,
		&account.Status, &account.CreatedAt, &account.UpdatedAt, &account.LastUsedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return account, err
}

// ListAccounts 列出账号
func (s *PostgresStore) ListAccounts(ctx context.Context) ([]*model.Account, error) {
	query := `SELECT id, name, agent_type_id, node_id, volume_name, status, created_at, updated_at, last_used_at 
			  FROM accounts ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*model.Account
	for rows.Next() {
		account := &model.Account{}
		if err := rows.Scan(&account.ID, &account.Name, &account.AgentTypeID, &account.NodeID, &account.VolumeName,
			&account.Status, &account.CreatedAt, &account.UpdatedAt, &account.LastUsedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

// ListAccountsByNode 列出指定节点的账号
func (s *PostgresStore) ListAccountsByNode(ctx context.Context, nodeID string) ([]*model.Account, error) {
	query := `SELECT id, name, agent_type_id, node_id, volume_name, status, created_at, updated_at, last_used_at 
			  FROM accounts WHERE node_id = $1 ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*model.Account
	for rows.Next() {
		account := &model.Account{}
		if err := rows.Scan(&account.ID, &account.Name, &account.AgentTypeID, &account.NodeID, &account.VolumeName,
			&account.Status, &account.CreatedAt, &account.UpdatedAt, &account.LastUsedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

// UpdateAccountStatus 更新账号状态
func (s *PostgresStore) UpdateAccountStatus(ctx context.Context, id string, status model.AccountStatus) error {
	query := `UPDATE accounts SET status = $1 WHERE id = $2`
	_, err := s.db.ExecContext(ctx, query, status, id)
	return err
}

// UpdateAccountVolume 更新账号的 Volume 名称
func (s *PostgresStore) UpdateAccountVolume(ctx context.Context, id string, volumeName string) error {
	query := `UPDATE accounts SET volume_name = $1 WHERE id = $2`
	_, err := s.db.ExecContext(ctx, query, volumeName, id)
	return err
}

// DeleteAccount 删除账号
func (s *PostgresStore) DeleteAccount(ctx context.Context, id string) error {
	query := `DELETE FROM accounts WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// === AuthTask 操作 ===

// CreateAuthTask 创建认证任务
func (s *PostgresStore) CreateAuthTask(ctx context.Context, task *model.AuthTask) error {
	status := string(task.Status)
	if status == "" {
		status = "pending"
	}
	query := `
		INSERT INTO auth_tasks (id, account_id, method, node_id, status, terminal_port, terminal_url, container_name, message, created_at, updated_at, expires_at)
		VALUES ($1, $2, $3, $4, $5::varchar, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := s.db.ExecContext(ctx, query,
		task.ID, task.AccountID, task.Method, task.NodeID, status,
		task.TerminalPort, task.TerminalURL, task.ContainerName, task.Message,
		task.CreatedAt, task.UpdatedAt, task.ExpiresAt)
	return err
}

// GetAuthTask 获取认证任务
func (s *PostgresStore) GetAuthTask(ctx context.Context, id string) (*model.AuthTask, error) {
	query := `SELECT id, account_id, method, node_id, status, terminal_port, terminal_url, container_name, message, created_at, updated_at, expires_at 
			  FROM auth_tasks WHERE id = $1`
	task := &model.AuthTask{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID, &task.AccountID, &task.Method, &task.NodeID, &task.Status,
		&task.TerminalPort, &task.TerminalURL, &task.ContainerName, &task.Message,
		&task.CreatedAt, &task.UpdatedAt, &task.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return task, err
}

// GetAuthTaskByAccountID 根据账号 ID 获取最新的认证任务
func (s *PostgresStore) GetAuthTaskByAccountID(ctx context.Context, accountID string) (*model.AuthTask, error) {
	query := `SELECT id, account_id, method, node_id, status, terminal_port, terminal_url, container_name, message, created_at, updated_at, expires_at 
			  FROM auth_tasks WHERE account_id = $1 ORDER BY created_at DESC LIMIT 1`
	task := &model.AuthTask{}
	err := s.db.QueryRowContext(ctx, query, accountID).Scan(
		&task.ID, &task.AccountID, &task.Method, &task.NodeID, &task.Status,
		&task.TerminalPort, &task.TerminalURL, &task.ContainerName, &task.Message,
		&task.CreatedAt, &task.UpdatedAt, &task.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return task, err
}

// ListRecentAuthTasks 列出最近的认证任务（用于监控）
func (s *PostgresStore) ListRecentAuthTasks(ctx context.Context, limit int) ([]*model.AuthTask, error) {
	query := `SELECT id, account_id, method, node_id, status, terminal_port, terminal_url, container_name, message, created_at, updated_at, expires_at 
			  FROM auth_tasks ORDER BY created_at DESC LIMIT $1`
	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*model.AuthTask
	for rows.Next() {
		task := &model.AuthTask{}
		if err := rows.Scan(&task.ID, &task.AccountID, &task.Method, &task.NodeID, &task.Status,
			&task.TerminalPort, &task.TerminalURL, &task.ContainerName, &task.Message,
			&task.CreatedAt, &task.UpdatedAt, &task.ExpiresAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

// ListPendingAuthTasks 列出待调度的认证任务
func (s *PostgresStore) ListPendingAuthTasks(ctx context.Context, limit int) ([]*model.AuthTask, error) {
	query := `SELECT id, account_id, method, node_id, status, terminal_port, terminal_url, container_name, message, created_at, updated_at, expires_at 
			  FROM auth_tasks WHERE status = 'pending' AND expires_at > NOW() ORDER BY created_at ASC LIMIT $1`
	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*model.AuthTask
	for rows.Next() {
		task := &model.AuthTask{}
		if err := rows.Scan(&task.ID, &task.AccountID, &task.Method, &task.NodeID, &task.Status,
			&task.TerminalPort, &task.TerminalURL, &task.ContainerName, &task.Message,
			&task.CreatedAt, &task.UpdatedAt, &task.ExpiresAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

// ListAuthTasksByNode 列出分配给节点的认证任务
func (s *PostgresStore) ListAuthTasksByNode(ctx context.Context, nodeID string) ([]*model.AuthTask, error) {
	query := `SELECT id, account_id, method, node_id, status, terminal_port, terminal_url, container_name, message, created_at, updated_at, expires_at 
			  FROM auth_tasks WHERE node_id = $1 AND status IN ('assigned', 'running', 'waiting_user') AND expires_at > NOW() ORDER BY created_at ASC`
	rows, err := s.db.QueryContext(ctx, query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*model.AuthTask
	for rows.Next() {
		task := &model.AuthTask{}
		if err := rows.Scan(&task.ID, &task.AccountID, &task.Method, &task.NodeID, &task.Status,
			&task.TerminalPort, &task.TerminalURL, &task.ContainerName, &task.Message,
			&task.CreatedAt, &task.UpdatedAt, &task.ExpiresAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

// UpdateAuthTaskAssignment 更新认证任务的调度信息
func (s *PostgresStore) UpdateAuthTaskAssignment(ctx context.Context, id string, nodeID string) error {
	query := `UPDATE auth_tasks SET node_id = $1, status = 'assigned' WHERE id = $2`
	_, err := s.db.ExecContext(ctx, query, nodeID, id)
	return err
}

// UpdateAuthTaskStatus 更新认证任务状态（Node Agent 调用）
func (s *PostgresStore) UpdateAuthTaskStatus(ctx context.Context, id string, status model.AuthTaskStatus, terminalPort *int, terminalURL *string, containerName *string, message *string) error {
	query := `UPDATE auth_tasks SET status = $1::varchar, terminal_port = $2, terminal_url = $3, container_name = $4, message = $5 WHERE id = $6`
	_, err := s.db.ExecContext(ctx, query, string(status), terminalPort, terminalURL, containerName, message, id)
	return err
}

// === Proxy 操作 ===

// CreateProxy 创建代理
func (s *PostgresStore) CreateProxy(ctx context.Context, proxy *model.Proxy) error {
	query := `
		INSERT INTO proxies (id, name, type, host, port, username, password, no_proxy, is_default, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := s.db.ExecContext(ctx, query,
		proxy.ID, proxy.Name, proxy.Type, proxy.Host, proxy.Port,
		proxy.Username, proxy.Password, proxy.NoProxy,
		proxy.IsDefault, proxy.Status, proxy.CreatedAt, proxy.UpdatedAt)
	return err
}

// GetProxy 获取代理
func (s *PostgresStore) GetProxy(ctx context.Context, id string) (*model.Proxy, error) {
	query := `SELECT id, name, type, host, port, username, password, no_proxy, is_default, status, created_at, updated_at 
			  FROM proxies WHERE id = $1`
	proxy := &model.Proxy{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&proxy.ID, &proxy.Name, &proxy.Type, &proxy.Host, &proxy.Port,
		&proxy.Username, &proxy.Password, &proxy.NoProxy,
		&proxy.IsDefault, &proxy.Status, &proxy.CreatedAt, &proxy.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return proxy, err
}

// ListProxies 列出所有代理
func (s *PostgresStore) ListProxies(ctx context.Context) ([]*model.Proxy, error) {
	query := `SELECT id, name, type, host, port, username, password, no_proxy, is_default, status, created_at, updated_at 
			  FROM proxies ORDER BY is_default DESC, created_at DESC`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var proxies []*model.Proxy
	for rows.Next() {
		proxy := &model.Proxy{}
		if err := rows.Scan(&proxy.ID, &proxy.Name, &proxy.Type, &proxy.Host, &proxy.Port,
			&proxy.Username, &proxy.Password, &proxy.NoProxy,
			&proxy.IsDefault, &proxy.Status, &proxy.CreatedAt, &proxy.UpdatedAt); err != nil {
			return nil, err
		}
		proxies = append(proxies, proxy)
	}
	return proxies, rows.Err()
}

// GetDefaultProxy 获取默认代理
func (s *PostgresStore) GetDefaultProxy(ctx context.Context) (*model.Proxy, error) {
	query := `SELECT id, name, type, host, port, username, password, no_proxy, is_default, status, created_at, updated_at 
			  FROM proxies WHERE is_default = TRUE AND status = 'active' LIMIT 1`
	proxy := &model.Proxy{}
	err := s.db.QueryRowContext(ctx, query).Scan(
		&proxy.ID, &proxy.Name, &proxy.Type, &proxy.Host, &proxy.Port,
		&proxy.Username, &proxy.Password, &proxy.NoProxy,
		&proxy.IsDefault, &proxy.Status, &proxy.CreatedAt, &proxy.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return proxy, err
}

// UpdateProxy 更新代理
func (s *PostgresStore) UpdateProxy(ctx context.Context, proxy *model.Proxy) error {
	query := `UPDATE proxies SET name = $1, type = $2, host = $3, port = $4, 
			  username = $5, password = $6, no_proxy = $7, status = $8 WHERE id = $9`
	_, err := s.db.ExecContext(ctx, query,
		proxy.Name, proxy.Type, proxy.Host, proxy.Port,
		proxy.Username, proxy.Password, proxy.NoProxy, proxy.Status, proxy.ID)
	return err
}

// SetDefaultProxy 设置默认代理（会清除其他代理的默认标记）
func (s *PostgresStore) SetDefaultProxy(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 清除所有代理的默认标记
	if _, err := tx.ExecContext(ctx, `UPDATE proxies SET is_default = FALSE WHERE is_default = TRUE`); err != nil {
		return err
	}

	// 设置指定代理为默认
	if _, err := tx.ExecContext(ctx, `UPDATE proxies SET is_default = TRUE WHERE id = $1`, id); err != nil {
		return err
	}

	return tx.Commit()
}

// ClearDefaultProxy 清除默认代理标记
func (s *PostgresStore) ClearDefaultProxy(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `UPDATE proxies SET is_default = FALSE WHERE is_default = TRUE`)
	return err
}

// DeleteProxy 删除代理
func (s *PostgresStore) DeleteProxy(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM proxies WHERE id = $1`, id)
	return err
}

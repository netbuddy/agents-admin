// Package repository Account 和 AuthTask 相关的存储操作
package repository

import (
	"context"
	"database/sql"

	"agents-admin/internal/shared/model"
)

// === Account 操作 ===

// CreateAccount 创建账号
func (s *Store) CreateAccount(ctx context.Context, account *model.Account) error {
	query := s.rebind(`
		INSERT INTO accounts (id, name, agent_type_id, node_id, volume_name, status, created_at, updated_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`)
	_, err := s.db.ExecContext(ctx, query,
		account.ID, account.Name, account.AgentTypeID, account.NodeID, account.VolumeName,
		account.Status, account.CreatedAt, account.UpdatedAt, account.LastUsedAt)
	return err
}

// GetAccount 获取账号
func (s *Store) GetAccount(ctx context.Context, id string) (*model.Account, error) {
	query := s.rebind(`SELECT id, name, agent_type_id, node_id, volume_name, status, created_at, updated_at, last_used_at 
			  FROM accounts WHERE id = $1`)
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
func (s *Store) ListAccounts(ctx context.Context) ([]*model.Account, error) {
	query := `SELECT id, name, agent_type_id, node_id, volume_name, status, created_at, updated_at, last_used_at 
			  FROM accounts ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAccounts(rows)
}

// ListAccountsByNode 列出指定节点的账号
func (s *Store) ListAccountsByNode(ctx context.Context, nodeID string) ([]*model.Account, error) {
	query := s.rebind(`SELECT id, name, agent_type_id, node_id, volume_name, status, created_at, updated_at, last_used_at 
			  FROM accounts WHERE node_id = $1 ORDER BY created_at DESC`)
	rows, err := s.db.QueryContext(ctx, query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAccounts(rows)
}

// UpdateAccountStatus 更新账号状态
func (s *Store) UpdateAccountStatus(ctx context.Context, id string, status model.AccountStatus) error {
	query := s.rebind(`UPDATE accounts SET status = $1 WHERE id = $2`)
	_, err := s.db.ExecContext(ctx, query, status, id)
	return err
}

// UpdateAccountVolume 更新账号的 Volume 名称
func (s *Store) UpdateAccountVolume(ctx context.Context, id string, volumeName string) error {
	query := s.rebind(`UPDATE accounts SET volume_name = $1 WHERE id = $2`)
	_, err := s.db.ExecContext(ctx, query, volumeName, id)
	return err
}

// DeleteAccount 删除账号
func (s *Store) DeleteAccount(ctx context.Context, id string) error {
	query := s.rebind(`DELETE FROM accounts WHERE id = $1`)
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

func scanAccounts(rows *sql.Rows) ([]*model.Account, error) {
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

// === AuthTask 操作 ===

// CreateAuthTask 创建认证任务
func (s *Store) CreateAuthTask(ctx context.Context, task *model.AuthTask) error {
	status := string(task.Status)
	if status == "" {
		status = "pending"
	}
	query := s.rebind(`
		INSERT INTO auth_tasks (id, account_id, method, node_id, status, terminal_port, terminal_url, container_name, message, created_at, updated_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`)
	_, err := s.db.ExecContext(ctx, query,
		task.ID, task.AccountID, task.Method, task.NodeID, status,
		task.TerminalPort, task.TerminalURL, task.ContainerName, task.Message,
		task.CreatedAt, task.UpdatedAt, task.ExpiresAt)
	return err
}

// GetAuthTask 获取认证任务
func (s *Store) GetAuthTask(ctx context.Context, id string) (*model.AuthTask, error) {
	query := s.rebind(`SELECT id, account_id, method, node_id, status, terminal_port, terminal_url, container_name, message, created_at, updated_at, expires_at 
			  FROM auth_tasks WHERE id = $1`)
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
func (s *Store) GetAuthTaskByAccountID(ctx context.Context, accountID string) (*model.AuthTask, error) {
	query := s.rebind(`SELECT id, account_id, method, node_id, status, terminal_port, terminal_url, container_name, message, created_at, updated_at, expires_at 
			  FROM auth_tasks WHERE account_id = $1 ORDER BY created_at DESC LIMIT 1`)
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

// ListRecentAuthTasks 列出最近的认证任务
func (s *Store) ListRecentAuthTasks(ctx context.Context, limit int) ([]*model.AuthTask, error) {
	query := s.rebind(`SELECT id, account_id, method, node_id, status, terminal_port, terminal_url, container_name, message, created_at, updated_at, expires_at 
			  FROM auth_tasks ORDER BY created_at DESC LIMIT $1`)
	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAuthTasks(rows)
}

// ListPendingAuthTasks 列出待调度的认证任务
func (s *Store) ListPendingAuthTasks(ctx context.Context, limit int) ([]*model.AuthTask, error) {
	nowExpr := s.now()
	query := s.rebind(`SELECT id, account_id, method, node_id, status, terminal_port, terminal_url, container_name, message, created_at, updated_at, expires_at 
			  FROM auth_tasks WHERE status = 'pending' AND expires_at > ` + nowExpr + ` ORDER BY created_at ASC LIMIT $1`)
	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAuthTasks(rows)
}

// ListAuthTasksByNode 列出分配给节点的认证任务
func (s *Store) ListAuthTasksByNode(ctx context.Context, nodeID string) ([]*model.AuthTask, error) {
	nowExpr := s.now()
	query := s.rebind(`SELECT id, account_id, method, node_id, status, terminal_port, terminal_url, container_name, message, created_at, updated_at, expires_at 
			  FROM auth_tasks WHERE node_id = $1 AND status IN ('assigned', 'running', 'waiting_user') AND expires_at > ` + nowExpr + ` ORDER BY created_at ASC`)
	rows, err := s.db.QueryContext(ctx, query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAuthTasks(rows)
}

// UpdateAuthTaskAssignment 更新认证任务的调度信息
func (s *Store) UpdateAuthTaskAssignment(ctx context.Context, id string, nodeID string) error {
	query := s.rebind(`UPDATE auth_tasks SET node_id = $1, status = 'assigned' WHERE id = $2`)
	_, err := s.db.ExecContext(ctx, query, nodeID, id)
	return err
}

// UpdateAuthTaskStatus 更新认证任务状态
func (s *Store) UpdateAuthTaskStatus(ctx context.Context, id string, status model.AuthTaskStatus, terminalPort *int, terminalURL *string, containerName *string, message *string) error {
	query := s.rebind(`UPDATE auth_tasks SET status = $1, terminal_port = $2, terminal_url = $3, container_name = $4, message = $5 WHERE id = $6`)
	_, err := s.db.ExecContext(ctx, query, string(status), terminalPort, terminalURL, containerName, message, id)
	return err
}

func scanAuthTasks(rows *sql.Rows) ([]*model.AuthTask, error) {
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

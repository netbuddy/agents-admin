// Package repository TerminalSession 相关的存储操作
package repository

import (
	"context"
	"database/sql"

	"agents-admin/internal/shared/model"
)

// CreateTerminalSession 创建终端会话
func (s *Store) CreateTerminalSession(ctx context.Context, session *model.TerminalSession) error {
	query := s.rebind(`
		INSERT INTO terminal_sessions (id, instance_id, container_name, node_id, port, url, status, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`)
	_, err := s.db.ExecContext(ctx, query,
		session.ID, session.InstanceID, session.ContainerName, session.NodeID,
		session.Port, session.URL, session.Status, session.CreatedAt, session.ExpiresAt)
	return err
}

// GetTerminalSession 获取终端会话
func (s *Store) GetTerminalSession(ctx context.Context, id string) (*model.TerminalSession, error) {
	query := s.rebind(`SELECT id, instance_id, container_name, node_id, port, url, status, created_at, expires_at 
			  FROM terminal_sessions WHERE id = $1`)
	session := &model.TerminalSession{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&session.ID, &session.InstanceID, &session.ContainerName, &session.NodeID,
		&session.Port, &session.URL, &session.Status, &session.CreatedAt, &session.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return session, err
}

// ListTerminalSessions 列出所有终端会话
func (s *Store) ListTerminalSessions(ctx context.Context) ([]*model.TerminalSession, error) {
	query := `SELECT id, instance_id, container_name, node_id, port, url, status, created_at, expires_at 
			  FROM terminal_sessions ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTerminalSessions(rows)
}

// ListTerminalSessionsByNode 列出指定节点的终端会话
func (s *Store) ListTerminalSessionsByNode(ctx context.Context, nodeID string) ([]*model.TerminalSession, error) {
	query := s.rebind(`SELECT id, instance_id, container_name, node_id, port, url, status, created_at, expires_at 
			  FROM terminal_sessions WHERE node_id = $1 ORDER BY created_at DESC`)
	rows, err := s.db.QueryContext(ctx, query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTerminalSessions(rows)
}

// ListPendingTerminalSessions 列出待处理的终端会话
func (s *Store) ListPendingTerminalSessions(ctx context.Context, nodeID string) ([]*model.TerminalSession, error) {
	query := s.rebind(`SELECT id, instance_id, container_name, node_id, port, url, status, created_at, expires_at 
			  FROM terminal_sessions WHERE node_id = $1 AND status IN ('pending', 'starting') ORDER BY created_at ASC`)
	rows, err := s.db.QueryContext(ctx, query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTerminalSessions(rows)
}

// UpdateTerminalSession 更新终端会话
func (s *Store) UpdateTerminalSession(ctx context.Context, id string, status model.TerminalSessionStatus, port *int, url *string) error {
	query := s.rebind(`UPDATE terminal_sessions SET status = $1, port = $2, url = $3 WHERE id = $4`)
	result, err := s.db.ExecContext(ctx, query, status, port, url, id)
	if err != nil {
		return err
	}
	if rows, err := result.RowsAffected(); err == nil && rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteTerminalSession 删除终端会话
func (s *Store) DeleteTerminalSession(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, s.rebind(`DELETE FROM terminal_sessions WHERE id = $1`), id)
	return err
}

// CleanupExpiredTerminalSessions 清理过期的终端会话
func (s *Store) CleanupExpiredTerminalSessions(ctx context.Context) (int64, error) {
	nowExpr := s.now()
	result, err := s.db.ExecContext(ctx, `DELETE FROM terminal_sessions WHERE expires_at < `+nowExpr+` AND status != 'closed'`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func scanTerminalSessions(rows *sql.Rows) ([]*model.TerminalSession, error) {
	var sessions []*model.TerminalSession
	for rows.Next() {
		session := &model.TerminalSession{}
		if err := rows.Scan(&session.ID, &session.InstanceID, &session.ContainerName, &session.NodeID,
			&session.Port, &session.URL, &session.Status, &session.CreatedAt, &session.ExpiresAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

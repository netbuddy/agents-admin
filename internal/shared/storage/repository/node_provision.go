package repository

import (
	"context"
	"database/sql"
	"fmt"

	"agents-admin/internal/shared/model"
)

// CreateNodeProvision 创建节点部署记录
func (s *Store) CreateNodeProvision(ctx context.Context, p *model.NodeProvision) error {
	query := s.rebind(`
		INSERT INTO node_provisions (id, node_id, host, port, ssh_user, auth_method, status, version, github_repo, api_server_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`)
	_, err := s.db.ExecContext(ctx, query,
		p.ID, p.NodeID, p.Host, p.Port, p.SSHUser, p.AuthMethod,
		p.Status, p.Version, p.GithubRepo, p.APIServerURL,
		p.CreatedAt, p.UpdatedAt)
	return err
}

// UpdateNodeProvision 更新节点部署记录（状态和错误信息）
func (s *Store) UpdateNodeProvision(ctx context.Context, p *model.NodeProvision) error {
	query := s.rebind(`
		UPDATE node_provisions SET status = $1, error_message = $2, updated_at = $3 WHERE id = $4
	`)
	_, err := s.db.ExecContext(ctx, query, p.Status, p.ErrorMessage, p.UpdatedAt, p.ID)
	return err
}

// GetNodeProvision 获取节点部署记录
func (s *Store) GetNodeProvision(ctx context.Context, id string) (*model.NodeProvision, error) {
	query := s.rebind(`
		SELECT id, node_id, host, port, ssh_user, auth_method, status,
		       COALESCE(error_message, ''), version, github_repo, api_server_url,
		       created_at, updated_at
		FROM node_provisions WHERE id = $1
	`)
	p := &model.NodeProvision{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.NodeID, &p.Host, &p.Port, &p.SSHUser, &p.AuthMethod,
		&p.Status, &p.ErrorMessage, &p.Version, &p.GithubRepo, &p.APIServerURL,
		&p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

// ListNodeProvisions 列出所有部署记录
func (s *Store) ListNodeProvisions(ctx context.Context) ([]*model.NodeProvision, error) {
	query := `
		SELECT id, node_id, host, port, ssh_user, auth_method, status,
		       COALESCE(error_message, ''), version, github_repo, api_server_url,
		       created_at, updated_at
		FROM node_provisions ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list node provisions: %w", err)
	}
	defer rows.Close()

	var result []*model.NodeProvision
	for rows.Next() {
		p := &model.NodeProvision{}
		if err := rows.Scan(
			&p.ID, &p.NodeID, &p.Host, &p.Port, &p.SSHUser, &p.AuthMethod,
			&p.Status, &p.ErrorMessage, &p.Version, &p.GithubRepo, &p.APIServerURL,
			&p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

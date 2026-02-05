// Package repository Instance 相关的存储操作
package repository

import (
	"context"
	"database/sql"

	"agents-admin/internal/shared/model"
)

// CreateInstance 创建实例
func (s *Store) CreateInstance(ctx context.Context, instance *model.Instance) error {
	query := s.rebind(`
		INSERT INTO instances (id, name, account_id, agent_type_id, container_name, node_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`)
	_, err := s.db.ExecContext(ctx, query,
		instance.ID, instance.Name, instance.AccountID, instance.AgentTypeID,
		instance.ContainerName, instance.NodeID, instance.Status,
		instance.CreatedAt, instance.UpdatedAt)
	return err
}

// GetInstance 获取实例
func (s *Store) GetInstance(ctx context.Context, id string) (*model.Instance, error) {
	query := s.rebind(`SELECT id, name, account_id, agent_type_id, container_name, node_id, status, created_at, updated_at 
			  FROM instances WHERE id = $1`)
	instance := &model.Instance{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&instance.ID, &instance.Name, &instance.AccountID, &instance.AgentTypeID,
		&instance.ContainerName, &instance.NodeID, &instance.Status,
		&instance.CreatedAt, &instance.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return instance, err
}

// ListInstances 列出所有实例
func (s *Store) ListInstances(ctx context.Context) ([]*model.Instance, error) {
	query := `SELECT id, name, account_id, agent_type_id, container_name, node_id, status, created_at, updated_at 
			  FROM instances ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInstances(rows)
}

// ListInstancesByNode 列出指定节点的实例
func (s *Store) ListInstancesByNode(ctx context.Context, nodeID string) ([]*model.Instance, error) {
	query := s.rebind(`SELECT id, name, account_id, agent_type_id, container_name, node_id, status, created_at, updated_at 
			  FROM instances WHERE node_id = $1 ORDER BY created_at DESC`)
	rows, err := s.db.QueryContext(ctx, query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInstances(rows)
}

// ListPendingInstances 列出待处理的实例
func (s *Store) ListPendingInstances(ctx context.Context, nodeID string) ([]*model.Instance, error) {
	query := s.rebind(`SELECT id, name, account_id, agent_type_id, container_name, node_id, status, created_at, updated_at 
			  FROM instances WHERE node_id = $1 AND status IN ('pending', 'creating', 'stopping') ORDER BY created_at ASC`)
	rows, err := s.db.QueryContext(ctx, query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInstances(rows)
}

// UpdateInstance 更新实例
func (s *Store) UpdateInstance(ctx context.Context, id string, status model.InstanceStatus, containerName *string) error {
	if containerName != nil {
		query := s.rebind(`UPDATE instances SET status = $1, container_name = $2 WHERE id = $3`)
		result, err := s.db.ExecContext(ctx, query, status, *containerName, id)
		if err != nil {
			return err
		}
		if rows, err := result.RowsAffected(); err == nil && rows == 0 {
			return sql.ErrNoRows
		}
		return nil
	}
	query := s.rebind(`UPDATE instances SET status = $1 WHERE id = $2`)
	result, err := s.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return err
	}
	if rows, err := result.RowsAffected(); err == nil && rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteInstance 删除实例
func (s *Store) DeleteInstance(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, s.rebind(`DELETE FROM instances WHERE id = $1`), id)
	return err
}

func scanInstances(rows *sql.Rows) ([]*model.Instance, error) {
	var instances []*model.Instance
	for rows.Next() {
		instance := &model.Instance{}
		if err := rows.Scan(&instance.ID, &instance.Name, &instance.AccountID, &instance.AgentTypeID,
			&instance.ContainerName, &instance.NodeID, &instance.Status,
			&instance.CreatedAt, &instance.UpdatedAt); err != nil {
			return nil, err
		}
		instances = append(instances, instance)
	}
	return instances, rows.Err()
}

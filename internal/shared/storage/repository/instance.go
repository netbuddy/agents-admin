// Package repository Agent 实例相关的存储操作（原 Instance，已重命名对齐领域模型）
package repository

import (
	"context"
	"database/sql"

	"agents-admin/internal/shared/model"
)

// CreateAgentInstance 创建 Agent 实例
func (s *Store) CreateAgentInstance(ctx context.Context, instance *model.Instance) error {
	query := s.rebind(`
		INSERT INTO agents (id, name, account_id, agent_type_id, template_id, container_name, node_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`)
	_, err := s.db.ExecContext(ctx, query,
		instance.ID, instance.Name, instance.AccountID, instance.AgentTypeID,
		instance.TemplateID, instance.ContainerName, instance.NodeID, instance.Status,
		instance.CreatedAt, instance.UpdatedAt)
	return err
}

// GetAgentInstance 获取 Agent 实例
func (s *Store) GetAgentInstance(ctx context.Context, id string) (*model.Instance, error) {
	query := s.rebind(`SELECT id, name, account_id, agent_type_id, template_id, container_name, node_id, status, created_at, updated_at 
			  FROM agents WHERE id = $1`)
	instance := &model.Instance{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&instance.ID, &instance.Name, &instance.AccountID, &instance.AgentTypeID,
		&instance.TemplateID, &instance.ContainerName, &instance.NodeID, &instance.Status,
		&instance.CreatedAt, &instance.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return instance, err
}

// ListAgentInstances 列出所有 Agent 实例
func (s *Store) ListAgentInstances(ctx context.Context) ([]*model.Instance, error) {
	query := `SELECT id, name, account_id, agent_type_id, template_id, container_name, node_id, status, created_at, updated_at 
			  FROM agents ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInstances(rows)
}

// ListAgentInstancesByNode 列出指定节点的 Agent 实例
func (s *Store) ListAgentInstancesByNode(ctx context.Context, nodeID string) ([]*model.Instance, error) {
	query := s.rebind(`SELECT id, name, account_id, agent_type_id, template_id, container_name, node_id, status, created_at, updated_at 
			  FROM agents WHERE node_id = $1 ORDER BY created_at DESC`)
	rows, err := s.db.QueryContext(ctx, query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInstances(rows)
}

// ListPendingAgentInstances 列出待处理的 Agent 实例
func (s *Store) ListPendingAgentInstances(ctx context.Context, nodeID string) ([]*model.Instance, error) {
	query := s.rebind(`SELECT id, name, account_id, agent_type_id, template_id, container_name, node_id, status, created_at, updated_at 
			  FROM agents WHERE node_id = $1 AND status IN ('pending', 'creating', 'stopping') ORDER BY created_at ASC`)
	rows, err := s.db.QueryContext(ctx, query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInstances(rows)
}

// UpdateAgentInstance 更新 Agent 实例
func (s *Store) UpdateAgentInstance(ctx context.Context, id string, status model.InstanceStatus, containerName *string) error {
	if containerName != nil {
		query := s.rebind(`UPDATE agents SET status = $1, container_name = $2 WHERE id = $3`)
		result, err := s.db.ExecContext(ctx, query, status, *containerName, id)
		if err != nil {
			return err
		}
		if rows, err := result.RowsAffected(); err == nil && rows == 0 {
			return sql.ErrNoRows
		}
		return nil
	}
	query := s.rebind(`UPDATE agents SET status = $1 WHERE id = $2`)
	result, err := s.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return err
	}
	if rows, err := result.RowsAffected(); err == nil && rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteAgentInstance 删除 Agent 实例
func (s *Store) DeleteAgentInstance(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, s.rebind(`DELETE FROM agents WHERE id = $1`), id)
	return err
}

func scanInstances(rows *sql.Rows) ([]*model.Instance, error) {
	var instances []*model.Instance
	for rows.Next() {
		instance := &model.Instance{}
		if err := rows.Scan(&instance.ID, &instance.Name, &instance.AccountID, &instance.AgentTypeID,
			&instance.TemplateID, &instance.ContainerName, &instance.NodeID, &instance.Status,
			&instance.CreatedAt, &instance.UpdatedAt); err != nil {
			return nil, err
		}
		instances = append(instances, instance)
	}
	return instances, rows.Err()
}

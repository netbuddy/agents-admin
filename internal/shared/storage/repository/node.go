// Package repository Node 相关的存储操作
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"agents-admin/internal/shared/model"
)

// UpsertNode 更新或插入节点
func (s *Store) UpsertNode(ctx context.Context, node *model.Node) error {
	conflict := s.dialect.UpsertConflict("id", []string{
		"status = EXCLUDED.status",
		"labels = EXCLUDED.labels",
		"capacity = EXCLUDED.capacity",
		"last_heartbeat = EXCLUDED.last_heartbeat",
	})
	query := s.rebind(fmt.Sprintf(`
		INSERT INTO nodes (id, status, labels, capacity, last_heartbeat, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		%s
	`, conflict))
	_, err := s.db.ExecContext(ctx, query,
		node.ID, node.Status, node.Labels, node.Capacity,
		node.LastHeartbeat, node.CreatedAt, node.UpdatedAt)
	return err
}

// UpsertNodeHeartbeat 心跳专用的 upsert
func (s *Store) UpsertNodeHeartbeat(ctx context.Context, node *model.Node) error {
	nowExpr := s.dialect.CurrentTimestamp()
	conflict := s.dialect.UpsertConflict("id", []string{
		"labels = EXCLUDED.labels",
		"capacity = EXCLUDED.capacity",
		"last_heartbeat = EXCLUDED.last_heartbeat",
		"updated_at = " + nowExpr,
	})
	query := s.rebind(fmt.Sprintf(`
		INSERT INTO nodes (id, status, labels, capacity, last_heartbeat, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		%s
	`, conflict))
	_, err := s.db.ExecContext(ctx, query,
		node.ID, node.Status, node.Labels, node.Capacity,
		node.LastHeartbeat, node.CreatedAt, node.UpdatedAt)
	return err
}

// GetNode 获取节点
func (s *Store) GetNode(ctx context.Context, id string) (*model.Node, error) {
	query := s.rebind(`SELECT id, status, COALESCE(labels, '{}'), COALESCE(capacity, '{}'), last_heartbeat, created_at, updated_at FROM nodes WHERE id = $1`)
	node := &model.Node{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&node.ID, &node.Status, &node.Labels, &node.Capacity,
		&node.LastHeartbeat, &node.CreatedAt, &node.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return node, err
}

// ListAllNodes 列出所有节点
func (s *Store) ListAllNodes(ctx context.Context) ([]*model.Node, error) {
	query := `SELECT id, status, COALESCE(labels, '{}'), COALESCE(capacity, '{}'), last_heartbeat, created_at, updated_at 
			  FROM nodes ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodes(rows)
}

// ListOnlineNodes 列出在线节点
func (s *Store) ListOnlineNodes(ctx context.Context) ([]*model.Node, error) {
	query := `SELECT id, status, COALESCE(labels, '{}'), COALESCE(capacity, '{}'), last_heartbeat, created_at, updated_at 
			  FROM nodes WHERE status = 'online' ORDER BY last_heartbeat DESC`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodes(rows)
}

// DeleteNode 删除节点
func (s *Store) DeleteNode(ctx context.Context, id string) error {
	query := s.rebind(`DELETE FROM nodes WHERE id = $1`)
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

func scanNodes(rows *sql.Rows) ([]*model.Node, error) {
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

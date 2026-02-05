// Package repository Proxy 相关的存储操作
package repository

import (
	"context"
	"database/sql"

	"agents-admin/internal/shared/model"
)

// CreateProxy 创建代理
func (s *Store) CreateProxy(ctx context.Context, proxy *model.Proxy) error {
	query := s.rebind(`
		INSERT INTO proxies (id, name, type, host, port, username, password, no_proxy, is_default, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`)
	_, err := s.db.ExecContext(ctx, query,
		proxy.ID, proxy.Name, proxy.Type, proxy.Host, proxy.Port,
		proxy.Username, proxy.Password, proxy.NoProxy,
		proxy.IsDefault, proxy.Status, proxy.CreatedAt, proxy.UpdatedAt)
	return err
}

// GetProxy 获取代理
func (s *Store) GetProxy(ctx context.Context, id string) (*model.Proxy, error) {
	query := s.rebind(`SELECT id, name, type, host, port, username, password, no_proxy, is_default, status, created_at, updated_at 
			  FROM proxies WHERE id = $1`)
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
func (s *Store) ListProxies(ctx context.Context) ([]*model.Proxy, error) {
	query := `SELECT id, name, type, host, port, username, password, no_proxy, is_default, status, created_at, updated_at 
			  FROM proxies ORDER BY is_default DESC, created_at DESC`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProxies(rows)
}

// GetDefaultProxy 获取默认代理
func (s *Store) GetDefaultProxy(ctx context.Context) (*model.Proxy, error) {
	query := s.rebind(`SELECT id, name, type, host, port, username, password, no_proxy, is_default, status, created_at, updated_at 
			  FROM proxies WHERE is_default = ` + s.dialect.BooleanLiteral(true) + ` AND status = 'active' LIMIT 1`)
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
func (s *Store) UpdateProxy(ctx context.Context, proxy *model.Proxy) error {
	query := s.rebind(`UPDATE proxies SET name = $1, type = $2, host = $3, port = $4, 
			  username = $5, password = $6, no_proxy = $7, status = $8 WHERE id = $9`)
	_, err := s.db.ExecContext(ctx, query,
		proxy.Name, proxy.Type, proxy.Host, proxy.Port,
		proxy.Username, proxy.Password, proxy.NoProxy, proxy.Status, proxy.ID)
	return err
}

// SetDefaultProxy 设置默认代理
func (s *Store) SetDefaultProxy(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	bTrue := s.dialect.BooleanLiteral(true)
	bFalse := s.dialect.BooleanLiteral(false)

	if _, err := tx.ExecContext(ctx, `UPDATE proxies SET is_default = `+bFalse+` WHERE is_default = `+bTrue); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, s.rebind(`UPDATE proxies SET is_default = `+bTrue+` WHERE id = $1`), id); err != nil {
		return err
	}

	return tx.Commit()
}

// ClearDefaultProxy 清除默认代理标记
func (s *Store) ClearDefaultProxy(ctx context.Context) error {
	bTrue := s.dialect.BooleanLiteral(true)
	bFalse := s.dialect.BooleanLiteral(false)
	_, err := s.db.ExecContext(ctx, `UPDATE proxies SET is_default = `+bFalse+` WHERE is_default = `+bTrue)
	return err
}

// DeleteProxy 删除代理
func (s *Store) DeleteProxy(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, s.rebind(`DELETE FROM proxies WHERE id = $1`), id)
	return err
}

func scanProxies(rows *sql.Rows) ([]*model.Proxy, error) {
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

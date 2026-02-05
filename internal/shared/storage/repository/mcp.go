// Package repository MCP Server 相关的存储操作
package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"agents-admin/internal/shared/model"
)

// CreateMCPServer 创建 MCP Server
func (s *Store) CreateMCPServer(ctx context.Context, server *model.MCPServer) error {
	argsJSON, _ := json.Marshal(server.Args)
	headersJSON, _ := json.Marshal(server.Headers)
	capsJSON, _ := json.Marshal(server.Capabilities)
	tagsJSON, _ := json.Marshal(server.Tags)

	query := s.rebind(`
		INSERT INTO mcp_servers (id, name, description, source, transport, command, args, url, headers, capabilities, version, author, repository, is_builtin, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`)
	_, err := s.db.ExecContext(ctx, query,
		server.ID, server.Name, server.Description, server.Source, server.Transport, server.Command,
		argsJSON, server.URL, headersJSON, capsJSON, server.Version, server.Author, server.Repository,
		server.IsBuiltin, tagsJSON, server.CreatedAt, server.UpdatedAt)
	return err
}

// GetMCPServer 获取 MCP Server
func (s *Store) GetMCPServer(ctx context.Context, id string) (*model.MCPServer, error) {
	query := s.rebind(`SELECT id, name, description, source, transport, command, args, url, headers, capabilities, version, author, repository, is_builtin, tags, created_at, updated_at
			  FROM mcp_servers WHERE id = $1`)
	server := &model.MCPServer{}
	var argsJSON, headersJSON, capsJSON, tagsJSON []byte
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&server.ID, &server.Name, &server.Description, &server.Source, &server.Transport, &server.Command,
		&argsJSON, &server.URL, &headersJSON, &capsJSON, &server.Version, &server.Author, &server.Repository,
		&server.IsBuiltin, &tagsJSON, &server.CreatedAt, &server.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(argsJSON) > 0 {
		json.Unmarshal(argsJSON, &server.Args)
	}
	if len(headersJSON) > 0 {
		json.Unmarshal(headersJSON, &server.Headers)
	}
	if len(capsJSON) > 0 {
		json.Unmarshal(capsJSON, &server.Capabilities)
	}
	if len(tagsJSON) > 0 {
		json.Unmarshal(tagsJSON, &server.Tags)
	}
	return server, nil
}

// ListMCPServers 列出 MCP Server
func (s *Store) ListMCPServers(ctx context.Context, source string) ([]*model.MCPServer, error) {
	var query string
	var args []interface{}

	if source != "" {
		query = s.rebind(`SELECT id, name, description, source, transport, command, args, url, headers, capabilities, version, author, repository, is_builtin, tags, created_at, updated_at
				 FROM mcp_servers WHERE source = $1 ORDER BY name`)
		args = []interface{}{source}
	} else {
		query = `SELECT id, name, description, source, transport, command, args, url, headers, capabilities, version, author, repository, is_builtin, tags, created_at, updated_at
				 FROM mcp_servers ORDER BY name`
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []*model.MCPServer
	for rows.Next() {
		server := &model.MCPServer{}
		var argsJSON, headersJSON, capsJSON, tagsJSON []byte
		if err := rows.Scan(&server.ID, &server.Name, &server.Description, &server.Source, &server.Transport, &server.Command,
			&argsJSON, &server.URL, &headersJSON, &capsJSON, &server.Version, &server.Author, &server.Repository,
			&server.IsBuiltin, &tagsJSON, &server.CreatedAt, &server.UpdatedAt); err != nil {
			return nil, err
		}
		if len(argsJSON) > 0 {
			json.Unmarshal(argsJSON, &server.Args)
		}
		if len(headersJSON) > 0 {
			json.Unmarshal(headersJSON, &server.Headers)
		}
		if len(capsJSON) > 0 {
			json.Unmarshal(capsJSON, &server.Capabilities)
		}
		if len(tagsJSON) > 0 {
			json.Unmarshal(tagsJSON, &server.Tags)
		}
		servers = append(servers, server)
	}
	return servers, rows.Err()
}

// DeleteMCPServer 删除 MCP Server
func (s *Store) DeleteMCPServer(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, s.rebind(`DELETE FROM mcp_servers WHERE id = $1`), id)
	return err
}

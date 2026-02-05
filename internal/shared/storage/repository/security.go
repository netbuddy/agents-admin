// Package repository SecurityPolicy 相关的存储操作
package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"agents-admin/internal/shared/model"
)

// CreateSecurityPolicy 创建安全策略
func (s *Store) CreateSecurityPolicy(ctx context.Context, policy *model.SecurityPolicyEntity) error {
	toolPermsJSON, _ := json.Marshal(policy.ToolPermissions)
	limitsJSON, _ := json.Marshal(policy.ResourceLimits)
	networkJSON, _ := json.Marshal(policy.NetworkPolicy)
	sandboxJSON, _ := json.Marshal(policy.SandboxPolicy)

	query := s.rebind(`
		INSERT INTO security_policies (id, name, description, tool_permissions, resource_limits, network_policy, sandbox_policy, is_builtin, category, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`)
	_, err := s.db.ExecContext(ctx, query,
		policy.ID, policy.Name, policy.Description, toolPermsJSON, limitsJSON, networkJSON, sandboxJSON,
		policy.IsBuiltin, policy.Category, policy.CreatedAt, policy.UpdatedAt)
	return err
}

// GetSecurityPolicy 获取安全策略
func (s *Store) GetSecurityPolicy(ctx context.Context, id string) (*model.SecurityPolicyEntity, error) {
	query := s.rebind(`SELECT id, name, description, tool_permissions, resource_limits, network_policy, sandbox_policy, is_builtin, category, created_at, updated_at
			  FROM security_policies WHERE id = $1`)
	policy := &model.SecurityPolicyEntity{}
	var toolPermsJSON, limitsJSON, networkJSON, sandboxJSON []byte
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&policy.ID, &policy.Name, &policy.Description, &toolPermsJSON, &limitsJSON, &networkJSON, &sandboxJSON,
		&policy.IsBuiltin, &policy.Category, &policy.CreatedAt, &policy.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(toolPermsJSON) > 0 {
		json.Unmarshal(toolPermsJSON, &policy.ToolPermissions)
	}
	if len(limitsJSON) > 0 {
		json.Unmarshal(limitsJSON, &policy.ResourceLimits)
	}
	if len(networkJSON) > 0 {
		json.Unmarshal(networkJSON, &policy.NetworkPolicy)
	}
	if len(sandboxJSON) > 0 {
		json.Unmarshal(sandboxJSON, &policy.SandboxPolicy)
	}
	return policy, nil
}

// ListSecurityPolicies 列出安全策略
func (s *Store) ListSecurityPolicies(ctx context.Context, category string) ([]*model.SecurityPolicyEntity, error) {
	var query string
	var args []interface{}

	if category != "" {
		query = s.rebind(`SELECT id, name, description, tool_permissions, resource_limits, network_policy, sandbox_policy, is_builtin, category, created_at, updated_at
				 FROM security_policies WHERE category = $1 ORDER BY name`)
		args = []interface{}{category}
	} else {
		query = `SELECT id, name, description, tool_permissions, resource_limits, network_policy, sandbox_policy, is_builtin, category, created_at, updated_at
				 FROM security_policies ORDER BY name`
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []*model.SecurityPolicyEntity
	for rows.Next() {
		policy := &model.SecurityPolicyEntity{}
		var toolPermsJSON, limitsJSON, networkJSON, sandboxJSON []byte
		if err := rows.Scan(&policy.ID, &policy.Name, &policy.Description, &toolPermsJSON, &limitsJSON, &networkJSON, &sandboxJSON,
			&policy.IsBuiltin, &policy.Category, &policy.CreatedAt, &policy.UpdatedAt); err != nil {
			return nil, err
		}
		if len(toolPermsJSON) > 0 {
			json.Unmarshal(toolPermsJSON, &policy.ToolPermissions)
		}
		if len(limitsJSON) > 0 {
			json.Unmarshal(limitsJSON, &policy.ResourceLimits)
		}
		if len(networkJSON) > 0 {
			json.Unmarshal(networkJSON, &policy.NetworkPolicy)
		}
		if len(sandboxJSON) > 0 {
			json.Unmarshal(sandboxJSON, &policy.SandboxPolicy)
		}
		policies = append(policies, policy)
	}
	return policies, rows.Err()
}

// DeleteSecurityPolicy 删除安全策略
func (s *Store) DeleteSecurityPolicy(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, s.rebind(`DELETE FROM security_policies WHERE id = $1`), id)
	return err
}

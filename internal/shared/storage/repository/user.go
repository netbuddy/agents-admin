package repository

import (
	"context"
	"database/sql"

	"agents-admin/internal/shared/model"
)

// CreateUser 创建用户
func (r *Store) CreateUser(ctx context.Context, user *model.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, email, username, password_hash, role, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		user.ID, user.Email, user.Username, user.PasswordHash,
		user.Role, user.Status, user.CreatedAt, user.UpdatedAt,
	)
	return err
}

// GetUserByEmail 通过邮箱查找用户
func (r *Store) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	user := &model.User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, username, password_hash, role, status, created_at, updated_at
		 FROM users WHERE email = $1`, email,
	).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash,
		&user.Role, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return user, err
}

// GetUserByID 通过 ID 查找用户
func (r *Store) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	user := &model.User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, username, password_hash, role, status, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash,
		&user.Role, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return user, err
}

// UpdateUserPassword 更新用户密码
func (r *Store) UpdateUserPassword(ctx context.Context, id, passwordHash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`,
		passwordHash, id,
	)
	return err
}

// ListUsers 列出所有用户
func (r *Store) ListUsers(ctx context.Context) ([]*model.User, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, email, username, password_hash, role, status, created_at, updated_at
		 FROM users ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		u := &model.User{}
		if err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash,
			&u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

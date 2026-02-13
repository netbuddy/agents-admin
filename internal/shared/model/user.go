package model

import "time"

// UserRole 用户角色
type UserRole string

const (
	UserRoleAdmin UserRole = "admin"
	UserRoleUser  UserRole = "user"
)

// UserStatus 用户状态
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
)

// User 用户
type User struct {
	ID           string     `json:"id" bson:"_id" db:"id"`
	Email        string     `json:"email" bson:"email" db:"email"`
	Username     string     `json:"username" bson:"username" db:"username"`
	PasswordHash string     `json:"-" bson:"password_hash" db:"password_hash"` // never expose in JSON
	Role         UserRole   `json:"role" bson:"role" db:"role"`
	Status       UserStatus `json:"status" bson:"status" db:"status"`
	CreatedAt    time.Time  `json:"created_at" bson:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" bson:"updated_at" db:"updated_at"`
}

// Package storage 定义存储层领域错误
//
// 这些错误用于隔离业务层与底层存储引擎的错误类型，
// 各驱动实现（sqlstore/mongostore/memstore）负责将底层错误转换为这些领域错误。
package storage

import "errors"

var (
	// ErrNotFound 实体不存在
	// 替代 sql.ErrNoRows / mongo.ErrNoDocuments
	ErrNotFound = errors.New("entity not found")

	// ErrConflict 并发冲突（乐观锁失败）
	ErrConflict = errors.New("conflict: concurrent modification detected")

	// ErrDuplicate 唯一键冲突（INSERT 重复 ID）
	ErrDuplicate = errors.New("duplicate: entity already exists")
)

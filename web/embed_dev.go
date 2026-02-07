//go:build dev
// +build dev

// Package web 提供前端静态文件的嵌入支持（开发模式）
//
// 开发模式下不嵌入静态文件，前端由 Next.js dev server 独立提供。
// 使用方式：go run -tags dev ./cmd/api-server
package web

import "io/fs"

// StaticFS 开发模式下返回 nil（不嵌入文件）
func StaticFS() (fs.FS, error) {
	return nil, nil
}

// IsEmbedded 返回 false 表示当前为开发模式（前端未嵌入）
func IsEmbedded() bool {
	return false
}

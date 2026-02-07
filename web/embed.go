//go:build !dev
// +build !dev

// Package web 提供前端静态文件的嵌入支持（生产模式）
//
// 使用 Go embed 将 Next.js 静态导出的 out/ 目录嵌入到二进制文件中。
// 注意：必须使用 all: 前缀，否则 _next/ 目录会被排除（以 _ 开头）。
//
// 构建前需要先执行：cd web && STATIC_EXPORT=true npm run build
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:out
var staticFiles embed.FS

// StaticFS 返回前端静态文件的文件系统，以 out/ 为根目录
func StaticFS() (fs.FS, error) {
	return fs.Sub(staticFiles, "out")
}

// IsEmbedded 返回 true 表示当前为生产模式（前端已嵌入）
func IsEmbedded() bool {
	return true
}

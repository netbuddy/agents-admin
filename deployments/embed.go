// Package deployments 嵌入部署相关文件到二进制
//
// 包含：
//   - init-db.sql: PostgreSQL 全量建表脚本
//   - migrations/*.sql: 增量迁移脚本
package deployments

import (
	"embed"
)

// InitDBSQL PostgreSQL 全量初始化脚本（全新安装使用）
//
//go:embed init-db.sql
var InitDBSQL string

// MigrationFiles 增量迁移脚本（升级使用）
//
//go:embed migrations/*.sql
var MigrationFiles embed.FS

// DockerComposeInfra 基础设施 Docker Compose 模板（Setup Wizard 一键部署使用）
//
//go:embed docker-compose.infra.yml
var DockerComposeInfra string

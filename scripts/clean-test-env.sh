#!/usr/bin/env bash
# ============================================================
# clean-test-env.sh — 清理测试环境中 agents-admin 的所有痕迹
#
# 用法: sudo bash scripts/clean-test-env.sh
#
# 清理内容:
#   1. 停止并移除 systemd 服务
#   2. 停止并移除 Docker Compose 基础设施容器和数据卷
#   3. 删除二进制文件
#   4. 删除配置文件和数据目录
#   5. 删除 agents-admin 系统用户
# ============================================================

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; }

if [ "$(id -u)" -ne 0 ]; then
    error "请以 root 权限运行: sudo bash $0"
    exit 1
fi

echo "========================================"
echo "  Agents Admin — 测试环境清理脚本"
echo "========================================"
echo ""

# --- 1. 停止 systemd 服务 ---
info "停止 systemd 服务..."
for svc in agents-admin-api-server agents-admin-node-manager; do
    if systemctl is-active --quiet "$svc" 2>/dev/null; then
        systemctl stop "$svc"
        info "  已停止 $svc"
    fi
    if systemctl is-enabled --quiet "$svc" 2>/dev/null; then
        systemctl disable "$svc" 2>/dev/null || true
        info "  已禁用 $svc"
    fi
    if [ -f "/etc/systemd/system/${svc}.service" ]; then
        rm -f "/etc/systemd/system/${svc}.service"
        info "  已删除 /etc/systemd/system/${svc}.service"
    fi
done
systemctl daemon-reload 2>/dev/null || true

# --- 2. 杀掉残留进程 ---
info "检查残留进程..."
for name in agents-admin-api-server agents-admin-node-manager api-server nodemanager; do
    pids=$(pgrep -f "$name" 2>/dev/null || true)
    if [ -n "$pids" ]; then
        echo "$pids" | xargs kill -9 2>/dev/null || true
        info "  已杀死 $name 进程: $pids"
    fi
done

# --- 3. 停止 Docker Compose 基础设施 ---
info "停止 Docker Compose 基础设施..."

# 查找可能的 infra 目录
for dir in /etc/agents-admin/infra /var/lib/agents-admin/infra /tmp/agents-admin-infra; do
    if [ -d "$dir" ] && [ -f "$dir/docker-compose.yml" ]; then
        info "  发现 infra 目录: $dir"
        (cd "$dir" && docker compose down -v 2>/dev/null) || true
        info "  已停止并移除容器和卷"
    fi
done

# 也检查 agents-* 容器（可能是手动启动的）
for container in agents-mongo agents-redis agents-minio agents-postgres agents-api agents-executor; do
    if docker ps -a --format '{{.Names}}' 2>/dev/null | grep -q "^${container}$"; then
        docker rm -f "$container" 2>/dev/null || true
        info "  已移除容器 $container"
    fi
done

# 移除 Docker volumes
for vol in $(docker volume ls -q 2>/dev/null | grep -E "^(infra_|agents-)" || true); do
    docker volume rm "$vol" 2>/dev/null || true
    info "  已移除卷 $vol"
done

# --- 4. 删除二进制文件 ---
info "删除二进制文件..."
for bin in /usr/local/bin/agents-admin-api-server /usr/local/bin/agents-admin-node-manager \
           /usr/bin/agents-admin-api-server /usr/bin/agents-admin-node-manager; do
    if [ -f "$bin" ]; then
        rm -f "$bin"
        info "  已删除 $bin"
    fi
done

# --- 5. 删除配置和数据目录 ---
info "删除配置和数据目录..."
for dir in /etc/agents-admin /var/lib/agents-admin /var/log/agents-admin; do
    if [ -d "$dir" ]; then
        rm -rf "$dir"
        info "  已删除 $dir"
    fi
done

# --- 6. 删除系统用户 ---
if id agents-admin &>/dev/null; then
    userdel -r agents-admin 2>/dev/null || userdel agents-admin 2>/dev/null || true
    info "已删除系统用户 agents-admin"
fi

# --- 7. 清理 MongoDB 数据（Docker volume） ---
info "清理 Docker named volumes..."
for vol in $(docker volume ls -q 2>/dev/null | grep -E "mongo_data|redis_data|minio_data" || true); do
    docker volume rm "$vol" 2>/dev/null || true
    info "  已移除卷 $vol"
done

echo ""
info "========================================"
info "  清理完成！环境已恢复干净状态。"
info "========================================"

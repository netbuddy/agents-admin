#!/bin/bash
# Qwen-Code 多账户 Docker 管理脚本
#
# 参考 OpenAI Codex 多账户方案设计
# 支持创建、登录、启动、停止、删除容器
#
# 使用方式:
#   ./qwencode-docker.sh create <账户名>
#   ./qwencode-docker.sh login <账户名> [--device]
#   ./qwencode-docker.sh start <账户名>
#   ./qwencode-docker.sh stop <账户名>
#   ./qwencode-docker.sh run <账户名> "<命令>"
#   ./qwencode-docker.sh delete <账户名> [--purge]
#   ./qwencode-docker.sh list
#   ./qwencode-docker.sh status <账户名>

set -e

# 配置
IMAGE_NAME="${QWENCODE_IMAGE:-runners/qwencode:latest}"
CONTAINER_PREFIX="qwencode"
VOLUME_SUFFIX="_data"
AUTH_PORT=1455

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 账户名转容器名（转义特殊字符）
account_to_container_name() {
    local account="$1"
    echo "${CONTAINER_PREFIX}_$(echo "$account" | sed 's/@/_at_/g; s/\./_/g; s/-/_/g')"
}

# 账户名转卷名
account_to_volume_name() {
    local account="$1"
    echo "$(account_to_container_name "$account")${VOLUME_SUFFIX}"
}

# 检查容器是否存在
container_exists() {
    docker container inspect "$1" >/dev/null 2>&1
}

# 检查容器是否运行
container_running() {
    docker container inspect -f '{{.State.Running}}' "$1" 2>/dev/null | grep -q true
}

# 检查是否已登录
is_logged_in() {
    local container="$1"
    docker exec "$container" test -f /home/agent/.qwen/auth.json 2>/dev/null
}

# 创建账户容器
cmd_create() {
    local account="$1"
    if [ -z "$account" ]; then
        log_error "请指定账户名"
        echo "用法: $0 create <账户名>"
        exit 1
    fi

    local container=$(account_to_container_name "$account")
    local volume=$(account_to_volume_name "$account")

    if container_exists "$container"; then
        log_warn "容器 $container 已存在"
        return 0
    fi

    log_info "创建数据卷: $volume"
    docker volume create "$volume" >/dev/null

    log_info "创建容器: $container"
    docker run -d \
        --name "$container" \
        --hostname "$container" \
        -v "$volume:/home/node/.qwen" \
        -v "${PWD}:/workspace" \
        -p "${AUTH_PORT}:${AUTH_PORT}" \
        --restart unless-stopped \
        "$IMAGE_NAME" >/dev/null

    log_success "容器 $container 创建成功"
    log_info "请运行 '$0 login $account' 完成登录"
}

# 登录
cmd_login() {
    local account="$1"
    local use_device="$2"
    
    if [ -z "$account" ]; then
        log_error "请指定账户名"
        exit 1
    fi

    local container=$(account_to_container_name "$account")

    if ! container_exists "$container"; then
        log_error "容器不存在，请先运行: $0 create $account"
        exit 1
    fi

    if ! container_running "$container"; then
        log_info "启动容器..."
        docker start "$container" >/dev/null
    fi

    if is_logged_in "$container"; then
        log_success "账户已登录"
        return 0
    fi

    log_info "开始登录流程..."
    
    if [ "$use_device" = "--device" ]; then
        log_info "使用设备码登录模式"
        echo ""
        echo "=========================================="
        echo "请在浏览器中完成以下步骤:"
        echo "1. 访问显示的验证 URL"
        echo "2. 输入终端显示的设备码"
        echo "3. 登录您的 Qwen 账户完成授权"
        echo "=========================================="
        echo ""
        docker exec -it "$container" qwen login --device-auth
    else
        log_info "使用浏览器 OAuth 登录模式"
        log_info "确保端口 $AUTH_PORT 可访问"
        docker exec -it "$container" qwen login
    fi

    if is_logged_in "$container"; then
        log_success "登录成功！"
    else
        log_warn "登录可能未完成，请检查 auth.json"
    fi
}

# 启动容器并进入交互模式
cmd_start() {
    local account="$1"
    if [ -z "$account" ]; then
        log_error "请指定账户名"
        exit 1
    fi

    local container=$(account_to_container_name "$account")

    if ! container_exists "$container"; then
        log_error "容器不存在，请先运行: $0 create $account"
        exit 1
    fi

    if ! container_running "$container"; then
        log_info "启动容器..."
        docker start "$container" >/dev/null
    fi

    log_info "进入容器交互模式..."
    docker exec -it "$container" bash
}

# 停止容器
cmd_stop() {
    local account="$1"
    if [ -z "$account" ]; then
        log_error "请指定账户名"
        exit 1
    fi

    local container=$(account_to_container_name "$account")

    if container_running "$container"; then
        log_info "停止容器..."
        docker stop "$container" >/dev/null
        log_success "容器已停止"
    else
        log_warn "容器未运行"
    fi
}

# 一次性运行命令
cmd_run() {
    local account="$1"
    shift
    local command="$*"

    if [ -z "$account" ] || [ -z "$command" ]; then
        log_error "请指定账户名和命令"
        echo "用法: $0 run <账户名> \"<命令>\""
        exit 1
    fi

    local container=$(account_to_container_name "$account")

    if ! container_exists "$container"; then
        log_info "容器不存在，自动创建..."
        cmd_create "$account"
    fi

    if ! container_running "$container"; then
        docker start "$container" >/dev/null
    fi

    if ! is_logged_in "$container"; then
        log_error "账户未登录，请先运行: $0 login $account"
        exit 1
    fi

    log_info "执行命令: $command"
    docker exec "$container" $command
}

# 删除容器
cmd_delete() {
    local account="$1"
    local purge="$2"

    if [ -z "$account" ]; then
        log_error "请指定账户名"
        exit 1
    fi

    local container=$(account_to_container_name "$account")
    local volume=$(account_to_volume_name "$account")

    if container_exists "$container"; then
        log_info "删除容器: $container"
        docker rm -f "$container" >/dev/null
        log_success "容器已删除"
    fi

    if [ "$purge" = "--purge" ]; then
        log_warn "删除数据卷（包含认证信息）: $volume"
        docker volume rm "$volume" 2>/dev/null || true
        log_success "数据卷已删除"
    else
        log_info "数据卷已保留，如需删除请使用 --purge 参数"
    fi
}

# 列出所有容器
cmd_list() {
    echo "Qwen-Code 容器列表:"
    echo "----------------------------------------"
    docker ps -a --filter "name=${CONTAINER_PREFIX}_" \
        --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
}

# 查看容器状态
cmd_status() {
    local account="$1"
    if [ -z "$account" ]; then
        log_error "请指定账户名"
        exit 1
    fi

    local container=$(account_to_container_name "$account")
    local volume=$(account_to_volume_name "$account")

    echo "账户: $account"
    echo "容器: $container"
    echo "数据卷: $volume"
    echo "----------------------------------------"

    if ! container_exists "$container"; then
        echo "状态: 未创建"
        return
    fi

    if container_running "$container"; then
        echo "状态: 运行中"
    else
        echo "状态: 已停止"
    fi

    if is_logged_in "$container"; then
        echo "登录: 已登录"
    else
        echo "登录: 未登录"
    fi
}

# 显示帮助
cmd_help() {
    echo "Qwen-Code 多账户 Docker 管理脚本"
    echo ""
    echo "用法: $0 <命令> [参数]"
    echo ""
    echo "命令:"
    echo "  create <账户名>              创建新容器"
    echo "  login <账户名> [--device]    登录账户（--device 使用设备码模式）"
    echo "  start <账户名>               启动容器并进入交互模式"
    echo "  stop <账户名>                停止容器"
    echo "  run <账户名> \"<命令>\"        在容器中执行命令"
    echo "  delete <账户名> [--purge]    删除容器（--purge 同时删除数据卷）"
    echo "  list                         列出所有容器"
    echo "  status <账户名>              查看容器状态"
    echo "  help                         显示帮助"
    echo ""
    echo "示例:"
    echo "  $0 create user@example.com"
    echo "  $0 login user@example.com --device"
    echo "  $0 run user@example.com \"qwen -p '优化代码性能'\""
}

# 主入口
case "${1:-help}" in
    create) cmd_create "$2" ;;
    login)  cmd_login "$2" "$3" ;;
    start)  cmd_start "$2" ;;
    stop)   cmd_stop "$2" ;;
    run)    shift; cmd_run "$@" ;;
    delete) cmd_delete "$2" "$3" ;;
    list)   cmd_list ;;
    status) cmd_status "$2" ;;
    help|*) cmd_help ;;
esac

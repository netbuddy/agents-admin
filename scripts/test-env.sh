#!/bin/bash
# =============================================================================
# 测试环境管理脚本
# 用法:
#   ./scripts/test-env.sh setup    # 创建测试环境（PostgreSQL + Redis）
#   ./scripts/test-env.sh teardown # 删除测试环境
#   ./scripts/test-env.sh reset    # 重置测试数据（清空数据，保留结构）
#   ./scripts/test-env.sh status   # 查看测试环境状态
#
# 环境隔离策略（集成测试 + E2E 共用）:
#   - PostgreSQL: agents_admin_test
#   - Redis: DB 1
# =============================================================================

set -e

# PostgreSQL 配置
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-agents}"
DB_PASSWORD="${DB_PASSWORD:-agents_dev_password}"
DB_NAME_PROD="agents_admin"
DB_NAME_TEST="agents_admin_test"

# Redis 配置
REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6380}"
REDIS_DB_TEST=1    # 测试环境使用 DB 1

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# PostgreSQL 连接命令（通过 Docker 执行）
PG_CONTAINER="agents-postgres"
PSQL="docker exec $PG_CONTAINER psql -U $DB_USER"

# Redis 连接命令（通过 Docker 执行）
REDIS_CONTAINER="agents-redis"
REDIS_CLI="docker exec $REDIS_CONTAINER redis-cli"

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
MIGRATIONS_DIR="$PROJECT_ROOT/deployments/migrations"
INIT_SQL="$PROJECT_ROOT/deployments/init-db.sql"

# =============================================================================
# 创建测试数据库
# =============================================================================
setup_test_db() {
    local db_name=$1
    echo -e "${YELLOW}>>> 创建测试数据库: $db_name${NC}"
    
    # 检查 PostgreSQL 容器是否运行
    if ! docker ps --format '{{.Names}}' | grep -q "^${PG_CONTAINER}$"; then
        echo -e "${RED}错误: PostgreSQL 容器 $PG_CONTAINER 未运行${NC}"
        echo "请先启动: cd deployments && docker-compose up -d postgres"
        return 1
    fi
    
    # 检查数据库是否已存在
    if $PSQL -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$db_name'" | grep -q 1; then
        echo -e "${GREEN}数据库 $db_name 已存在，跳过创建${NC}"
    else
        # 创建数据库
        $PSQL -d postgres -c "CREATE DATABASE $db_name"
        echo -e "${GREEN}数据库 $db_name 创建成功${NC}"
    fi
    
    # 复制 SQL 文件到容器并执行
    echo ">>> 执行初始化脚本: init-db.sql"
    docker cp "$INIT_SQL" $PG_CONTAINER:/tmp/init-db.sql
    $PSQL -d $db_name -f /tmp/init-db.sql 2>/dev/null || true
    
    # 执行迁移脚本（按顺序）
    echo ">>> 执行迁移脚本..."
    for migration in $(ls "$MIGRATIONS_DIR"/*.sql 2>/dev/null | sort); do
        local filename=$(basename $migration)
        echo "    - $filename"
        docker cp "$migration" $PG_CONTAINER:/tmp/$filename
        $PSQL -d $db_name -f /tmp/$filename 2>/dev/null || true
    done
    
    echo -e "${GREEN}✓ 测试数据库 $db_name 初始化完成${NC}"
}

# =============================================================================
# 删除测试数据库
# =============================================================================
teardown_test_db() {
    local db_name=$1
    echo -e "${YELLOW}>>> 删除测试数据库: $db_name${NC}"
    
    # 强制断开所有连接
    $PSQL -d postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='$db_name'" 2>/dev/null || true
    
    # 删除数据库
    $PSQL -d postgres -c "DROP DATABASE IF EXISTS $db_name"
    echo -e "${GREEN}✓ 测试数据库 $db_name 已删除${NC}"
}

# =============================================================================
# 清空 Redis 测试数据库
# =============================================================================
flush_redis_db() {
    local db_index=$1
    echo -e "${YELLOW}>>> 清空 Redis DB $db_index${NC}"
    
    $REDIS_CLI -n $db_index FLUSHDB 2>/dev/null || echo -e "${RED}警告: 无法连接 Redis${NC}"
    echo -e "${GREEN}✓ Redis DB $db_index 已清空${NC}"
}

# =============================================================================
# 重置测试数据库（清空数据，保留结构）
# =============================================================================
reset_test_db() {
    local db_name=$1
    echo -e "${YELLOW}>>> 重置测试数据库: $db_name${NC}"
    
    # 获取所有表并清空
    tables=$($PSQL -d $db_name -tAc "SELECT tablename FROM pg_tables WHERE schemaname='public'" 2>/dev/null)
    for table in $tables; do
        $PSQL -d $db_name -c "TRUNCATE TABLE $table CASCADE" 2>/dev/null || true
    done
    
    echo -e "${GREEN}✓ 测试数据库 $db_name 已重置${NC}"
}

# =============================================================================
# 检查 Redis 连接
# =============================================================================
check_redis() {
    if docker ps --format '{{.Names}}' | grep -q "^${REDIS_CONTAINER}$"; then
        if $REDIS_CLI PING 2>/dev/null | grep -q PONG; then
            return 0
        fi
    fi
    return 1
}

# =============================================================================
# 查看状态
# =============================================================================
show_status() {
    echo -e "${YELLOW}>>> 测试环境状态${NC}"
    echo ""
    
    # 检查 PostgreSQL 容器
    echo "PostgreSQL (container: $PG_CONTAINER):"
    if docker ps --format '{{.Names}}' | grep -q "^${PG_CONTAINER}$"; then
        for db in $DB_NAME_PROD $DB_NAME_TEST; do
            if $PSQL -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$db'" 2>/dev/null | grep -q 1; then
                table_count=$($PSQL -d $db -tAc "SELECT COUNT(*) FROM pg_tables WHERE schemaname='public'" 2>/dev/null | tr -d ' ' || echo "0")
                echo -e "  ${GREEN}✓${NC} $db (${table_count} 张表)"
            else
                echo -e "  ${RED}✗${NC} $db (不存在)"
            fi
        done
    else
        echo -e "  ${RED}✗${NC} 容器未运行"
    fi
    
    echo ""
    echo "Redis (container: $REDIS_CONTAINER):"
    if check_redis; then
        for db_idx in 0 $REDIS_DB_TEST; do
            key_count=$($REDIS_CLI -n $db_idx DBSIZE 2>/dev/null | grep -oE '[0-9]+' || echo "0")
            case $db_idx in
                0) db_name="生产/开发" ;;
                $REDIS_DB_TEST) db_name="测试环境" ;;
            esac
            echo -e "  ${GREEN}✓${NC} DB $db_idx ($db_name, ${key_count} 个键)"
        done
    else
        echo -e "  ${RED}✗${NC} 容器未运行"
    fi
    echo ""
}

# =============================================================================
# 同步表结构（从生产库复制到测试库）
# =============================================================================
sync_schema() {
    echo -e "${YELLOW}>>> 同步表结构${NC}"
    echo "注意: 使用迁移脚本可以保证所有数据库结构一致"
    echo "只需对所有数据库执行相同的迁移脚本即可"
    echo ""
    echo "推荐做法:"
    echo "  1. 修改迁移脚本 (deployments/migrations/)"
    echo "  2. 运行: ./scripts/test-env.sh teardown"
    echo "  3. 运行: ./scripts/test-env.sh setup"
}

# =============================================================================
# 主逻辑
# =============================================================================
case "${1:-help}" in
    setup)
        echo -e "${YELLOW}=== 创建测试环境 ===${NC}"
        echo ""
        
        # PostgreSQL
        setup_test_db $DB_NAME_TEST
        
        # Redis（清空测试 DB 确保干净）
        if check_redis; then
            flush_redis_db $REDIS_DB_TEST
        else
            echo -e "${RED}警告: Redis 不可用，跳过 Redis 初始化${NC}"
        fi
        
        echo ""
        echo -e "${GREEN}=== 测试环境创建完成 ===${NC}"
        echo ""
        echo "PostgreSQL: $DB_NAME_TEST"
        echo "Redis: DB $REDIS_DB_TEST"
        echo ""
        echo "使用方法:"
        echo "  集成测试: APP_ENV=test go test ./tests/integration/..."
        echo "  E2E 测试: APP_ENV=test go test ./tests/e2e/..."
        ;;
    teardown)
        echo -e "${YELLOW}=== 释放测试环境 ===${NC}"
        echo ""
        
        # PostgreSQL
        teardown_test_db $DB_NAME_TEST
        
        # Redis
        if check_redis; then
            flush_redis_db $REDIS_DB_TEST
        fi
        
        echo ""
        echo -e "${GREEN}=== 测试环境已释放 ===${NC}"
        ;;
    reset)
        echo -e "${YELLOW}=== 重置测试数据 ===${NC}"
        echo ""
        
        # PostgreSQL
        reset_test_db $DB_NAME_TEST
        
        # Redis
        if check_redis; then
            flush_redis_db $REDIS_DB_TEST
        fi
        
        echo ""
        echo -e "${GREEN}=== 测试数据已清空 ===${NC}"
        ;;
    status)
        show_status
        ;;
    sync)
        sync_schema
        ;;
    *)
        echo "测试环境管理脚本"
        echo ""
        echo "用法: $0 <command>"
        echo ""
        echo "命令:"
        echo "  setup     创建测试环境（PostgreSQL + Redis）"
        echo "  teardown  释放测试环境"
        echo "  reset     重置测试数据（清空数据，保留表结构）"
        echo "  status    查看测试环境状态"
        echo "  sync      显示表结构同步说明"
        ;;
esac

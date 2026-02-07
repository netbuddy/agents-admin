#!/bin/bash
# =============================================================================
# Agent Admin MVP v1.0 端到端测试套件
# =============================================================================
#
# 使用 agent-browser 进行浏览器自动化测试
# 基于用户使用手册(docs/user-guide/)编写
#
# 使用方法:
#   bash tests/e2e/browser/run_all_tests_v2.sh
#
# 环境变量:
#   BASE_URL  - 前端地址 (默认 http://localhost:3002)
#   API_URL   - API 地址 (默认 http://localhost:8080)
#
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_URL="${BASE_URL:-http://localhost:3002}"
API_URL="${API_URL:-http://localhost:8080}"

export BASE_URL API_URL

echo "============================================================"
echo "  Agent Admin MVP v1.0 — 端到端测试套件"
echo "============================================================"
echo ""
echo "配置:"
echo "  前端地址: $BASE_URL"
echo "  API 地址: $API_URL"
echo "  测试目录: $SCRIPT_DIR"
echo ""

# ---- 前置检查 ----
echo "前置检查..."

# 检查 agent-browser
if ! command -v agent-browser &> /dev/null; then
    echo "❌ agent-browser 未安装"
    exit 1
fi
echo "  ✅ agent-browser 已安装"

# 检查 curl
if ! command -v curl &> /dev/null; then
    echo "❌ curl 未安装"
    exit 1
fi
echo "  ✅ curl 已安装"

# 检查 API Server
echo "  检查 API Server..."
API_READY=false
for i in {1..10}; do
    if curl -s "$API_URL/health" > /dev/null 2>&1; then
        API_READY=true
        break
    fi
    sleep 1
done
if [ "$API_READY" = true ]; then
    echo "  ✅ API Server ($API_URL) 已就绪"
else
    echo "  ❌ API Server ($API_URL) 未就绪"
    exit 1
fi

# 检查前端
echo "  检查前端..."
FE_READY=false
for i in {1..10}; do
    if curl -s "$BASE_URL" > /dev/null 2>&1; then
        FE_READY=true
        break
    fi
    sleep 1
done
if [ "$FE_READY" = true ]; then
    echo "  ✅ 前端 ($BASE_URL) 已就绪"
else
    echo "  ❌ 前端 ($BASE_URL) 未就绪"
    exit 1
fi

echo ""

# ---- 运行测试 ----
TOTAL=0
PASSED=0
FAILED=0
FAILED_TESTS=""

run_test() {
    local name="$1"
    local script="$2"
    TOTAL=$((TOTAL + 1))

    echo ""
    echo "============================================================"
    echo "  [$TOTAL] $name"
    echo "============================================================"

    # 确保没有残留的 browser session
    agent-browser close 2>/dev/null || true

    if bash "$script"; then
        PASSED=$((PASSED + 1))
        echo "  >>> ✅ $name 通过"
    else
        FAILED=$((FAILED + 1))
        FAILED_TESTS="$FAILED_TESTS\n    - $name"
        echo "  >>> ❌ $name 失败"
    fi

    # 清理 browser session
    agent-browser close 2>/dev/null || true
}

# 按序运行
run_test "健康检查与基础验证"   "$SCRIPT_DIR/test_00_health.sh"
run_test "任务看板页面"          "$SCRIPT_DIR/test_01_kanban.sh"
run_test "任务生命周期"          "$SCRIPT_DIR/test_02_task_lifecycle.sh"
run_test "节点管理"              "$SCRIPT_DIR/test_03_node_management.sh"
run_test "代理管理"              "$SCRIPT_DIR/test_04_proxy_management.sh"
run_test "账号与实例管理"        "$SCRIPT_DIR/test_05_account_instance.sh"
run_test "监控页面与系统统计"    "$SCRIPT_DIR/test_06_monitoring.sh"
run_test "智能体页面"            "$SCRIPT_DIR/test_07_agent_page.sh"

# ---- 结果汇总 ----
echo ""
echo "============================================================"
echo "  测试结果汇总"
echo "============================================================"
echo "  总计: $TOTAL"
echo "  通过: $PASSED"
echo "  失败: $FAILED"
if [ $FAILED -gt 0 ]; then
    echo ""
    echo "  失败的测试:"
    echo -e "$FAILED_TESTS"
fi
echo "============================================================"

if [ $FAILED -gt 0 ]; then
    exit 1
fi

echo ""
echo "🎉 所有测试通过！MVP v1.0 发布就绪。"
exit 0

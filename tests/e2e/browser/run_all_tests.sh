#!/bin/bash
# 运行所有浏览器端到端测试
# 使用 agent-browser 进行浏览器自动化测试

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_URL="${BASE_URL:-http://localhost:3001}"
API_URL="${API_URL:-http://localhost:8080}"

echo "=============================================="
echo "  Agent Admin 浏览器端到端测试套件"
echo "=============================================="
echo ""
echo "配置:"
echo "  前端地址: $BASE_URL"
echo "  API 地址: $API_URL"
echo "  测试目录: $SCRIPT_DIR"
echo ""

# 检查 agent-browser 是否可用
if ! command -v agent-browser &> /dev/null; then
    echo "❌ agent-browser 命令未找到"
    echo "请确保 agent-browser 已安装并在 PATH 中"
    exit 1
fi
echo "✅ agent-browser 已安装"

# 检查 Playwright 浏览器是否已安装
if ! agent-browser open "about:blank" 2>/dev/null; then
    echo "⚠️ Playwright 浏览器需要安装，正在安装..."
    npx playwright install chromium || {
        echo "❌ 无法安装 Playwright 浏览器"
        exit 1
    }
fi
agent-browser close 2>/dev/null || true

# 检查服务是否就绪
echo ""
echo "检查服务状态..."

# 检查 API Server
if curl -s "$API_URL/health" > /dev/null 2>&1; then
    echo "✅ API Server ($API_URL) 已就绪"
else
    echo "❌ API Server ($API_URL) 未就绪"
    echo "请先启动 API Server: make run-api"
    exit 1
fi

# 检查前端
if curl -s "$BASE_URL" > /dev/null 2>&1; then
    echo "✅ 前端 ($BASE_URL) 已就绪"
else
    echo "❌ 前端 ($BASE_URL) 未就绪"
    echo "请先启动前端: cd web && npm run dev"
    exit 1
fi

# 运行测试
PASSED=0
FAILED=0

run_test() {
    local test_name="$1"
    local test_script="$2"
    
    echo ""
    echo "=============================================="
    echo "运行测试: $test_name"
    echo "=============================================="
    
    if bash "$test_script"; then
        echo "✅ $test_name 通过"
        ((PASSED++))
    else
        echo "❌ $test_name 失败"
        ((FAILED++))
    fi
}

# 运行各个测试
run_test "节点管理页面" "$SCRIPT_DIR/test_nodes_page.sh"
run_test "账号管理页面" "$SCRIPT_DIR/test_accounts_page.sh"

# 输出结果
echo ""
echo "=============================================="
echo "  测试结果汇总"
echo "=============================================="
echo "  通过: $PASSED"
echo "  失败: $FAILED"
echo "=============================================="

if [ $FAILED -gt 0 ]; then
    exit 1
fi

exit 0

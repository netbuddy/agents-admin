#!/bin/bash
# =============================================================================
# 测试 06: 监控页面与系统统计
# =============================================================================
#
# 测试说明：
#   验证监控和运维功能，包括：
#   1. 监控页面正常加载
#   2. 系统统计 API 返回正确数据
#   3. 工作流列表 API 可用
#   4. 系统设置页面可访问
#   5. 所有导航页面可访问
#
# 对应使用手册：06-monitoring.md
#
# 前置条件：
#   - API Server 和前端服务运行中
#
# =============================================================================

set -e

BASE_URL="${BASE_URL:-http://localhost:3002}"
API_URL="${API_URL:-http://localhost:8080}"
ERRORS=0

log_pass() { echo "  ✅ $1"; }
log_fail() { echo "  ❌ $1"; ERRORS=$((ERRORS + 1)); }
log_step() { echo ""; echo "--- $1 ---"; }

echo "=========================================="
echo "  测试 06: 监控页面与系统统计"
echo "=========================================="

# ----- 测试 1: 监控页面加载 -----
log_step "测试 1: 监控页面加载"
agent-browser open "$BASE_URL/monitor"
agent-browser wait --load networkidle
agent-browser wait 2000

SNAPSHOT=$(agent-browser snapshot -i 2>/dev/null || echo "")
if echo "$SNAPSHOT" | grep -qi "监控\|工作流\|monitor\|workflow"; then
    log_pass "监控页面加载成功"
else
    log_fail "监控页面加载异常"
fi

# ----- 测试 2: 系统统计 API -----
log_step "测试 2: 系统统计 API"
STATS_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/v1/monitor/stats")
if [ "$STATS_CODE" = "200" ]; then
    log_pass "GET /api/v1/monitor/stats 返回 200"
else
    log_fail "GET /api/v1/monitor/stats 返回 $STATS_CODE"
fi

# ----- 测试 3: 工作流列表 API -----
log_step "测试 3: 工作流列表 API"
WF_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/v1/monitor/workflows")
if [ "$WF_CODE" = "200" ]; then
    log_pass "GET /api/v1/monitor/workflows 返回 200"
else
    log_fail "GET /api/v1/monitor/workflows 返回 $WF_CODE"
fi

# ----- 测试 4: 系统设置页面 -----
log_step "测试 4: 系统设置页面"
agent-browser open "$BASE_URL/settings"
agent-browser wait --load networkidle
agent-browser wait 2000

SETTINGS_SNAP=$(agent-browser snapshot -i 2>/dev/null || echo "")
if echo "$SETTINGS_SNAP" | grep -qi "设置\|settings\|系统"; then
    log_pass "系统设置页面加载成功"
else
    log_fail "系统设置页面加载异常"
fi

# ----- 测试 5: 所有导航页面可访问 -----
log_step "测试 5: 所有导航页面可访问"

PAGES=("/" "/accounts" "/instances" "/nodes" "/proxies" "/monitor" "/settings")
PAGE_NAMES=("任务看板" "账号管理" "实例管理" "节点管理" "代理管理" "工作流监控" "系统设置")

for i in "${!PAGES[@]}"; do
    PAGE="${PAGES[$i]}"
    NAME="${PAGE_NAMES[$i]}"
    
    agent-browser open "$BASE_URL$PAGE"
    agent-browser wait --load networkidle
    agent-browser wait 1500
    
    PAGE_TITLE=$(agent-browser get title 2>/dev/null || echo "")
    if [ -n "$PAGE_TITLE" ]; then
        log_pass "$NAME ($PAGE) 可访问"
    else
        log_fail "$NAME ($PAGE) 无法访问"
    fi
done

# ----- 测试 6: 扩展 API 端点可用性（非核心，仅警告） -----
log_step "测试 6: 扩展 API 端点检查"

TEMPLATE_APIS=(
    "/api/v1/task-templates"
    "/api/v1/agent-templates"
    "/api/v1/skills"
    "/api/v1/mcp-servers"
    "/api/v1/security-policies"
)
TEMPLATE_NAMES=(
    "任务模板"
    "Agent 模板"
    "Skills"
    "MCP 服务器"
    "安全策略"
)

for i in "${!TEMPLATE_APIS[@]}"; do
    API="${TEMPLATE_APIS[$i]}"
    NAME="${TEMPLATE_NAMES[$i]}"
    
    CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL$API" 2>/dev/null || echo "000")
    if [ "$CODE" = "200" ]; then
        log_pass "$NAME API ($API) 返回 200"
    else
        # 模板 API 非 MVP 核心功能，仅警告不计为失败
        echo "  ⚠️  $NAME API ($API) 返回 $CODE（非核心，跳过）"
    fi
done

# ----- 清理 -----
agent-browser close 2>/dev/null || true

# ----- 结果 -----
echo ""
echo "=========================================="
if [ $ERRORS -eq 0 ]; then
    echo "  ✅ 测试 06 全部通过"
else
    echo "  ❌ 测试 06 有 $ERRORS 项失败"
fi
echo "=========================================="

exit $ERRORS

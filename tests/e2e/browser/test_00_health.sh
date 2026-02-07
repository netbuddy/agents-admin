#!/bin/bash
# =============================================================================
# 测试 00: 健康检查与基础验证
# =============================================================================
#
# 测试说明：
#   验证系统基础服务是否正常运行，包括：
#   1. API Server 健康检查端点返回 200
#   2. 前端页面可正常访问
#   3. Prometheus 指标端点可用
#
# 对应使用手册：06-monitoring.md - 健康检查
#
# 前置条件：
#   - API Server 运行在 API_URL (默认 http://localhost:8080)
#   - 前端运行在 BASE_URL (默认 http://localhost:3002)
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
echo "  测试 00: 健康检查与基础验证"
echo "=========================================="
echo "API: $API_URL | 前端: $BASE_URL"

# ----- 测试 1: API 健康检查 -----
log_step "测试 1: API 健康检查"
HEALTH=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/health" 2>/dev/null || echo "000")
if [ "$HEALTH" = "200" ]; then
    log_pass "GET /health 返回 200"
else
    log_fail "GET /health 返回 $HEALTH（期望 200）"
fi

# 验证响应体
HEALTH_BODY=$(curl -s "$API_URL/health" 2>/dev/null || echo "{}")
if echo "$HEALTH_BODY" | grep -q '"status"'; then
    log_pass "健康检查响应包含 status 字段"
else
    log_fail "健康检查响应缺少 status 字段"
fi

# ----- 测试 2: Prometheus 指标端点 -----
log_step "测试 2: Prometheus 指标端点"
METRICS_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/metrics" 2>/dev/null || echo "000")
if [ "$METRICS_CODE" = "200" ]; then
    log_pass "GET /metrics 返回 200"
else
    log_fail "GET /metrics 返回 $METRICS_CODE（期望 200）"
fi

# ----- 测试 3: 前端首页加载 -----
log_step "测试 3: 前端首页加载"
agent-browser open "$BASE_URL"
agent-browser wait --load networkidle
TITLE=$(agent-browser get title 2>/dev/null || echo "")
if [ -n "$TITLE" ]; then
    log_pass "前端首页可访问，标题: $TITLE"
else
    log_fail "前端首页无法访问"
fi

# ----- 测试 4: 前端页面包含关键元素 -----
log_step "测试 4: 前端看板页面包含关键元素"
SNAPSHOT=$(agent-browser snapshot -i 2>/dev/null || echo "")
if echo "$SNAPSHOT" | grep -qi "新建\|任务\|看板\|Agent"; then
    log_pass "看板页面包含预期的 UI 元素"
else
    log_fail "看板页面缺少预期的 UI 元素"
fi

# ----- 测试 5: API 端点基础验证 -----
log_step "测试 5: 核心 API 端点可用性"

# 任务列表 API
TASKS_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/v1/tasks" 2>/dev/null || echo "000")
if [ "$TASKS_CODE" = "200" ]; then
    log_pass "GET /api/v1/tasks 返回 200"
else
    log_fail "GET /api/v1/tasks 返回 $TASKS_CODE"
fi

# 节点列表 API
NODES_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/v1/nodes" 2>/dev/null || echo "000")
if [ "$NODES_CODE" = "200" ]; then
    log_pass "GET /api/v1/nodes 返回 200"
else
    log_fail "GET /api/v1/nodes 返回 $NODES_CODE"
fi

# 账号列表 API
ACCOUNTS_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/v1/accounts" 2>/dev/null || echo "000")
if [ "$ACCOUNTS_CODE" = "200" ]; then
    log_pass "GET /api/v1/accounts 返回 200"
else
    log_fail "GET /api/v1/accounts 返回 $ACCOUNTS_CODE"
fi

# 实例列表 API
INSTANCES_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/v1/instances" 2>/dev/null || echo "000")
if [ "$INSTANCES_CODE" = "200" ]; then
    log_pass "GET /api/v1/instances 返回 200"
else
    log_fail "GET /api/v1/instances 返回 $INSTANCES_CODE"
fi

# 代理列表 API
PROXIES_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/v1/proxies" 2>/dev/null || echo "000")
if [ "$PROXIES_CODE" = "200" ]; then
    log_pass "GET /api/v1/proxies 返回 200"
else
    log_fail "GET /api/v1/proxies 返回 $PROXIES_CODE"
fi

# Agent 类型 API
TYPES_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/v1/agent-types" 2>/dev/null || echo "000")
if [ "$TYPES_CODE" = "200" ]; then
    log_pass "GET /api/v1/agent-types 返回 200"
else
    log_fail "GET /api/v1/agent-types 返回 $TYPES_CODE"
fi

# ----- 清理 -----
agent-browser close 2>/dev/null || true

# ----- 结果 -----
echo ""
echo "=========================================="
if [ $ERRORS -eq 0 ]; then
    echo "  ✅ 测试 00 全部通过"
else
    echo "  ❌ 测试 00 有 $ERRORS 项失败"
fi
echo "=========================================="

exit $ERRORS

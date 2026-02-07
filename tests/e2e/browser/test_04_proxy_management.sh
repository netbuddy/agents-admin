#!/bin/bash
# =============================================================================
# 测试 04: 代理管理页面
# =============================================================================
#
# 测试说明：
#   验证代理管理功能，包括：
#   1. 代理管理页面正常加载
#   2. 通过 API 创建代理
#   3. 页面显示已创建的代理
#   4. 设置默认代理
#   5. 清除默认代理
#   6. 删除代理
#
# 对应使用手册：05-proxy-management.md
#
# 前置条件：
#   - API Server 和前端服务运行中
#
# =============================================================================

set -e

BASE_URL="${BASE_URL:-http://localhost:3002}"
API_URL="${API_URL:-http://localhost:8080}"
ERRORS=0
PROXY_ID=""

log_pass() { echo "  ✅ $1"; }
log_fail() { echo "  ❌ $1"; ERRORS=$((ERRORS + 1)); }
log_step() { echo ""; echo "--- $1 ---"; }

echo "=========================================="
echo "  测试 04: 代理管理页面"
echo "=========================================="

# ----- 测试 1: 代理管理页面加载 -----
log_step "测试 1: 代理管理页面加载"
agent-browser open "$BASE_URL/proxies"
agent-browser wait --load networkidle
agent-browser wait 2000

SNAPSHOT=$(agent-browser snapshot -i 2>/dev/null || echo "")
if echo "$SNAPSHOT" | grep -qi "代理\|proxy\|添加"; then
    log_pass "代理管理页面加载成功"
else
    log_fail "代理管理页面加载异常"
fi

# ----- 测试 2: API 创建代理 -----
log_step "测试 2: API 创建代理"
CREATE_RESP=$(curl -s -X POST "$API_URL/api/v1/proxies" \
    -H "Content-Type: application/json" \
    -d '{
        "name": "E2E-Test-Proxy",
        "type": "http",
        "host": "127.0.0.1",
        "port": 18080,
        "is_default": false
    }')

PROXY_ID=$(echo "$CREATE_RESP" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -n "$PROXY_ID" ]; then
    log_pass "代理创建成功: $PROXY_ID"
else
    log_fail "代理创建失败: $CREATE_RESP"
fi

# ----- 测试 3: API 获取代理详情 -----
log_step "测试 3: API 获取代理详情"
GET_RESP=$(curl -s "$API_URL/api/v1/proxies/$PROXY_ID")
PROXY_NAME=$(echo "$GET_RESP" | grep -o '"name":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ "$PROXY_NAME" = "E2E-Test-Proxy" ]; then
    log_pass "代理详情获取正确"
else
    log_fail "代理详情不匹配: $PROXY_NAME"
fi

# ----- 测试 4: API 列出代理 -----
log_step "测试 4: API 列出代理"
LIST_RESP=$(curl -s "$API_URL/api/v1/proxies")
if echo "$LIST_RESP" | grep -q "E2E-Test-Proxy"; then
    log_pass "代理列表包含测试代理"
else
    log_fail "代理列表缺少测试代理"
fi

# ----- 测试 5: 页面刷新后显示代理 -----
log_step "测试 5: 页面显示新代理"
agent-browser open "$BASE_URL/proxies"
agent-browser wait --load networkidle
agent-browser wait 3000

PAGE_TEXT=$(agent-browser get text body 2>/dev/null || echo "")
if echo "$PAGE_TEXT" | grep -qi "E2E-Test-Proxy\|127.0.0.1\|18080"; then
    log_pass "页面显示新创建的代理"
else
    log_fail "页面未显示新代理"
fi

# ----- 测试 6: 设置默认代理 -----
log_step "测试 6: 设置默认代理"
SET_DEFAULT_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_URL/api/v1/proxies/$PROXY_ID/set-default")
if [ "$SET_DEFAULT_CODE" = "200" ]; then
    log_pass "设置默认代理成功"
else
    log_fail "设置默认代理返回 $SET_DEFAULT_CODE（期望 200）"
fi

# 验证默认状态
GET_DEFAULT=$(curl -s "$API_URL/api/v1/proxies/$PROXY_ID")
if echo "$GET_DEFAULT" | grep -q '"is_default":true'; then
    log_pass "代理已标记为默认"
else
    log_fail "代理未正确标记为默认"
fi

# ----- 测试 7: 清除默认代理 -----
log_step "测试 7: 清除默认代理"
CLEAR_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_URL/api/v1/proxies/clear-default")
if [ "$CLEAR_CODE" = "200" ]; then
    log_pass "清除默认代理成功"
else
    log_fail "清除默认代理返回 $CLEAR_CODE（期望 200）"
fi

# ----- 测试 8: 更新代理 -----
log_step "测试 8: 更新代理"
UPDATE_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$API_URL/api/v1/proxies/$PROXY_ID" \
    -H "Content-Type: application/json" \
    -d '{
        "name": "E2E-Test-Proxy-Updated",
        "type": "http",
        "host": "127.0.0.1",
        "port": 18081,
        "is_default": false
    }')

if [ "$UPDATE_CODE" = "200" ]; then
    log_pass "代理更新成功"
else
    log_fail "代理更新返回 $UPDATE_CODE（期望 200）"
fi

# ----- 测试 9: 删除代理 -----
log_step "测试 9: 删除代理"
if [ -n "$PROXY_ID" ]; then
    DEL_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$API_URL/api/v1/proxies/$PROXY_ID")
    if [ "$DEL_CODE" = "204" ] || [ "$DEL_CODE" = "200" ]; then
        log_pass "代理删除成功"
    else
        log_fail "代理删除返回 $DEL_CODE（期望 200/204）"
    fi
fi

# ----- 清理 -----
agent-browser close 2>/dev/null || true

# ----- 结果 -----
echo ""
echo "=========================================="
if [ $ERRORS -eq 0 ]; then
    echo "  ✅ 测试 04 全部通过"
else
    echo "  ❌ 测试 04 有 $ERRORS 项失败"
fi
echo "=========================================="

exit $ERRORS

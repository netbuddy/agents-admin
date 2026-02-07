#!/bin/bash
# =============================================================================
# 测试 05: 账号与实例管理页面
# =============================================================================
#
# 测试说明：
#   验证账号和实例管理功能，包括：
#   1. 账号管理页面正常加载
#   2. Agent 类型 API 返回预置类型列表
#   3. 账号列表 API 正常
#   4. 实例管理页面正常加载
#   5. 实例列表 API 正常
#   6. 页面间导航正常（账号→实例）
#
# 对应使用手册：03-account-instance.md
#
# 注意：OAuth/DeviceCode 认证流程需要实际 Agent 服务，
#       本测试仅验证 UI 和 API 基础可用性，不执行实际认证。
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
echo "  测试 05: 账号与实例管理页面"
echo "=========================================="

# ----- 测试 1: Agent 类型 API -----
log_step "测试 1: Agent 类型 API"
TYPES_RESP=$(curl -s "$API_URL/api/v1/agent-types")
if echo "$TYPES_RESP" | grep -q "agent_types"; then
    log_pass "Agent 类型列表 API 正常"
else
    log_fail "Agent 类型列表 API 异常: $TYPES_RESP"
fi

# 验证包含预置类型
if echo "$TYPES_RESP" | grep -q "qwen-code"; then
    log_pass "包含 qwen-code 类型"
else
    log_fail "缺少 qwen-code 类型"
fi

# ----- 测试 2: 账号管理页面加载 -----
log_step "测试 2: 账号管理页面加载"
agent-browser open "$BASE_URL/accounts"
agent-browser wait --load networkidle
agent-browser wait 2000

SNAPSHOT=$(agent-browser snapshot -i 2>/dev/null || echo "")
if echo "$SNAPSHOT" | grep -qi "账号\|account\|添加"; then
    log_pass "账号管理页面加载成功"
else
    log_fail "账号管理页面加载异常"
fi

# ----- 测试 3: 账号页面"添加账号"按钮 -----
log_step "测试 3: 添加账号按钮"
agent-browser find text "添加账号" click 2>/dev/null || agent-browser find text "添加" click 2>/dev/null || {
    log_fail "找不到添加账号按钮"
}
agent-browser wait 1000

MODAL_SNAP=$(agent-browser snapshot -i 2>/dev/null || echo "")
if echo "$MODAL_SNAP" | grep -qi "Agent.*类型\|agent.*type\|认证\|账号名称"; then
    log_pass "添加账号对话框已打开，包含必要字段"
else
    log_fail "添加账号对话框异常"
fi

# 关闭对话框
agent-browser find text "取消" click 2>/dev/null || agent-browser press Escape 2>/dev/null || true
agent-browser wait 500

# ----- 测试 4: 账号列表 API -----
log_step "测试 4: 账号列表 API"
ACCOUNTS_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/v1/accounts")
if [ "$ACCOUNTS_CODE" = "200" ]; then
    log_pass "账号列表 API 返回 200"
else
    log_fail "账号列表 API 返回 $ACCOUNTS_CODE"
fi

# 带过滤参数
FILTERED_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/v1/accounts?agent_type=qwen-code")
if [ "$FILTERED_CODE" = "200" ]; then
    log_pass "账号过滤 API 正常"
else
    log_fail "账号过滤 API 返回 $FILTERED_CODE"
fi

# ----- 测试 5: 实例管理页面加载 -----
log_step "测试 5: 实例管理页面加载"
agent-browser open "$BASE_URL/instances"
agent-browser wait --load networkidle
agent-browser wait 2000

INST_SNAPSHOT=$(agent-browser snapshot -i 2>/dev/null || echo "")
if echo "$INST_SNAPSHOT" | grep -qi "实例\|instance\|创建"; then
    log_pass "实例管理页面加载成功"
else
    log_fail "实例管理页面加载异常"
fi

# ----- 测试 6: 实例列表 API -----
log_step "测试 6: 实例列表 API"
INST_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/v1/instances")
if [ "$INST_CODE" = "200" ]; then
    log_pass "实例列表 API 返回 200"
else
    log_fail "实例列表 API 返回 $INST_CODE"
fi

# 带过滤参数
INST_FILTERED=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/v1/instances?agent_type=qwen-code")
if [ "$INST_FILTERED" = "200" ]; then
    log_pass "实例过滤 API 正常"
else
    log_fail "实例过滤 API 返回 $INST_FILTERED"
fi

# ----- 测试 7: 侧边栏导航（账号→实例→看板） -----
log_step "测试 7: 侧边栏导航"
# 导航到账号页面
agent-browser open "$BASE_URL/accounts"
agent-browser wait --load networkidle
agent-browser wait 1500

URL_ACCOUNTS=$(agent-browser get url 2>/dev/null || echo "")
if echo "$URL_ACCOUNTS" | grep -q "accounts"; then
    log_pass "导航到账号页面成功"
else
    log_fail "导航到账号页面失败: $URL_ACCOUNTS"
fi

# 通过侧边栏链接导航到实例（使用 snapshot 获取 ref）
NAV_SNAP=$(agent-browser snapshot -i 2>/dev/null || echo "")
INST_REF=$(echo "$NAV_SNAP" | grep 'link "实例管理"' | grep -o 'ref=e[0-9]*' | head -1 | sed 's/ref=/@/')
if [ -n "$INST_REF" ]; then
    agent-browser click "$INST_REF" 2>/dev/null || true
    agent-browser wait --load networkidle
    agent-browser wait 1500
    URL_INST=$(agent-browser get url 2>/dev/null || echo "")
    if echo "$URL_INST" | grep -q "instances"; then
        log_pass "侧边栏导航到实例页面成功"
    else
        log_fail "侧边栏导航到实例页面失败: $URL_INST"
    fi
else
    log_fail "侧边栏找不到实例管理链接"
fi

# 导航回看板
NAV_SNAP2=$(agent-browser snapshot -i 2>/dev/null || echo "")
HOME_REF=$(echo "$NAV_SNAP2" | grep 'link "任务看板"' | grep -o 'ref=e[0-9]*' | head -1 | sed 's/ref=/@/')
if [ -n "$HOME_REF" ]; then
    agent-browser click "$HOME_REF" 2>/dev/null || true
    agent-browser wait --load networkidle
    agent-browser wait 1500
    URL_HOME=$(agent-browser get url 2>/dev/null || echo "")
    if echo "$URL_HOME" | grep -qE "/$|localhost:[0-9]+/?$"; then
        log_pass "导航回看板成功"
    else
        log_fail "导航回看板失败: $URL_HOME"
    fi
else
    log_fail "侧边栏找不到任务看板链接"
fi

# ----- 清理 -----
agent-browser close 2>/dev/null || true

# ----- 结果 -----
echo ""
echo "=========================================="
if [ $ERRORS -eq 0 ]; then
    echo "  ✅ 测试 05 全部通过"
else
    echo "  ❌ 测试 05 有 $ERRORS 项失败"
fi
echo "=========================================="

exit $ERRORS

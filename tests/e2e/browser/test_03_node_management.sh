#!/bin/bash
# =============================================================================
# 测试 03: 节点管理页面
# =============================================================================
#
# 测试说明：
#   验证节点管理功能，包括：
#   1. 节点管理页面正常加载
#   2. 通过 API 注册节点心跳
#   3. 页面显示已注册节点
#   4. 节点状态正确显示（在线/离线）
#   5. 节点详情可查看
#   6. 节点可删除
#
# 对应使用手册：04-node-management.md
#
# 前置条件：
#   - API Server 和前端服务运行中
#
# =============================================================================

set -e

BASE_URL="${BASE_URL:-http://localhost:3002}"
API_URL="${API_URL:-http://localhost:8080}"
ERRORS=0
NODE_ID="e2e-node-$(date +%s)"

log_pass() { echo "  ✅ $1"; }
log_fail() { echo "  ❌ $1"; ERRORS=$((ERRORS + 1)); }
log_step() { echo ""; echo "--- $1 ---"; }

echo "=========================================="
echo "  测试 03: 节点管理页面"
echo "=========================================="

# ----- 测试 1: 注册测试节点 -----
log_step "测试 1: 通过 API 注册测试节点"
HB_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_URL/api/v1/nodes/heartbeat" \
    -H "Content-Type: application/json" \
    -d "{
        \"node_id\": \"$NODE_ID\",
        \"status\": \"online\",
        \"labels\": {\"os\": \"linux\", \"arch\": \"amd64\", \"env\": \"e2e-test\"},
        \"capacity\": {\"max_concurrent\": 4, \"available\": 4}
    }")

if [ "$HB_CODE" = "200" ]; then
    log_pass "节点心跳注册成功: $NODE_ID"
else
    log_fail "节点心跳返回 $HB_CODE（期望 200）"
fi

# ----- 测试 2: API 验证节点已注册 -----
log_step "测试 2: API 验证节点列表"
NODES_RESP=$(curl -s "$API_URL/api/v1/nodes")
if echo "$NODES_RESP" | grep -q "$NODE_ID"; then
    log_pass "节点列表包含测试节点"
else
    log_fail "节点列表缺少测试节点"
fi

# ----- 测试 3: API 获取节点详情 -----
log_step "测试 3: API 获取节点详情"
NODE_RESP=$(curl -s "$API_URL/api/v1/nodes/$NODE_ID")
NODE_STATUS=$(echo "$NODE_RESP" | grep -o '"status":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ "$NODE_STATUS" = "online" ]; then
    log_pass "节点状态正确: online"
else
    log_fail "节点状态异常: $NODE_STATUS（期望 online）"
fi

# ----- 测试 4: Web UI 节点管理页面 -----
log_step "测试 4: 打开节点管理页面"
agent-browser open "$BASE_URL/nodes"
agent-browser wait --load networkidle
agent-browser wait 3000

SNAPSHOT=$(agent-browser snapshot -i 2>/dev/null || echo "")
if echo "$SNAPSHOT" | grep -qi "节点\|node\|在线\|online"; then
    log_pass "节点管理页面加载成功"
else
    log_fail "节点管理页面加载异常"
fi

# ----- 测试 5: 页面显示测试节点 -----
log_step "测试 5: 页面显示测试节点"
PAGE_TEXT=$(agent-browser get text body 2>/dev/null || echo "")
if echo "$PAGE_TEXT" | grep -q "$NODE_ID"; then
    log_pass "页面显示测试节点"
else
    log_fail "页面未显示测试节点 $NODE_ID"
fi

# ----- 测试 6: 更新节点标签 -----
log_step "测试 6: 更新节点"
UPDATE_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "$API_URL/api/v1/nodes/$NODE_ID" \
    -H "Content-Type: application/json" \
    -d "{\"labels\": {\"os\": \"linux\", \"arch\": \"amd64\", \"env\": \"e2e-updated\"}}")

if [ "$UPDATE_CODE" = "200" ]; then
    log_pass "节点更新成功"
else
    log_fail "节点更新返回 $UPDATE_CODE（期望 200）"
fi

# ----- 测试 7: 获取节点 Run 列表 -----
log_step "测试 7: 获取节点 Run 列表"
RUNS_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/v1/nodes/$NODE_ID/runs")
if [ "$RUNS_CODE" = "200" ]; then
    log_pass "节点 Run 列表 API 正常"
else
    log_fail "节点 Run 列表返回 $RUNS_CODE（期望 200）"
fi

# ----- 测试 8: 删除测试节点 -----
log_step "测试 8: 删除测试节点"
DEL_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$API_URL/api/v1/nodes/$NODE_ID")
if [ "$DEL_CODE" = "204" ] || [ "$DEL_CODE" = "200" ]; then
    log_pass "节点删除成功"
else
    log_fail "节点删除返回 $DEL_CODE（期望 204）"
fi

# ----- 清理 -----
agent-browser close 2>/dev/null || true

# ----- 结果 -----
echo ""
echo "=========================================="
if [ $ERRORS -eq 0 ]; then
    echo "  ✅ 测试 03 全部通过"
else
    echo "  ❌ 测试 03 有 $ERRORS 项失败"
fi
echo "=========================================="

exit $ERRORS

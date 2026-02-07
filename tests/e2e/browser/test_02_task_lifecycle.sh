#!/bin/bash
# =============================================================================
# 测试 02: 任务生命周期（创建 → 执行 → 事件 → 取消 → 删除）
# =============================================================================
#
# 测试说明：
#   验证任务的完整生命周期，包括：
#   1. 通过 API 创建任务
#   2. 创建 Run（启动执行）
#   3. 模拟事件上报
#   4. 验证事件获取
#   5. 取消 Run
#   6. 在 Web UI 验证任务状态变化
#   7. 删除任务并验证
#
# 对应使用手册：02-task-management.md - 执行任务、查看执行详情、取消执行
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
echo "  测试 02: 任务生命周期"
echo "=========================================="

# ----- 测试 1: 创建任务 -----
log_step "测试 1: 创建任务"
TASK_NAME="Lifecycle-Test-$(date +%s)"
CREATE_RESP=$(curl -s -X POST "$API_URL/api/v1/tasks" \
    -H "Content-Type: application/json" \
    -d "{\"name\": \"$TASK_NAME\", \"prompt\": \"Test lifecycle prompt for E2E\"}")

TASK_ID=$(echo "$CREATE_RESP" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
TASK_STATUS=$(echo "$CREATE_RESP" | grep -o '"status":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -n "$TASK_ID" ] && [ "$TASK_STATUS" = "pending" ]; then
    log_pass "任务创建成功: $TASK_ID (status=pending)"
else
    log_fail "任务创建失败: $CREATE_RESP"
    agent-browser close 2>/dev/null || true
    exit 1
fi

# ----- 测试 2: 获取任务详情 -----
log_step "测试 2: 获取任务详情"
GET_RESP=$(curl -s "$API_URL/api/v1/tasks/$TASK_ID")
GET_NAME=$(echo "$GET_RESP" | grep -o '"name":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ "$GET_NAME" = "$TASK_NAME" ]; then
    log_pass "任务详情获取正确"
else
    log_fail "任务详情不匹配: $GET_NAME != $TASK_NAME"
fi

# ----- 测试 3: 创建 Run -----
log_step "测试 3: 创建 Run"
RUN_RESP=$(curl -s -X POST "$API_URL/api/v1/tasks/$TASK_ID/runs" \
    -H "Content-Type: application/json")

RUN_ID=$(echo "$RUN_RESP" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
RUN_STATUS=$(echo "$RUN_RESP" | grep -o '"status":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -n "$RUN_ID" ] && [ "$RUN_STATUS" = "queued" ]; then
    log_pass "Run 创建成功: $RUN_ID (status=queued)"
else
    log_fail "Run 创建失败: $RUN_RESP"
fi

# ----- 测试 4: 获取 Run 详情 -----
log_step "测试 4: 获取 Run 详情"
RUN_GET=$(curl -s "$API_URL/api/v1/runs/$RUN_ID")
RUN_GET_STATUS=$(echo "$RUN_GET" | grep -o '"status":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ "$RUN_GET_STATUS" = "queued" ] || [ "$RUN_GET_STATUS" = "running" ]; then
    log_pass "Run 详情正确: status=$RUN_GET_STATUS"
else
    log_fail "Run 状态异常: $RUN_GET_STATUS"
fi

# ----- 测试 5: 模拟事件上报 -----
log_step "测试 5: 模拟事件上报"
NOW=$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")
EVENTS_RESP=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_URL/api/v1/runs/$RUN_ID/events" \
    -H "Content-Type: application/json" \
    -d "{
        \"events\": [
            {\"seq\": 1, \"type\": \"run_started\", \"timestamp\": \"$NOW\", \"payload\": {\"node_id\": \"e2e-test\"}},
            {\"seq\": 2, \"type\": \"message\", \"timestamp\": \"$NOW\", \"payload\": {\"role\": \"assistant\", \"content\": \"E2E test output\"}},
            {\"seq\": 3, \"type\": \"tool_use_start\", \"timestamp\": \"$NOW\", \"payload\": {\"tool_id\": \"t1\", \"tool_name\": \"Read\"}}
        ]
    }")

if [ "$EVENTS_RESP" = "201" ]; then
    log_pass "事件上报成功 (3 events)"
else
    log_fail "事件上报返回 $EVENTS_RESP（期望 201）"
fi

# ----- 测试 6: 获取事件 -----
log_step "测试 6: 获取事件列表"
EVENTS_GET=$(curl -s "$API_URL/api/v1/runs/$RUN_ID/events")
EVENT_COUNT=$(echo "$EVENTS_GET" | grep -o '"count":[0-9]*' | head -1 | cut -d':' -f2)

if [ -n "$EVENT_COUNT" ] && [ "$EVENT_COUNT" -ge 3 ]; then
    log_pass "获取到 $EVENT_COUNT 个事件"
else
    log_fail "事件数量不足: $EVENT_COUNT（期望 >= 3）"
fi

# ----- 测试 7: 列出任务的 Run -----
log_step "测试 7: 列出任务的 Run"
RUNS_LIST=$(curl -s "$API_URL/api/v1/tasks/$TASK_ID/runs")
RUNS_COUNT=$(echo "$RUNS_LIST" | grep -o '"count":[0-9]*' | head -1 | cut -d':' -f2)

if [ -n "$RUNS_COUNT" ] && [ "$RUNS_COUNT" -ge 1 ]; then
    log_pass "任务有 $RUNS_COUNT 个 Run"
else
    log_fail "Run 列表异常: $RUNS_LIST"
fi

# ----- 测试 8: 在 Web UI 验证任务显示 -----
log_step "测试 8: Web UI 验证任务"
agent-browser open "$BASE_URL"
agent-browser wait --load networkidle
agent-browser wait 3000

PAGE_TEXT=$(agent-browser get text body 2>/dev/null || echo "")
if echo "$PAGE_TEXT" | grep -q "$TASK_NAME"; then
    log_pass "Web UI 显示测试任务"
else
    log_fail "Web UI 未显示测试任务"
fi

# ----- 测试 9: 取消 Run -----
log_step "测试 9: 取消 Run"
CANCEL_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_URL/api/v1/runs/$RUN_ID/cancel" \
    -H "Content-Type: application/json")

if [ "$CANCEL_CODE" = "200" ] || [ "$CANCEL_CODE" = "204" ]; then
    log_pass "Run 取消成功"
else
    log_fail "Run 取消返回 $CANCEL_CODE"
fi

# ----- 测试 10: 删除任务 -----
log_step "测试 10: 删除任务"
DEL_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$API_URL/api/v1/tasks/$TASK_ID")
if [ "$DEL_CODE" = "204" ]; then
    log_pass "任务删除成功"
else
    log_fail "任务删除返回 $DEL_CODE（期望 204）"
fi

# 验证删除后获取返回 404
VERIFY_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/v1/tasks/$TASK_ID")
if [ "$VERIFY_CODE" = "404" ]; then
    log_pass "已删除任务返回 404"
else
    log_fail "已删除任务返回 $VERIFY_CODE（期望 404）"
fi

# ----- 清理 -----
agent-browser close 2>/dev/null || true

# ----- 结果 -----
echo ""
echo "=========================================="
if [ $ERRORS -eq 0 ]; then
    echo "  ✅ 测试 02 全部通过"
else
    echo "  ❌ 测试 02 有 $ERRORS 项失败"
fi
echo "=========================================="

exit $ERRORS

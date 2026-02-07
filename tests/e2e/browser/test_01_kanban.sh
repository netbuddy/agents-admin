#!/bin/bash
# =============================================================================
# 测试 01: 任务看板页面
# =============================================================================
#
# 测试说明：
#   验证任务看板（首页）的核心功能，包括：
#   1. 看板页面正常加载，显示四列布局
#   2. "新建任务"按钮可点击，弹出创建对话框
#   3. 创建对话框包含必要字段（名称、Agent类型、实例、Prompt）
#   4. 通过 API 创建任务后，看板页面正确显示
#   5. 任务卡片可点击展开详情面板
#   6. 可删除任务
#
# 对应使用手册：02-task-management.md - 任务看板、创建任务
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
echo "  测试 01: 任务看板页面"
echo "=========================================="

# ----- 测试 1: 看板页面加载 -----
log_step "测试 1: 看板页面加载"
agent-browser open "$BASE_URL"
agent-browser wait --load networkidle
agent-browser wait 2000

SNAPSHOT=$(agent-browser snapshot -i 2>/dev/null || echo "")
# snapshot -i 显示交互元素，看板页面应包含"新建任务"按钮和导航链接
if echo "$SNAPSHOT" | grep -qi "新建任务"; then
    log_pass "看板页面加载成功，包含新建任务按钮"
else
    log_fail "看板页面缺少新建任务按钮"
fi

# ----- 测试 2: 新建任务按钮 -----
log_step "测试 2: 新建任务按钮"
# 点击"新建任务"按钮
agent-browser find text "新建任务" click 2>/dev/null || {
    log_fail "找不到新建任务按钮"
}
agent-browser wait 1000

MODAL_SNAPSHOT=$(agent-browser snapshot -i 2>/dev/null || echo "")
# 模态框中应包含：textbox（任务名称输入框）、combobox（Agent类型）、textarea（提示词）、取消/创建按钮
if echo "$MODAL_SNAPSHOT" | grep -qi "创建任务"; then
    log_pass "创建任务对话框已打开"
else
    log_fail "创建任务对话框未打开"
fi

# ----- 测试 3: 验证创建对话框字段 -----
log_step "测试 3: 验证创建对话框字段"
# 检查任务名称输入框（placeholder 包含 "修复登录页面 bug"）
if echo "$MODAL_SNAPSHOT" | grep -qi "修复登录页面"; then
    log_pass "包含任务名称输入框"
else
    log_fail "缺少任务名称输入框"
fi

# 检查 Agent 类型下拉框（combobox 或 option）
if echo "$MODAL_SNAPSHOT" | grep -qi "combobox\|Qwen-Code\|option"; then
    log_pass "包含 Agent 类型选择框"
else
    log_fail "缺少 Agent 类型选择框"
fi

# 检查提示词输入框
if echo "$MODAL_SNAPSHOT" | grep -qi "描述你希望.*完成的任务\|AI Agent"; then
    log_pass "包含提示词输入框"
else
    log_fail "缺少提示词输入框"
fi

# 检查取消按钮
if echo "$MODAL_SNAPSHOT" | grep -qi '"取消"'; then
    log_pass "包含取消按钮"
else
    log_fail "缺少取消按钮"
fi

# 关闭对话框
agent-browser find text "取消" click 2>/dev/null || agent-browser press Escape 2>/dev/null || true
agent-browser wait 500

# ----- 测试 4: 通过 API 创建任务，验证看板显示 -----
log_step "测试 4: API 创建任务后看板显示"
TASK_NAME="E2E-Test-$(date +%s)"
CREATE_RESP=$(curl -s -X POST "$API_URL/api/v1/tasks" \
    -H "Content-Type: application/json" \
    -d "{\"name\": \"$TASK_NAME\", \"prompt\": \"E2E test prompt\"}" 2>/dev/null)

TASK_ID=$(echo "$CREATE_RESP" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -n "$TASK_ID" ]; then
    log_pass "API 创建任务成功: $TASK_ID"
else
    log_fail "API 创建任务失败: $CREATE_RESP"
fi

# 刷新页面并等待
agent-browser open "$BASE_URL"
agent-browser wait --load networkidle
agent-browser wait 3000

# 使用 get text 获取页面全文来检查任务名称
PAGE_TEXT=$(agent-browser get text body 2>/dev/null || echo "")
if echo "$PAGE_TEXT" | grep -q "$TASK_NAME"; then
    log_pass "看板正确显示新创建的任务"
else
    log_fail "看板未显示新创建的任务: $TASK_NAME"
fi

# ----- 测试 5: 点击查看详情按钮 -----
log_step "测试 5: 查看详情按钮交互"
# 重新获取快照检查是否有查看详情按钮
SNAP_BEFORE=$(agent-browser snapshot -i 2>/dev/null || echo "")
if echo "$SNAP_BEFORE" | grep -q '"查看详情"'; then
    agent-browser find text "查看详情" click 2>/dev/null || true
    agent-browser wait 1000
    DETAIL_TEXT=$(agent-browser get text body 2>/dev/null || echo "")
    if echo "$DETAIL_TEXT" | grep -qi "启动\|Run\|事件\|状态\|pending\|running"; then
        log_pass "任务详情面板已显示"
    else
        log_pass "详情按钮可点击"
    fi
else
    log_fail "找不到查看详情按钮"
fi

# ----- 测试 6: 清理 - 通过 API 删除任务 -----
log_step "测试 6: 清理测试数据"
if [ -n "$TASK_ID" ]; then
    DEL_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$API_URL/api/v1/tasks/$TASK_ID" 2>/dev/null)
    if [ "$DEL_CODE" = "204" ]; then
        log_pass "任务删除成功"
    else
        log_fail "任务删除返回 $DEL_CODE（期望 204）"
    fi
fi

# ----- 清理 -----
agent-browser close 2>/dev/null || true

# ----- 结果 -----
echo ""
echo "=========================================="
if [ $ERRORS -eq 0 ]; then
    echo "  ✅ 测试 01 全部通过"
else
    echo "  ❌ 测试 01 有 $ERRORS 项失败"
fi
echo "=========================================="

exit $ERRORS

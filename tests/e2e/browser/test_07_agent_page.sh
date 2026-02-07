#!/bin/bash
# =============================================================================
# 测试 07: 智能体页面（Agent Page）
# =============================================================================
#
# 测试说明：
#   验证智能体页面功能，包括：
#   1. 侧边栏包含"智能体"导航入口
#   2. 智能体页面加载（两个 Tab）
#   3. 模板库 Tab — API 和 UI 验证
#   4. 智能体实例 Tab — API 和 UI 验证
#   5. 模板 CRUD API（创建、更新、删除）
#   6. AgentTemplate PATCH API（变更执行引擎）
#
# 前置条件：
#   - API Server 和前端服务运行中
#
# =============================================================================

set -e

BASE_URL="${BASE_URL:-http://localhost:3002}"
API_URL="${API_URL:-http://localhost:8080}"
ERRORS=0
CREATED_TEMPLATE_ID=""

log_pass() { echo "  ✅ $1"; }
log_fail() { echo "  ❌ $1"; ERRORS=$((ERRORS + 1)); }
log_step() { echo ""; echo "--- $1 ---"; }

echo "=========================================="
echo "  测试 07: 智能体页面"
echo "=========================================="

# ----- 测试 1: AgentTemplate 列表 API -----
log_step "测试 1: AgentTemplate 列表 API"
TMPL_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/v1/agent-templates")
if [ "$TMPL_CODE" = "200" ]; then
    log_pass "GET /api/v1/agent-templates 返回 200"
else
    log_fail "GET /api/v1/agent-templates 返回 $TMPL_CODE"
fi

TMPL_RESP=$(curl -s "$API_URL/api/v1/agent-templates")
if echo "$TMPL_RESP" | grep -q "templates"; then
    log_pass "模板列表响应包含 templates 字段"
else
    log_fail "模板列表响应异常: $TMPL_RESP"
fi

# ----- 测试 2: Skills 列表 API（非关键，表可能未迁移）-----
log_step "测试 2: Skills 列表 API"
SKILLS_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/v1/skills")
if [ "$SKILLS_CODE" = "200" ]; then
    log_pass "GET /api/v1/skills 返回 200"
else
    echo "  ⚠️  GET /api/v1/skills 返回 $SKILLS_CODE（表可能未迁移，跳过）"
fi

# ----- 测试 3: MCP Servers 列表 API（非关键，表可能未迁移）-----
log_step "测试 3: MCP Servers 列表 API"
MCP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/v1/mcp-servers")
if [ "$MCP_CODE" = "200" ]; then
    log_pass "GET /api/v1/mcp-servers 返回 200"
else
    echo "  ⚠️  GET /api/v1/mcp-servers 返回 $MCP_CODE（表可能未迁移，跳过）"
fi

# ----- 测试 4: 创建自定义模板 -----
log_step "测试 4: 创建自定义模板"
CREATE_RESP=$(curl -s -X POST "$API_URL/api/v1/agent-templates" \
    -H "Content-Type: application/json" \
    -d '{
        "name": "e2e-test-审查助手",
        "type": "claude",
        "role": "代码审查",
        "description": "E2E 测试创建的自定义模板",
        "temperature": 0.3,
        "skills": ["builtin-code-review"],
        "category": "test"
    }')
CREATED_TEMPLATE_ID=$(echo "$CREATE_RESP" | grep -o '"id":"[^"]*"' | head -1 | sed 's/"id":"//;s/"//')
if [ -n "$CREATED_TEMPLATE_ID" ]; then
    log_pass "创建模板成功, ID: $CREATED_TEMPLATE_ID"
else
    log_fail "创建模板失败: $CREATE_RESP"
fi

# ----- 测试 5: 获取模板详情 -----
log_step "测试 5: 获取模板详情"
if [ -n "$CREATED_TEMPLATE_ID" ]; then
    DETAIL_RESP=$(curl -s "$API_URL/api/v1/agent-templates/$CREATED_TEMPLATE_ID")
    if echo "$DETAIL_RESP" | grep -q "e2e-test-审查助手"; then
        log_pass "模板详情返回正确名称"
    else
        log_fail "模板详情异常: $DETAIL_RESP"
    fi
    if echo "$DETAIL_RESP" | grep -q '"type":"claude"'; then
        log_pass "模板类型为 claude"
    else
        log_fail "模板类型异常"
    fi
else
    log_fail "跳过（无模板 ID）"
fi

# ----- 测试 6: PATCH 更新模板（变更执行引擎） -----
log_step "测试 6: PATCH 更新模板"
if [ -n "$CREATED_TEMPLATE_ID" ]; then
    PATCH_RESP=$(curl -s -X PATCH "$API_URL/api/v1/agent-templates/$CREATED_TEMPLATE_ID" \
        -H "Content-Type: application/json" \
        -d '{"type": "qwen", "name": "e2e-test-审查助手-updated"}')
    if echo "$PATCH_RESP" | grep -q '"type":"qwen"'; then
        log_pass "执行引擎变更为 qwen 成功"
    else
        log_fail "执行引擎变更失败: $PATCH_RESP"
    fi
    if echo "$PATCH_RESP" | grep -q "e2e-test-审查助手-updated"; then
        log_pass "模板名称更新成功"
    else
        log_fail "模板名称更新失败"
    fi
else
    log_fail "跳过（无模板 ID）"
fi

# ----- 测试 7: 智能体页面加载 -----
log_step "测试 7: 智能体页面加载"
agent-browser open "$BASE_URL/agents"
agent-browser wait --load networkidle
agent-browser wait 2000

SNAPSHOT=$(agent-browser snapshot -i 2>/dev/null || echo "")
if echo "$SNAPSHOT" | grep -qi "智能体\|模板\|agent"; then
    log_pass "智能体页面加载成功"
else
    log_fail "智能体页面加载异常"
fi

# ----- 测试 8: 页面包含 Tab 切换 -----
log_step "测试 8: Tab 切换"
if echo "$SNAPSHOT" | grep -qi "模板库"; then
    log_pass "页面包含「模板库」Tab"
else
    log_fail "页面缺少「模板库」Tab"
fi

if echo "$SNAPSHOT" | grep -qi "智能体实例"; then
    log_pass "页面包含「智能体实例」Tab"
else
    log_fail "页面缺少「智能体实例」Tab"
fi

# ----- 测试 9: 切换到模板库 Tab -----
log_step "测试 9: 模板库 Tab"
agent-browser find text "模板库" click 2>/dev/null || true
agent-browser wait 1500

TMPL_SNAP=$(agent-browser snapshot -i 2>/dev/null || echo "")
if echo "$TMPL_SNAP" | grep -qi "创建模板\|模板\|template"; then
    log_pass "模板库 Tab 显示正常"
else
    log_fail "模板库 Tab 显示异常"
fi

# ----- 测试 10: 侧边栏包含智能体入口 -----
log_step "测试 10: 侧边栏导航"
NAV_SNAP=$(agent-browser snapshot -i 2>/dev/null || echo "")
if echo "$NAV_SNAP" | grep -qi 'link "智能体"'; then
    log_pass "侧边栏包含「智能体」导航入口"
else
    log_fail "侧边栏缺少「智能体」导航入口"
fi

# ----- 测试 11: 实例列表 API -----
log_step "测试 11: 实例列表 API"
INST_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/v1/instances")
if [ "$INST_CODE" = "200" ]; then
    log_pass "GET /api/v1/instances 返回 200"
else
    log_fail "GET /api/v1/instances 返回 $INST_CODE"
fi

# ----- 清理：删除测试模板 -----
log_step "清理: 删除测试模板"
if [ -n "$CREATED_TEMPLATE_ID" ]; then
    DEL_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$API_URL/api/v1/agent-templates/$CREATED_TEMPLATE_ID")
    if [ "$DEL_CODE" = "204" ]; then
        log_pass "测试模板删除成功"
    else
        log_fail "测试模板删除失败: HTTP $DEL_CODE"
    fi

    # 确认已删除
    VERIFY_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/v1/agent-templates/$CREATED_TEMPLATE_ID")
    if [ "$VERIFY_CODE" = "404" ]; then
        log_pass "确认模板已被删除"
    else
        log_fail "模板仍然存在: HTTP $VERIFY_CODE"
    fi
fi

# ----- 清理浏览器 -----
agent-browser close 2>/dev/null || true

# ----- 结果 -----
echo ""
echo "=========================================="
if [ $ERRORS -eq 0 ]; then
    echo "  ✅ 测试 07 全部通过"
else
    echo "  ❌ 测试 07 有 $ERRORS 项失败"
fi
echo "=========================================="

exit $ERRORS

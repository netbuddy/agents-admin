#!/bin/bash
# 节点管理页面端到端测试
# 使用 agent-browser 进行浏览器自动化测试

set -e

BASE_URL="${BASE_URL:-http://localhost:3001}"
API_URL="${API_URL:-http://localhost:8080}"

echo "=== 节点管理页面 E2E 测试 ==="
echo "前端地址: $BASE_URL"
echo "API 地址: $API_URL"

# 等待前端就绪
echo "等待前端服务就绪..."
for i in {1..30}; do
    if curl -s "$BASE_URL" > /dev/null 2>&1; then
        echo "前端服务已就绪"
        break
    fi
    sleep 1
done

# 1. 打开节点管理页面
echo ""
echo "=== 测试 1: 打开节点管理页面 ==="
agent-browser open "$BASE_URL/nodes"
sleep 2

# 2. 获取页面快照
echo ""
echo "=== 测试 2: 检查页面元素 ==="
agent-browser snapshot -i

# 3. 验证页面标题
echo ""
echo "=== 测试 3: 验证页面标题 ==="
TITLE=$(agent-browser get title)
echo "页面标题: $TITLE"
if [[ "$TITLE" != *"Agent Kanban"* ]]; then
    echo "❌ 页面标题不正确"
    exit 1
fi
echo "✅ 页面标题正确"

# 4. 检查节点列表是否存在
echo ""
echo "=== 测试 4: 检查节点状态显示 ==="
agent-browser snapshot -i | grep -E "(在线|离线|未知)" && echo "✅ 节点状态标签存在" || echo "⚠️ 没有找到节点状态标签（可能没有节点）"

# 5. 截图保存
echo ""
echo "=== 测试 5: 保存截图 ==="
SCREENSHOT_PATH="/tmp/e2e_nodes_page_$(date +%Y%m%d_%H%M%S).png"
agent-browser screenshot "$SCREENSHOT_PATH"
echo "截图已保存: $SCREENSHOT_PATH"

# 6. 关闭浏览器
echo ""
echo "=== 清理 ==="
agent-browser close

echo ""
echo "=== 节点管理页面测试完成 ==="

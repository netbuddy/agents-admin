#!/bin/bash
# 账号管理页面端到端测试
# 使用 agent-browser 进行浏览器自动化测试

set -e

BASE_URL="${BASE_URL:-http://localhost:3001}"
API_URL="${API_URL:-http://localhost:8080}"

echo "=== 账号管理页面 E2E 测试 ==="
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

# 1. 打开账号管理页面
echo ""
echo "=== 测试 1: 打开账号管理页面 ==="
agent-browser open "$BASE_URL/accounts"
sleep 2

# 2. 获取页面快照
echo ""
echo "=== 测试 2: 检查页面元素 ==="
agent-browser snapshot -i

# 3. 验证页面标题
echo ""
echo "=== 测试 3: 验证页面内容 ==="
SNAPSHOT=$(agent-browser snapshot -i)
echo "$SNAPSHOT"

# 4. 点击"添加账号"按钮
echo ""
echo "=== 测试 4: 点击添加账号按钮 ==="
# 查找添加账号按钮
agent-browser find text "添加账号" click || {
    echo "⚠️ 没有找到'添加账号'按钮，尝试其他方式"
    agent-browser snapshot -i
}
sleep 1

# 5. 检查对话框是否打开
echo ""
echo "=== 测试 5: 检查创建账号对话框 ==="
SNAPSHOT=$(agent-browser snapshot -i)
echo "$SNAPSHOT"

# 验证对话框包含必要字段
if echo "$SNAPSHOT" | grep -q "Agent 类型"; then
    echo "✅ Agent 类型选择框存在"
else
    echo "❌ Agent 类型选择框不存在"
fi

if echo "$SNAPSHOT" | grep -q "节点"; then
    echo "✅ 节点选择框存在"
else
    echo "❌ 节点选择框不存在 - 这是本次修复的重点！"
fi

if echo "$SNAPSHOT" | grep -q "账号名称"; then
    echo "✅ 账号名称输入框存在"
else
    echo "❌ 账号名称输入框不存在"
fi

# 6. 截图保存
echo ""
echo "=== 测试 6: 保存对话框截图 ==="
SCREENSHOT_PATH="/tmp/e2e_accounts_dialog_$(date +%Y%m%d_%H%M%S).png"
agent-browser screenshot "$SCREENSHOT_PATH"
echo "截图已保存: $SCREENSHOT_PATH"

# 7. 关闭对话框
echo ""
echo "=== 测试 7: 关闭对话框 ==="
agent-browser find text "取消" click || echo "没有找到取消按钮"
sleep 1

# 8. 关闭浏览器
echo ""
echo "=== 清理 ==="
agent-browser close

echo ""
echo "=== 账号管理页面测试完成 ==="

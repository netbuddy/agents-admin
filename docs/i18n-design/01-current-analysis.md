# 现状分析与改造范围

> 返回 [README](./README.md)

---

## 一、代码架构现状

### 技术栈

| 技术 | 版本 | 说明 |
|------|------|------|
| Next.js | 14.1.0 | App Router, `output: 'export'` 静态导出 |
| React | 18.2.0 | 所有页面均为 `'use client'` 客户端组件 |
| TypeScript | 5.3.3 | 严格类型检查 |
| Go embed | 1.24.0 | 编译时嵌入前端静态文件到二进制 |

### 前端目录结构

```
web/
├── app/
│   ├── layout.tsx              ← html lang="zh" 硬编码
│   ├── page.tsx                ← 首页（看板）
│   ├── accounts/page.tsx       ← 账号管理
│   ├── instances/page.tsx      ← 实例管理
│   ├── monitor/page.tsx        ← 工作流监控
│   ├── nodes/page.tsx          ← 节点管理
│   ├── proxies/page.tsx        ← 代理管理
│   ├── runners/page.tsx        ← Runner 管理
│   ├── settings/page.tsx       ← 系统设置
│   └── tasks/detail/
│       ├── page.tsx            ← 任务详情入口
│       └── TaskDetailClient.tsx ← 任务详情主组件
├── components/
│   ├── layout/
│   │   ├── Sidebar.tsx         ← 导航菜单（7 个中文标签）
│   │   └── Header.tsx          ← 顶栏（刷新、通知等）
│   ├── kanban/
│   │   ├── TaskCard.tsx        ← 任务卡片
│   │   ├── TaskDetailPanel.tsx ← 任务详情面板（最多中文：71处）
│   │   └── KanbanColumn.tsx    ← 看板列
│   ├── agent-output/
│   │   ├── AgentOutput.tsx     ← Agent 输出展示
│   │   ├── DebugPanel.tsx      ← 调试面板
│   │   ├── MessageBlock.tsx    ← 消息块
│   │   ├── CommandBlock.tsx    ← 命令块
│   │   ├── FileBlock.tsx       ← 文件操作块
│   │   └── ToolBlock.tsx       ← 工具调用块
│   └── CreateTaskModal.tsx     ← 创建任务弹窗
└── package.json
```

---

## 二、硬编码中文字符串统计

### 按文件分布

| 文件 | 中文匹配数 | 主要内容 |
|------|-----------|---------|
| **components/kanban/TaskDetailPanel.tsx** | 71 | 运行记录、配置面板、状态文案、操作按钮 |
| **app/accounts/page.tsx** | 54 | 账号列表、添加/编辑弹窗、状态文案、认证方式 |
| **components/agent-output/DebugPanel.tsx** | 47 | 调试信息、时间戳、状态标签 |
| **app/monitor/page.tsx** | 47 | 工作流列表、状态筛选、详情展示 |
| **app/instances/page.tsx** | 41 | 实例列表、创建/配置弹窗 |
| **app/proxies/page.tsx** | 29 | 代理列表、添加/测试弹窗 |
| **app/runners/page.tsx** | 24 | Runner 列表、终端操作、确认对话框 |
| **components/CreateTaskModal.tsx** | 24 | 表单标签、验证提示、按钮文案 |
| **components/agent-output/AgentOutput.tsx** | 21 | 事件类型标签、Token 使用信息 |
| **app/page.tsx** | 18 | 看板列头、任务统计、状态标签 |
| **components/kanban/TaskCard.tsx** | 16 | 卡片状态、操作按钮 |
| **components/layout/Sidebar.tsx** | 13 | 7 个导航标签 + 注释 |
| **app/nodes/page.tsx** | 13 | 节点列表、状态文案 |
| **components/agent-output/MessageBlock.tsx** | 12 | 消息类型标签 |
| **components/agent-output/CommandBlock.tsx** | 8 | 命令执行状态 |
| **components/layout/Header.tsx** | 6 | 刷新、通知 |
| **components/agent-output/FileBlock.tsx** | 6 | 读取/修改文件标签 |
| **components/agent-output/ToolBlock.tsx** | 5 | 工具调用标签 |
| **app/settings/page.tsx** | 7 | 设置项标签 |
| **app/tasks/detail/TaskDetailClient.tsx** | 22 | 任务详情操作 |
| **app/tasks/detail/page.tsx** | 2 | 缺少 ID 提示 |
| **components/kanban/KanbanColumn.tsx** | 2 | 列标题 |
| **app/layout.tsx** | 1 | metadata description |
| **合计** | **~489** | — |

### 按字符串类别分类

#### 1. 导航与布局标签（~20 个）

```
任务看板, 工作流监控, 账号管理, 实例管理, 节点管理, 代理管理, 系统设置
刷新, 通知, 返回看板, 打开菜单
Agent Kanban（logo，保持英文）
```

#### 2. 操作按钮文案（~30 个）

```
新建任务, 添加账号, 添加代理, 添加账户, 查看详情, 开始执行
删除, 取消, 保存, 编辑, 测试, 创建, 确认
新建 Run, 创建中..., 保存中...
返回首页, 前往创建实例
```

#### 3. 表单标签与占位符（~25 个）

```
任务名称, Agent 类型, 选择实例, 任务提示词
账号名称, 代理地址, 端口, 用户名, 密码
例如：修复登录页面 bug
描述你希望 AI Agent 完成的任务...
请选择一个运行中的实例
```

#### 4. 状态文案（~30 个）

```
待处理, 运行中, 已完成, 失败, 已取消
在线, 离线, 空闲, 忙碌
活跃, 已过期, 已禁用
等待事件..., 暂无运行记录
```

#### 5. 数据展示文案（~40 个）

```
共 X 个任务, 运行记录 (N), 任务配置
最后心跳: ..., 上次使用: ...
创建于 ..., 执行时间: ...
节点: ..., 账号: ...
Token 使用: ...
```

#### 6. 确认/错误消息（~15 个）

```
缺少任务 ID, 任务不存在
确定删除 Runner "xxx"？（包括认证数据）
无法打开终端，请确保后端服务正在运行
没有可用的 xxx 实例
需要先创建一个运行中的实例才能创建任务
错误
```

#### 7. 代码注释中的中文（不计入翻译，但建议保留）

```
// 数据加载状态
// 获取 Agent 类型和实例列表
// 移动端操作栏
// 桌面端运行记录侧栏
```

---

## 三、日期时间格式化现状

当前代码中硬编码了 `'zh-CN'` locale，需要统一改为动态 locale：

| 文件 | 代码 | 格式 |
|------|------|------|
| TaskDetailClient.tsx (×3) | `new Date(x).toLocaleString('zh-CN')` | 完整日期时间 |
| TaskCard.tsx | `new Date(x).toLocaleString('zh-CN', {...})` | 月/日 时:分 |
| monitor/page.tsx | `date.toLocaleString('zh-CN', {...})` | 月/日 时:分:秒 |
| DebugPanel.tsx | `new Date(x).toLocaleTimeString('zh-CN', {...})` | 时:分:秒.毫秒 |
| AgentOutput.tsx | `new Date(x).toLocaleTimeString('zh-CN')` | 时:分:秒 |
| TaskDetailPanel.tsx | `new Date(x).toLocaleString('zh-CN')` | 完整日期时间 |
| accounts/page.tsx | `new Date(x).toLocaleDateString()` | 日期（无指定 locale） |
| nodes/page.tsx | `new Date(x).toLocaleString()` | 完整日期时间（无指定 locale） |

**改造方向**：提取统一的 `useFormatDate` hook，根据当前 i18n locale 自动格式化。

---

## 四、HTML 元数据

```tsx
// app/layout.tsx
<html lang="zh">  // ← 需要动态设置
export const metadata: Metadata = {
  title: 'Agent Kanban',
  description: 'AI Agent 任务编排与可观测平台',  // ← 需要翻译
}
```

**注意**：Next.js 静态导出的 metadata 是构建时确定的，无法运行时切换。
解决方案：在客户端用 `document.documentElement.lang = locale` 动态更新 `lang` 属性，
metadata description 保留为默认语言（管理后台不需要 SEO）。

---

## 五、需要特殊处理的模式

### 5.1 带变量的字符串插值

```tsx
// 当前
`共 ${count} 个任务`
`运行记录 (${runs.length})`
`确定删除 Runner "${account}"？`
`节点: ${selectedRun.node_id}`

// 改造后（react-i18next 插值语法）
t('tasks.total', { count })          → "共 {{count}} 个任务" / "{{count}} tasks total"
t('runs.count', { count: runs.length }) → "运行记录 ({{count}})" / "Run history ({{count}})"
t('runners.confirmDelete', { name: account }) → '确定删除 Runner "{{name}}"？'
```

### 5.2 条件拼接

```tsx
// 当前
{account.last_used_at && ` · 上次使用: ${new Date(account.last_used_at).toLocaleDateString()}`}

// 改造后
{account.last_used_at && ` · ${t('accounts.lastUsed')}: ${formatDate(account.last_used_at)}`}
```

### 5.3 alert() / confirm() 对话框

```tsx
// 当前
alert('无法打开终端，请确保后端服务正在运行')
if (!confirm(`确定删除 Runner "${account}"？`)) return

// 改造后：建议同步替换为 UI 组件（Toast/Dialog），但可先用 t() 包裹
alert(t('runners.terminalError'))
if (!confirm(t('runners.confirmDelete', { name: account }))) return
```

### 5.4 枚举映射

```tsx
// 当前（散落在各文件中）
const statusText = (s: string) => {
  switch (s) {
    case 'active': return '活跃'
    case 'expired': return '已过期'
    ...
  }
}

// 改造后：使用 t() 统一映射
const statusText = (s: string) => t(`status.${s}`)
// zh.json: { "status": { "active": "活跃", "expired": "已过期" } }
// en.json: { "status": { "active": "Active", "expired": "Expired" } }
```

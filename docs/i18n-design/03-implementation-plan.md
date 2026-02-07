# 实施方案详解

> 返回 [README](./README.md)

---

## 一、整体架构

```
┌─────────────────────────────────────────────────────────┐
│  浏览器                                                  │
│                                                         │
│  1. 页面加载                                             │
│       │                                                 │
│       ▼                                                 │
│  2. i18next 初始化                                       │
│       │                                                 │
│       ├─ 检查 localStorage 中是否有 saved locale         │
│       │    │ 有 → 使用 saved locale                      │
│       │    │ 无 → 继续检测                               │
│       │                                                 │
│       ├─ 读取 navigator.language (如 "zh-CN", "en-US")   │
│       │    │                                            │
│       │    ▼                                            │
│       ├─ 匹配支持的 locale: zh, en                       │
│       │    "zh-CN" → "zh"                               │
│       │    "en-US" → "en"                               │
│       │    "ja"    → fallback "en"                      │
│       │                                                 │
│       ▼                                                 │
│  3. 加载对应翻译资源（静态 import，无网络请求）           │
│       │                                                 │
│       ▼                                                 │
│  4. I18nextProvider 注入 React 树                        │
│       │                                                 │
│       ▼                                                 │
│  5. 组件中 useTranslation() → t('key') 返回翻译文本     │
│       │                                                 │
│       ▼                                                 │
│  6. 用户点击语言切换器                                   │
│       │                                                 │
│       ▼                                                 │
│  7. i18next.changeLanguage('en')                        │
│       │                                                 │
│       ├─ 更新 localStorage                              │
│       ├─ 更新 document.documentElement.lang             │
│       └─ 触发所有使用 t() 的组件重渲染                   │
└─────────────────────────────────────────────────────────┘
```

---

## 二、目录结构设计

```
web/
├── i18n/
│   ├── config.ts               ← i18next 初始化配置
│   ├── provider.tsx            ← I18nProvider 组件
│   ├── useFormatDate.ts        ← 日期格式化 hook（locale-aware）
│   └── locales/
│       ├── zh/
│       │   ├── common.json     ← 通用：导航、按钮、状态
│       │   ├── tasks.json      ← 任务看板 + 任务详情
│       │   ├── accounts.json   ← 账号管理
│       │   ├── instances.json  ← 实例管理
│       │   ├── monitor.json    ← 工作流监控
│       │   ├── runners.json    ← Runner 管理
│       │   ├── nodes.json      ← 节点管理
│       │   ├── proxies.json    ← 代理管理
│       │   └── settings.json   ← 系统设置
│       └── en/
│           ├── common.json
│           ├── tasks.json
│           ├── accounts.json
│           ├── instances.json
│           ├── monitor.json
│           ├── runners.json
│           ├── nodes.json
│           ├── proxies.json
│           └── settings.json
├── components/
│   └── LanguageSwitcher.tsx    ← 语言切换下拉菜单
└── app/
    └── layout.tsx              ← 包裹 I18nProvider
```

### namespace 拆分策略

| namespace | 内容 | 使用文件 |
|-----------|------|---------|
| `common` | 导航标签、通用按钮（取消/保存/删除）、状态文案、错误提示 | 所有页面 |
| `tasks` | 看板、任务卡片、任务详情、创建任务弹窗 | page.tsx, TaskCard, TaskDetailPanel, CreateTaskModal |
| `accounts` | 账号列表、添加/编辑弹窗、认证方式 | accounts/page.tsx |
| `instances` | 实例列表、创建弹窗、配置项 | instances/page.tsx |
| `monitor` | 工作流列表、状态筛选、Agent 输出组件 | monitor/page.tsx, agent-output/* |
| `runners` | Runner 列表、终端操作 | runners/page.tsx |
| `nodes` | 节点列表 | nodes/page.tsx |
| `proxies` | 代理列表、测试弹窗 | proxies/page.tsx |
| `settings` | 系统设置项 | settings/page.tsx |

---

## 三、核心模块伪码

### 3.1 i18n/config.ts — 初始化配置

```
引入 i18next, react-i18next, browser-languagedetector

定义支持的语言列表 = ['zh', 'en']
定义默认语言 = 'zh'
定义 namespace 列表 = ['common', 'tasks', 'accounts', ...]

静态引入所有翻译 JSON:
  zh_common = import('./locales/zh/common.json')
  zh_tasks  = import('./locales/zh/tasks.json')
  en_common = import('./locales/en/common.json')
  ...

初始化 i18next:
  使用 react-i18next 插件
  使用 browser-languagedetector 插件（检测顺序: localStorage → navigator）
  
  配置:
    fallbackLng = 'en'
    supportedLngs = ['zh', 'en']
    defaultNS = 'common'
    ns = ['common', 'tasks', 'accounts', ...]
    interpolation.escapeValue = false  (React 已自动转义)
    
    resources = {
      zh: { common: zh_common, tasks: zh_tasks, ... },
      en: { common: en_common, tasks: en_tasks, ... }
    }
    
    detection = {
      order: ['localStorage', 'navigator']
      lookupLocalStorage: 'i18n-lang'
      caches: ['localStorage']
    }

导出 i18next 实例
```

**为什么用静态 import 而非异步加载？**

翻译 JSON 文件很小（每个 locale 估计 20-50KB），通过静态 import 会被 webpack 打包
进 JS bundle。这样做的好处：
1. 消除语言切换的异步加载延迟
2. 消除首次加载的闪烁问题（翻译资源随 JS 同步可用）
3. 不需要额外的 HTTP 请求
4. 文件自动被 Go embed 嵌入（作为 `_next/static/chunks/` 的一部分）

### 3.2 i18n/provider.tsx — React Provider

```
'use client'

引入 I18nextProvider from react-i18next
引入 i18next 实例 from ./config

组件 I18nProvider({ children }):
  副作用(初始化时):
    设置 document.documentElement.lang = i18next.language
    
  副作用(语言变化时):
    监听 i18next 的 'languageChanged' 事件
    更新 document.documentElement.lang

  返回:
    <I18nextProvider i18n={i18next}>
      {children}
    </I18nextProvider>
```

### 3.3 app/layout.tsx — 根布局集成

```
引入 I18nProvider from '@/i18n/provider'

组件 RootLayout({ children }):
  返回:
    <html lang="zh">   // 默认 zh，运行时由 provider 动态更新
      <body>
        <I18nProvider>
          {children}
        </I18nProvider>
      </body>
    </html>
```

**注意**：`<html lang="zh">` 是静态导出时的默认值。`I18nProvider` 会在客户端
hydration 后立即更新为用户实际语言。

### 3.4 components/LanguageSwitcher.tsx — 语言切换器

```
'use client'

引入 useTranslation from react-i18next

支持的语言 = [
  { code: 'zh', label: '中文', flag: '🇨🇳' },
  { code: 'en', label: 'English', flag: '🇺🇸' }
]

组件 LanguageSwitcher():
  const { i18n } = useTranslation()
  const [open, setOpen] = state(false)
  
  当前语言 = 支持的语言.find(l => l.code === i18n.language)
  
  切换语言(code):
    i18n.changeLanguage(code)
    setOpen(false)

  返回:
    <下拉按钮 显示当前语言 flag + label>
      <下拉菜单 显示所有语言选项>
        对每个语言:
          <选项 onClick=切换语言(code)>
            {flag} {label} {当前语言 ? '✓' : ''}
          </选项>
    </下拉按钮>
```

**放置位置**：Header 组件右侧，通知按钮旁边。

### 3.5 i18n/useFormatDate.ts — 日期格式化 hook

```
引入 useTranslation from react-i18next

导出 hook useFormatDate():
  const { i18n } = useTranslation()
  const locale = i18n.language === 'zh' ? 'zh-CN' : 'en-US'
  
  返回 {
    formatDateTime(date):
      new Date(date).toLocaleString(locale)
    
    formatDate(date):
      new Date(date).toLocaleDateString(locale)
    
    formatTime(date):
      new Date(date).toLocaleTimeString(locale)
    
    formatShortDate(date):
      new Date(date).toLocaleString(locale, { month:'numeric', day:'numeric', hour:'2-digit', minute:'2-digit' })
    
    formatRelative(date):
      // "3 分钟前" / "3 minutes ago"
      计算与当前时间差
      if < 1分钟: t('common.time.justNow')
      if < 1小时: t('common.time.minutesAgo', { count })
      if < 1天:   t('common.time.hoursAgo', { count })
      else:       formatDateTime(date)
  }
```

---

## 四、翻译文件示例

### zh/common.json

```json
{
  "nav": {
    "taskBoard": "任务看板",
    "monitor": "工作流监控",
    "accounts": "账号管理",
    "instances": "实例管理",
    "nodes": "节点管理",
    "proxies": "代理管理",
    "settings": "系统设置"
  },
  "action": {
    "save": "保存",
    "cancel": "取消",
    "delete": "删除",
    "edit": "编辑",
    "create": "创建",
    "refresh": "刷新",
    "confirm": "确认",
    "goBack": "返回",
    "viewDetail": "查看详情",
    "startExecution": "开始执行",
    "saving": "保存中...",
    "creating": "创建中..."
  },
  "status": {
    "pending": "待处理",
    "running": "运行中",
    "completed": "已完成",
    "failed": "失败",
    "cancelled": "已取消",
    "online": "在线",
    "offline": "离线",
    "idle": "空闲",
    "busy": "忙碌",
    "active": "活跃",
    "expired": "已过期",
    "disabled": "已禁用"
  },
  "error": {
    "notFound": "未找到",
    "unknown": "未知错误",
    "networkError": "网络错误"
  },
  "time": {
    "justNow": "刚刚",
    "minutesAgo": "{{count}} 分钟前",
    "hoursAgo": "{{count}} 小时前",
    "daysAgo": "{{count}} 天前",
    "createdAt": "创建于",
    "lastUsed": "上次使用",
    "lastHeartbeat": "最后心跳",
    "executionTime": "执行时间",
    "neverReported": "从未上报"
  },
  "confirm": {
    "deleteTitle": "确认删除",
    "deleteMessage": "确定要删除吗？此操作不可撤销。"
  }
}
```

### en/common.json

```json
{
  "nav": {
    "taskBoard": "Task Board",
    "monitor": "Workflow Monitor",
    "accounts": "Accounts",
    "instances": "Instances",
    "nodes": "Nodes",
    "proxies": "Proxies",
    "settings": "Settings"
  },
  "action": {
    "save": "Save",
    "cancel": "Cancel",
    "delete": "Delete",
    "edit": "Edit",
    "create": "Create",
    "refresh": "Refresh",
    "confirm": "Confirm",
    "goBack": "Go Back",
    "viewDetail": "View Details",
    "startExecution": "Start",
    "saving": "Saving...",
    "creating": "Creating..."
  },
  "status": {
    "pending": "Pending",
    "running": "Running",
    "completed": "Completed",
    "failed": "Failed",
    "cancelled": "Cancelled",
    "online": "Online",
    "offline": "Offline",
    "idle": "Idle",
    "busy": "Busy",
    "active": "Active",
    "expired": "Expired",
    "disabled": "Disabled"
  },
  "error": {
    "notFound": "Not Found",
    "unknown": "Unknown Error",
    "networkError": "Network Error"
  },
  "time": {
    "justNow": "Just now",
    "minutesAgo": "{{count}} min ago",
    "hoursAgo": "{{count}} hr ago",
    "daysAgo": "{{count}} days ago",
    "createdAt": "Created",
    "lastUsed": "Last used",
    "lastHeartbeat": "Last heartbeat",
    "executionTime": "Execution time",
    "neverReported": "Never reported"
  },
  "confirm": {
    "deleteTitle": "Confirm Delete",
    "deleteMessage": "Are you sure? This action cannot be undone."
  }
}
```

### zh/tasks.json

```json
{
  "board": {
    "title": "任务看板",
    "total": "共 {{count}} 个任务",
    "newTask": "新建任务",
    "noTasks": "暂无任务"
  },
  "card": {
    "viewDetail": "查看详情",
    "startExecution": "开始执行",
    "delete": "删除"
  },
  "create": {
    "title": "新建任务",
    "name": "任务名称",
    "namePlaceholder": "例如：修复登录页面 bug",
    "agentType": "Agent 类型",
    "selectInstance": "选择实例",
    "prompt": "任务提示词",
    "promptPlaceholder": "描述你希望 AI Agent 完成的任务...",
    "noInstance": "没有可用的 {{type}} 实例",
    "needInstance": "需要先创建一个运行中的实例才能创建任务",
    "goCreateInstance": "前往创建实例",
    "selectRunningInstance": "请选择一个运行中的实例",
    "createTask": "创建任务"
  },
  "detail": {
    "missingId": "缺少任务 ID",
    "notFound": "任务不存在",
    "goHome": "返回首页",
    "newRun": "新建 Run",
    "runHistory": "运行记录 ({{count}})",
    "taskConfig": "任务配置",
    "noRuns": "暂无运行记录",
    "cancel": "取消",
    "selectRun": "选择一个 Run 查看详情",
    "liveConnection": "实时连接",
    "node": "节点",
    "waitingEvents": "等待事件...",
    "error": "错误"
  }
}
```

---

## 五、组件改造模式

### 5.1 基本文本替换

```tsx
// 改造前
<h2 className="font-semibold">运行记录 ({runs.length})</h2>

// 改造后
const { t } = useTranslation('tasks')
<h2 className="font-semibold">{t('detail.runHistory', { count: runs.length })}</h2>
```

### 5.2 状态映射

```tsx
// 改造前
const statusText = (s: string) => {
  switch (s) {
    case 'active': return '活跃'
    case 'expired': return '已过期'
    default: return s
  }
}

// 改造后
const { t } = useTranslation()
const statusText = (s: string) => t(`status.${s}`, { defaultValue: s })
```

### 5.3 日期格式化

```tsx
// 改造前
{new Date(task.created_at).toLocaleString('zh-CN')}

// 改造后
const { formatDateTime } = useFormatDate()
{formatDateTime(task.created_at)}
```

### 5.4 确认对话框

```tsx
// 改造前
if (!confirm(`确定删除 Runner "${account}"？`)) return

// 改造后
const { t } = useTranslation('runners')
if (!confirm(t('confirmDelete', { name: account }))) return
```

### 5.5 Sidebar 导航

```tsx
// 改造前
const navigation = [
  { name: '任务看板', href: '/', icon: LayoutDashboard },
  { name: '工作流监控', href: '/monitor', icon: Activity },
  ...
]

// 改造后
const { t } = useTranslation()
const navigation = [
  { name: t('nav.taskBoard'), href: '/', icon: LayoutDashboard },
  { name: t('nav.monitor'), href: '/monitor', icon: Activity },
  ...
]
```

---

## 六、静态导出 + Go embed 兼容性

### 构建流程不变

```
STATIC_EXPORT=true npm run build
  → Next.js 静态导出到 web/out/
  → 翻译 JSON 已打包在 _next/static/chunks/*.js 中
  → go build 嵌入所有文件

运行时：
  浏览器下载 JS bundle（已包含所有语言的翻译）
  i18next 从内存中读取翻译，无额外请求
```

### 为什么不会增大二进制体积？

翻译 JSON 通过 `import` 语句被 webpack 打包进 JS bundle，而不是作为独立文件。
因此不会产生新的静态文件，只是 JS chunk 稍微变大（估计 +50KB 左右）。

### 验证清单

```
□ STATIC_EXPORT=true npm run build 正常完成
□ go build ./cmd/api-server 正常编译
□ 启动服务器，页面正常渲染
□ 浏览器语言设为英文，自动显示英文界面
□ 浏览器语言设为中文，自动显示中文界面
□ 语言切换器点击切换正常
□ 刷新页面后语言选择持久化
□ 所有页面路由正常工作
□ API 请求不受影响
```

---

## 七、TypeScript 类型安全（可选增强）

react-i18next 支持通过 TypeScript 声明文件确保翻译 key 的类型安全：

```
web/
├── i18n/
│   └── i18next.d.ts    ← 类型声明

声明文件内容（伪码）：
  引入 common.json 的类型
  引入 tasks.json 的类型
  ...
  
  声明 react-i18next 的 CustomTypeOptions:
    defaultNS = 'common'
    resources = {
      common: typeof common_json
      tasks: typeof tasks_json
      ...
    }

效果：
  t('nav.taskBoard')   ← ✅ 有自动补全
  t('nav.typo')        ← ❌ TypeScript 报错
```

这可以在基础改造完成后作为增强项添加，初期不阻塞。

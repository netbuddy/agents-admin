# 迁移指南与任务拆分

> 返回 [README](./README.md)

---

## 一、迁移原则

1. **渐进式迁移**：先搭建基础设施（任务 1-4），再逐页面替换文案（任务 5-16）
2. **不破坏现有功能**：每个任务完成后必须通过构建验证
3. **保持中文为默认语言**：`fallbackLng: 'zh'`，未翻译的 key 显示中文原文
4. **先迁移高频组件**：Sidebar → Header → 首页 → 各子页面
5. **翻译 JSON 与代码同步提交**：避免 key 不匹配

---

## 二、阶段划分

### 阶段 1：基础设施搭建（任务 1-4）

目标：i18n 框架就绪，语言切换可用，但页面文案尚未替换。

```
预计工作量：2-3 小时
验证标准：
  ✓ 语言切换器在 Header 中可见
  ✓ 点击切换后 document.lang 更新
  ✓ 刷新页面后语言选择保持
  ✓ STATIC_EXPORT=true npm run build 正常
  ✓ go build ./cmd/api-server 正常
```

### 阶段 2：核心组件迁移（任务 5-8）

目标：导航、头部、首页看板完成 i18n，用户可以感知语言切换效果。

```
预计工作量：3-4 小时
验证标准：
  ✓ Sidebar 7 个导航标签随语言切换
  ✓ Header 按钮文案随语言切换
  ✓ 首页看板标题、状态、按钮随语言切换
  ✓ 创建任务弹窗随语言切换
```

### 阶段 3：子页面全量迁移（任务 9-16）

目标：所有页面完成 i18n。

```
预计工作量：6-8 小时
验证标准：
  ✓ 所有页面无硬编码中文（代码注释除外）
  ✓ 所有 toLocaleString('zh-CN') 替换为 useFormatDate
  ✓ E2E 测试通过
```

### 阶段 4：收尾与验证（任务 17-20）

目标：日期格式化统一、HTML lang 动态更新、完整 E2E 测试。

```
预计工作量：2-3 小时
验证标准：
  ✓ 日期时间显示随语言切换
  ✓ <html lang> 动态更新
  ✓ 完整构建 + Go embed + 浏览器测试通过
```

---

## 三、逐任务详细说明

### 任务 1：安装依赖 + 初始化 i18n 配置

```
操作：
  1. cd web && npm install react-i18next i18next i18next-browser-languagedetector
  2. 创建 web/i18n/config.ts
  3. 配置 i18next 实例（语言检测、资源加载、namespace）

创建文件：
  web/i18n/config.ts

验证：
  npm run build 正常
```

### 任务 2：创建翻译文件结构

```
操作：
  1. 创建 web/i18n/locales/zh/ 和 web/i18n/locales/en/ 目录
  2. 创建 common.json（两种语言）
  3. 创建各 namespace JSON 文件（初始可为空对象 {}）

创建文件：
  web/i18n/locales/zh/common.json
  web/i18n/locales/zh/tasks.json
  web/i18n/locales/zh/accounts.json
  web/i18n/locales/zh/instances.json
  web/i18n/locales/zh/monitor.json
  web/i18n/locales/zh/runners.json
  web/i18n/locales/zh/nodes.json
  web/i18n/locales/zh/proxies.json
  web/i18n/locales/zh/settings.json
  （en/ 同理）

验证：
  TypeScript 编译无错误
```

### 任务 3：创建 I18nProvider 包装 RootLayout

```
操作：
  1. 创建 web/i18n/provider.tsx（I18nProvider 组件）
  2. 修改 web/app/layout.tsx，用 I18nProvider 包裹 children
  3. 确保 provider 在客户端初始化时更新 document.documentElement.lang

修改文件：
  web/app/layout.tsx
创建文件：
  web/i18n/provider.tsx

验证：
  浏览器控制台无错误
  document.documentElement.lang 等于浏览器语言
```

### 任务 4：实现语言切换器组件

```
操作：
  1. 创建 web/components/LanguageSwitcher.tsx
  2. 在 Header.tsx 中添加语言切换器（通知按钮右侧）
  3. 支持 zh/en 两种语言

创建文件：
  web/components/LanguageSwitcher.tsx
修改文件：
  web/components/layout/Header.tsx

验证：
  语言切换器下拉菜单可见
  点击切换后 localStorage 中写入语言偏好
  刷新后保持选择
```

### 任务 5：迁移 Sidebar 导航文案

```
修改文件：
  web/components/layout/Sidebar.tsx
翻译文件：
  zh/common.json（nav 节点）
  en/common.json（nav 节点）

改造点：
  navigation 数组中的 name 字段：7 个导航标签
  代码注释保持中文不变

改造模式：
  const { t } = useTranslation()
  { name: '任务看板', ... }  →  { name: t('nav.taskBoard'), ... }
```

### 任务 6：迁移 Header 组件文案

```
修改文件：
  web/components/layout/Header.tsx
翻译文件：
  zh/common.json（action 节点补充）
  en/common.json

改造点：
  "刷新" 按钮 title
  "通知" 按钮 title
  "打开菜单" aria-label
```

### 任务 7：迁移首页（看板）文案

```
修改文件：
  web/app/page.tsx
  web/components/kanban/TaskCard.tsx
  web/components/kanban/KanbanColumn.tsx
翻译文件：
  zh/tasks.json, en/tasks.json

改造点：
  看板列标题（待处理/运行中/已完成/失败）
  任务统计（共 X 个任务）
  新建任务按钮
  任务卡片中的状态标签、操作按钮
  日期格式化
```

### 任务 8：迁移 CreateTaskModal 文案

```
修改文件：
  web/components/CreateTaskModal.tsx
翻译文件：
  zh/tasks.json（create 节点）
  en/tasks.json

改造点：
  弹窗标题："新建任务"
  表单标签：任务名称、Agent 类型、选择实例、任务提示词
  占位符文本
  按钮：取消、创建任务、创建中...
  提示消息：没有可用实例、需要先创建实例
  alert() 调用
```

### 任务 9-16：子页面迁移

每个子页面的迁移模式相同：

```
步骤：
  1. 提取该页面的所有中文字符串到对应的 namespace JSON
  2. 在组件中引入 useTranslation(namespace)
  3. 逐一替换硬编码字符串为 t('key')
  4. 处理带变量的插值字符串
  5. 替换 toLocaleString('zh-CN') 为 useFormatDate
  6. 验证页面功能正常

按优先级排序（中文字符串越多越先迁移）：
  P1: accounts (54), monitor (47), instances (41)
  P1: TaskDetailPanel (71) + TaskDetailClient (22)
  P1: agent-output 组件群 (47+21+12+8+6+5 = 99)
  P1: proxies (29), runners (24)
  P2: nodes (13), settings (7)
```

### 任务 17：统一日期时间格式化

```
操作：
  1. 创建 web/i18n/useFormatDate.ts hook
  2. 在所有使用 toLocaleString('zh-CN') 的地方替换

创建文件：
  web/i18n/useFormatDate.ts
修改文件：
  所有包含日期格式化的组件（约 7 个文件）

改造模式：
  // 前
  {new Date(task.created_at).toLocaleString('zh-CN')}
  // 后
  const { formatDateTime } = useFormatDate()
  {formatDateTime(task.created_at)}
```

### 任务 18：更新 `<html lang>` 为动态值

```
修改文件：
  web/i18n/provider.tsx（已在任务 3 中预留）

操作：
  确保 I18nProvider 在语言变化时执行：
  document.documentElement.lang = i18n.language === 'zh' ? 'zh' : 'en'
```

### 任务 19：验证静态导出 + Go embed 兼容性

```
操作：
  1. STATIC_EXPORT=true npm run build
  2. 检查 web/out/ 目录结构，确认无异常
  3. go build ./cmd/api-server
  4. 启动服务器，用浏览器测试所有页面
  5. 确认二进制体积增长在预期范围内

验证清单：
  □ 静态导出成功，页面数量不变（不应翻倍）
  □ Go 编译成功
  □ 服务器启动正常
  □ 所有页面路由可访问
  □ API 端点不受影响
  □ 二进制体积增长 < 200KB
```

### 任务 20：E2E 测试

```
操作：
  使用 agent-browser 测试以下场景

测试用例：
  1. 默认语言检测
     - 设置浏览器语言为 en
     - 打开首页，验证导航标签为英文
  
  2. 默认语言检测（中文）
     - 设置浏览器语言为 zh-CN
     - 打开首页，验证导航标签为中文
  
  3. 手动切换语言
     - 打开首页（默认中文）
     - 点击语言切换器，选择 English
     - 验证导航标签变为英文
     - 验证按钮文案变为英文
  
  4. 语言持久化
     - 切换为英文
     - 刷新页面
     - 验证仍为英文
  
  5. 各页面路由 + 语言
     - 依次访问 /accounts, /nodes, /settings 等
     - 验证每个页面的文案都是当前语言
  
  6. 日期格式化
     - 中文模式：日期显示为 "2025/2/6 19:00:00" 格式
     - 英文模式：日期显示为 "2/6/2025, 7:00:00 PM" 格式
```

---

## 四、注意事项

### 4.1 不翻译的内容

| 内容 | 原因 |
|------|------|
| 代码注释中的中文 | 仅开发者可见，不影响用户 |
| API 返回的数据（任务名称、提示词等） | 用户输入的内容，不应翻译 |
| 技术术语（Agent、Runner、Run、Token） | 保持英文，技术词汇无需翻译 |
| Logo "Agent Kanban" | 品牌名，保持英文 |
| console.log 日志 | 仅开发者可见 |

### 4.2 翻译 key 命名规范

```
规范：
  namespace.section.key
  使用 camelCase
  
示例：
  common.nav.taskBoard       ✓
  common.action.save         ✓
  tasks.create.title          ✓
  tasks.detail.runHistory    ✓
  
反例：
  common.导航.任务看板        ✗ （不用中文 key）
  Common.Nav.TaskBoard       ✗ （不用 PascalCase）
  task-board                  ✗ （不用 kebab-case）
```

### 4.3 新增页面/组件的 i18n 开发流程

```
1. 在对应的 namespace JSON 中添加中文和英文翻译
2. 在组件中使用 useTranslation(namespace)
3. 使用 t('key') 代替硬编码字符串
4. 日期用 useFormatDate()
5. 提交前确认两种语言的 JSON 都已更新
```

### 4.4 未来扩展

```
新增语言（如日语）：
  1. 创建 web/i18n/locales/ja/ 目录
  2. 复制 en/*.json 并翻译
  3. 在 i18n/config.ts 中注册新语言
  4. 在 LanguageSwitcher 中添加选项
  5. 重新构建即可

不需要修改任何组件代码或 Go 后端代码。
```

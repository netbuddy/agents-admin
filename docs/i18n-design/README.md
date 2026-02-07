# 前端多语言（i18n）改造方案

> **状态**：📋 方案设计阶段  
> **最后更新**：2025-02  
> **目标**：支持浏览器语言自动检测 + 手动切换，初期支持中文（zh）和英文（en）

---

## 目录

1. [现状分析与改造范围](./01-current-analysis.md)  
   当前代码中硬编码中文字符串的完整清单、分类统计、日期格式化现状
2. [技术方案对比](./02-approach-comparison.md)  
   三种主流 i18n 方案的详细对比与推荐理由
3. [实施方案详解](./03-implementation-plan.md)  
   推荐方案 react-i18next 的架构设计、目录结构、核心代码伪码
4. [迁移指南与任务拆分](./04-migration-guide.md)  
   逐文件迁移清单、优先级排序、TODO 检查表

---

## 核心结论

### 推荐方案：纯客户端 react-i18next

| 维度 | 说明 |
|------|------|
| **方案** | react-i18next + i18next-browser-languagedetector |
| **原理** | 翻译 JSON 文件随静态资源嵌入，浏览器运行时加载并替换文本 |
| **路由** | 无需改动，不引入 `[locale]` 路径段 |
| **二进制体积** | 增加约 50-100KB（翻译 JSON 文件），不会翻倍 |
| **Go 后端** | 无需任何改动 |
| **检测方式** | 首次访问：浏览器 `navigator.language` → 匹配支持的 locale |
| **持久化** | localStorage 记住用户选择 |
| **切换方式** | Header 中的语言切换器组件 |

### 为什么不选 `[locale]` 子路径方案？

本项目是**管理后台**（非公开网站），SEO 无意义。`[locale]` 方案会导致：
- 静态 HTML 文件数量翻倍（每个 locale 一套），二进制体积翻倍
- 所有页面文件迁移到 `app/[locale]/` 目录
- Go SPA handler 需要处理 locale 前缀路由
- 所有内部链接需要添加 locale 前缀
- 改造工作量大且收益低

---

## 改造范围概览

| 类别 | 文件数 | 中文字符串数 | 说明 |
|------|--------|-------------|------|
| 页面组件（app/） | 11 | ~258 | 各管理页面 |
| 通用组件（components/） | 12 | ~231 | 侧边栏、模态框、看板卡片等 |
| 布局（layout.tsx） | 1 | 1 | `<html lang="zh">` + metadata |
| 日期格式化 | 7处 | — | 硬编码 `'zh-CN'` locale |
| **合计** | **23+** | **~489** | — |

---

## TODO 检查表

> 每个任务代表一个可独立交付的功能切片

| # | 任务 | 状态 | 优先级 | 备注 |
|---|------|------|--------|------|
| 1 | 安装依赖 + 初始化 i18n 配置 | ⬜ | P0 | react-i18next + languagedetector |
| 2 | 创建翻译文件结构（zh.json / en.json） | ⬜ | P0 | 按模块组织 namespace |
| 3 | 创建 I18nProvider 包装 RootLayout | ⬜ | P0 | 自动检测浏览器语言 |
| 4 | 实现语言切换器组件（Header 中） | ⬜ | P0 | 下拉菜单 + localStorage 持久化 |
| 5 | 迁移 Sidebar 导航文案 | ⬜ | P1 | 7 个导航项 |
| 6 | 迁移 Header 组件文案 | ⬜ | P1 | 刷新、通知等 |
| 7 | 迁移首页（看板）文案 | ⬜ | P1 | 任务状态、操作按钮 |
| 8 | 迁移 CreateTaskModal 文案 | ⬜ | P1 | 表单标签、按钮、提示 |
| 9 | 迁移 accounts 页面文案 | ⬜ | P1 | 最多中文字符串（54处） |
| 10 | 迁移 monitor 页面文案 | ⬜ | P1 | 47处 |
| 11 | 迁移 instances 页面文案 | ⬜ | P1 | 41处 |
| 12 | 迁移 proxies 页面文案 | ⬜ | P1 | 29处 |
| 13 | 迁移 runners 页面文案 | ⬜ | P1 | 24处 + alert() 需替换 |
| 14 | 迁移 TaskDetailPanel + TaskDetailClient | ⬜ | P1 | 71+22处 |
| 15 | 迁移 agent-output 组件群 | ⬜ | P1 | DebugPanel/AgentOutput/FileBlock/CommandBlock 等 |
| 16 | 迁移 nodes/settings 页面文案 | ⬜ | P2 | 较少字符串 |
| 17 | 统一日期时间格式化（useLocale hook） | ⬜ | P1 | 替换硬编码 'zh-CN' |
| 18 | 更新 `<html lang>` 为动态值 | ⬜ | P1 | 根据当前语言设置 |
| 19 | 验证静态导出 + Go embed 兼容性 | ⬜ | P0 | 确保 i18n 不破坏构建 |
| 20 | E2E 测试：语言切换功能 | ⬜ | P1 | agent-browser 测试 |

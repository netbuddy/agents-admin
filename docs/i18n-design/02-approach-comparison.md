# 技术方案对比

> 返回 [README](./README.md)

---

## 概述

针对本项目的技术约束（Next.js 14.1.0 + `output: 'export'` + Go embed + 全客户端组件），
调研了三种主流 i18n 方案，并从多个维度进行对比。

---

## 方案一：`[locale]` 子路径 + next-international

### 工作原理

```
app/
├── [locale]/           ← 所有页面移入此目录
│   ├── layout.tsx      ← I18nProviderClient 包裹
│   ├── page.tsx
│   ├── accounts/page.tsx
│   └── ...
└── locales/
    ├── client.ts       ← createI18nClient({ zh, en })
    ├── server.ts       ← createI18nServer({ zh, en })
    ├── zh.ts
    └── en.ts

静态导出时，generateStaticParams 返回 [{locale:'zh'}, {locale:'en'}]
每个页面生成两份 HTML：/zh/accounts.html, /en/accounts.html
URL: http://localhost:8080/zh/accounts
```

### 优点
- 标准 Next.js 模式，社区最推荐
- URL 直接体现语言（`/en/accounts`），可分享带语言的链接
- SEO 友好（对搜索引擎有意义的场景）
- 构建时确定语言，无闪烁

### 缺点
- **所有页面文件必须迁移到 `app/[locale]/` 目录**
- **静态 HTML 翻倍**：当前 44 文件 × 2 语言 = 88 文件，二进制增大约 1.9MB
- **Go SPA handler 需要大改**：必须处理 locale 前缀路由、默认语言重定向
- **所有内部链接需要更新**：`href="/accounts"` → `href="/zh/accounts"` 或使用特殊 Link 组件
- **Next.js 14.1.0 的 `generateStaticParams` 已有兼容性问题**（tasks/[id] 的教训）
- 新增语言需要重新构建二进制

### 对本项目的适用性：⚠️ 不推荐

原因：改造成本高，且本项目是管理后台，不需要 SEO。`generateStaticParams` 在 14.1.0 上
已有兼容性问题（参见 tasks/[id] 的踩坑记录），引入 `[locale]` 动态段可能再次触发。

---

## 方案二：纯客户端 react-i18next（⭐ 推荐）

### 工作原理

```
web/
├── app/                    ← 目录结构完全不变
│   ├── layout.tsx          ← 添加 I18nextProvider
│   ├── page.tsx            ← 中文字符串替换为 t('key')
│   └── ...
├── i18n/
│   ├── config.ts           ← i18next 初始化 + 语言检测配置
│   ├── locales/
│   │   ├── zh/
│   │   │   ├── common.json ← 通用文案
│   │   │   ├── tasks.json  ← 任务相关
│   │   │   └── ...
│   │   └── en/
│   │       ├── common.json
│   │       ├── tasks.json
│   │       └── ...
│   └── provider.tsx        ← React context provider
└── components/
    └── LanguageSwitcher.tsx ← 语言切换下拉菜单

运行时流程：
  1. 浏览器打开页面
  2. i18next-browser-languagedetector 检测 navigator.language
  3. 匹配到 zh/en，加载对应 JSON 翻译文件
  4. React 组件中 t('key') 返回翻译文本
  5. 用户切换语言 → i18next.changeLanguage('en') → 所有组件重渲染
  6. localStorage 记住选择，下次访问自动使用
```

### 优点
- **零路由改动**：不引入 `[locale]` 路径段，URL 保持 `/accounts`
- **零 Go 后端改动**：SPA handler 完全不需要修改
- **最小二进制体积增长**：只增加翻译 JSON（估计 50-100KB），不翻倍 HTML
- **最大生态系统**：react-i18next 周下载量 400 万+，社区支持最广
- **TypeScript 支持好**：可通过类型推导确保翻译 key 不遗漏
- **namespace 支持**：按模块拆分翻译文件，按需加载
- **插值/复数/嵌套 key 完善**：处理各种复杂翻译场景
- **浏览器语言检测内置**：i18next-browser-languagedetector
- **渐进式迁移**：可以逐个组件替换，不需要一次性全改

### 缺点
- **首次加载可能闪烁**：翻译 JSON 加载完成前可能短暂显示 key 或空白
  - **缓解**：将翻译 JSON 内联到 bundle（静态 import），消除异步加载
- **不利于 SEO**：HTML 源码中没有翻译内容（但管理后台不需要 SEO）

### 对本项目的适用性：✅ 强烈推荐

原因：
1. 管理后台不需要 SEO
2. 所有组件已经是客户端组件
3. 改造量最小、风险最低
4. 不影响现有 Go embed 架构
5. 生态成熟，遇到问题容易找到解决方案

---

## 方案三：next-export-i18n

### 工作原理

```
纯客户端方案，使用 query param (?lang=en) 或 localStorage 存储语言选择。
提供 useTranslation hook 和 LanguageSwitcher 组件。
翻译文件为 JSON 格式。
```

### 优点
- 专为 `next export` 设计
- 零路由改动
- 简单轻量

### 缺点
- **生态较小**：npm 周下载量仅 ~2000，远小于 react-i18next
- **query param 模式有侵入性**：URL 变为 `/accounts?lang=en`
- **所有内部 Link 需要用 `LinkWithLocale` 替代**
- **无 namespace 支持**：所有翻译放在一个大 JSON 中
- **无 TypeScript 类型推导**
- **不支持复数、嵌套插值等高级特性**

### 对本项目的适用性：⚠️ 可用但不推荐

原因：生态太小，功能有限。react-i18next 在同样「纯客户端」的前提下提供了更完善的功能。

---

## 方案对比总结

| 维度 | [locale] 子路径 | react-i18next ⭐ | next-export-i18n |
|------|----------------|-------------------|------------------|
| **路由改动** | 所有文件迁移到 [locale]/ | 无 | 无 |
| **Go 后端改动** | 需要处理 locale 前缀 | 无 | 无 |
| **二进制体积** | ×N（每 locale 一套 HTML） | +50-100KB | +50-100KB |
| **URL 结构** | `/en/accounts` | `/accounts` | `/accounts?lang=en` |
| **SEO** | 好 | 无（不需要） | 无（不需要） |
| **闪烁** | 无 | 可消除（静态 import） | 可能有 |
| **TypeScript** | 好 | 好 | 弱 |
| **namespace** | 支持 | 支持 | 不支持 |
| **插值/复数** | 支持 | 支持 | 基础 |
| **NPM 周下载** | ~50K | ~4M | ~2K |
| **浏览器检测** | 需自行实现 | 内置插件 | 内置 |
| **迁移工作量** | 高 | 中 | 中 |
| **风险** | 高（generateStaticParams） | 低 | 低 |

---

## 最终推荐

**方案二：react-i18next + i18next-browser-languagedetector**

核心理由：
1. 本项目是管理后台，SEO 无意义 → 排除 [locale] 方案
2. 所有组件已是客户端组件 → 纯客户端方案天然契合
3. react-i18next 生态最大、功能最全 → 优于 next-export-i18n
4. 零路由改动 + 零 Go 后端改动 → 最低风险
5. 渐进式迁移 → 可按优先级逐步替换

### 需要安装的依赖

```bash
npm install react-i18next i18next i18next-browser-languagedetector
```

- **i18next**：核心国际化框架
- **react-i18next**：React 绑定层（提供 `useTranslation` hook、`I18nextProvider`）
- **i18next-browser-languagedetector**：浏览器语言自动检测

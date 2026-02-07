# Go后端与前端合并为单一二进制文件技术方案

> **状态**：✅ 已实现  
> **最后更新**：2025-02

---

## 一、go:embed 工作原理

> **核心结论：`go:embed` 不是启动前端进程，而是在编译时将文件字节写入二进制，运行时从内存读取。**

### 1.1 编译时：文件 → 二进制 data 段

```
┌─────────────────────────────────────────────────────────────┐
│  go build ./cmd/api-server                                  │
│                                                             │
│  1. 编译器扫描源码，发现 //go:embed all:out 指令            │
│  2. 递归读取 web/out/ 目录下所有文件（含 _next/）            │
│  3. 将每个文件的原始字节写入二进制文件的 .rodata 段          │
│  4. 在 embed.FS 结构体中记录文件名→偏移量的映射表            │
│                                                             │
│  结果：api-server 二进制 = Go代码 + 前端静态文件字节         │
│        （31MB = ~29MB Go + ~1.9MB 前端）                    │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 运行时：从内存读取，零磁盘 I/O

```
┌─────────────────────────────────────────────────────────────┐
│  ./api-server                                               │
│                                                             │
│  浏览器请求 GET /_next/static/chunks/app/page-xxx.js        │
│       │                                                     │
│       ▼                                                     │
│  SPA Handler                                                │
│       │                                                     │
│       ▼                                                     │
│  embed.FS.Open("_next/static/chunks/app/page-xxx.js")       │
│       │                                                     │
│       ▼                                                     │
│  从二进制 .rodata 段中按偏移量直接读取字节（内存操作）        │
│       │                                                     │
│       ▼                                                     │
│  http.FileServer 设置 Content-Type 并返回给浏览器            │
│                                                             │
│  整个过程：无磁盘读取、无子进程、无网络请求                  │
└─────────────────────────────────────────────────────────────┘
```

### 1.3 关键特性

| 特性 | 说明 |
|------|------|
| **编译时嵌入** | 文件内容在 `go build` 时读取，之后二进制不再依赖原始文件 |
| **只读** | `embed.FS` 是不可变的，运行时无法修改嵌入的文件 |
| **零运行时开销** | 从内存读取，性能等同于读取程序常量 |
| **线程安全** | `embed.FS` 可在多个 goroutine 中并发使用 |
| **标准 fs.FS 接口** | 兼容 `http.FileServer`、`template.ParseFS` 等标准库 |
| **路径规则** | 只能引用源文件所在目录及子目录，不能用 `../` |
| **`all:` 前缀** | 包含 `.` 和 `_` 开头的文件/目录（默认排除） |

### 1.4 与其他方案的对比

| 方案 | 部署产物 | 运行时依赖 | 前端更新方式 |
|------|---------|-----------|-------------|
| **go:embed（本方案）** | 单一二进制 | 无 | 重新编译 |
| Node.js + Go 分离 | 两个进程 | Node.js 运行时 | 独立部署 |
| Docker 多服务 | 容器镜像 | Docker | 重建镜像 |
| Nginx 反向代理 | Go 二进制 + 静态文件目录 | Nginx + 文件系统 | 替换文件 |

> **参考来源**：[Go embed 官方文档](https://pkg.go.dev/embed)、[JetBrains Go embed 教程](https://blog.jetbrains.com/go/2021/06/09/how-to-use-go-embed-in-go-1-16/)

---

## 二、模式分离策略

| 模式 | 前端 | 后端 | 构建命令 |
|------|------|------|----------|
| **开发模式** | Next.js dev server (`make run-web`) | `make run-api-dev`（`-tags dev`） | 各自独立 |
| **生产模式** | 静态导出嵌入二进制 | 单一可执行文件 | `make release` |

### 开发模式

```bash
# 终端1 - 前端（带 API 反向代理到 :8080）
make run-web          # http://localhost:3002

# 终端2 - 后端（不嵌入前端）
make run-api-dev      # http://localhost:8080
```

### 生产模式

```bash
# 一键构建（前端静态导出 → embed → Go 二进制）
make release          # 输出 bin/api-server-linux-amd64 等

# 运行（前后端一体，单端口）
./bin/api-server-linux-amd64   # http://localhost:8080
```

---

## 二、实际项目结构

```
agents-admin/
├── cmd/api-server/
│   ├── main.go              # 入口：根据 web.IsEmbedded() 决定是否包装 SPA handler
│   └── static.go            # 【新增】SPA handler：静态文件服务 + 路由回退
├── web/
│   ├── embed.go             # 【新增】生产模式：//go:embed all:out
│   ├── embed_dev.go         # 【新增】开发模式：空实现（-tags dev）
│   ├── next.config.js       # 【修改】条件 output: 'export'
│   ├── out/                 # 静态导出产物（gitignore）
│   └── app/
│       └── tasks/detail/    # 【修改】原 [id] 动态路由改为查询参数
├── Makefile                 # 【修改】新增 web-build / release 目标
└── .gitignore               # 已含 web/out/ 和 bin/
```

---

## 三、核心实现文件

### 1. web/embed.go（生产模式）

```go
//go:build !dev

package web

import (
    "embed"
    "io/fs"
)

// 必须使用 all: 前缀，否则 _next/ 目录被排除（_ 开头）
//go:embed all:out
var staticFiles embed.FS

func StaticFS() (fs.FS, error) {
    return fs.Sub(staticFiles, "out")
}

func IsEmbedded() bool { return true }
```

### 2. web/embed_dev.go（开发模式）

```go
//go:build dev

package web

import "io/fs"

func StaticFS() (fs.FS, error) { return nil, nil }
func IsEmbedded() bool         { return false }
```

### 3. cmd/api-server/static.go（SPA handler 详解）

#### 核心思想

`newSPAHandler` 是一个 HTTP 中间件，它**包装**了原有的 API 路由 handler。
所有请求先经过它，按优先级决定由谁处理。

**关键：它不是基于文件扩展名匹配，而是直接尝试从 embed.FS 中打开文件。**
如果文件存在就服务，不存在就回退——因此新增任何文件类型都自动生效，无需修改代码。

#### 请求处理流程图

```
浏览器请求 GET /some/path
       │
       ▼
  ┌─ 步骤1: 是否为后端路由？ ─────────────────────────┐
  │  匹配: /api/*、/ws/*、/ttyd/*、/health、/metrics   │
  │  判断方式: 字符串前缀匹配                          │
  └────────────────────────────────────────────────────┘
       │ 是                              │ 否
       ▼                                 ▼
  apiHandler.ServeHTTP()      ┌─ 步骤2: 是根路径 / 吗？ ─┐
  （Go 后端处理）              └─────────────────────────────┘
                                   │ 是              │ 否
                                   ▼                 ▼
                            直接返回 index.html  ┌─ 步骤3: embed.FS 中有这个文件？ ─┐
                            （预加载在内存中）    │  方式: fsys.Open(path) 尝试打开    │
                                                │  检查: 打开成功 && 不是目录          │
                                                └──────────────────────────────────────┘
                                                     │ 有                     │ 没有
                                                     ▼                        ▼
                                              fileServer.ServeHTTP()  ┌─ 步骤4: 加 .html 后有？ ─┐
                                              （返回 JS/CSS/图片等）   │  /accounts → accounts.html │
                                                                      │  只对无扩展名路径尝试       │
                                                                      └────────────────────────────┘
                                                                           │ 有            │ 没有
                                                                           ▼               ▼
                                                                    直接读取 .html      直接返回 index.html
                                                                    文件内容返回         （SPA 客户端路由接管）

  注意：步骤4和步骤5的 HTML 返回都是直接从 fs.FS 读取内容写入 response，
  而不是通过 http.FileServer。因为 FileServer 对 /index.html 会发送
  301 重定向到 ./（相对路径），导致非根路径产生无限重定向循环。
```

#### 各步骤的具体匹配示例

| 请求 URL | 步骤1 | 步骤2 | 步骤3 | 最终处理 |
|----------|-------|-------|-------|---------|
| `GET /api/v1/tasks` | ✅ 匹配 `/api/` | — | — | Go 后端 API |
| `GET /ws/runs/xxx/events` | ✅ 匹配 `/ws/` | — | — | Go WebSocket |
| `GET /_next/static/chunks/page-xxx.js` | ❌ | ✅ 文件存在 | — | 返回 JS 文件 |
| `GET /_next/static/css/xxx.css` | ❌ | ✅ 文件存在 | — | 返回 CSS 文件 |
| `GET /favicon.ico` | ❌ | ✅/❌ 取决于是否存在 | — | 返回文件或 index.html |
| `GET /accounts` | ❌ | ❌ 无此文件 | ✅ `accounts.html` 存在 | 返回 accounts 页面 |
| `GET /tasks/detail` | ❌ | ❌ 无此文件 | ✅ `tasks/detail.html` 存在 | 返回任务详情页 |
| `GET /tasks/detail?id=abc` | ❌ | ❌ 无此文件 | ✅ `tasks/detail.html` 存在 | 返回任务详情页 |
| `GET /some/unknown/path` | ❌ | ❌ 无此文件 | ❌ 无对应 .html | 返回 index.html（SPA 兜底） |

#### 为什么不会因新增文件类型而出问题？

```
tryServeFile() 的判断逻辑：
  1. fsys.Open(path) → 能打开就说明文件存在
  2. stat.IsDir() → 确认不是目录
  3. 无任何扩展名检查、无 MIME 类型白名单

http.FileServer 的 Content-Type 处理：
  Go 标准库自动根据扩展名设置 Content-Type（内置 MIME 类型表）
  即使是未知扩展名也会返回 application/octet-stream
```

**结论：无论新增 `.wasm`、`.webp`、`.json`、`.map` 还是任何其他类型的文件，只要它在 `web/out/` 目录中被构建出来，就会被自动嵌入并正确服务。**

### 4. cmd/api-server/main.go（集成）

```go
import "agents-admin/web"

// 根据嵌入状态包装 handler
var handler http.Handler = h.Router()
if web.IsEmbedded() {
    staticFS, _ := web.StaticFS()
    handler = newSPAHandler(handler, staticFS)
}
```

### 5. web/next.config.js（条件导出）

```javascript
const isStaticExport = process.env.STATIC_EXPORT === 'true'

const nextConfig = {
  ...(isStaticExport ? { output: 'export' } : {}),
  ...(!isStaticExport ? { async rewrites() { /* 代理到 :8080 */ } } : {}),
}
```

---

## 四、关键技术决策

### 1. embed 文件位置

| 原方案（错误） | 实际方案 |
|----------------|----------|
| `cmd/api-server/static/` 中 `//go:embed web/dist/*` | `web/embed.go` 中 `//go:embed all:out` |

**原因**：`//go:embed` 只能引用相对于源文件的路径，不支持 `../`。embed 文件必须与目标目录同级。

### 2. `all:` 前缀

**必须使用 `//go:embed all:out`**，因为 Next.js 输出的 `_next/` 目录以 `_` 开头，默认被排除。

### 3. 动态路由处理：为什么只有 tasks 需要特殊处理？

#### 前端路由 vs 后端 API 路由——两个完全不同的东西

```
前端页面路由（Next.js App Router）：        后端 API 路由（Go HTTP Router）：
  /                    ← 静态               /api/v1/tasks/:id      ← Go 处理
  /accounts            ← 静态               /api/v1/tasks/:id/runs ← Go 处理
  /nodes               ← 静态               /api/v1/runs/:id       ← Go 处理
  /runners             ← 静态               /ws/runs/:id/events    ← Go 处理
  /proxies             ← 静态               /health                ← Go 处理
  /instances           ← 静态
  /monitor             ← 静态
  /settings            ← 静态
  /tasks/[id]          ← ❌ 动态路由！

静态导出只影响「前端页面路由」，与后端 API 路由无关。
后端 API 的 :id 参数由 Go HTTP router 在运行时解析，不受 Next.js 影响。
```

#### 为什么 /tasks/[id] 是唯一的问题？

在整个前端中，**只有 `app/tasks/[id]/page.tsx` 使用了 Next.js 的动态路由段**（方括号 `[id]` 语法）。这意味着 URL 中的 `id` 是路径的一部分（如 `/tasks/abc123`），Next.js 需要在构建时知道所有可能的 `id` 值来生成对应的 HTML 文件。

其他所有页面（`/accounts`、`/nodes` 等）的 URL 是固定的，不含任何动态参数。它们通过 `useEffect` + `fetch()` 在客户端运行时获取数据，不依赖 URL 路径中的参数。

#### 静态导出对动态路由的要求

```
Next.js output: 'export' 要求：
  所有动态路由必须提供 generateStaticParams()
  编译器根据返回值预生成 HTML（如 /tasks/abc → tasks/abc.html）

问题：
  Next.js 14.1.0 对 generateStaticParams() 的检测存在 bug
  即使正确导出该函数，构建仍报错 "missing generateStaticParams()"
  尝试过的方案：同步/异步函数、空数组返回、dynamicParams=false，均失败
```

#### 解决方案：查询参数代替路径参数

```
原路由:  /tasks/abc123          → app/tasks/[id]/page.tsx
新路由:  /tasks/detail?id=abc123 → app/tasks/detail/page.tsx

detail/ 是固定路径（静态路由），id 通过 ?query 传递
useSearchParams() 在客户端运行时读取，不影响静态导出
```

### 4. Rewrites 与静态导出互斥

`output: 'export'` 不支持 `rewrites()`。通过环境变量 `STATIC_EXPORT` 条件切换：
- 开发模式：启用 rewrites（代理 `/api/*` 到 Go 后端）
- 生产模式：禁用 rewrites（同源，无需代理）

---

## 五、构建验证结果

| 验证项 | 结果 |
|--------|------|
| `STATIC_EXPORT=true npm run build` | ✅ 12 页面全部静态导出 |
| `go build ./cmd/api-server`（生产） | ✅ 编译成功，含嵌入前端 |
| `go build -tags dev ./cmd/api-server`（开发） | ✅ 编译成功，无嵌入 |
| `npm run build`（常规开发构建） | ✅ 编译成功 |
| 二进制体积 | 31MB（后端 ~29MB + 前端 ~1.9MB） |
| 前端静态资源 | 1.9MB / 44 文件 |

---

## 六、Makefile 命令速查

```bash
make web-build       # 构建前端静态文件
make web-clean       # 清理前端产物
make release         # 完整生产构建（多平台）
make release-linux   # 仅 Linux 版本
make run-api-dev     # 开发模式后端（-tags dev）
make run-web         # 前端开发服务器
make clean           # 清理所有产物
```

---

## 七、风险与缓解

| 风险项 | 影响 | 缓解措施 |
|--------|------|----------|
| Next.js 静态导出限制 | 中 | 动态路由已改为查询参数；所有页面均为客户端渲染 |
| 二进制体积增大 (+1.9MB) | 低 | 前端资源经 Next.js 优化压缩 |
| 构建流程依赖 Node.js | 低 | 仅构建时需要，CI/CD 已配置 |
| `_next/` 目录排除 | 高 | 已使用 `all:` 前缀解决 |

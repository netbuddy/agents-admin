# Node Manager 单二进制 + Web 初始化向导 技术方案

> **版本**: v2（根据讨论反馈修订）

## 1. 需求概述

将 Node Manager 设计为一个**自包含的单二进制程序**，具备以下特性：

1. 二进制内嵌一个 Web 配置页面
2. **首次运行**时启动 Web 服务，用户通过浏览器完成所有运行时参数配置
3. 配置保存为 YAML 文件后，Node Manager 重启进入**工作模式**
4. 进入工作模式后，Web 配置页面**不可再访问**
5. 同时支持**直接运行二进制**和 **deb 包安装后 systemd 自动运行**两种部署方式

## 2. 业界参考

### 2.1 AdGuard Home（最佳参考）

AdGuard Home 是 Go 编写的单二进制程序，实现了几乎完全一致的模式：

| 阶段 | 行为 |
|------|------|
| **首次运行**（无 config.yaml） | 启动 Web 安装向导（默认 `:3000`），注册 `/install/*` 路由 |
| **配置完成** | 写入 config.yaml，同进程内切换到正常控制路由 |
| **后续运行**（有 config.yaml） | 跳过向导，直接启动 DNS + Web 管理面板 |

关键设计点：
- **检测方式**：检查配置文件是否存在（`detectFirstRun`）
- **API 设计**：`GET /install/get_addresses` → `POST /install/check_config` → `POST /install/configure`
- **切换机制**：配置写入后，同进程设置 `firstRun=false`，后续请求走正常 handler

### 2.2 Jupyter Notebook Token 安全模型（安全参考）

Jupyter 在远程部署场景下的安全设计：
- 启动时生成随机 token，打印到终端/journal
- 所有请求必须携带 `?token=xxx` 或 `Authorization: token xxx`
- 首次通过 token 访问后设置 Cookie，后续请求自动认证
- 这是**无预设密码、无证书、远程访问**场景下的业界标准做法

### 2.3 与本项目的差异

| | AdGuard Home / Gitea | 本方案 |
|---|---|---|
| 配置后 Web 状态 | 切换为管理面板 | **完全关闭**（Node Manager 不需要 Web 管理） |
| 重配置方式 | 通过 Web 管理面板 | 删除配置文件 → 重启 / `--reconfigure` |
| 前端技术 | React SPA 嵌入 | 轻量 HTML（无需 Next.js） |
| 监听方式 | 默认 localhost | **默认 0.0.0.0**（远程部署为主要场景） |
| 安全模型 | 无特殊保护 | **Jupyter 式 Token 认证**（适配远程场景） |

## 3. 可行性分析

### 3.1 方案优势

- **✅ 部署极简**：一个二进制 → 运行 → 浏览器配置 → 完成，零手动编辑配置文件
- **✅ 降低出错率**：表单校验 + 连接测试，避免 YAML 手写错误
- **✅ 业界验证**：AdGuard Home、Gitea 等成熟项目已验证此模式
- **✅ 技术可复用**：项目已有 `go:embed` 嵌入前端的完整方案（api-server）
- **✅ 二进制体积可控**：配置向导只需轻量 HTML/JS（~50KB），远小于完整前端（~2MB）

### 3.2 潜在风险与应对

| 风险 | 严重度 | 应对策略 |
|------|--------|---------|
| **安全暴露**：0.0.0.0 监听时网络上任何人可尝试访问 | 高 | Setup Token 认证（参见 4.7 节） |
| **配置错误无法修改**：Web 关闭后用户无法通过 UI 修正 | 中 | 提供 `--reconfigure` 命令行参数重新启动向导 |
| **deb 安装自动启动**：systemd 启动时无人交互 | 中 | 首次启动检测到无配置 → 打印 URL+Token 到 journal → 等待配置 |
| **多实例冲突**：多个 Node Manager 用同一端口 | 低 | 配置向导端口可通过 `--setup-port` 指定 |

### 3.3 不建议的替代方案

| 替代方案 | 否决原因 |
|---------|---------|
| 纯命令行交互式配置 | 不适合 deb 包 systemd 自动启动场景；无法做连接测试的友好 UI |
| 始终保留 Web 管理页 | Node Manager 定位是无头 worker，长期暴露 HTTP 端口增加攻击面 |
| 预写配置文件模板 | 回到手动编辑 YAML 的老路，且模板参数因环境而异 |

## 4. 架构设计

### 4.1 运行时状态机

```
┌─────────────────────────────────────────────────────────┐
│                      程序启动                             │
│                        │                                 │
│             检查配置文件是否存在                            │
│               /    \                                      │
│             不存在    存在                                  │
│              │        │                                   │
│     ┌────────▼───────┐ │                                   │
│     │  Setup Mode    │ │                                   │
│     │                │ │                                   │
│     │ HTTP 0.0.0.0   │ │                                   │
│     │ :15700         │ │                                   │
│     │ Token 认证     │ │                                   │
│     │ /setup/*       │ │                                   │
│     │ 等待用户配置    │ │                                   │
│     └────────┬───────┘ │                                   │
│              │         │                                   │
│        写入配置文件     │                                   │
│              │         │                                   │
│    ┌─────────▼─────────▼──┐                                │
│    │    Worker Mode       │                                │
│    │                      │                                │
│    │ 心跳 + 任务轮询       │                                │
│    │ 无 HTTP 端口          │                                │
│    └──────────────────────┘                                │
└─────────────────────────────────────────────────────────┘
```

### 4.2 模块划分

```
cmd/nodemanager/
├── main.go                  # 入口：状态检测 + 分支启动
├── setup.go                 # Setup Mode: HTTP server + API handlers + Token 认证
└── setup_static.go          # 嵌入静态文件声明

internal/nodemanager/
├── setup/                   # Setup 向导模块（新增）
│   ├── wizard.go            # 向导核心逻辑
│   ├── validator.go         # 配置验证（连接测试等）
│   └── config_writer.go     # 配置文件写入
├── manager.go               # NodeManager 主体（现有，不变）
└── ...

web/nodemanager-setup/       # 配置向导前端（新增）
├── index.html               # 单页配置表单
├── setup.js                 # 表单逻辑 + API 调用
└── setup.css                # 样式
```

### 4.3 前端方案选型

**推荐：纯 HTML + Vanilla JS（无框架）**

理由：
- 配置向导只有一个页面，无需 React/Vue/Next.js
- 体积极小（~30-50KB），对比完整 SPA（~2MB）
- 无构建步骤，直接 `go:embed`
- 维护简单，不引入前端工具链依赖

页面结构（单页分步表单）：
```
Step 1: 基本信息
  - Node ID（自动生成 hostname-随机后缀，可修改）
  - API Server URL（http:// 或 https://）
  - 工作空间目录（默认 /var/lib/agents-admin/workspaces 或 /tmp/agents-workspaces）

Step 2: 中间件连接（可选）
  - Redis URL（可选，留空则退化为 HTTP 轮询模式）

Step 3: HTTPS 客户端配置（当 API Server URL 为 https:// 时显示）
  - CA 证书获取方式：
    - 选项 A: 从 MinIO 下载（输入 MinIO endpoint + bucket + object key）
    - 选项 B: 指定本地文件路径
    - 选项 C: 跳过证书验证（仅开发环境）

Step 4: 确认 & 连接测试
  - 显示完整配置摘要
  - "测试连接"按钮（验证 API Server / Redis 可达性）
  - "保存并启动"按钮
```

### 4.4 配置向导 API 设计

所有 API 请求都需要携带 Setup Token（参见 4.7 节）。

```
# 获取系统信息（预填表单默认值）
GET /setup/api/info
→ {
    hostname: "worker-node-03",
    ips: ["192.168.1.100", "10.0.0.5"],
    default_workspace_dir: "/var/lib/agents-admin/workspaces",
    generated_node_id: "worker-node-03-a7f3"
  }

# 验证配置（不写入，仅测试连接）
POST /setup/api/validate
← {
    node: { id: "...", api_server_url: "...", workspace_dir: "..." },
    redis: { url: "..." },
    tls: { ca_source: "minio|file|skip", ca_file: "...", minio: {...} }
  }
→ {
    valid: true/false,
    checks: {
      api_server: { ok: true, message: "Connected (200 OK)" },
      redis: { ok: false, message: "Connection refused" }
    }
  }

# 保存配置并触发重启
POST /setup/api/apply
← { ... 同 validate 请求体 ... }
→ { success: true, config_path: "/etc/agents-admin/nodemanager.yaml" }
# 响应后：
#   1. 写入配置文件
#   2. 关闭 HTTP server
#   3. deb 场景：exit(0)，systemd 自动重启进入 Worker Mode
#   4. 二进制场景：提示用户手动重启
```

### 4.5 首次运行检测逻辑

```
func main():
    configPath = resolveConfigPath()  // --config 参数 或 默认搜索路径
    
    if --reconfigure:
        // 强制重新配置，忽略现有配置文件
        startSetupServer(configPath)
        return

    if configFileExists(configPath):
        // Worker Mode: 现有逻辑不变
        cfg = loadConfig(configPath)
        startNodeManager(cfg)
    else:
        // Setup Mode: 启动配置向导
        token = generateRandomToken()
        log("====================================================")
        log("首次运行，请访问以下地址完成配置：")
        log("  http://<IP>:15700/setup?token=" + token)
        log("====================================================")
        startSetupServer(configPath, token)
```

### 4.6 配置文件路径策略

| 场景 | 配置文件路径 | 说明 |
|------|------------|------|
| 直接运行二进制 | `./nodemanager.yaml` | 当前工作目录 |
| `--config /path/to/dir` | `/path/to/dir/nodemanager.yaml` | 用户指定 |
| deb 包安装 | `/etc/agents-admin/nodemanager.yaml` | systemd 传入 `--config /etc/agents-admin` |

deb 包安装后：
1. systemd 启动 node-manager，`--config /etc/agents-admin`
2. 检测到 `/etc/agents-admin/nodemanager.yaml` 不存在（deb 不再预装配置文件模板）
3. 启动 Setup 向导，日志输出带 Token 的完整 URL
4. 管理员通过 `journalctl -u agents-admin-node-manager -n 20` 看到 URL + Token
5. 浏览器远程访问完成配置
6. 配置写入 → 进程 exit(0) → systemd 自动重启 → 进入 Worker Mode

### 4.7 安全设计

#### 4.7.1 Setup Token 认证（Jupyter 模型）

由于 Node Manager 的部署场景**以远程为主**，Setup 模式必须监听 `0.0.0.0`。
采用 **Jupyter Notebook 的 Token 安全模型**，这是远程首次配置场景的业界标准做法：

```
程序启动
  │
  ├─ 生成 32 字节 crypto/rand 随机 token（hex 编码，64 字符）
  │
  ├─ 打印到 stdout / systemd journal:
  │   ┌──────────────────────────────────────────────────────────┐
  │   │ Setup wizard: http://192.168.1.100:15700/setup?token=a3f│
  │   │ Token: a3f7b2c9d4e5...（完整 64 字符）                    │
  │   └──────────────────────────────────────────────────────────┘
  │
  ├─ HTTP 中间件：
  │   每个请求检查 token 参数（URL query 或 Cookie）
  │     ├─ token 正确 → 设置 HttpOnly Cookie（有效期 1 小时）→ 放行
  │     └─ token 错误/缺失 → 返回 403 Forbidden
  │
  └─ 配置完成后：
      Token 失效，HTTP server 关闭，不再暴露任何端口
```

**安全保障层次：**

| 层次 | 机制 | 防护目标 |
|------|------|---------|
| **L1: Token** | 64 字符随机 hex，仅打印在终端/journal | 阻止未授权访问 |
| **L2: Cookie** | 首次 token 验证后设置 HttpOnly Cookie | 避免 token 在 URL 中反复暴露 |
| **L3: 超时** | Setup 模式启动后 30 分钟无操作自动退出 | 防止长时间暴露 |
| **L4: 一次性** | 配置完成后 HTTP server 关闭 | 零持久攻击面 |

**为什么不用 HTTPS？** Setup 向导本身不传输敏感业务数据（仅配置参数），且自签证书在远程首次访问场景下反而增加复杂度（浏览器证书警告）。Token 机制已提供足够的认证保护。

#### 4.7.2 配置完成后
- HTTP server 关闭，不暴露任何端口
- 配置文件权限设为 `0640`（owner + group 可读）

#### 4.7.3 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--config <dir>` | `configs/` | 配置文件目录 |
| `--setup-port <port>` | `15700` | Setup 向导监听端口 |
| `--setup-listen <addr>` | `0.0.0.0` | Setup 向导监听地址 |
| `--reconfigure` | false | 强制重新进入配置向导 |

deb 包场景下修改端口的方式：
- 编辑 `/etc/agents-admin/node-manager.env`，添加 `SETUP_PORT=15800`
- 或通过 systemd override: `systemctl edit agents-admin-node-manager`，在 `[Service]` 中添加 `ExecStart=` 行覆盖参数

### 4.8 HTTPS 客户端配置

Node Manager 是**TLS 客户端**（连接 API Server），不是 TLS 服务端。
当用户在 Step 1 填写的 API Server URL 以 `https://` 开头时，Step 3 自动展开 HTTPS 配置：

**需要配置的内容：如何信任 API Server 的 CA 证书**

三种 CA 证书获取方式：

| 方式 | 适用场景 | 向导中的配置项 |
|------|---------|--------------|
| **A. 从 MinIO 下载** | 集中管理 CA 证书的生产环境 | MinIO Endpoint, Bucket, Object Key, Access Key, Secret Key |
| **B. 本地文件路径** | CA 证书已预先分发到节点 | CA 文件路径（如 `/etc/agents-admin/certs/ca.pem`） |
| **C. 跳过验证** | 开发/测试环境 | 无需配置（`InsecureSkipVerify: true`） |

**MinIO 方案的工作流：**
```
配置向导 validate 阶段
  │
  ├─ 用户选择 "从 MinIO 下载 CA"
  ├─ 填写 MinIO endpoint + credentials + bucket/key
  │
  ├─ 后端调用 MinIO API 下载 ca.pem
  ├─ 保存到本地（如 /etc/agents-admin/certs/ca.pem）
  ├─ 用下载的 CA 测试连接 API Server
  │
  └─ 写入配置文件：
      tls:
        enabled: true
        ca_file: /etc/agents-admin/certs/ca.pem
```

配置文件中**不保存** MinIO credentials，仅保存下载后的 CA 文件路径。

### 4.9 与现有架构的集成

#### 不变的部分
- `internal/nodemanager/manager.go` — 核心逻辑不变
- `internal/nodemanager/adapter/` — 适配器不变（硬编码注册 qwencode/gemini/claude，与配置无关）
- 现有 `nodeManagerYAML` 配置结构基本不变（移除 etcd 相关字段）
- Makefile `build` 和 `release-linux` 目标不变

#### 变更的部分

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `cmd/nodemanager/main.go` | 修改 | 增加首次运行检测分支 + 命令行参数 |
| `cmd/nodemanager/setup.go` | 新增 | Setup HTTP server + Token 中间件 + API handlers |
| `web/nodemanager-setup/` | 新增 | 配置向导前端（HTML/JS/CSS） |
| `web/nodemanager-setup/embed.go` | 新增 | `go:embed` 声明 |
| `internal/nodemanager/setup/` | 新增 | 验证、写入逻辑、MinIO CA 下载 |
| `deployments/deb/build-deb.sh` | 修改 | 不再预装 nodemanager.yaml 模板 |
| `deployments/deb/config/nodemanager.yaml` | 删除 | 改为由向导生成 |
| systemd service | 修改 | 添加 `Restart=always`（Setup 完成后需自动重启） |
| `cmd/nodemanager/main.go` (etcd 部分) | 清理 | 移除 etcd EventBus 相关代码（未实际使用） |

## 5. 关键设计决策

### 5.1 配置完成后的重启策略

| 部署方式 | 重启机制 | 用户体验 |
|---------|---------|---------|
| **deb 包 (systemd)** | 进程 `exit(0)` → systemd `Restart=always` 自动重启 | 无需人工干预 |
| **直接运行二进制** | 前端提示"配置已保存，请手动重启程序" | 用户 Ctrl+C 后重新运行 |

选择**不使用 `syscall.Exec` 自动重启**的理由：
- `syscall.Exec` 替换当前进程，如果配置有误会进入"启动→失败→无法重新配置"的死循环
- 手动重启让用户有机会检查日志、确认配置是否正确
- systemd 场景下 `Restart=always` 已覆盖自动重启需求

### 5.2 前端国际化

配置向导前端独立于主项目前端（Next.js），但仍应遵循多语言策略：
- 由于页面极简（1个页面、~20个文本），直接在 JS 中内置中英文 JSON 即可
- 根据 `navigator.language` 自动切换

### 5.3 `--reconfigure` 重新配置

用户修改配置的两种路径：
1. **手动编辑**：直接编辑 `nodemanager.yaml`，重启服务
2. **向导重配**：`agents-admin-node-manager --reconfigure` → 重新启动向导（生成新 Token）

deb 包场景下使用 `--reconfigure`：
```bash
# 停止服务
sudo systemctl stop agents-admin-node-manager

# 以重配置模式手动运行
sudo agents-admin-node-manager --config /etc/agents-admin --reconfigure

# 配置完成后 Ctrl+C，重新启动服务
sudo systemctl start agents-admin-node-manager
```

### 5.4 etcd 清理

代码审查确认：`nm.eventBus`（etcd EventBus）虽然在 `manager.go` 中声明和赋值，但从未被实际调用任何方法。Node Manager 唯一使用的中间件是 **Redis**（用于 NodeRunQueue 任务分发）。

清理范围：
- `cmd/nodemanager/main.go` — 移除 etcd 初始化代码
- `nodeManagerYAML` — 移除 `Etcd` 配置段
- `configs/nodemanager.yaml` — 移除 etcd 配置
- `manager.go` — 保留 `SetEventBus` 接口（未来可能用其他实现替代）

## 6. 开发计划（TODO）

- [ ] **T1: 基础骨架** — 首次运行检测 + Setup HTTP server + Token 认证中间件
- [ ] **T2: 配置向导前端** — 分步表单 UI + API 调用 + 国际化
- [ ] **T3: 后端 API** — `/setup/api/info` + `/setup/api/validate` + `/setup/api/apply`
- [ ] **T4: 连接测试** — API Server / Redis 连通性验证
- [ ] **T5: HTTPS 客户端配置** — MinIO CA 下载 + 本地文件 + 跳过验证
- [ ] **T6: 配置写入 + 重启** — YAML 生成 + 双模式重启策略
- [ ] **T7: deb 包适配** — 移除预装配置模板，调整 systemd service
- [ ] **T8: etcd 清理** — 移除未使用的 etcd 相关代码
- [ ] **T9: 测试** — 单元测试 + 集成测试 + 手动端到端验证

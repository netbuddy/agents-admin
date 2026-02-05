# NodeManager（节点管理器）

Node Agent 数据平面组件，负责在节点上执行 Agent 任务。

## 目录结构

```
nodemanager/
│
├── README.md                 # 本文件
│
│  ══════ 核心入口 ══════
├── manager.go                # NodeManager 主体
├── manager_test.go
│
│  ══════ Agent CLI 适配器 ══════
├── adapter/                  # Agent CLI 适配层
│   ├── adapter.go            # Adapter 接口定义
│   ├── types.go              # TaskSpec, AgentConfig 等类型
│   ├── event.go              # CanonicalEvent 统一事件
│   ├── claude/               # Claude Code 适配器
│   ├── gemini/               # Gemini CLI 适配器
│   └── qwencode/             # Qwencode 适配器
│
│  ══════ Agent 认证 ══════
├── auth/                     # Agent 认证模块
│   ├── authenticator.go      # Authenticator 接口
│   ├── container.go          # 认证流程容器操作
│   └── qwencode/             # Qwencode OAuth 认证
│
│  ══════ 运行时环境 ══════
├── runtime/                  # Agent 运行时环境
│   ├── runtime.go            # Runtime 接口定义
│   └── docker/               # Docker 容器运行时
│
│  ══════ 功能模块 ══════
├── auth_controller.go        # 认证控制器
├── container_instance.go     # Instance 容器管理
├── container_terminal.go     # Terminal 会话管理
├── workspace_manager.go      # 工作空间准备
├── heartbeat_service.go      # 心跳服务
├── metrics_prometheus.go     # Prometheus 指标
│
│  ══════ Handler 框架 ══════
└── handler/                  # Handler 插件框架
    ├── interface.go          # Handler 接口
    ├── registry.go           # Handler 注册表
    └── ...
```

## 架构关系

```
┌─────────────────────────────────────────────────────────────┐
│                      NodeManager                             │
│                                                              │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐   │
│  │   adapter/    │  │    auth/      │  │   runtime/    │   │
│  │               │  │               │  │               │   │
│  │ - Claude      │  │ - Qwencode    │  │ - Docker      │   │
│  │ - Gemini      │  │   OAuth       │  │ - VM (未来)   │   │
│  │ - Qwencode    │  │               │  │ - Bare (未来) │   │
│  │               │  │               │  │               │   │
│  │ "如何调用CLI" │  │  "如何认证"   │  │ "在哪里运行"  │   │
│  └───────────────┘  └───────────────┘  └───────────────┘   │
│                              │                               │
│                              ▼                               │
│                    ┌───────────────┐                        │
│                    │  manager.go   │                        │
│                    │   (协调器)    │                        │
│                    └───────────────┘                        │
└─────────────────────────────────────────────────────────────┘
```

## 模块职责

### adapter/ - Agent CLI 适配器

将平台统一的 TaskSpec 转换为具体 CLI 的启动命令，并将 CLI 输出解析为统一的事件格式。

```go
type Adapter interface {
    Name() string
    Validate(agent *AgentConfig) error
    BuildCommand(ctx, spec, agent) (*RunConfig, error)
    ParseEvent(line string) (*CanonicalEvent, error)
    CollectArtifacts(ctx, workspaceDir) (*Artifacts, error)
}
```

### auth/ - Agent 认证

处理 Agent 的 OAuth/DeviceCode/APIKey 等认证流程。

- `authenticator.go` - Authenticator 接口
- `container.go` - 认证流程的容器操作（临时容器）
- `qwencode/` - Qwencode OAuth 实现

### runtime/ - 运行时环境

Agent 任务执行的环境抽象，支持多种运行时：

- `docker/` - Docker 容器运行时（当前实现）
- `vm/` - 虚拟机运行时（未来）
- `bare/` - 物理机运行时（未来）

```go
type Runtime interface {
    Name() string
    Create(ctx, config) (*Instance, error)
    Start(ctx, instanceID) error
    Stop(ctx, instanceID) error
    Exec(ctx, instanceID, cmd) (*ExecResult, error)
    // ...
}
```

## 与 auth/container.go 和 runtime/docker 的区别

| 模块 | 用途 | 容器特点 |
|------|------|----------|
| `auth/container.go` | 认证流程 | 临时容器，完成 OAuth 后销毁 |
| `runtime/docker/` | 任务执行 | 长期运行，执行 Agent 任务 |

## 阅读顺序

1. `manager.go` - 理解整体协调逻辑
2. `adapter/adapter.go` - 理解 CLI 适配机制
3. `runtime/runtime.go` - 理解运行时抽象
4. `auth/authenticator.go` - 理解认证机制

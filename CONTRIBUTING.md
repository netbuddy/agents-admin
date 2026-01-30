# 贡献指南

感谢你对 Agent Kanban 项目的兴趣！

## 开发环境

### 前置要求

- Go 1.22+
- Docker & Docker Compose
- Make

### 本地开发

```bash
# 克隆仓库
git clone <repo-url>
cd agents-admin

# 启动基础设施
make dev-up

# 运行 API Server（开发模式）
make run-api

# 运行 Node Agent（开发模式）
make run-agent

# 运行测试
make test

# 代码检查
make lint
```

## 分支策略

- `main` - 生产就绪代码
- `develop` - 开发分支
- `feature/*` - 功能分支
- `fix/*` - 修复分支

## 提交规范

使用 [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

类型：
- `feat`: 新功能
- `fix`: 修复
- `docs`: 文档
- `refactor`: 重构
- `test`: 测试
- `chore`: 杂项

示例：
```
feat(driver): add Claude Code driver implementation

- Implement BuildCommand for headless mode
- Add event parsing for stream-json output
- Add unit tests

Closes #123
```

## 代码风格

- 使用 `gofmt` 格式化代码
- 使用 `golangci-lint` 检查代码
- 遵循 [Effective Go](https://go.dev/doc/effective_go)

## 测试

- 所有新功能需要单元测试
- 集成测试放在 `*_integration_test.go`
- 测试覆盖率目标 80%+

```bash
# 运行所有测试
make test

# 运行带覆盖率的测试
make test-coverage

# 运行特定包的测试
go test -v ./pkg/driver/...
```

## PR 流程

1. Fork 仓库
2. 创建功能分支
3. 编写代码和测试
4. 确保 CI 通过
5. 提交 PR 到 `develop`
6. 等待 Code Review

## 目录结构

```
agents-admin/
├── api/
│   └── schemas/          # JSON Schema 定义
├── cmd/
│   ├── api-server/       # API Server 入口
│   └── node-agent/       # Node Agent 入口
├── deployments/          # Docker、K8s 配置
├── docs/                 # 文档
├── internal/             # 内部包（不导出）
│   ├── api/              # HTTP handlers
│   ├── scheduler/        # 调度器
│   └── storage/          # 存储层
├── pkg/                  # 公共包（可导出）
│   └── driver/           # Driver 接口与实现
└── web/                  # 前端代码
```

## 联系方式

如有问题，请通过 Issue 讨论。

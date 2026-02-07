# 账号与实例管理

> 本文介绍如何管理 Agent 账号和运行实例。

## 核心概念

```
Agent 类型 ──1:N──► 账号 ──1:N──► 实例
(qwen-code)        (已认证凭据)    (运行容器)
```

- **Agent 类型**：系统预置，不可修改（qwen-code、gemini-cli、claude-code、openai-codex）
- **账号**：一个 Agent 类型可以有多个账号（如多个 Google 账号）
- **实例**：一个账号可以创建多个实例（如同时执行多个任务）

## Agent 类型

系统内置以下 Agent 类型：

| 类型 ID | 名称 | 认证方式 | 说明 |
|---------|------|----------|------|
| `qwen-code` | Qwen Code | OAuth | 阿里云 Qwen 代码助手 |
| `gemini-cli` | Gemini CLI | OAuth | Google Gemini CLI |
| `claude-code` | Claude Code | OAuth | Anthropic Claude Code |
| `openai-codex` | OpenAI Codex | Device Code / API Key | OpenAI Codex CLI |

### 查看 Agent 类型

1. 访问 **「账号管理」** 页面
2. 在创建账号对话框中可以看到所有可用的 Agent 类型

## 账号管理

### 查看账号列表

1. 点击左侧导航栏的 **「账号管理」**
2. 页面显示所有已创建的账号
3. 每个账号卡片显示：名称、Agent 类型、状态、所属节点

### 账号状态

| 状态 | 说明 |
|------|------|
| `pending` | 待认证 |
| `authenticating` | 认证中（等待用户完成 OAuth 流程） |
| `authenticated` | 已认证，可创建实例 |
| `expired` | 凭据已过期，需重新认证 |

### 创建账号

1. 在账号管理页面，点击 **「添加账号」**
2. 填写以下信息：

| 字段 | 必填 | 说明 |
|------|------|------|
| **Agent 类型** | 是 | 选择 Agent 产品类型 |
| **认证方式** | 是 | OAuth / Device Code / API Key |
| **账号名称** | 是 | 显示名称（如邮箱） |
| **节点** | 是 | 选择账号绑定的执行节点 |
| **API Key** | 仅 API Key 模式 | Agent 的 API 密钥 |

3. 点击 **「创建」**
4. 对于 OAuth 认证：
   - 系统会在目标节点启动认证容器
   - 页面会显示 ttyd 终端窗口
   - 按终端提示完成认证（如浏览器登录）
   - 认证成功后账号状态变为 `authenticated`

### 删除账号

1. 在账号卡片上点击 **删除按钮**
2. 确认删除
3. **注意**：删除账号前应先删除关联的实例

## 实例管理

### 查看实例列表

1. 点击左侧导航栏的 **「实例管理」**
2. 页面显示所有实例及其状态
3. 可按 Agent 类型筛选

### 实例状态

| 状态 | 说明 |
|------|------|
| `pending` | 已创建，等待启动 |
| `starting` | 正在启动容器 |
| `running` | 运行中，可接受任务 |
| `stopping` | 正在停止 |
| `stopped` | 已停止 |
| `error` | 启动或运行出错 |

### 创建实例

1. 在实例管理页面，点击 **「创建实例」**
2. 填写信息：

| 字段 | 必填 | 说明 |
|------|------|------|
| **实例名称** | 否 | 可选，留空自动生成 |
| **账号** | 是 | 选择已认证（authenticated）的账号 |
| **节点** | 否 | 默认使用账号所属节点 |

3. 点击 **「创建」**

### 启动实例

1. 在实例卡片上点击 **「启动」** 按钮
2. 系统通知 NodeManager 启动 Docker 容器
3. 等待状态变为 **「运行中」**（通常几秒钟）

### 停止实例

1. 在实例卡片上点击 **「停止」** 按钮
2. 系统通知 NodeManager 停止容器
3. 实例状态变为 **「已停止」**

### 删除实例

1. 确保实例已停止
2. 点击 **删除按钮**
3. 确认删除

## 典型工作流

```
1. 查看 Agent 类型     →  确认支持的 Agent
2. 创建账号           →  OAuth / API Key 认证
3. 等待认证完成       →  账号状态变为 authenticated
4. 创建实例           →  选择已认证账号
5. 启动实例           →  状态变为 running
6. 创建任务时选择实例  →  任务绑定到具体实例执行
```

## API 参考

| 操作 | 方法 | 路径 |
|------|------|------|
| 列出 Agent 类型 | GET | `/api/v1/agent-types` |
| 获取 Agent 类型 | GET | `/api/v1/agent-types/{id}` |
| 列出账号 | GET | `/api/v1/accounts` |
| 获取账号 | GET | `/api/v1/accounts/{id}` |
| 删除账号 | DELETE | `/api/v1/accounts/{id}` |
| 创建认证操作 | POST | `/api/v1/operations` |
| 列出实例 | GET | `/api/v1/instances` |
| 创建实例 | POST | `/api/v1/instances` |
| 获取实例 | GET | `/api/v1/instances/{id}` |
| 启动实例 | POST | `/api/v1/instances/{id}/start` |
| 停止实例 | POST | `/api/v1/instances/{id}/stop` |
| 删除实例 | DELETE | `/api/v1/instances/{id}` |

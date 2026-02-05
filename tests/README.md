# 测试目录结构

本项目采用测试金字塔模型，包含以下测试类型：

```
                    ╱╲
                   ╱  ╲         tests/e2e/
                  ╱────╲        E2E Tests (浏览器自动化)
                 ╱      ╲
                ╱────────╲      tests/integration/
               ╱          ╲     Integration Tests (真实 HTTP + 真实 DB)
              ╱────────────╲
             ╱              ╲   tests/handler/
            ╱────────────────╲  Handler Unit Tests (内存调用 + 可 Mock)
           ╱                  ╲
          ╱────────────────────╲  internal/*_test.go
         ╱                      ╲ Pure Unit Tests (函数级别)
```

## 测试类型详解

### 1. Pure Unit Tests（纯单元测试）

**位置**: `internal/**/*_test.go`（与源码同目录）

**特点**:
- 测试单个函数或方法
- 完全隔离，不依赖外部系统
- 使用 Mock 替代所有外部依赖
- 速度最快（微秒级）

**示例**:
```go
func TestGenerateID(t *testing.T) {
    id := generateID("task")
    if !strings.HasPrefix(id, "task-") {
        t.Errorf("expected prefix 'task-', got %s", id)
    }
}
```

**运行**: `go test ./internal/...`

---

### 2. Handler Unit Tests（处理器单元测试）

**位置**: `tests/handler/`

**特点**:
- 使用 `router.ServeHTTP(w, req)` 直接调用 Handler
- 跳过网络层，在内存中完成
- 可以使用 Mock Database 隔离外部依赖
- 速度极快（毫秒级）

**适用场景**:
- 测试 Handler 的业务逻辑
- 测试参数校验和错误处理
- 测试边界条件
- 需要快速反馈的 TDD 开发

**架构图**:
```
┌──────────────┐        ┌─────────────┐
│  Test Code   │──────→ │   Handler   │ ──→ Mock DB (可选)
│ ServeHTTP()  │        │  Create()   │
└──────────────┘        └─────────────┘
      ↑ 内存中直接调用，无网络开销
```

**示例**:
```go
req := httptest.NewRequest("POST", "/api/v1/tasks", body)
w := httptest.NewRecorder()
router.ServeHTTP(w, req)  // 直接调用，跳过网络层
```

**运行**: `go test ./tests/handler/...`

---

### 3. Integration Tests（集成测试）

**位置**: `tests/integration/`

**特点**:
- 使用 `httptest.NewServer` 启动真实 HTTP 服务器
- 请求经过完整的 TCP/IP 网络栈
- 使用真实的 PostgreSQL 数据库
- 测试组件间的真实交互

**适用场景**:
- 验证 API 端到端流程
- 测试中间件（CORS、认证、指标）
- 验证数据库读写正确性
- 测试多组件协作

**架构图**:
```
┌──────────────┐  HTTP  ┌─────────────┐
│  Test Code   │──────→ │ HTTP Server │ ──→ Real PostgreSQL
│ http.Post()  │ TCP/IP │  :random    │
└──────────────┘        └─────────────┘
      ↑ 真实网络请求，完整流程
```

**示例**:
```go
server := httptest.NewServer(handler.Router())
defer server.Close()

resp, err := http.Post(server.URL+"/api/v1/tasks", "application/json", body)
```

**运行**: `go test ./tests/integration/...`

---

### 4. E2E Tests（端到端测试）

**位置**: `tests/e2e/`

**特点**:
- 启动完整的系统（API Server + Web UI）
- 使用浏览器自动化工具（如 Playwright）
- 模拟真实用户操作
- 速度最慢，但最接近真实场景

**适用场景**:
- 验证关键用户流程
- 测试前后端集成
- 验收测试

**运行**: `./tests/e2e/browser/run_all_tests.sh`

---

### 5. Regression Tests（回归测试）

**位置**: `tests/regression/`

**特点**:
- 针对已修复的 Bug 编写的测试
- 防止 Bug 重新出现
- 通常与 Integration Tests 类似，但更聚焦特定场景

**运行**: `go test ./tests/regression/...`

---

## 目录命名规则

测试目录和文件名与 `docs/business-flow/` 中的文档保持一致：

| 文档路径 | Handler 测试路径 | 集成测试路径 |
|----------|------------------|--------------|
| `docs/business-flow/10-任务管理/01-任务创建.md` | `tests/handler/10-task-management/01-task-create_test.go` | `tests/integration/10-task-management/01-task-create_test.go` |

规则：
- 保留数字前缀（如 `10-`）
- 中文翻译为英文（如 `任务管理` → `task-management`）
- 测试文件以 `_test.go` 结尾

---

## 何时使用哪种测试？

| 场景 | 推荐测试类型 |
|------|--------------|
| 开发新功能，快速验证逻辑 | Handler Unit Test |
| 验证参数校验和错误处理 | Handler Unit Test |
| 验证 API 完整流程 | Integration Test |
| 验证数据库操作正确性 | Integration Test |
| 验证前后端集成 | E2E Test |
| 验证关键用户流程 | E2E Test |
| 修复 Bug 后防止回归 | Regression Test |

---

## 运行所有测试

```bash
# 运行所有单元测试
go test ./internal/...

# 运行 Handler 单元测试
go test ./tests/handler/...

# 运行集成测试（需要数据库）
go test ./tests/integration/...

# 运行回归测试
go test ./tests/regression/...

# 运行所有测试
make test-all
```

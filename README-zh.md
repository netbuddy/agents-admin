<a name="readme-top"></a>
[![Contributors][contributors-shield]][contributors-url]
[![Forks][forks-shield]][forks-url]
[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]
[![MIT License][license-shield]][license-url]

<!-- PROJECT LOGO -->
<br />
<div align="center">
  <a href="https://github.com/netbuddy/agents-admin">
    <img src="docs/images/logo.png" alt="Logo" width="80" height="80">
  </a>

<h3 align="center">Agent Kanban</h3>

  <p align="center">
    AI Agent 任务编排与可观测平台
    <br />
    <a href="https://github.com/netbuddy/agents-admin"><strong>探索项目文档 »</strong></a>
    <br />
    <br />
    <a href="https://github.com/netbuddy/agents-admin">查看 Demo</a>
    ·
    <a href="https://github.com/netbuddy/agents-admin/issues/new?labels=bug&template=bug-report---.md">报告 Bug</a>
    ·
    <a href="https://github.com/netbuddy/agents-admin/issues/new?labels=enhancement&template=feature-request---.md">请求功能</a>
  </p>
</div>

<!-- TABLE OF CONTENTS -->
<details>
  <summary>目录</summary>
  <ol>
    <li>
      <a href="#关于项目">关于项目</a>
      <ul>
        <li><a href="#构建技术">构建技术</a></li>
      </ul>
    </li>
    <li>
      <a href="#快速开始">快速开始</a>
      <ul>
        <li><a href="#前置条件">前置条件</a></li>
        <li><a href="#安装">安装</a></li>
      </ul>
    </li>
    <li><a href="#使用说明">使用说明</a></li>
    <li><a href="#路线图">路线图</a></li>
    <li><a href="#贡献指南">贡献指南</a></li>
    <li><a href="#许可证">许可证</a></li>
    <li><a href="#联系方式">联系方式</a></li>
    <li><a href="#致谢">致谢</a></li>
  </ol>
</details>

<!-- ABOUT THE PROJECT -->
## 关于项目

[![产品截图][product-screenshot]](https://example.com)

Agent Kanban 是一个分布式容器化 AI Agent 编排与看板监控系统，支持多种 AI Agent CLI（Claude Code、Gemini CLI、Codex）的统一管理和实时监控。

核心优势：
* 标准化 TaskSpec 管理不同 Agent 的任务，实现统一编排
* Event-First 架构，通过 WebSocket 实时推送事件流，实现实时监控
* 三层防御（容器隔离、CLI 权限、平台熔断）保障安全
* Driver 抽象层设计，轻松接入新的 Agent CLI，具备高度可扩展性

<p align="right">(<a href="#readme-top">返回顶部</a>)</p>

### 构建技术

本项目主要基于以下技术栈构建：

* [![Go][Go]][Go-url]
* [![Next][Next.js]][Next-url]
* [![React][React.js]][React-url]
* [![TypeScript][TypeScript]][TypeScript-url]
* [![PostgreSQL][PostgreSQL]][PostgreSQL-url]
* [![Redis][Redis]][Redis-url]
* [![Docker][Docker]][Docker-url]
* [![TailwindCSS][TailwindCSS]][Tailwind-url]

<p align="right">(<a href="#readme-top">返回顶部</a>)</p>

<!-- GETTING STARTED -->
## 快速开始

以下是设置本地开发环境的步骤。

### 前置条件

开始之前，请确保已安装以下软件：

* Go 1.24+
  ```sh
  # 使用官方安装包或包管理器安装
  # https://go.dev/doc/install
  ```

* Node.js 20+ 和 npm
  ```sh
  # 推荐使用 nvm 安装
  nvm install 20
  nvm use 20
  ```

* Docker & Docker Compose
  ```sh
  # https://docs.docker.com/get-docker/
  ```

* Make
  ```sh
  # Ubuntu/Debian
  sudo apt-get install make

  # macOS
  xcode-select --install
  ```

### 安装

1. 克隆仓库
   ```sh
   git clone https://github.com/netbuddy/agents-admin.git
   cd agents-admin
   ```

2. 复制环境变量文件
   ```sh
   cp .env.example .env
   # 根据你的环境编辑 .env 文件
   ```

3. 启动基础设施服务（PostgreSQL、Redis、MinIO）
   ```sh
   make dev-up
   ```

4. 安装前端依赖
   ```sh
   cd web && npm install
   ```

5. 生成 OpenAPI 代码
   ```sh
   make generate-api
   ```

6. 构建项目
   ```sh
   make build
   ```

<p align="right">(<a href="#readme-top">返回顶部</a>)</p>

<!-- USAGE EXAMPLES -->
## 使用说明

### 启动开发服务

1. 启动 API Server
   ```sh
   make run-api
   # 或使用热加载模式
   make watch-api
   ```

2. 启动 NodeManager
   ```sh
   make run-nodemanager
   # 或使用热加载模式
   make watch-nodemanager
   ```

3. 启动 Web UI
   ```sh
   make run-web
   ```

### 服务访问地址

| 服务 | 地址 | 说明 |
|------|------|------|
| API Server | http://localhost:8080 | REST API |
| Web UI | http://localhost:3002 | 前端界面 |
| MinIO Console | http://localhost:9001 | 对象存储管理 |
| PostgreSQL | localhost:5432 | 数据库 |
| Redis | localhost:6380 | 缓存/消息队列 |

### 运行测试

```sh
# 单元测试
make test

# 集成测试（需要数据库）
make test-integration

# E2E 测试（需要运行中的服务）
make test-e2e

# 测试覆盖率
make test-coverage
```

更多使用示例，请参考 [文档](./docs/)

<p align="right">(<a href="#readme-top">返回顶部</a>)</p>

<!-- ROADMAP -->
## 路线图

- [x] 基础 API Server 架构
- [x] NodeManager 节点管理
- [x] WebSocket 实时事件推送
- [x] PostgreSQL + Redis 存储层
- [x] Docker 容器化执行环境
- [x] OpenAPI 规范与代码生成
- [x] Web UI 看板界面
- [ ] 多租户支持
- [ ] 更丰富的监控指标
- [ ] 任务调度优化算法
- [ ] 更多 Agent CLI 驱动（Gemini、Codex 等）
- [ ] 工作流编排功能

查看 [open issues](https://github.com/netbuddy/agents-admin/issues) 获取完整的功能请求列表和已知问题。

<p align="right">(<a href="#readme-top">返回顶部</a>)</p>

<!-- CONTRIBUTING -->
## 贡献指南

贡献让开源社区成为学习、激励和创造的绝佳场所。非常感谢您的任何贡献！

如果您有能让项目变得更好的建议，请 Fork 本仓库并创建一个 Pull Request。您也可以简单地打开一个带有 "enhancement" 标签的 Issue。别忘了给项目点个 Star！再次感谢！

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

更多详情，请查看 [CONTRIBUTING.md](./CONTRIBUTING.md)

<p align="right">(<a href="#readme-top">返回顶部</a>)</p>

<!-- LICENSE -->
## 许可证

根据 MIT 许可证分发。查看 `LICENSE` 文件了解更多信息。

<p align="right">(<a href="#readme-top">返回顶部</a>)</p>

<!-- CONTACT -->
## 联系方式

NetBuddy Team - [@netbuddy](https://github.com/netbuddy)

项目链接: [https://github.com/netbuddy/agents-admin](https://github.com/netbuddy/agents-admin)

<p align="right">(<a href="#readme-top">返回顶部</a>)</p>

<!-- ACKNOWLEDGMENTS -->
## 致谢

感谢以下开源项目和资源：

* [Choose an Open Source License](https://choosealicense.com)
* [Img Shields](https://shields.io)
* [GitHub Pages](https://pages.github.com)
* [Font Awesome](https://fontawesome.com)
* [React Icons](https://react-icons.github.io/react-icons/search)
* [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen)
* [Air](https://github.com/cosmtrek/air) - Go 热加载工具
* [Lucide React](https://lucide.dev) - 图标库

<p align="right">(<a href="#readme-top">返回顶部</a>)</p>

<!-- MARKDOWN LINKS & IMAGES -->
<!-- https://www.markdownguide.org/basic-syntax/#reference-style-links -->
[contributors-shield]: https://img.shields.io/github/contributors/netbuddy/agents-admin.svg?style=for-the-badge
[contributors-url]: https://github.com/netbuddy/agents-admin/graphs/contributors
[forks-shield]: https://img.shields.io/github/forks/netbuddy/agents-admin.svg?style=for-the-badge
[forks-url]: https://github.com/netbuddy/agents-admin/network/members
[stars-shield]: https://img.shields.io/github/stars/netbuddy/agents-admin.svg?style=for-the-badge
[stars-url]: https://github.com/netbuddy/agents-admin/stargazers
[issues-shield]: https://img.shields.io/github/issues/netbuddy/agents-admin.svg?style=for-the-badge
[issues-url]: https://github.com/netbuddy/agents-admin/issues
[license-shield]: https://img.shields.io/github/license/netbuddy/agents-admin.svg?style=for-the-badge
[license-url]: https://github.com/netbuddy/agents-admin/blob/master/LICENSE.txt
[product-screenshot]: docs/images/screenshot.png
[Go]: https://img.shields.io/badge/go-00ADD8?style=for-the-badge&logo=go&logoColor=white
[Go-url]: https://go.dev/
[Next.js]: https://img.shields.io/badge/next.js-000000?style=for-the-badge&logo=nextdotjs&logoColor=white
[Next-url]: https://nextjs.org/
[React.js]: https://img.shields.io/badge/React-20232A?style=for-the-badge&logo=react&logoColor=61DAFB
[React-url]: https://reactjs.org/
[TypeScript]: https://img.shields.io/badge/TypeScript-007ACC?style=for-the-badge&logo=typescript&logoColor=white
[TypeScript-url]: https://www.typescriptlang.org/
[PostgreSQL]: https://img.shields.io/badge/PostgreSQL-316192?style=for-the-badge&logo=postgresql&logoColor=white
[PostgreSQL-url]: https://www.postgresql.org/
[Redis]: https://img.shields.io/badge/Redis-DC382D?style=for-the-badge&logo=redis&logoColor=white
[Redis-url]: https://redis.io/
[Docker]: https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white
[Docker-url]: https://www.docker.com/
[TailwindCSS]: https://img.shields.io/badge/Tailwind_CSS-38B2AC?style=for-the-badge&logo=tailwind-css&logoColor=white
[Tailwind-url]: https://tailwindcss.com/

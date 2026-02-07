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
    AI Agent Task Orchestration & Observability Platform
    <br />
    <a href="https://github.com/netbuddy/agents-admin"><strong>Explore the docs »</strong></a>
    <br />
    <br />
    <a href="https://github.com/netbuddy/agents-admin">View Demo</a>
    ·
    <a href="https://github.com/netbuddy/agents-admin/issues/new?labels=bug&template=bug-report---.md">Report Bug</a>
    ·
    <a href="https://github.com/netbuddy/agents-admin/issues/new?labels=enhancement&template=feature-request---.md">Request Feature</a>
  </p>
</div>

<!-- TABLE OF CONTENTS -->
<details>
  <summary>Table of Contents</summary>
  <ol>
    <li>
      <a href="#about-the-project">About The Project</a>
      <ul>
        <li><a href="#built-with">Built With</a></li>
      </ul>
    </li>
    <li>
      <a href="#getting-started">Getting Started</a>
      <ul>
        <li><a href="#prerequisites">Prerequisites</a></li>
        <li><a href="#installation">Installation</a></li>
      </ul>
    </li>
    <li><a href="#usage">Usage</a></li>
    <li><a href="#roadmap">Roadmap</a></li>
    <li><a href="#contributing">Contributing</a></li>
    <li><a href="#license">License</a></li>
    <li><a href="#contact">Contact</a></li>
    <li><a href="#acknowledgments">Acknowledgments</a></li>
  </ol>
</details>

<!-- ABOUT THE PROJECT -->
## About The Project

[![Product Screenshot][product-screenshot]](https://example.com)

Agent Kanban is a distributed containerized AI Agent orchestration and kanban monitoring system that supports unified management and real-time monitoring of multiple AI Agent CLIs (Claude Code, Gemini CLI, Codex).

Key Features:
* **Unified Orchestration**: Manage tasks from different Agents through standardized TaskSpec
* **Real-time Monitoring**: Event-First architecture with WebSocket real-time event streaming
* **Security Isolation**: Three-layer defense (container isolation, CLI permissions, platform circuit breaker)
* **Extensible**: Driver abstraction layer for easy integration of new Agent CLIs

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Built With

This project is primarily built with the following technologies:

* [![Go][Go]][Go-url]
* [![Next][Next.js]][Next-url]
* [![React][React.js]][React-url]
* [![TypeScript][TypeScript]][TypeScript-url]
* [![PostgreSQL][PostgreSQL]][PostgreSQL-url]
* [![Redis][Redis]][Redis-url]
* [![Docker][Docker]][Docker-url]
* [![TailwindCSS][TailwindCSS]][Tailwind-url]

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- GETTING STARTED -->
## Getting Started

Follow these steps to set up your local development environment.

### Prerequisites

Ensure you have the following software installed before starting:

* Go 1.24+
  ```sh
  # Use official installer or package manager
  # https://go.dev/doc/install
  ```

* Node.js 20+ and npm
  ```sh
  # Recommended to use nvm
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

### Installation

1. Clone the repository
   ```sh
   git clone https://github.com/netbuddy/agents-admin.git
   cd agents-admin
   ```

2. Copy environment variables file
   ```sh
   cp .env.example .env
   # Edit .env file according to your environment
   ```

3. Start infrastructure services (PostgreSQL, Redis, MinIO)
   ```sh
   make dev-up
   ```

4. Install frontend dependencies
   ```sh
   cd web && npm install
   ```

5. Generate OpenAPI code
   ```sh
   make generate-api
   ```

6. Build the project
   ```sh
   make build
   ```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- USAGE EXAMPLES -->
## Usage

### Start Development Services

1. Start API Server
   ```sh
   make run-api
   # Or use hot reload mode
   make watch-api
   ```

2. Start NodeManager
   ```sh
   make run-nodemanager
   # Or use hot reload mode
   make watch-nodemanager
   ```

3. Start Web UI
   ```sh
   make run-web
   ```

### Service Endpoints

| Service | URL | Description |
|---------|-----|-------------|
| API Server | http://localhost:8080 | REST API |
| Web UI | http://localhost:3002 | Frontend Interface |
| MinIO Console | http://localhost:9001 | Object Storage Management |
| PostgreSQL | localhost:5432 | Database |
| Redis | localhost:6380 | Cache/Message Queue |

### Running Tests

```sh
# Unit tests
make test

# Integration tests (requires database)
make test-integration

# E2E tests (requires running services)
make test-e2e

# Test coverage
make test-coverage
```

For more examples, please refer to the [Documentation](./docs/)

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- ROADMAP -->
## Roadmap

- [x] Basic API Server Architecture
- [x] NodeManager Implementation
- [x] WebSocket Real-time Event Streaming
- [x] PostgreSQL + Redis Storage Layer
- [x] Docker Containerized Execution Environment
- [x] OpenAPI Specification & Code Generation
- [x] Web UI Kanban Interface
- [ ] Multi-tenant Support
- [ ] Enhanced Monitoring Metrics
- [ ] Task Scheduling Optimization Algorithm
- [ ] Additional Agent CLI Drivers (Gemini, Codex, etc.)
- [ ] Workflow Orchestration Features

See the [open issues](https://github.com/netbuddy/agents-admin/issues) for a full list of proposed features and known issues.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- CONTRIBUTING -->
## Contributing

Contributions are what make the open source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

If you have a suggestion that would make this better, please fork the repo and create a pull request. You can also simply open an issue with the tag "enhancement". Don't forget to give the project a star! Thanks again!

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

For more details, see [CONTRIBUTING.md](./CONTRIBUTING.md)

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- LICENSE -->
## License

Distributed under the MIT License. See `LICENSE` for more information.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- CONTACT -->
## Contact

NetBuddy Team - [@netbuddy](https://github.com/netbuddy)

Project Link: [https://github.com/netbuddy/agents-admin](https://github.com/netbuddy/agents-admin)

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- ACKNOWLEDGMENTS -->
## Acknowledgments

We extend our gratitude to the following open source projects and resources:

* [Choose an Open Source License](https://choosealicense.com)
* [Img Shields](https://shields.io)
* [GitHub Pages](https://pages.github.com)
* [Font Awesome](https://fontawesome.com)
* [React Icons](https://react-icons.github.io/react-icons/search)
* [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen)
* [Air](https://github.com/cosmtrek/air) - Go live reload tool
* [Lucide React](https://lucide.dev) - Icon library

<p align="right">(<a href="#readme-top">back to top</a>)</p>

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


.PHONY: all build test lint clean dev-up dev-down run-api run-nodemanager run-web stop-api stop-nodemanager stop-web monitoring-up monitoring-down watch-api watch-nodemanager generate-api generate-api-force generate-api-models generate-api-server generate-api-spec bundle-openapi

# ========== OpenAPI 代码生成 ==========
OAPI_CODEGEN := $(HOME)/go/bin/oapi-codegen
OPENAPI_DIR := api/openapi
CODEGEN_DIR := api/codegen
GENERATED_DIR := api/generated/go

# 源文件（拆分的 yaml 文件，不包括 bundled.yaml）
OPENAPI_SOURCES := $(filter-out $(OPENAPI_DIR)/bundled.yaml,$(shell find $(OPENAPI_DIR) -name '*.yaml' 2>/dev/null))
OPENAPI_MAIN := $(OPENAPI_DIR)/openapi.yaml
OPENAPI_BUNDLED := $(OPENAPI_DIR)/bundled.yaml

# 生成的文件
GEN_MODELS := $(GENERATED_DIR)/models.gen.go
GEN_SERVER := $(GENERATED_DIR)/server.gen.go
GEN_SPEC := $(GENERATED_DIR)/spec.gen.go

# 合并 OpenAPI 规范（将拆分的文件合并为单个文件）
$(OPENAPI_BUNDLED): $(OPENAPI_SOURCES)
	@echo "Bundling OpenAPI specs..."
	npx @redocly/cli bundle $(OPENAPI_MAIN) -o $(OPENAPI_BUNDLED)

bundle-openapi: $(OPENAPI_BUNDLED)

# Models（所有模型定义）
$(GEN_MODELS): $(OPENAPI_BUNDLED) $(CODEGEN_DIR)/models.yaml
	@echo "Generating models..."
	@mkdir -p $(GENERATED_DIR)
	$(OAPI_CODEGEN) --config $(CODEGEN_DIR)/models.yaml $(OPENAPI_BUNDLED)

# Server（服务接口）
$(GEN_SERVER): $(OPENAPI_BUNDLED) $(CODEGEN_DIR)/server.yaml $(GEN_MODELS)
	@echo "Generating server interface..."
	$(OAPI_CODEGEN) --config $(CODEGEN_DIR)/server.yaml $(OPENAPI_BUNDLED)

# Spec（嵌入规范）
$(GEN_SPEC): $(OPENAPI_BUNDLED) $(CODEGEN_DIR)/spec.yaml
	@echo "Generating embedded spec..."
	$(OAPI_CODEGEN) --config $(CODEGEN_DIR)/spec.yaml $(OPENAPI_BUNDLED)

# 快捷目标
generate-api-models: $(GEN_MODELS)
generate-api-server: $(GEN_SERVER)
generate-api-spec: $(GEN_SPEC)

# 生成所有（增量）
generate-api: $(GEN_MODELS) $(GEN_SERVER) $(GEN_SPEC)

# 强制重新生成所有
generate-api-force:
	@echo "Force regenerating all OpenAPI code..."
	@rm -f $(OPENAPI_BUNDLED)
	@rm -rf $(GENERATED_DIR)
	@$(MAKE) generate-api

# 默认目标
all: lint test build

# 构建（自动检查 OpenAPI 是否需要重新生成）
build: generate-api
	@echo "Building API Server..."
	CGO_ENABLED=0 go build -o bin/api-server ./cmd/api-server
	@echo "Building NodeManager..."
	CGO_ENABLED=0 go build -o bin/nodemanager ./cmd/nodemanager

# ========== 前端构建 ==========
.PHONY: web-build web-clean

web-build: ## 构建前端静态文件（用于嵌入 Go 二进制）
	@echo "Building frontend for static export..."
	cd web && STATIC_EXPORT=true npm run build
	@echo "Frontend built: web/out/"

web-clean: ## 清理前端构建产物
	rm -rf web/out web/.next

# ========== 生产构建（前后端合一） ==========
.PHONY: release release-linux release-darwin release-windows

release: web-build generate-api ## 完整生产构建（前端嵌入，多平台）
	@echo "Building release binaries with embedded frontend..."
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/api-server-linux-amd64 ./cmd/api-server
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bin/api-server-darwin-amd64 ./cmd/api-server
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o bin/api-server-darwin-arm64 ./cmd/api-server
	@echo "Release binaries built in bin/"

release-linux: web-build generate-api ## 仅构建 Linux 版本
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/api-server-linux-amd64 ./cmd/api-server
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/nodemanager-linux-amd64 ./cmd/nodemanager

# 测试
test:
	go test -v -race ./internal/... ./pkg/...

test-all:
	go test -v -race ./...

test-coverage:
	go test -v -race -coverprofile=coverage.out ./internal/... ./pkg/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-integration:
	@echo "Running integration tests (requires database)..."
	TEST_DATABASE_URL="postgres://agents:agents_dev_password@localhost:5432/agents_admin?sslmode=disable" \
	go test -v ./tests/integration/...

test-e2e:
	@echo "Running E2E tests (requires running API server)..."
	API_BASE_URL="http://localhost:8080" \
	go test -v ./tests/e2e/...

# 代码检查
lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .

# 开发环境
dev-up:
	docker compose -f deployments/docker-compose.yml up -d postgres redis minio
	@echo "Waiting for services..."
	@sleep 5
	@echo "Services started:"
	@docker compose -f deployments/docker-compose.yml ps

dev-down:
	docker compose -f deployments/docker-compose.yml down

dev-logs:
	docker compose -f deployments/docker-compose.yml logs -f

# 运行（开发模式）
run-api:
	@echo "Starting API Server..."
	DATABASE_URL="postgres://agents:agents_dev_password@localhost:5432/agents_admin?sslmode=disable" \
	REDIS_URL="redis://localhost:6380/0" \
	go run ./cmd/api-server

stop-api:
	@echo "Stopping API Server on port 8080..."
	@PID=$$(lsof -ti tcp:8080 || true); \
	if [ -n "$$PID" ]; then \
		kill $$PID && echo "API Server stopped (pid: $$PID)"; \
	else \
		echo "No process listening on 8080"; \
	fi

run-nodemanager:
	@echo "Starting NodeManager..."
	API_SERVER_URL="http://localhost:8080" \
	NODE_ID="dev-node-01" \
	WORKSPACE_DIR="/tmp/agents-workspaces" \
	go run ./cmd/nodemanager

stop-nodemanager:
	@echo "Stopping NodeManager on port 18000-18099 (ttyd)..."
	@PORTS=$$(seq 18000 18099); \
	FOUND=0; \
	for p in $$PORTS; do \
		PID=$$(lsof -ti tcp:$$p || true); \
		if [ -n "$$PID" ]; then \
			FOUND=1; \
			kill $$PID && echo "Killed process $$PID on port $$p"; \
		fi; \
	done; \
	if [ "$$FOUND" -eq 0 ]; then \
		echo "No nodemanager-related ports (18000-18099) in use"; \
	fi

run-web:
	@echo "Starting Web UI on port 3002..."
	cd web && npm run dev

run-api-dev: ## 开发模式运行后端（不嵌入前端，需单独启动 Next.js dev server）
	@echo "Starting API Server in dev mode (no embedded frontend)..."
	DATABASE_URL="postgres://agents:agents_dev_password@localhost:5432/agents_admin?sslmode=disable" \
	REDIS_URL="redis://localhost:6380/0" \
	go run -tags dev ./cmd/api-server

stop-web:
	@echo "Stopping Web UI on port 3002..."
	@PID=$$(lsof -ti tcp:3002 || true); \
	if [ -n "$$PID" ]; then \
		kill $$PID && echo "Web UI stopped (pid: $$PID)"; \
	else \
		echo "No process listening on 3002"; \
	fi

# 热加载运行（修改代码后自动重启）
watch-api:
	@echo "Starting API Server with hot reload (air)..."
	@echo "修改 Go 代码后会自动重新编译并重启"
	air -c .air.api.toml

watch-nodemanager:
	@echo "Starting Executor with hot reload (air)..."
	@echo "修改 Go 代码后会自动重新编译并重启"
	air -c .air.nodemanager.toml

# 监控栈
monitoring-up:
	@echo "Starting monitoring stack (Prometheus, Grafana, Loki, Jaeger, etcd)..."
	docker compose -f deployments/docker-compose.monitoring.yml up -d
	@echo "Waiting for services..."
	@sleep 3
	@echo "Monitoring stack started:"
	@docker compose -f deployments/docker-compose.monitoring.yml ps
	@echo ""
	@echo "URLs:"
	@echo "  Prometheus: http://localhost:9090"
	@echo "  Grafana:    http://localhost:3001 (admin/admin)"
	@echo "  Jaeger:     http://localhost:16686"
	@echo "  etcd:       localhost:2379"

monitoring-down:
	docker compose -f deployments/docker-compose.monitoring.yml down

monitoring-logs:
	docker compose -f deployments/docker-compose.monitoring.yml logs -f

# Deb 打包
.PHONY: deb

deb: release-linux ## 构建 Linux 二进制并打包为 .deb
	@echo "Building .deb packages..."
	./deployments/deb/build-deb.sh

# Docker 构建
docker-build:
	docker build -f deployments/Dockerfile.api -t agents-admin/api-server:dev .
	docker build -f deployments/Dockerfile.agent -t agents-admin/executor:dev .

# 清理
clean:
	rm -rf bin/
	rm -rf web/out web/.next
	rm -f coverage.out coverage.html
	docker compose -f deployments/docker-compose.yml down -v

# 帮助
help:
	@echo "Available targets:"
	@echo ""
	@echo "  Frontend:"
	@echo "    web-build        - Build frontend for static export (embed into Go)"
	@echo "    web-clean        - Clean frontend build artifacts"
	@echo ""
	@echo "  Release (frontend + backend combined):"
	@echo "    release          - Build multi-platform release binaries"
	@echo "    release-linux    - Build Linux release binary only"
	@echo ""
	@echo "  Code Generation (by type):"
	@echo "    generate-api         - Generate all OpenAPI code (incremental)"
	@echo "    generate-api-force   - Force regenerate all OpenAPI code"
	@echo "    generate-api-models  - Generate models only"
	@echo "    generate-api-server  - Generate server interface only"
	@echo "    generate-api-spec    - Generate embedded spec only"
	@echo "    bundle-openapi       - Bundle split OpenAPI files into one"
	@echo ""
	@echo "  Build & Test:"
	@echo "    build            - Build binaries (auto-generates API if needed)"
	@echo "    test             - Run unit tests"
	@echo "    test-all         - Run all tests"
	@echo "    test-coverage    - Run tests with coverage"
	@echo "    test-integration - Run integration tests"
	@echo "    test-e2e         - Run E2E tests"
	@echo "    lint             - Run linter"
	@echo ""
	@echo "  Development:"
	@echo "    dev-up           - Start dev infrastructure"
	@echo "    dev-down         - Stop dev infrastructure"
	@echo "    run-api          - Run API Server locally"
	@echo "    run-nodemanager  - Run NodeManager locally"
	@echo "    run-web          - Run Web UI locally"
	@echo "    run-api-dev      - Run API Server in dev mode (no embedded frontend)"
	@echo "    watch-api        - Run API Server with hot reload (air)"
	@echo "    watch-nodemanager - Run NodeManager with hot reload (air)"
	@echo ""
	@echo "  Infrastructure:"
	@echo "    monitoring-up    - Start monitoring stack"
	@echo "    monitoring-down  - Stop monitoring stack"
	@echo "    docker-build     - Build Docker images"
	@echo "    clean            - Clean build artifacts"

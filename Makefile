.PHONY: all build test lint clean dev-up dev-down run-api run-executor run-web stop-api stop-executor stop-web monitoring-up monitoring-down watch-api watch-executor

# 默认目标
all: lint test build

# 构建
build:
	@echo "Building API Server..."
	CGO_ENABLED=0 go build -o bin/api-server ./cmd/api-server
	@echo "Building Executor..."
	CGO_ENABLED=0 go build -o bin/executor ./cmd/executor

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

run-executor:
	@echo "Starting Executor..."
	API_SERVER_URL="http://localhost:8080" \
	NODE_ID="dev-node-01" \
	WORKSPACE_DIR="/tmp/agents-workspaces" \
	go run ./cmd/executor

stop-executor:
	@echo "Stopping Executor on port 18000-18099 (ttyd) and any 8080 client connections..."
	@# 终止 ttyd 暴露端口
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
		echo "No executor-related ports (18000-18099) in use"; \
	fi

run-web:
	@echo "Starting Web UI on port 3002..."
	cd web && npm run dev

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

watch-executor:
	@echo "Starting Executor with hot reload (air)..."
	@echo "修改 Go 代码后会自动重新编译并重启"
	air -c .air.executor.toml

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

# Docker 构建
docker-build:
	docker build -f deployments/Dockerfile.api -t agents-admin/api-server:dev .
	docker build -f deployments/Dockerfile.agent -t agents-admin/executor:dev .

# 清理
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
	docker compose -f deployments/docker-compose.yml down -v

# 帮助
help:
	@echo "Available targets:"
	@echo "  build            - Build binaries"
	@echo "  test             - Run unit tests"
	@echo "  test-all         - Run all tests"
	@echo "  test-coverage    - Run tests with coverage"
	@echo "  test-integration - Run integration tests"
	@echo "  test-e2e         - Run E2E tests"
	@echo "  lint             - Run linter"
	@echo "  dev-up           - Start dev infrastructure"
	@echo "  dev-down         - Stop dev infrastructure"
	@echo "  run-api          - Run API Server locally"
	@echo "  run-executor     - Run Executor locally"
	@echo "  run-web          - Run Web UI locally (支持热加载)"
	@echo "  watch-api        - Run API Server with hot reload (air)"
	@echo "  watch-executor   - Run Executor with hot reload (air)"
	@echo "  monitoring-up    - Start monitoring stack"
	@echo "  monitoring-down  - Stop monitoring stack"
	@echo "  docker-build     - Build Docker images"
	@echo "  clean            - Clean build artifacts"

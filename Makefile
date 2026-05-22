.PHONY: help infra-up infra-down server worker test lint tidy sync-types schema-gen schema-init clean

help: ## 显示帮助
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ====== 基础设施 ======

infra-up: ## 启动 Dgraph + OpenViking + NATS + Postgres + Jaeger + Prometheus
	docker compose -f deployments/docker-compose.yml up -d

infra-down: ## 停止全部基础设施
	docker compose -f deployments/docker-compose.yml down

infra-logs: ## 查看基础设施日志
	docker compose -f deployments/docker-compose.yml logs -f

infra-ps: ## 查看基础设施状态
	docker compose -f deployments/docker-compose.yml ps

# ====== 应用启动 ======

server: ## 启动 HTTP 服务（默认 :8080）
	go run ./cmd/server

worker: ## 启动 Agent Worker
	go run ./cmd/worker

# ====== 开发工具 ======

test: ## 跑全部测试
	go test ./...

test-core: ## 跑核心测试（provenance / dag）
	go test ./internal/provenance/... ./internal/dag/...

lint: ## 静态检查
	go vet ./...
	gofmt -l .

tidy: ## 整理依赖
	go mod tidy

build: ## 编译验证
	go build ./...

# ====== Schema 同步 ======

schema-gen: ## 从 schema.yaml 生成 Go Struct + Dgraph schema（重要！修改 ontology 后必跑）
	cd internal/schema && go run -tags=generator generator.go

schema-init: ## 把 Schema 注入到 Dgraph
	go run ./cmd/server -init-schema

sync-types: ## 同步类型到前端（运行 tygo）
	@command -v tygo >/dev/null 2>&1 || $$(go env GOPATH)/bin/tygo --help >/dev/null 2>&1 || \
		(echo "tygo not installed, running go install..." && go install github.com/gzuidhof/tygo@latest)
	@$$(command -v tygo || echo $$(go env GOPATH)/bin/tygo) generate
	@echo "✅ 前端 types/api.ts 已更新，记得跑 npm run type-check"

# ====== 清理 ======

clean: ## 清理编译产物
	go clean -cache
	rm -rf bin/

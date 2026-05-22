# 本地开发环境

## Prerequisites

- Go 1.22+
- Docker Desktop（运行中）
- Node.js 18+（前端用）
- 一个 OpenAI 兼容 API Key（推荐 DeepSeek）

## 环境变量（`.env`）

```bash
# LLM
OPENAI_API_KEY=sk-xxx
OPENAI_BASE_URL=https://api.deepseek.com/v1
OPENAI_MODEL_NAME=deepseek-chat

# Storage
DGRAPH_GRPC=localhost:9080
VIKING_BASE_URL=http://localhost:8000
VIKING_API_KEY=local-dev
POSTGRES_DSN=postgres://competify:competify@localhost:5432/competify

# Messaging
NATS_URL=nats://localhost:4222

# Observability
JAEGER_ENDPOINT=http://localhost:14268/api/traces
PROMETHEUS_PORT=9090

# Server
SERVER_PORT=8080
HMAC_SECRET=local-dev-secret
```

## 启动顺序

```bash
# 1. 启动基础设施（Docker）
make infra-up

# 2. 等容器健康后，初始化 Dgraph Schema
make schema-init

# 3. 同步类型到前端（如有改动）
make sync-types

# 4. 启动后端服务
make server     # 终端 1
make worker     # 终端 2

# 5. 启动前端（另一个仓库）
cd ../competify-frontend && npm run dev
```

## 验证

```bash
curl http://localhost:8080/healthz                              # 应返回 200
curl -X POST http://localhost:8080/api/v1/tasks \
  -H 'Content-Type: application/json' \
  -d '{"competitor_name": "Cursor", "dimensions": ["feature"]}' # 应返回 task_id
```

## 常见问题

- **Dgraph 启动慢**：第一次拉镜像可能要 1-2 分钟，`docker compose logs dgraph` 看日志
- **OpenViking 连不上**：MVP 阶段可以先跳过，所有 Viking 调用做降级处理
- **LLM 调用失败**：检查 `OPENAI_API_KEY` 是否设置，DeepSeek 余额是否充足

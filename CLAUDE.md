# CompetifyAI Backend — CLAUDE.md

> 这是给 Claude Code 看的项目指南。每次开始 vibe coding 前请确认你已经读过这个文件。

---

## 项目简介

CompetifyAI 后端 — 基于 **Go + Eino** 的多 Agent 编排系统，做 AI 工具领域的竞品分析。
7 个 Agent 协作（Orchestrator → Collector ×5 → Cleaner → Analyzer ×4 → Cross-Reviewer → Writer → Final-Reviewer），输出带 Merkle Tree 溯源链的 Markdown 报告。

**配套设计文档**（必读）：
- `../CompetifyAI 技术架构方案/01_CompetifyAI_技术架构方案.md` — 为什么这样设计
- `../CompetifyAI 工程实现指南.md` — 完整代码实现参考

---

## 快速导航 — 目录职责

| 目录 | 职责 | 禁止事项 |
|------|------|---------|
| `internal/schema/` | 全部 Agent 间数据契约（Go Struct） + YAML 本体 | 写业务逻辑、有 import 第三方 SDK |
| `internal/agent/` | Agent 业务逻辑（Execute 方法） | 直接写 Eino 编排代码 |
| `internal/dag/` | Eino Graph + Workflow 编排 | 写业务逻辑（应只调用 agent） |
| `internal/storage/` | DB / 缓存 / 消息访问层 | 写 Agent 业务、组装报告 |
| `internal/provenance/` | Merkle Tree + 置信度 + 审计链 | 依赖 Agent 包 |
| `internal/messaging/` | NATS JetStream 事件循环 | 直接调用 Agent.Execute |
| `internal/observability/` | OpenTelemetry + Prometheus | 业务逻辑 |
| `internal/handler/` | HTTP / WebSocket 路由 | 直接访问 storage |
| `cmd/server/` | HTTP 服务入口（Hertz） | 写实现细节 |
| `cmd/worker/` | Agent Worker 入口 | 同上 |

**铁律**：禁止跨层直接调用。`handler` 不能直接调 `dgraph.Client`，必须经过 `agent` 或 `dag` 层。

---

## Go 编码约束（防堆屎山的护栏）

1. **单文件 ≤ 400 行**，超了立刻拆。
2. **嵌套 ≤ 4 层**，第 5 层出现 = 需要重构。
3. **导出函数禁用 `interface{}` 参数**，必须用 `internal/schema` 里的具体类型。
4. **依赖注入**：所有客户端（DB / LLM / NATS）通过构造函数参数传入，**禁止 `package-level var`**。
5. **错误包装**：统一用 `fmt.Errorf("pkg.Func: %w", err)`，禁止吞错误。
6. **命名约定**：Agent 文件名 = 角色名小写（`orchestrator.go` / `cross.go`）。
7. **并发安全**：DAG 节点函数必须无副作用（幂等），共享状态通过 channel 传递。
8. **只写 WHY 注释**，不写 WHAT。代码自身要能说明做了什么。

---

## Skill 速查（遇到对应场景请先调用）

| 场景 | Skill |
|------|-------|
| 新增任何 Agent | `/new-agent` |
| 新增本体类型（YAML → Dgraph → Go 三层同步） | `/add-ontology` |
| 修改 DAG 编排后 | `/verify-dag` |
| 改完 `internal/schema/types.go` 后 | `/sync-types`（同步到前端） |

---

## 规范文档导航（编码前必读对应文档）

| 任务 | 读哪个文档 |
|------|-----------|
| 写 Agent 业务逻辑 | `docs/agent-patterns.md` |
| 改 Eino DAG 结构 | `docs/dag-patterns.md` |
| 新增存储查询 | `docs/storage-patterns.md` |
| 不确定代码风格 | `docs/code-style.md` |
| 不知道某个 Agent 角色干嘛的 | `docs/agents.md` |
| 不知道某个 API 路径 | `docs/api.md` |

---

## 如何运行

```bash
make infra-up          # 启动 Dgraph + OpenViking + NATS + Postgres + Jaeger
make server            # 启动 HTTP 服务（默认 :8080）
make worker            # 启动 Agent Worker
make test              # 跑测试（重点：provenance/ 和 dag/）
make sync-types        # 把 schema/types.go 同步成前端 types/api.ts
make schema-gen        # 从 YAML 生成 Go Struct
```

---

## 跨仓约定

前端仓库在 `../competify-frontend/`。

**类型契约**：`internal/schema/types.go` 是单一真相源。改完字段后**必须立刻**运行 `make sync-types`，否则前端编译会失败。

**API 契约**：HTTP/WebSocket 路由文档在 `docs/api.md`，前端基于此实现 API 客户端。

---

## 提交规范（Commit Discipline）

**分支策略**：单线推进，不建 feature 分支，所有改动直接在 `main` 上提交。回滚用 `git revert`，不 force push。

**提交前必做 checklist**（完成一个 Phase/模块后，commit 前跑一遍）：

```bash
make test        # 测试必须绿
make lint        # go vet + gofmt
```

1. **代码可用性**：`go build ./...` 零报错、`go test ./...` 全通过。
2. **Skill 辅助验证**：提交前调用 `/verification-before-completion` 做最终检查；遇到 bug 先用 `/systematic-debugging`。
3. **边界条件**：检查 nil 指针、空切片、并发安全（Race）、错误是否被吞。
4. **架构可维护性**：单文件是否 ≤ 400 行、嵌套是否 ≤ 4 层、是否跨层调用、是否有 package-level var。

**Commit 拆分原则**：
- 一个 commit 只做一件事（如"feat: 实现 DevilsAdvocate 3 策略"、"test: 补充 provenance 篡改检测"）。
- 禁止把多个不相关的改动塞进同一个 commit。
- **用 `/commit-message-zh` 生成中文 Conventional Commit**，不手写。

---

## 语言约定（来自全局 CLAUDE.md）

- 回复用中文，代码注释用英文，commit 用中文 Conventional Commits
- 输出追求简洁，推理过程详尽
- 优先编辑文件，不重写整个文件
- 复杂任务先 plan

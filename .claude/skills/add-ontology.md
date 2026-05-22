---
name: add-ontology
description: 新增本体类型时执行 YAML → Go Struct → Dgraph Schema 三层同步，避免漏改
---

# /add-ontology — 新增本体类型的三层同步

触发时机：用户说「加一个本体」「新增 ObjectType」「新加一个实体类型 XXX」「Schema 改了」。

## 上下文

本体是单一真相源 `internal/schema/schema.yaml`，通过 `go generate` 产出 Go Struct，再注入 Dgraph。
**三层任何一层不同步都会出诡异 bug**，所以必须按顺序走完。

## 前置阅读

1. `docs/storage-patterns.md` — Schema 三层同步规范
2. `internal/schema/schema.yaml` — 看现有 ObjectType 长什么样
3. `internal/schema/generator.go` — 代码生成器入口
4. `internal/storage/dgraph/schema.go` — Dgraph Schema 注入逻辑

## 执行步骤

### Step 1：YAML 层 — 编辑 `internal/schema/schema.yaml`

按现有 ObjectType 的格式加。最小示例：

```yaml
object_types:
  - name: NewEntity
    description: "新实体说明（业务语义）"
    fields:
      - name: id
        type: string
        required: true
      - name: name
        type: string
        required: true
      - name: created_at
        type: time.Time
        required: true
    relations:
      - name: belongs_to
        target: Competitor
        cardinality: one
```

### Step 2：Go 层 — 跑代码生成

```bash
make schema-gen
# 等价于：go generate ./internal/schema/
```

会基于 schema.yaml 重写 `internal/schema/types.go` 里 ObjectType 部分（generator 实现细节见 `generator.go`）。

**如果 generator 还没实现（Phase 1 之前）**：手动在 `types.go` 加对应 Go Struct，命名匹配 YAML：

```go
type NewEntity struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
}
```

### Step 3：Dgraph 层 — 更新 schema.go

在 `internal/storage/dgraph/schema.go` 的 `InitializeSchema()` 里追加 DQL Schema：

```go
new_entity.id: string @index(exact) .
new_entity.name: string @index(term) .
new_entity.created_at: dateTime @index(hour) .
new_entity.belongs_to: uid @reverse .

type NewEntity {
    new_entity.id
    new_entity.name
    new_entity.created_at
    new_entity.belongs_to
}
```

### Step 4：注入到 Dgraph

```bash
make schema-init
# 等价于：go run ./cmd/server -init-schema
```

如果 Dgraph 没起，先：
```bash
make infra-up
```

### Step 5：前端类型同步

```bash
make sync-types
```

会调用 tygo 把 `types.go` → 前端 `competify-frontend/src/types/api.ts`。改完提醒用户：
- 切到前端跑 `npm run type-check` 确认没断
- 如果有 page/component 用到这个类型，可能需要补字段处理

## 完成检查清单

- [ ] `schema.yaml` 加了新 ObjectType
- [ ] `make schema-gen` 通过（或手动改了 `types.go`）
- [ ] `dgraph/schema.go` 的 `InitializeSchema()` 加了 DQL
- [ ] `make schema-init` 成功注入
- [ ] `make sync-types` 同步到前端
- [ ] `go build ./...` 通过

## 反模式（禁止）

- 只改 `types.go` 不改 `schema.yaml`（YAML 是源，下次跑 generator 会被覆盖）
- 只改 `schema.yaml` 不更新 dgraph/schema.go（Dgraph 不认识新字段，写入会报错）
- 改完不跑 `sync-types`（前端编译会断）

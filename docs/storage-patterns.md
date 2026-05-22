# 存储层扩展规范

## 新增 Dgraph 查询

**位置**：`internal/storage/dgraph/query.go`，**不允许在 Agent 层写 DQL**。

```go
// internal/storage/dgraph/query.go

func (c *Client) QueryFeaturesByCategory(ctx context.Context, category string) ([]*schema.Feature, error) {
    query := `query FeaturesByCategory($cat: string) {
        features(func: eq(category, $cat)) {
            uid feature_name description availability maturity
        }
    }`
    resp, err := c.dg.NewTxn().QueryWithVars(ctx, query, map[string]string{"$cat": category})
    if err != nil {
        return nil, fmt.Errorf("dgraph.QueryFeaturesByCategory: %w", err)
    }
    return parseFeatures(resp.Json)
}
```

**铁律**：Agent 调用 `dgraphClient.QueryFeaturesByCategory(...)`，**禁止 Agent 拼接 DQL 字符串**。

## 新增 Viking 资源

通过 `viking.CompetifyVikingPaths` 命名空间生成路径，**禁止硬编码 `viking://`**：

```go
// ✅ 正确
paths := viking.NewCompetifyPaths()
uri := paths.TaskCollectorPath(taskID, "web")  // viking://competify/tasks/{id}/collectors/web/

// ❌ 错误
uri := fmt.Sprintf("viking://competify/tasks/%s/collectors/web/", taskID)  // 禁止
```

如需新增目录命名空间，先改 `internal/storage/viking/resource.go` 添加方法。

## 新增 pgvector 索引

**位置**：`internal/storage/pgvector/index.go`。

**命名规范**：集合 = `competify_{用途}_embeddings`

```go
// ✅ 正确命名
"competify_collector_embeddings"
"competify_review_embeddings"
"competify_tech_embeddings"

// ❌ 错误命名
"embeddings"          // 太通用
"my_index"            // 没有 competify_ 前缀
"CollectorEmbeddings" // 不是 snake_case
```

## Schema 变更（三层同步）

**铁律**：YAML 是单一真相源。改 Schema 永远从 YAML 开始。

```bash
# 1. 先改 internal/schema/schema.yaml（唯一真相源）
vim internal/schema/schema.yaml

# 2. 从 YAML 生成 Go Struct
make schema-gen           # 等价于 go generate ./internal/schema/

# 3. 同步更新 Dgraph Schema 常量
# 编辑 internal/storage/dgraph/schema.go

# 4. 同步前端类型
make sync-types           # 等价于 tygo generate

# 5. 编译验证
go build ./...

# 6. 重新初始化 Dgraph Schema（影响线上数据）
go run ./cmd/server -init-schema
```

**禁止**：直接改 `internal/schema/types.go`（它是生成的）。可以加新的非 ontology 类型（如 API 请求/响应），但 ontology 类型必须经 YAML。

## 三个存储的边界

| 数据类型 | 存哪里 | 不存哪里 |
|---------|-------|---------|
| 任务执行历史、原始数据、辩论记录 | Viking | Dgraph |
| 竞品本体（Competitor / Product / Feature）+ 关系 | Dgraph | Viking |
| 语义向量、Embedding | pgvector | Dgraph |

**禁止**：把 Embedding 存 Dgraph 的 string 字段、把原始 HTML 存 Dgraph。

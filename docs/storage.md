# 存储层指南

## 三级存储职责

| 层级 | 组件 | 包 | 数据类型 |
|------|------|------|---------|
| L1 记忆层 | OpenViking | `internal/storage/viking/` | Agent 会话历史、原始数据、溯源链 |
| L2 图谱层 | Dgraph | `internal/storage/dgraph/` | 竞品本体（Competitor / Product / Feature / ...） |
| L3 向量层 | pgvector (替代 VikingDB) | `internal/storage/pgvector/` | 语义 Embedding |

## viking:// 命名空间

```
viking://competify/
├── tasks/{task_id}/
│   ├── orchestrator/    # 任务计划
│   ├── collectors/{source_type}/
│   ├── cleaner/
│   ├── analyzers/{dimension}/
│   ├── reviewer/        # 辩论记录
│   └── writer/
├── ontology/competitors/{name}/
├── provenance/{conclusion_id}/
└── schema/
```

**铁律**：禁止硬编码 viking:// 路径。通过 `viking.CompetifyVikingPaths` 的方法生成。

## 连接配置（环境变量）

```
DGRAPH_GRPC=localhost:9080
VIKING_BASE_URL=http://localhost:8000
VIKING_API_KEY=<from .env>
POSTGRES_DSN=postgres://competify:competify@localhost:5432/competify
NATS_URL=nats://localhost:4222
```

## 集合命名规则（pgvector）

- `competify_collector_embeddings` — 采集数据向量
- `competify_review_embeddings` — 用户评论向量
- `competify_tech_embeddings` — 技术文档向量

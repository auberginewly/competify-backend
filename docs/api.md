# API 契约

> 前端必读。所有 API 必须与此文档保持同步，新增 API 先改本文档再写代码。

## 任务管理

| Method | Path | 用途 |
|--------|------|------|
| POST | `/api/v1/tasks` | 创建竞品分析任务，body = `UserQuery` |
| GET | `/api/v1/tasks/:id` | 查询任务状态，response = `TaskStatus` |
| GET (WS) | `/api/v1/tasks/:id/dag` | 实时 DAG 状态推送（WebSocket） |

## 报告管理

| Method | Path | 用途 |
|--------|------|------|
| GET | `/api/v1/reports/:id` | 获取报告（Markdown） |
| GET | `/api/v1/reports/:id/provenance` | 报告完整溯源链 |
| POST | `/api/v1/reports/:id/approve` | 终审审批，body 含 PM signature |

## 本体图谱

| Method | Path | 用途 |
|--------|------|------|
| GET | `/api/v1/ontology/competitors` | 竞品列表 |
| GET | `/api/v1/ontology/competitors/:name` | 单个竞品详情 |
| GET | `/api/v1/ontology/graph` | 图谱数据（Cytoscape.js 格式） |
| GET | `/api/v1/ontology/timeline?start=&end=` | 时序事件 |

## 审计

| Method | Path | 用途 |
|--------|------|------|
| GET | `/api/v1/audit/:provenance_id` | 单条溯源链详情 |
| GET | `/api/v1/audit/:provenance_id/verify` | Merkle 哈希校验 |

## 可观测性

| Method | Path | 用途 |
|--------|------|------|
| GET | `/metrics` | Prometheus 指标 |
| GET | `/healthz` | 健康检查 |

## WebSocket 消息格式

DAG 状态推送 JSON 结构（每个节点状态变化推送一次）：

```json
{
  "node_name": "collector_web",
  "status": "running",
  "progress": 0.45,
  "timestamp": "2026-05-22T13:00:00Z",
  "logs": ["开始抓取 cursor.com"]
}
```

`status` 枚举：`pending` / `running` / `review` / `error` / `done`

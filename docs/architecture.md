# 架构总览

> 详细 rationale 见 `../../CompetifyAI 技术架构方案/01_CompetifyAI_技术架构方案.md`，本文档只做工程导航。

## 四层架构

```
┌────────────────────────────────────────────────────────────────┐
│  编排层 Orchestration  ←  internal/dag/ (Eino Graph + Workflow) │
├────────────────────────────────────────────────────────────────┤
│  执行层 Execution      ←  internal/agent/ (7 Agent 角色)         │
├────────────────────────────────────────────────────────────────┤
│  存储层 Storage        ←  internal/storage/ (Dgraph/Viking/PG)  │
├────────────────────────────────────────────────────────────────┤
│  观测层 Observability  ←  internal/observability/ (OTel + Prom) │
└────────────────────────────────────────────────────────────────┘
```

## 三大设计哲学

1. **Go 原生优先** — 编译期类型安全，goroutine 并发，单二进制部署
2. **一切皆图** — 竞品知识用 Dgraph 本体图谱建模，事件驱动动态演化
3. **信任但验证** — Devil's Advocate 对抗辩论 + Merkle Tree 溯源链双重保障

## 数据流

```
UserQuery
  → Orchestrator → TaskDAGPlan
  → Collector ×5 (并发) → RawDataPack
  → Cleaner (SimHash 去重) → NormalizedDataset
  → Analyzer ×4 (并发) → AnalysisResult
  → Cross-Reviewer (Devil's Advocate 辩论) → ReviewReport
  → Writer (Markdown + 脚注) → DraftReport
  → Final-Reviewer (HMAC 签名) → FinalReport
```

## 横切关注点

- **溯源**：每个 Agent 输出附带 `provenance_id`，写入 `AuditChain`，根哈希锚定到 Merkle Tree
- **置信度**：4 维加权（source 0.30 / clean 0.25 / llm 0.25 / review 0.20），< 0.6 标记 SUSPICIOUS
- **观测**：每个 Agent 自动 Span 埋点，LLM 调用记录 Token，DAG 节点状态 Prometheus Gauge

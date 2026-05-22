# Agent 角色速查表

| 角色 | 文件 | 输入类型 | 输出类型 | 核心职责 |
|------|------|---------|---------|---------|
| Orchestrator | `internal/agent/orchestrator.go` | `UserQuery` | `TaskDAGPlan` | 解析需求，生成任务计划，指定 Schema 版本 |
| Collector (web) | `internal/agent/collector/web.go` | `TaskDAGPlan` | `RawDataPack` | 抓官网 / Product Hunt |
| Collector (social) | `internal/agent/collector/social.go` | `TaskDAGPlan` | `RawDataPack` | 抓 Twitter / X |
| Collector (financial) | `internal/agent/collector/financial.go` | `TaskDAGPlan` | `RawDataPack` | 抓融资动态 |
| Collector (review) | `internal/agent/collector/review.go` | `TaskDAGPlan` | `RawDataPack` | 抓 G2 / Capterra |
| Collector (api) | `internal/agent/collector/api.go` | `TaskDAGPlan` | `RawDataPack` | 调 GitHub API |
| Cleaner | `internal/agent/cleaner.go` | `[]RawDataPack` | `NormalizedDataset` | SimHash 去重 + 字段标准化 |
| Analyzer (feature) | `internal/agent/analyzer/feature.go` | `NormalizedDataset` | `AnalysisResult` | 功能矩阵分析 |
| Analyzer (pricing) | `internal/agent/analyzer/pricing.go` | `NormalizedDataset` | `AnalysisResult` | 定价对比分析 |
| Analyzer (tech) | `internal/agent/analyzer/tech.go` | `NormalizedDataset` | `AnalysisResult` | 技术栈分析 |
| Analyzer (market) | `internal/agent/analyzer/market.go` | `NormalizedDataset` | `AnalysisResult` | 市场地位分析 |
| Cross-Reviewer | `internal/agent/reviewer/cross.go` | `[]AnalysisResult` | `ReviewReport` | 多维交叉验证 + 辩论流程 |
| Devil's Advocate | `internal/agent/reviewer/devil.go` | `AnalysisResult` | `*Conflict` | 反方论证 |
| Writer | `internal/agent/writer.go` | `ReviewReport` | `DraftReport` | 渲染 Markdown + 脚注 + Merkle Root |
| Final-Reviewer | `internal/agent/reviewer/final.go` | `DraftReport` | `FinalReviewOutput` | 终审 + HMAC 签名 |

## 通用接口

所有 Agent 实现 `internal/agent/agent.go` 的 `Agent` interface：

```go
type Agent interface {
    Name() string
    Execute(ctx context.Context, input interface{}) (interface{}, error)
    HealthCheck(ctx context.Context) error
}
```

`BaseAgent` 提供通用基础设施（VikingClient / Tracer / Metrics），通过嵌入复用。

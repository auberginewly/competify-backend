# Eino DAG 编写规范

> Eino API 模式来自 `../Sea-mult-agent/scholar-agent/backend/internal/agent/coder.go`（只读参考）。  
> Eino 版本：v0.8.4

## Graph vs Workflow 选择

| 场景 | 模式 | 理由 |
|------|------|------|
| 顶层 DAG 骨架（节点串联 + 并行 + 分支） | Graph | 支持 `AddBranch` 条件分支 |
| 节点内部字段提取 / 多输入聚合 | Workflow | `MapFields` 字段级映射 |
| 简单线性链 | Graph (single path) | 不需要 Workflow 的复杂度 |

## 节点注册三要素

```go
// 1. 创建泛型 Graph（编译期类型安全）
graph := compose.NewGraph[schema.UserQuery, schema.FinalReport]()

// 2. 注册 Lambda 节点
_ = graph.AddLambdaNode("orchestrator",
    wrapAgent[schema.UserQuery, schema.TaskDAGPlan](orchestrator))

// 3. 连接边
_ = graph.AddEdge(compose.START, "orchestrator")
_ = graph.AddEdge("orchestrator", "collector_web")
```

## 并行节点写法

多个节点接收同一个上游 = 自动并行：

```go
for i, coll := range collectors {
    nodeName := fmt.Sprintf("collector_%d", i)
    _ = graph.AddLambdaNode(nodeName, wrapAgent(coll))
    _ = graph.AddEdge("orchestrator", nodeName)  // 5 个 collector 并发启动
    _ = graph.AddEdge(nodeName, "cleaner")       // 汇聚到 cleaner
}
```

## 条件分支写法

```go
reviewBranch := compose.NewGraphBranch(
    func(ctx context.Context, report *schema.ReviewReport) (string, error) {
        switch report.NextAction {
        case "APPROVE": return "writer", nil
        case "RETRY_AUTO": return "retry", nil
        case "REJECT_HUMAN": return "human_intervention", nil
        default: return "writer", nil
        }
    },
    map[string]bool{
        "writer": true, "retry": true, "human_intervention": true,
    },
)
_ = graph.AddBranch("cross_reviewer", reviewBranch)
_ = graph.AddBranchEdge("cross_reviewer", "writer", "writer")
_ = graph.AddBranchEdge("cross_reviewer", "human_intervention", "human_intervention")
```

**铁律**：`NewGraphBranch` 第二个参数的 map key 必须与已注册的 `AddLambdaNode` 节点名一一对应，否则编译失败。

## ChatModel 节点（LLM 调用）

```go
chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
    BaseURL: baseURL, APIKey: apiKey, Model: modelName,
})
graph.AddChatModelNode("LLM_Analyze", chatModel)
```

输入类型必须是 `[]*schema.Message`（来自 `github.com/cloudwego/eino/schema`，不是本项目的 schema 包）。

## 命名规范

- 节点名：`snake_case`，与 Agent 角色名保持一致
- 并行节点：`{role}_{idx}` 或 `{role}_{dimension}`，如 `collector_0` / `analyzer_pricing`
- 分支目标：必须是已注册的节点名

## 反模式

```go
// ❌ 错误：在节点 Lambda 内直接调其他节点
graph.AddLambdaNode("collector", compose.InvokableLambda(func(ctx, in) (..., error) {
    cleanerAgent.Execute(ctx, ...)  // 禁止！通过 AddEdge 连接，不直接调用
}))

// ❌ 错误：节点函数有副作用（写全局状态）
graph.AddLambdaNode("collector", compose.InvokableLambda(func(ctx, in) (..., error) {
    globalCache[in.ID] = result  // 禁止！节点必须幂等无副作用
}))

// ❌ 错误：忘记连接到 END
// 任何最终节点必须 `graph.AddEdge(lastNode, compose.END)`
```

## 编译验证检查清单

每次改动 `internal/dag/` 后必须做（通过 `/verify-dag` skill 自动）：

1. `go build ./...` — 类型不匹配立即失败
2. `go vet ./internal/dag/...` — 常见错误
3. 所有 `AddEdge` 的目标节点必须已 `AddLambdaNode` 注册
4. 所有 `AddBranch` 的 map 节点必须已注册
5. `compose.START` 和 `compose.END` 必须连通

## 推荐学习路径

1. 先看 `../Sea-mult-agent/scholar-agent/backend/internal/agent/chat.go` — 最简单的 3 节点 Graph
2. 再看 `coder.go` — 含 LLM + 自修复循环的复杂 Graph
3. 本项目 `internal/dag/graph.go` 是这两者的扩展版

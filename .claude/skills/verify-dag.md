---
name: verify-dag
description: 修改 DAG 编排后跑完整验证清单（编译/节点注册/边连通性/Compile 通过）
---

# /verify-dag — DAG 变更后的验证清单

触发时机：用户说「DAG 改完了」「Graph 加完节点了」「跑一下 verify-dag」，或刚改完 `internal/dag/graph.go` / `workflow.go` / `branch.go`。

## 上下文

Eino DAG 在 `Compile(ctx)` 时才检查：
- 节点是否都连通（孤立节点报错）
- 是否有环（除非显式 AddBranch）
- 输入输出类型是否对齐

编译时报错信息晦涩，所以建立这个 checklist 在编译前先肉眼/脚本扫一遍能省很多时间。

## 前置阅读

1. `docs/dag-patterns.md` — Eino Graph/Workflow 写法
2. `internal/dag/graph.go` — 顶层 `BuildCompetifyGraph`
3. `internal/dag/workflow.go` — Cleaner/Reviewer 内部 Workflow
4. `internal/dag/branch.go` — 条件分支（reviewCondition / awaitHumanIntervention）

## 验证清单（按顺序跑）

### 1. 编译验证

```bash
go build ./internal/dag/...
go build ./...
```

任何一个报错先修了再继续。常见错误：
- `wrapAgent` 函数签名不匹配 → 检查 Agent 是否实现了 `Execute(ctx, interface{}) (interface{}, error)`
- 类型参数不匹配 `NewGraph[I, O]()` → 检查 I/O 是不是 `schema` 包里的具体类型

### 2. 节点注册检查

肉眼或 grep 一下：

```bash
grep -n "AddLambdaNode\|AddChatModelNode" internal/dag/graph.go
```

每个 Agent 都应该有对应的 `AddXxxNode("agent_id", ...)`，**节点 ID 必须与 Agent.Name() 返回值完全一致**。

### 3. 边连通性

```bash
grep -n "AddEdge\|AddBranch" internal/dag/graph.go
```

按 7 Agent 流程检查（4 Agent 简化版同理）：

```
orchestrator → collector_* → cleaner → analyzer_* → cross_reviewer → writer → final_reviewer
```

- 每个节点必须至少有 1 条入边（除起点）和 1 条出边（除终点）
- 起点：`orchestrator`
- 终点：`final_reviewer` 或 `writer`

### 4. 分支条件检查

如果用了 `AddBranch`：
- 分支函数返回值（如 `"approve" / "retry" / "reject"`）必须与下游节点 ID 一一对应
- 检查每个分支返回值都有对应节点（漏 case 会 panic）

### 5. Compile 通过

```bash
go test ./internal/dag/...
```

如果 `dag_test.go` 还没写，临时跑一个 main 试试：

```go
ctx := context.Background()
runner, err := dag.BuildCompetifyGraph().Compile(ctx)
if err != nil {
    log.Fatal(err)
}
_ = runner
```

### 6. 类型断言扫雷

每个 `wrapAgent` 内的 type assertion 是常见崩点：

```bash
grep -rn "input.(" internal/agent/
```

确认每个 Agent 的输入类型断言匹配上游产出。

## 完成检查清单

- [ ] `go build ./...` 通过
- [ ] 节点 ID 与 Agent.Name() 一致
- [ ] 每个节点至少有 1 条入边 + 1 条出边
- [ ] 分支返回值对得上下游节点
- [ ] `Compile(ctx)` 不报错
- [ ] 类型断言路径都对齐

## 反模式（禁止）

- 在 Agent.Execute 里直接调用其他 Agent（应该让 DAG 编排）
- 节点名硬编码字符串散落各处（应该用 `const` 集中管理）
- DAG 节点函数有副作用（破坏幂等性，重跑会出错）

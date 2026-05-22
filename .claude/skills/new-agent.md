---
name: new-agent
description: 按 docs/agent-patterns.md 的 7 步流程新增一个 Agent，确保 schema、agent、dag 三层同步更新
---

# /new-agent — 新增 Agent 的标准流程

触发时机：用户说「新增一个 XXX Agent」「加一个 Collector」「实现 XXX 角色」。

## 前置阅读（必读）

1. `docs/agent-patterns.md` — Agent 7 步流程 + BaseAgent 使用
2. `docs/code-style.md` — Go 编码规范（单文件 ≤ 400 行 / 嵌套 ≤ 4 层）
3. `docs/dag-patterns.md` — Eino Graph 编排（最后挂载节点时要用）
4. `internal/agent/agent.go` — 看 Agent interface + BaseAgent 当前长什么样

## 执行步骤

### Step 1：确认 Agent 角色定位

问用户：
- 这个 Agent 属于哪一类？Orchestrator / Collector / Cleaner / Analyzer / Reviewer / Writer
- 输入是什么 schema 类型？输出是什么 schema 类型？
- 是否需要调用 LLM？是否需要访问 Dgraph / pgvector？

### Step 2：在 `internal/schema/types.go` 定义 I/O 类型

如果输入/输出类型还没有，先加到 `types.go`。命名规则：
- `XXXInput` / `XXXOutput`（如 `FeatureAnalyzerInput`）
- 字段加 `json` tag
- 改完立刻提醒用户跑 `make sync-types` 同步到前端

### Step 3：在 `internal/agent/<role>/` 下新建文件

命名：角色名小写 `.go`（如 `feature.go` / `devil.go`）。骨架：

```go
package analyzer  // or collector / reviewer

import (
    "context"
    "fmt"

    "github.com/competify-ai/competify-backend/internal/agent"
    "github.com/competify-ai/competify-backend/internal/schema"
)

type FeatureAnalyzer struct {
    agent.BaseAgent
    // 依赖通过构造函数注入
}

func NewFeatureAnalyzer( /* deps */ ) *FeatureAnalyzer {
    return &FeatureAnalyzer{
        BaseAgent: agent.BaseAgent{
            ID:   "feature_analyzer",
            Role: "analyzer",
        },
    }
}

func (a *FeatureAnalyzer) Name() string { return a.ID }

func (a *FeatureAnalyzer) Execute(ctx context.Context, input interface{}) (interface{}, error) {
    in, ok := input.(schema.FeatureAnalyzerInput)
    if !ok {
        return nil, fmt.Errorf("agent.FeatureAnalyzer: invalid input type %T", input)
    }
    // 业务逻辑
    _ = in
    return schema.FeatureAnalyzerOutput{}, nil
}

func (a *FeatureAnalyzer) HealthCheck(ctx context.Context) error { return nil }
```

### Step 4：依赖注入

如果需要 LLM / DB 客户端，**不要**用 package-level var，必须通过 `NewXxx(...)` 构造函数传入。

### Step 5：在 `internal/dag/graph.go` 挂载节点

```go
graph.AddLambdaNode("feature_analyzer", wrapAgent(featureAnalyzer))
graph.AddEdge("cleaner", "feature_analyzer")
graph.AddEdge("feature_analyzer", "cross_reviewer")
```

节点名 = Agent.ID（snake_case）。

### Step 6：写单测（最小可行）

在同目录加 `feature_test.go`，至少覆盖：
- 输入类型断言失败的错误路径
- Happy path 返回正确 Output

### Step 7：验证

```bash
go build ./...           # 必须无编译错误
go test ./internal/agent/analyzer/...  # 单测通过
```

改完 DAG 后调用 `/verify-dag`。

## 完成检查清单

- [ ] `internal/schema/types.go` 加了 I/O 类型
- [ ] `internal/agent/<role>/<name>.go` 实现了 Agent interface
- [ ] 依赖通过构造函数注入，无 package-level var
- [ ] `internal/dag/graph.go` 挂载了节点
- [ ] 写了单测（至少 Happy + Error 各 1 个）
- [ ] `go build ./...` 通过
- [ ] 提醒用户：如果改了 schema，跑 `make sync-types`

## 反模式（禁止）

- 在 Agent 内写 Eino 编排代码（属于 dag/ 层）
- 在 Agent 内直接 import handler/ 包
- `Execute` 用 `interface{}` 之外的具体类型签名（会破坏 Agent interface 统一性）

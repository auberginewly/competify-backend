# Agent 编写规范

> 新增任何 Agent 都按这个流程走。可以通过 `/new-agent` skill 自动引导。

## 标准 7 步流程

### 1. 在 `internal/schema/types.go` 定义输入输出类型

```go
// 新 Agent 叫 "Translator"，把英文报告翻译成中文
type TranslateInput struct {
    TaskID string `json:"task_id"`
    Source string `json:"source"`
    TargetLang string `json:"target_lang"`
}

type TranslateOutput struct {
    TaskID string `json:"task_id"`
    Result string `json:"result"`
    Confidence float64 `json:"confidence"`
}
```

### 2. 在 `internal/agent/` 下创建实现文件

文件名 = Agent 角色名小写：`internal/agent/translator.go`

```go
package agent

import (
    "context"
    "fmt"
    "github.com/competify-ai/competify-backend/internal/schema"
)

type Translator struct {
    BaseAgent
    // 注入依赖（LLM 客户端、Viking 等）
}

func NewTranslator(base BaseAgent) *Translator {
    return &Translator{BaseAgent: base}
}

func (t *Translator) Name() string { return "translator" }

func (t *Translator) Execute(ctx context.Context, input interface{}) (interface{}, error) {
    in, ok := input.(*schema.TranslateInput)
    if !ok {
        return nil, fmt.Errorf("translator.Execute: invalid input type")
    }
    // ... 业务逻辑
    return &schema.TranslateOutput{...}, nil
}

func (t *Translator) HealthCheck(ctx context.Context) error { return nil }
```

### 3. 在 `internal/dag/graph.go` 注册节点和边

```go
_ = graph.AddLambdaNode("translator",
    wrapAgent[schema.TranslateInput, schema.TranslateOutput](translator))
_ = graph.AddEdge("writer", "translator")
_ = graph.AddEdge("translator", "final_reviewer")
```

### 4. 如需内部字段映射，在 `dag/workflow.go` 建 Workflow

字段级数据映射（如把多个 Analyzer 的结果聚合）用 Workflow + `MapFields`。详见 `docs/dag-patterns.md`。

### 5. 如需条件分支，在 `dag/branch.go` 添加条件函数

```go
func translateCondition(ctx context.Context, in *schema.TranslateOutput) (string, error) {
    if in.Confidence < 0.7 {
        return "retry", nil
    }
    return "final_reviewer", nil
}
```

### 6. 在 `cmd/worker/main.go` 注入依赖并启动

```go
baseAgent := agent.BaseAgent{
    VikingClient: vikingClient,
    Tracer: tracer,
    Metrics: metrics,
}
translator := agent.NewTranslator(baseAgent)
```

### 7. 验证

```bash
go build ./...   # 类型安全编译通过
go vet ./...     # 静态检查通过
```

## BaseAgent 使用

所有 Agent 通过嵌入 `BaseAgent` 复用通用基础设施：

```go
type BaseAgent struct {
    ID string
    Name string
    VikingClient *viking.Client
    Tracer trace.Tracer
    Metrics *metrics.AgentMetrics
}
```

不要在每个 Agent 里重复定义这些字段。

## 反模式

```go
// ❌ 错误：在 Agent 里写 Eino 编排
func (a *MyAgent) Execute(ctx context.Context, input interface{}) (interface{}, error) {
    graph := compose.NewGraph[...]()  // 禁止！Eino 编排只能在 internal/dag/
    ...
}

// ❌ 错误：在 Agent 里直接读 DB
func (a *MyAgent) Execute(ctx context.Context, input interface{}) (interface{}, error) {
    dgraphClient.Query(...)  // 禁止！应通过构造函数注入
    ...
}

// ❌ 错误：在 Agent 里跨调其他 Agent
func (a *MyAgent) Execute(ctx context.Context, input interface{}) (interface{}, error) {
    otherAgent.Execute(...)  // 禁止！Agent 间通过 DAG 编排连接，不直接调用
    ...
}
```

## 测试

每个 Agent 至少要有：
- 一个 happy path 单测
- 一个错误处理单测（无效输入、依赖失败）

测试文件：`internal/agent/translator_test.go`

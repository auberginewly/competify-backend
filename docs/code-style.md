# Go 编码规范（本项目专用）

> 这是防堆屎山的护栏。提交代码前自查一遍。

## 1. 包职责一览

| 包 | 职责 | 禁止 |
|----|------|------|
| `internal/schema/` | 纯数据类型 + YAML 本体定义 | 业务逻辑、依赖第三方 SDK |
| `internal/agent/` | Agent.Execute 业务逻辑 | 直接写 Eino 编排 |
| `internal/dag/` | Eino Graph / Workflow 编排 | 写业务（只调用 agent） |
| `internal/storage/{dgraph,viking,pgvector}/` | DB 客户端封装 | 写 Agent 业务 |
| `internal/provenance/` | Merkle Tree + 置信度 + 审计链 | 依赖 agent 包 |
| `internal/messaging/` | NATS 事件循环 | 直接调 Agent.Execute |
| `internal/observability/` | OTel + Prometheus 埋点 | 业务逻辑 |
| `internal/handler/` | HTTP / WebSocket 路由 | 直接调 storage（必须经 dag 或 agent） |

## 2. 文件与函数大小

- **单文件 ≤ 400 行**。超过 = 必须拆，按职责拆成多个文件。
- **单函数 ≤ 80 行**。超过 = 抽辅助函数。
- **嵌套 ≤ 4 层**。第 5 层出现 = 提取子函数或用 early return。

## 3. 错误处理

```go
// ✅ 正确
if err != nil {
    return nil, fmt.Errorf("dgraph.UpsertCompetitor: %w", err)
}

// ❌ 错误（吞错）
if err != nil {
    log.Println(err)
    return nil, nil
}

// ❌ 错误（没有上下文）
return nil, err
```

错误前缀格式：`包名.函数名`，方便日志追溯。

## 4. 导出函数签名

```go
// ✅ 正确：使用 schema 包的具体类型
func (c *Cleaner) Execute(ctx context.Context, in *schema.RawDataPack) (*schema.NormalizedDataset, error)

// ❌ 错误：导出函数使用 interface{}
func (c *Cleaner) Execute(ctx context.Context, in interface{}) (interface{}, error)
```

`Agent` interface 内部用 `interface{}` 是为了统一签名，但**具体实现的 Execute 方法必须用具体类型**，通过类型断言转换。

## 5. 依赖注入

```go
// ✅ 正确：构造函数注入
type CrossReviewer struct {
    devil *DevilsAdvocate
    viking *viking.Client
    tracer trace.Tracer
}

func NewCrossReviewer(devil *DevilsAdvocate, viking *viking.Client, tracer trace.Tracer) *CrossReviewer {
    return &CrossReviewer{devil: devil, viking: viking, tracer: tracer}
}

// ❌ 错误：package-level 全局变量
var vikingClient *viking.Client  // 禁止
```

## 6. 命名约定

- Agent 文件名 = 角色名小写（`orchestrator.go` / `cross.go` / `devil.go`）
- Schema 类型名 = 数据流顺序命名（`UserQuery` → `TaskDAGPlan` → `RawDataPack` → ...）
- 构造函数：`NewXxx`，返回 `*Xxx`
- 私有辅助函数小写开头，导出函数大写开头（标准 Go 约定）

## 7. 并发安全

- DAG 节点函数（`InvokableLambda`）必须**无副作用**、**幂等**
- 共享状态只通过 channel 传递，禁止用 mutex 保护跨 Agent 的共享变量
- 所有 goroutine 必须有明确的退出机制（context.Done 或 channel close）

## 8. 注释

```go
// ✅ 正确：解释 WHY
// 使用 SimHash 而非 MD5：需要近似去重，MD5 改一个字节就完全不同
fingerprint := SimHash(content)

// ❌ 错误：复述 WHAT
// 计算 SimHash
fingerprint := SimHash(content)
```

**规则**：只在选择背后有非显然 trade-off 时才写注释。代码自身能说清楚的事情不写。

## 9. 提交前自查

```bash
go build ./...       # 编译通过
go vet ./...         # 静态检查通过
go test ./...        # 测试通过
gofmt -l .           # 输出空（格式正确）
```

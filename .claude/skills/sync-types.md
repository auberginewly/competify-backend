---
name: sync-types
description: 改完 internal/schema/types.go 后同步类型到前端，提醒前端验证编译
---

# /sync-types — Schema 类型跨仓同步

触发时机：用户改完 `internal/schema/types.go` / `schema.yaml`，或说「同步一下类型」「sync types」。

## 上下文

`internal/schema/types.go` 是前后端类型契约的单一真相源。tygo 工具读 Go Struct，自动生成前端 `../competify-frontend/src/types/api.ts`。

**不跑这一步前端会编译失败**，因为前端的 `api/` 层和 `types/api.ts` 强绑定。

## 前置检查

1. `tygo.yaml` 存在，且 `output_path` 指向 `../competify-frontend/src/types/api.ts`
2. tygo 已安装：`which tygo`，没装跑 `go install github.com/gzuidhof/tygo@latest`
3. 前端仓库目录存在：`ls ../competify-frontend/src/types/`

## 执行步骤

### Step 1：跑 sync-types

```bash
make sync-types
# 等价于：tygo generate
```

输出应看到：「✅ 前端 types/api.ts 已更新」。

### Step 2：肉眼扫一眼生成结果

```bash
head -50 ../competify-frontend/src/types/api.ts
```

确认：
- 新加的字段出现了
- 字段名 / 类型符合预期（`time.Time` 应该被映射成 `string`，`uuid.UUID` 也是 `string`）
- 没有奇怪的 `any` 漏掉

### Step 3：提醒用户切到前端验证

告诉用户：

> 类型已同步到 `competify-frontend/src/types/api.ts`。请切到前端跑：
> ```bash
> cd ../competify-frontend && npm run type-check
> ```
> 如果报错，可能是某个 `pages/` 或 `api/` 文件用了旧字段名，需要顺势改一下。

### Step 4（可选）：如果有 enum 改动

tygo 对 Go const 块的支持有限。如果 schema 里加了 enum（如 status 字符串常量），可能需要手动在前端补一份对照（写在 `competify-frontend/src/types/enums.ts`）。

## 完成检查清单

- [ ] `make sync-types` 命令成功跑完
- [ ] `types/api.ts` 包含新字段
- [ ] 提醒用户去前端跑 `npm run type-check`
- [ ] 如果有 enum，补了前端对应文件

## 反模式（禁止）

- 直接手改前端 `types/api.ts`（会被下次 sync 覆盖）
- 改 schema 不跑 sync，等前端编译挂了再排查
- 把 `time.Time` 用 `interface{}` 包，破坏 tygo 类型推导

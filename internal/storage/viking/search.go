package viking

// Find / Grep / Glob — OpenViking 检索接口封装。
//
// Phase 6 实现：
//   - Find(ctx, query, path, level, topK) — 深度语义检索
//   - Grep(ctx, pattern, path) — 文本搜索
//   - Glob(ctx, pattern, path) — 路径通配匹配
//
// Devil's Advocate 通过 Grep 搜反向证据（详见 docs/agent-patterns.md）。

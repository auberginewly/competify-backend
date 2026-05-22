# 溯源与置信度系统

## Merkle Tree 工作流

每个 Agent 执行后写入 `AuditLog`，所有 AuditLog 的 hash 作为叶子节点构成 Merkle Tree，根哈希锚定到 `DraftReport.MerkleRootHash`。

```
AuditLog ─┐
AuditLog ─┼─→ MerkleNode ─┐
AuditLog ─┘                ├─→ MerkleNode (Root) → DraftReport
AuditLog ─┐                │
AuditLog ─┼─→ MerkleNode ─┘
AuditLog ─┘
```

**验证机制**：任何 AuditLog 被篡改 → 重新计算的 Root 与原 Root 不匹配 → 篡改被立即检测。

## 置信度公式

```
C = w₁ × c_source + w₂ × c_clean + w₃ × c_llm + w₄ × c_review

默认权重：
  w₁ = 0.30 (Source — 数据源可信度)
  w₂ = 0.25 (Clean — 清洗质量)
  w₃ = 0.25 (LLM — 模型一致性)
  w₄ = 0.20 (Review — 审查通过度)
```

## 置信度等级

| Score | Label | 处理 |
|-------|-------|------|
| ≥ 0.9 | HIGH | 直接通过 |
| ≥ 0.7 | MEDIUM | 通过 |
| ≥ 0.6 | LOW | 通过但标注「建议复核」 |
| < 0.6 | SUSPICIOUS | 入人工复核队列，**不写入终版报告** |

## 人工介入触发条件（任一满足）

- 整体 Confidence < 0.6
- Cross-Reviewer 达到 MaxRounds (3) 仍未达成共识
- 任一 Conflict 的 Severity = "high"
- LLM 多次采样一致性 < 阈值

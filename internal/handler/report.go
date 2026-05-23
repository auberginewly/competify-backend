package handler

import (
	"context"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/competify-ai/competify-backend/internal/schema"
)

// GetReportHandler returns a handler for GET /api/v1/reports/:id.
func GetReportHandler(deps *Deps) func(context.Context, *app.RequestContext) {
	return func(ctx context.Context, c *app.RequestContext) {
		id := c.Param("id")

		// Try to read a real report from the memory store first.
		if r, ok := deps.ReportStore.Get(id); ok {
			c.JSON(200, r)
			return
		}

		// Fallback to mock data for backward compatibility.
		approvedAt, _ := time.Parse(time.RFC3339, "2026-05-22T13:00:00Z")
		c.JSON(200, schema.FinalReport{
			TaskID:     id,
			ReportID:   id,
			Content:    mockReportMarkdown,
			Status:     "published",
			Signature:  "0x7a3f9e2b1c8d4e5f6a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f",
			ApprovedBy: "final_reviewer",
			ApprovedAt: approvedAt,
			MerkleRoot: "0x7a3f9e2b1c8d4e5f6a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f",
		})
	}
}

// GetProvenance handles GET /api/v1/reports/:id/provenance.
func GetProvenance(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")
	c.JSON(200, map[string]any{
		"report_id": id,
		"chain": []map[string]any{
			{"agent": "collector_web", "step": "数据采集", "confidence": 0.92},
			{"agent": "analyzer_feat", "step": "功能分析", "confidence": 0.88},
			{"agent": "writer", "step": "报告撰写", "confidence": 0.95},
			{"agent": "final_reviewer", "step": "终审发布", "confidence": 1.0},
		},
	})
}

// ApproveReport handles POST /api/v1/reports/:id/approve.
func ApproveReport(ctx context.Context, c *app.RequestContext) {
	c.JSON(200, map[string]string{"status": "approved"})
}

const mockReportMarkdown = `# 竞品分析报告 — Cursor

## 1. 产品定位

Cursor 是基于 **VS Code fork** 的 AI-native 编辑器，主打 AI Pair Programming 体验。其差异化在于将 LLM 深度嵌入 IDE 工作流，而非简单的侧边栏 Chat。[^fn-1]

## 2. 核心功能矩阵

| 功能 | 可用性 | 成熟度 |
|------|--------|--------|
| Tab 自动补全 | ✅ | 5 |
| Composer 多文件编辑 | ✅ | 4 |
| Agent Mode 自主任务执行 | ✅ | 3 |
| @Codebase 上下文索引 | ✅ | 4 |

[^fn-1]: 功能数据来源于 cursor.sh 官网及 GitHub Releases 页面，置信度 0.88。

## 3. 定价策略

- **Free**: 有限次数的 fast request
- **Pro $20/月**: 500 次 fast + 无限 slow
- **Business $40/月**: 团队共享额度 + 管理员控制台

[^fn-2]: 定价数据抓取自 cursor.com/pricing，2026-05 有效。置信度 0.95。

## 4. 技术栈推断

编辑器底层为 Electron + VS Code fork，模型层混合自训练 Tab 模型与 GPT-4o / Claude Sonnet。[^fn-3]

---

*报告由 CompetifyAI 多 Agent 协作系统生成，每条结论附带 Merkle Proof 可验证。*
`

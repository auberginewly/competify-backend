// Writer renders ReviewReport into Markdown DraftReport with provenance footnotes.
package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
)

// competitorFromTaskID extracts the competitor name from a task ID.
// Task IDs have the form "task_<CompetitorName>_<unix_timestamp>".
func competitorFromTaskID(taskID string) string {
	s := strings.TrimPrefix(taskID, "task_")
	if idx := strings.LastIndex(s, "_"); idx > 0 {
		return s[:idx]
	}
	return s
}

// extractDimFromURI returns the dimension segment from a VikingURI.
// e.g. "viking://competify/tasks/.../analyzers/feature" → "feature"
func extractDimFromURI(uri string) string {
	parts := strings.Split(uri, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// Writer composes the final Markdown report and anchors it to the audit Merkle Tree.
type Writer struct {
	BaseAgent
}

// NewWriter creates a Writer with an optional audit chain.
func NewWriter(auditChain ...*provenance.AuditChain) *Writer {
	ba := BaseAgent{Role: "writer"}
	if len(auditChain) > 0 {
		ba.AuditChain = auditChain[0]
	}
	return &Writer{BaseAgent: ba}
}

func (w *Writer) Name() string { return "writer" }

// Execute accepts *schema.ReviewReport and returns *schema.DraftReport.
// It reads the shared AuditChain to obtain the MerkleRootHash and computes
// per-conclusion confidence scores.
func (w *Writer) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	report, ok := input.(*schema.ReviewReport)
	if !ok {
		return nil, fmt.Errorf("writer: expected *schema.ReviewReport, got %T", input)
	}

	// Derive MerkleRoot from the shared audit chain (injected via constructor).
	merkleRoot := ""
	if w.AuditChain != nil {
		merkleRoot = w.AuditChain.GetMerkleRoot()
	}

	// Build dynamic footnotes from analyzer outputs forwarded by CrossReviewer.
	var footnotes []schema.Footnote
	for i, a := range report.Analyses {
		if a == nil {
			continue
		}
		footnotes = append(footnotes, schema.Footnote{
			ID:           fmt.Sprintf("fn-%d", i+1),
			Conclusion:   a.Payload,
			Confidence:   a.Score,
			ProvenanceID: a.AnalyzerID,
			VikingURI:    fmt.Sprintf("viking://competify/tasks/%s/analyzers/%s", report.TaskID, a.Dimension),
		})
	}
	if len(footnotes) == 0 {
		// ultimate fallback so the pipeline never breaks
		footnotes = append(footnotes, schema.Footnote{
			ID:         "fn-0",
			Conclusion: "No analysis results available.",
			Confidence: 0.0,
		})
	}

	avgConfidence := calculateAverageConfidence(footnotes)

	competitor := competitorFromTaskID(report.TaskID)
	draft := &schema.DraftReport{
		TaskID:          report.TaskID,
		Title:           fmt.Sprintf("%s 竞品分析报告", competitor),
		MarkdownContent: buildMarkdown(report.TaskID, footnotes),
		Footnotes:       footnotes,
		MerkleRootHash:  merkleRoot,
		Confidence:      avgConfidence,
		GeneratedAt:     time.Now().UTC(),
		WriterID:        w.GenerateID(report.TaskID),
	}

	w.RecordAudit(report.TaskID, w.GenerateID(report.TaskID),
		report.NextAction, draft.MerkleRootHash,
		"Writer composed draft report with Merkle root and confidence", draft.Confidence)

	return draft, nil
}

func (w *Writer) HealthCheck(ctx context.Context) error { return nil }

// buildMarkdown assembles a structured Chinese report body grouped by analysis dimension.
func buildMarkdown(taskID string, footnotes []schema.Footnote) string {
	competitor := competitorFromTaskID(taskID)

	// Group footnotes by dimension extracted from VikingURI.
	sectionMap := map[string]schema.Footnote{}
	for _, fn := range footnotes {
		dim := extractDimFromURI(fn.VikingURI)
		sectionMap[dim] = fn
	}

	md := fmt.Sprintf("# %s 竞品分析报告\n\n", competitor)
	md += "> 由 CompetifyAI 多 Agent 协作系统生成 · Merkle Tree 溯源可验证\n\n"
	md += "---\n\n"

	sections := []struct{ dim, title string }{
		{"feature", "## 📦 功能矩阵分析"},
		{"pricing", "## 💰 定价策略"},
		{"tech", "## 🔧 技术栈推断"},
		{"market", "## 📊 市场定位"},
	}
	for _, s := range sections {
		md += s.title + "\n\n"
		if fn, ok := sectionMap[s.dim]; ok {
			md += fn.Conclusion + "\n\n"
		} else {
			md += "_数据不足，跳过此维度。_\n\n"
		}
	}

	md += "---\n\n## 数据溯源\n\n"
	for _, fn := range footnotes {
		md += fmt.Sprintf("- **置信度 %.0f%%**：%s\n", fn.Confidence*100, fn.Conclusion)
	}
	return md
}

// calculateAverageConfidence computes a simple mean for the stub.
// In production this would use the weighted provenance.CalculateConfidence.
func calculateAverageConfidence(footnotes []schema.Footnote) float64 {
	if len(footnotes) == 0 {
		return 0
	}
	var sum float64
	for _, fn := range footnotes {
		sum += fn.Confidence
	}
	avg := sum / float64(len(footnotes))
	if avg > 1.0 {
		avg = 1.0
	}
	return avg
}

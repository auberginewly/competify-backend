// Writer renders ReviewReport into Markdown DraftReport with provenance footnotes.
package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
)

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

	draft := &schema.DraftReport{
		TaskID:          report.TaskID,
		Title:           fmt.Sprintf("Competitor Analysis: %s", report.TaskID),
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

// buildMarkdown assembles a minimal report body from footnotes.
func buildMarkdown(taskID string, footnotes []schema.Footnote) string {
	md := fmt.Sprintf("# Competitor Analysis Report\n\n**Task ID:** %s\n\n", taskID)
	for _, fn := range footnotes {
		md += fmt.Sprintf("- %s (confidence: %.2f) [^%s]\n", fn.Conclusion, fn.Confidence, fn.ID)
	}
	md += "\n## Sources\n\n"
	for _, fn := range footnotes {
		md += fmt.Sprintf("[^%s]: %s — %s\n", fn.ID, fn.VikingURI, fn.ProvenanceID)
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

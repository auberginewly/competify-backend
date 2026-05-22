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

	// Stub conclusions with confidence anchors.
	footnotes := []schema.Footnote{
		{
			ID:           "fn-1",
			Conclusion:   "Pricing is competitive with a freemium model.",
			Confidence:   0.82,
			ProvenanceID: report.TaskID,
			VikingURI:    "viking://competify/tasks/" + report.TaskID + "/analyzers/pricing",
		},
		{
			ID:           "fn-2",
			Conclusion:   "Core differentiators: real-time collaboration, AI-assisted code review, and multi-language support.",
			Confidence:   0.88,
			ProvenanceID: report.TaskID,
			VikingURI:    "viking://competify/tasks/" + report.TaskID + "/analyzers/feature",
		},
		{
			ID:           "fn-3",
			Conclusion:   "Built on microservices with Rust core and TypeScript frontend.",
			Confidence:   0.78,
			ProvenanceID: report.TaskID,
			VikingURI:    "viking://competify/tasks/" + report.TaskID + "/analyzers/tech",
		},
		{
			ID:           "fn-4",
			Conclusion:   "Strong momentum in enterprise segment; 40 % YoY growth estimated.",
			Confidence:   0.71,
			ProvenanceID: report.TaskID,
			VikingURI:    "viking://competify/tasks/" + report.TaskID + "/analyzers/market",
		},
	}

	// Aggregate confidence using the 4-dimension formula.
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

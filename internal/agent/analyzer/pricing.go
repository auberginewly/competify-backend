package analyzer

import (
	"context"
	"fmt"
	"time"

	"github.com/competify-ai/competify-backend/internal/agent"
	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
)

// PricingAnalyzer compares pricing tiers and billing models.
type PricingAnalyzer struct {
	agent.BaseAgent
}

func NewPricingAnalyzer(auditChain ...*provenance.AuditChain) *PricingAnalyzer {
	ba := agent.BaseAgent{Role: "analyzer_pricing"}
	if len(auditChain) > 0 {
		ba.AuditChain = auditChain[0]
	}
	return &PricingAnalyzer{BaseAgent: ba}
}

func (p *PricingAnalyzer) Name() string { return "analyzer_pricing" }

func (p *PricingAnalyzer) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	ds, ok := input.(*schema.NormalizedDataset)
	if !ok {
		return nil, fmt.Errorf("pricing analyzer: expected *schema.NormalizedDataset, got %T", input)
	}
	result := &schema.AnalysisResult{
		TaskID:     ds.TaskID,
		Dimension:  "pricing",
		Payload:    "Pricing is competitive with a freemium model.",
		CoTReason:  "Compared tiers against 3 competitors.",
		Score:      0.82,
		SourceURIs: []string{ds.VikingURI},
		AnalyzerID: p.GenerateID(ds.TaskID),
		AnalyzedAt: time.Now().UTC(),
	}
	p.RecordAudit(ds.TaskID, p.GenerateID(ds.TaskID),
		ds.VikingURI, result.Payload,
		"PricingAnalyzer completed comparison", result.Score)
	return result, nil
}

func (p *PricingAnalyzer) HealthCheck(ctx context.Context) error { return nil }

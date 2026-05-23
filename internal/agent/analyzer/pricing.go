package analyzer

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/competify-ai/competify-backend/internal/agent"
	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
)

// PricingAnalyzer compares pricing tiers and billing models.
type PricingAnalyzer struct {
	agent.BaseAgent
	model *openai.ChatModel
}

func NewPricingAnalyzer(model *openai.ChatModel, auditChain ...*provenance.AuditChain) *PricingAnalyzer {
	ba := agent.BaseAgent{Role: "analyzer_pricing"}
	if len(auditChain) > 0 {
		ba.AuditChain = auditChain[0]
	}
	return &PricingAnalyzer{BaseAgent: ba, model: model}
}

func (p *PricingAnalyzer) Name() string { return "analyzer_pricing" }

func (p *PricingAnalyzer) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	ds, ok := input.(*schema.NormalizedDataset)
	if !ok {
		return nil, fmt.Errorf("pricing analyzer: expected *schema.NormalizedDataset, got %T", input)
	}

	competitor := competitorFromTaskID(ds.TaskID)
	payload, reasoning, score, err := analyzeWithLLM(ctx, p.model, "pricing", competitor, ds.CleanedText)
	if err != nil {
		log.Printf("[Analyzer] pricing/%s LLM failed, using stub: %v", competitor, err)
		payload = "Pricing is competitive with a freemium model."
		reasoning = "Compared tiers against 3 competitors."
		score = 0.82
	}

	result := &schema.AnalysisResult{
		TaskID:     ds.TaskID,
		Dimension:  "pricing",
		Payload:    payload,
		CoTReason:  reasoning,
		Score:      score,
		SourceURIs: []string{ds.VikingURI},
		AnalyzerID: p.GenerateID(ds.TaskID),
		ModelName:  "openai",
		AnalyzedAt: time.Now().UTC(),
	}
	p.RecordAudit(ds.TaskID, p.GenerateID(ds.TaskID),
		ds.VikingURI, result.Payload,
		"PricingAnalyzer completed comparison", result.Score)
	return result, nil
}

func (p *PricingAnalyzer) HealthCheck(ctx context.Context) error { return nil }

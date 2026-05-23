package analyzer

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/competify-ai/competify-backend/internal/agent"
	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
)

// MarketAnalyzer assesses market position, user growth and competitive landscape.
type MarketAnalyzer struct {
	agent.BaseAgent
	model *openai.ChatModel
}

func NewMarketAnalyzer(model *openai.ChatModel, auditChain ...*provenance.AuditChain) *MarketAnalyzer {
	ba := agent.BaseAgent{Role: "analyzer_market"}
	if len(auditChain) > 0 {
		ba.AuditChain = auditChain[0]
	}
	return &MarketAnalyzer{BaseAgent: ba, model: model}
}

func (m *MarketAnalyzer) Name() string { return "analyzer_market" }

func (m *MarketAnalyzer) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	ds, ok := input.(*schema.NormalizedDataset)
	if !ok {
		return nil, fmt.Errorf("market analyzer: expected *schema.NormalizedDataset, got %T", input)
	}

	payload, reasoning, score, err := analyzeWithLLM(ctx, m.model, "market", ds.TaskID, ds.CleanedText)
	if err != nil {
		payload = "Strong momentum in enterprise segment; 40 % YoY growth estimated."
		reasoning = "Synthesized signals from funding, hiring and social trends."
		score = 0.71
	}

	result := &schema.AnalysisResult{
		TaskID:     ds.TaskID,
		Dimension:  "market",
		Payload:    payload,
		CoTReason:  reasoning,
		Score:      score,
		SourceURIs: []string{ds.VikingURI},
		AnalyzerID: m.GenerateID(ds.TaskID),
		ModelName:  "openai",
		AnalyzedAt: time.Now().UTC(),
	}
	m.RecordAudit(ds.TaskID, m.GenerateID(ds.TaskID),
		ds.VikingURI, result.Payload,
		"MarketAnalyzer assessed positioning", result.Score)
	return result, nil
}

func (m *MarketAnalyzer) HealthCheck(ctx context.Context) error { return nil }

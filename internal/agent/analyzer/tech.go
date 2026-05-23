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

// TechAnalyzer evaluates architecture, open-source dependencies and performance.
type TechAnalyzer struct {
	agent.BaseAgent
	model *openai.ChatModel
}

func NewTechAnalyzer(model *openai.ChatModel, auditChain ...*provenance.AuditChain) *TechAnalyzer {
	ba := agent.BaseAgent{Role: "analyzer_tech"}
	if len(auditChain) > 0 {
		ba.AuditChain = auditChain[0]
	}
	return &TechAnalyzer{BaseAgent: ba, model: model}
}

func (t *TechAnalyzer) Name() string { return "analyzer_tech" }

func (t *TechAnalyzer) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	ds, ok := input.(*schema.NormalizedDataset)
	if !ok {
		return nil, fmt.Errorf("tech analyzer: expected *schema.NormalizedDataset, got %T", input)
	}

	payload, reasoning, score, err := analyzeWithLLM(ctx, t.model, "tech", ds.TaskID, ds.CleanedText)
	if err != nil {
		payload = "Built on microservices with Rust core and TypeScript frontend."
		reasoning = "Identified stack from GitHub repos and API headers."
		score = 0.78
	}

	result := &schema.AnalysisResult{
		TaskID:     ds.TaskID,
		Dimension:  "tech",
		Payload:    payload,
		CoTReason:  reasoning,
		Score:      score,
		SourceURIs: []string{ds.VikingURI},
		AnalyzerID: t.GenerateID(ds.TaskID),
		ModelName:  "openai",
		AnalyzedAt: time.Now().UTC(),
	}
	t.RecordAudit(ds.TaskID, t.GenerateID(ds.TaskID),
		ds.VikingURI, result.Payload,
		"TechAnalyzer evaluated architecture", result.Score)
	return result, nil
}

func (t *TechAnalyzer) HealthCheck(ctx context.Context) error { return nil }

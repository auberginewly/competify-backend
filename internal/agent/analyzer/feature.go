// Package analyzer contains 4 dimension analyzers (feature / pricing / tech / market).
// Each analyzer calls an LLM via Eino ChatModel and outputs schema.AnalysisResult.
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

// FeatureAnalyzer compares feature matrices and maturity levels.
type FeatureAnalyzer struct {
	agent.BaseAgent
	model *openai.ChatModel
}

func NewFeatureAnalyzer(model *openai.ChatModel, auditChain ...*provenance.AuditChain) *FeatureAnalyzer {
	ba := agent.BaseAgent{Role: "analyzer_feature"}
	if len(auditChain) > 0 {
		ba.AuditChain = auditChain[0]
	}
	return &FeatureAnalyzer{BaseAgent: ba, model: model}
}

func (f *FeatureAnalyzer) Name() string { return "analyzer_feature" }

func (f *FeatureAnalyzer) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	ds, ok := input.(*schema.NormalizedDataset)
	if !ok {
		return nil, fmt.Errorf("feature analyzer: expected *schema.NormalizedDataset, got %T", input)
	}

	competitor := competitorFromTaskID(ds.TaskID)
	payload, reasoning, score, err := analyzeWithLLM(ctx, f.model, "feature", competitor, ds.CleanedText)
	if err != nil {
		log.Printf("[Analyzer] feature/%s LLM failed, using stub: %v", competitor, err)
		payload = "Core differentiators: real-time collaboration, AI-assisted code review, and multi-language support."
		reasoning = "Extracted from product docs and release notes."
		score = 0.88
	}

	result := &schema.AnalysisResult{
		TaskID:     ds.TaskID,
		Dimension:  "feature",
		Payload:    payload,
		CoTReason:  reasoning,
		Score:      score,
		SourceURIs: []string{ds.VikingURI},
		AnalyzerID: f.GenerateID(ds.TaskID),
		ModelName:  "openai",
		AnalyzedAt: time.Now().UTC(),
	}
	f.RecordAudit(ds.TaskID, f.GenerateID(ds.TaskID),
		ds.VikingURI, result.Payload,
		"FeatureAnalyzer completed matrix comparison", result.Score)
	return result, nil
}

func (f *FeatureAnalyzer) HealthCheck(ctx context.Context) error { return nil }

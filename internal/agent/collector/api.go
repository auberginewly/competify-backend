package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/competify-ai/competify-backend/internal/agent"
	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
)

// APICollector queries GitHub / Crunchbase APIs.
type APICollector struct {
	agent.BaseAgent
}

func NewAPICollector(auditChain ...*provenance.AuditChain) *APICollector {
	ba := agent.BaseAgent{Role: "collector_api"}
	if len(auditChain) > 0 {
		ba.AuditChain = auditChain[0]
	}
	return &APICollector{BaseAgent: ba}
}

func (a *APICollector) Name() string { return "collector_api" }

func (a *APICollector) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	plan, ok := input.(*schema.TaskDAGPlan)
	if !ok {
		return nil, fmt.Errorf("api collector: expected *schema.TaskDAGPlan, got %T", input)
	}

	searchURL := fmt.Sprintf("https://api.github.com/search/repositories?q=%s&sort=stars&order=desc", plan.CompetitorName)
	body, statusCode, err := fetchText(ctx, searchURL)
	var rawContent string
	var confidence float64
	if err != nil || statusCode != 200 {
		rawContent = fmt.Sprintf("GitHub API failed: status=%d err=%v", statusCode, err)
		statusCode = 0
		confidence = 0.30
	} else {
		rawContent = body
		confidence = 0.85
	}

	pack := &schema.RawDataPack{
		TaskID:      plan.TaskID,
		SourceType:  "api",
		SourceURL:   searchURL,
		RawContent:  rawContent,
		StatusCode:  statusCode,
		CapturedAt:  time.Now().UTC(),
		CollectorID: a.GenerateID(plan.TaskID),
	}
	a.RecordAudit(plan.TaskID, a.GenerateID(plan.TaskID),
		plan.CompetitorName, pack.SourceURL,
		"APICollector queried GitHub API", confidence)
	return pack, nil
}

func (a *APICollector) HealthCheck(ctx context.Context) error { return nil }

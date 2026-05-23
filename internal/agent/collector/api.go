package collector

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/competify-ai/competify-backend/internal/agent"
	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
	"github.com/competify-ai/competify-backend/internal/storage/tavily"
)

// APICollector queries GitHub and tech API data for the competitor.
type APICollector struct {
	agent.BaseAgent
	TavilyClient *tavily.Client
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

	var rawContent string
	var confidence float64
	sourceURL := fmt.Sprintf("https://github.com/search?q=%s", plan.CompetitorName)

	if a.TavilyClient != nil {
		query := fmt.Sprintf("%s tech stack GitHub repositories programming language architecture open source", plan.CompetitorName)
		result, err := a.TavilyClient.Search(ctx, query)
		if err != nil {
			log.Printf("[APICollector] Tavily search failed for %s: %v", plan.CompetitorName, err)
			rawContent = fmt.Sprintf("Search unavailable. Use training knowledge about %s tech stack and GitHub presence.", plan.CompetitorName)
			confidence = 0.40
		} else {
			rawContent = result
			confidence = 0.85
			a.EmitEvent(ctx, plan.CompetitorName, "PRODUCT_LAUNCH",
				fmt.Sprintf("tavily api search: %d chars", len(result)), sourceURL)
		}
	} else {
		searchURL := fmt.Sprintf("https://api.github.com/search/repositories?q=%s&sort=stars&order=desc", plan.CompetitorName)
		body, statusCode, err := fetchText(ctx, searchURL)
		if err != nil || statusCode != 200 {
			rawContent = fmt.Sprintf("GitHub API unavailable (status=%d). Use training knowledge about %s tech stack and repositories.", statusCode, plan.CompetitorName)
			confidence = 0.40
		} else {
			rawContent = body
			confidence = 0.85
			a.EmitEvent(ctx, plan.CompetitorName, "PRODUCT_LAUNCH",
				fmt.Sprintf("github search: %d chars", len(body)), searchURL)
		}
	}

	pack := &schema.RawDataPack{
		TaskID:      plan.TaskID,
		SourceType:  "api",
		SourceURL:   sourceURL,
		RawContent:  rawContent,
		CapturedAt:  time.Now().UTC(),
		CollectorID: a.GenerateID(plan.TaskID),
	}
	a.RecordAudit(plan.TaskID, a.GenerateID(plan.TaskID),
		plan.CompetitorName, pack.SourceURL,
		"APICollector queried tech data", confidence)
	return pack, nil
}

func (a *APICollector) HealthCheck(ctx context.Context) error { return nil }

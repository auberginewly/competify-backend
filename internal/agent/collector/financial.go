package collector

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/competify-ai/competify-backend/internal/agent"
	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
	"github.com/competify-ai/competify-backend/internal/storage/tavily"
)

// FinancialCollector gathers funding and revenue data.
type FinancialCollector struct {
	agent.BaseAgent
	TavilyClient *tavily.Client
}

func NewFinancialCollector(auditChain ...*provenance.AuditChain) *FinancialCollector {
	ba := agent.BaseAgent{Role: "collector_financial"}
	if len(auditChain) > 0 {
		ba.AuditChain = auditChain[0]
	}
	return &FinancialCollector{BaseAgent: ba}
}

func (f *FinancialCollector) Name() string { return "collector_financial" }

func (f *FinancialCollector) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	plan, ok := input.(*schema.TaskDAGPlan)
	if !ok {
		return nil, fmt.Errorf("financial collector: expected *schema.TaskDAGPlan, got %T", input)
	}

	var rawContent string
	var confidence float64
	sourceURL := fmt.Sprintf("https://www.crunchbase.com/organization/%s", strings.ToLower(plan.CompetitorName))

	if f.TavilyClient != nil {
		query := fmt.Sprintf("%s funding rounds investors valuation revenue ARR team size crunchbase 2024", plan.CompetitorName)
		result, err := f.TavilyClient.Search(ctx, query)
		if err != nil {
			log.Printf("[FinancialCollector] Tavily search failed for %s: %v", plan.CompetitorName, err)
			rawContent = fmt.Sprintf("Search unavailable. Use training knowledge about %s funding and financial trajectory.", plan.CompetitorName)
			confidence = 0.40
		} else {
			rawContent = result
			confidence = 0.80
		}
	} else {
		body, statusCode, err := fetchText(ctx, sourceURL)
		if err != nil || statusCode != 200 {
			rawContent = fmt.Sprintf("Crunchbase unavailable (status=%d). Use training knowledge about %s funding rounds, investors, and valuation.", statusCode, plan.CompetitorName)
			confidence = 0.40
		} else {
			title := extractTitle(body)
			desc := extractMetaDescription(body)
			rawContent = fmt.Sprintf("Title: %s\nDescription: %s\nPreview: %s", title, desc, truncate(body, 600))
			confidence = 0.65
		}
	}

	pack := &schema.RawDataPack{
		TaskID:      plan.TaskID,
		SourceType:  "financial",
		SourceURL:   sourceURL,
		RawContent:  rawContent,
		CapturedAt:  time.Now().UTC(),
		CollectorID: f.GenerateID(plan.TaskID),
	}
	f.RecordAudit(plan.TaskID, f.GenerateID(plan.TaskID),
		plan.CompetitorName, pack.SourceURL,
		"FinancialCollector gathered funding data", confidence)
	return pack, nil
}

func (f *FinancialCollector) HealthCheck(ctx context.Context) error { return nil }

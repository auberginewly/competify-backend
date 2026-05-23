package collector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/competify-ai/competify-backend/internal/agent"
	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
)

// FinancialCollector gathers funding and revenue data.
type FinancialCollector struct {
	agent.BaseAgent
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

	url := fmt.Sprintf("https://www.crunchbase.com/organization/%s", strings.ToLower(plan.CompetitorName))
	body, statusCode, err := fetchText(ctx, url)
	var rawContent string
	var confidence float64
	if err != nil || statusCode != 200 {
		rawContent = fmt.Sprintf("Stub financial content for %s (fetch failed: status=%d err=%v)", plan.CompetitorName, statusCode, err)
		statusCode = 0
		confidence = 0.30
	} else {
		title := extractTitle(body)
		desc := extractMetaDescription(body)
		rawContent = fmt.Sprintf("Title: %s\nDescription: %s\nPreview: %s", title, desc, truncate(body, 600))
		confidence = 0.65
	}

	pack := &schema.RawDataPack{
		TaskID:      plan.TaskID,
		SourceType:  "financial",
		SourceURL:   url,
		RawContent:  rawContent,
		StatusCode:  statusCode,
		CapturedAt:  time.Now().UTC(),
		CollectorID: f.GenerateID(plan.TaskID),
	}
	f.RecordAudit(plan.TaskID, f.GenerateID(plan.TaskID),
		plan.CompetitorName, pack.SourceURL,
		"FinancialCollector gathered funding data", confidence)
	return pack, nil
}

func (f *FinancialCollector) HealthCheck(ctx context.Context) error { return nil }

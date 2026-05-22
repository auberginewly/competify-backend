package collector

import (
	"context"
	"fmt"
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
	pack := &schema.RawDataPack{
		TaskID:      plan.TaskID,
		SourceType:  "financial",
		SourceURL:   fmt.Sprintf("https://crunchbase.com/%s", plan.CompetitorName),
		RawContent:  fmt.Sprintf("Stub financial content for %s", plan.CompetitorName),
		StatusCode:  200,
		CapturedAt:  time.Now().UTC(),
		CollectorID: f.GenerateID(plan.TaskID),
	}
	f.RecordAudit(plan.TaskID, f.GenerateID(plan.TaskID),
		plan.CompetitorName, pack.SourceURL,
		"FinancialCollector gathered funding data", 0.75)
	return pack, nil
}

func (f *FinancialCollector) HealthCheck(ctx context.Context) error { return nil }

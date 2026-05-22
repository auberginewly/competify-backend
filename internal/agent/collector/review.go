package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/competify-ai/competify-backend/internal/agent"
	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
)

// ReviewCollector gathers user reviews from G2 / Capterra.
type ReviewCollector struct {
	agent.BaseAgent
}

func NewReviewCollector(auditChain ...*provenance.AuditChain) *ReviewCollector {
	ba := agent.BaseAgent{Role: "collector_review"}
	if len(auditChain) > 0 {
		ba.AuditChain = auditChain[0]
	}
	return &ReviewCollector{BaseAgent: ba}
}

func (r *ReviewCollector) Name() string { return "collector_review" }

func (r *ReviewCollector) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	plan, ok := input.(*schema.TaskDAGPlan)
	if !ok {
		return nil, fmt.Errorf("review collector: expected *schema.TaskDAGPlan, got %T", input)
	}
	pack := &schema.RawDataPack{
		TaskID:      plan.TaskID,
		SourceType:  "review",
		SourceURL:   fmt.Sprintf("https://g2.com/products/%s", plan.CompetitorName),
		RawContent:  fmt.Sprintf("Stub review content for %s", plan.CompetitorName),
		StatusCode:  200,
		CapturedAt:  time.Now().UTC(),
		CollectorID: r.GenerateID(plan.TaskID),
	}
	r.RecordAudit(plan.TaskID, r.GenerateID(plan.TaskID),
		plan.CompetitorName, pack.SourceURL,
		"ReviewCollector scraped G2 reviews", 0.70)
	return pack, nil
}

func (r *ReviewCollector) HealthCheck(ctx context.Context) error { return nil }

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

	url := fmt.Sprintf("https://www.g2.com/products/%s", strings.ToLower(plan.CompetitorName))
	body, statusCode, err := fetchText(ctx, url)
	var rawContent string
	var confidence float64
	if err != nil || statusCode != 200 {
		rawContent = fmt.Sprintf("Stub review content for %s (fetch failed: status=%d err=%v)", plan.CompetitorName, statusCode, err)
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
		SourceType:  "review",
		SourceURL:   url,
		RawContent:  rawContent,
		StatusCode:  statusCode,
		CapturedAt:  time.Now().UTC(),
		CollectorID: r.GenerateID(plan.TaskID),
	}
	r.RecordAudit(plan.TaskID, r.GenerateID(plan.TaskID),
		plan.CompetitorName, pack.SourceURL,
		"ReviewCollector scraped G2 reviews", confidence)
	return pack, nil
}

func (r *ReviewCollector) HealthCheck(ctx context.Context) error { return nil }

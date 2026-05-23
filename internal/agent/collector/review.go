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

// ReviewCollector gathers user reviews from G2 / Capterra / Reddit.
type ReviewCollector struct {
	agent.BaseAgent
	TavilyClient *tavily.Client
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

	var rawContent string
	var confidence float64
	sourceURL := fmt.Sprintf("https://www.g2.com/products/%s", strings.ToLower(plan.CompetitorName))

	if r.TavilyClient != nil {
		query := fmt.Sprintf("%s user reviews G2 Capterra pros cons complaints praise ratings 2024", plan.CompetitorName)
		result, err := r.TavilyClient.Search(ctx, query)
		if err != nil {
			log.Printf("[ReviewCollector] Tavily search failed for %s: %v", plan.CompetitorName, err)
			rawContent = fmt.Sprintf("Search unavailable. Use training knowledge about %s user reviews and ratings.", plan.CompetitorName)
			confidence = 0.40
		} else {
			rawContent = result
			confidence = 0.80
		}
	} else {
		body, statusCode, err := fetchText(ctx, sourceURL)
		if err != nil || statusCode != 200 {
			rawContent = fmt.Sprintf("G2 unavailable (status=%d). Use training knowledge about %s user reviews and ratings.", statusCode, plan.CompetitorName)
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
		SourceType:  "review",
		SourceURL:   sourceURL,
		RawContent:  rawContent,
		CapturedAt:  time.Now().UTC(),
		CollectorID: r.GenerateID(plan.TaskID),
	}
	r.RecordAudit(plan.TaskID, r.GenerateID(plan.TaskID),
		plan.CompetitorName, pack.SourceURL,
		"ReviewCollector gathered user reviews", confidence)
	return pack, nil
}

func (r *ReviewCollector) HealthCheck(ctx context.Context) error { return nil }

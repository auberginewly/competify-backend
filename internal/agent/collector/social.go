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

// SocialCollector gathers social media and community signals.
type SocialCollector struct {
	agent.BaseAgent
	TavilyClient *tavily.Client
}

func NewSocialCollector(auditChain ...*provenance.AuditChain) *SocialCollector {
	ba := agent.BaseAgent{Role: "collector_social"}
	if len(auditChain) > 0 {
		ba.AuditChain = auditChain[0]
	}
	return &SocialCollector{BaseAgent: ba}
}

func (s *SocialCollector) Name() string { return "collector_social" }

func (s *SocialCollector) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	plan, ok := input.(*schema.TaskDAGPlan)
	if !ok {
		return nil, fmt.Errorf("social collector: expected *schema.TaskDAGPlan, got %T", input)
	}

	var rawContent string
	var confidence float64
	sourceURL := fmt.Sprintf("https://www.producthunt.com/products/%s", strings.ToLower(plan.CompetitorName))

	if s.TavilyClient != nil {
		query := fmt.Sprintf("%s ProductHunt Reddit Twitter user sentiment community reviews social media buzz 2024", plan.CompetitorName)
		result, err := s.TavilyClient.Search(ctx, query)
		if err != nil {
			log.Printf("[SocialCollector] Tavily search failed for %s: %v", plan.CompetitorName, err)
			rawContent = fmt.Sprintf("Search unavailable. Use training knowledge about %s community reception and social media sentiment.", plan.CompetitorName)
			confidence = 0.40
		} else {
			rawContent = result
			confidence = 0.80
		}
	} else {
		body, statusCode, err := fetchText(ctx, sourceURL)
		if err != nil || statusCode != 200 {
			rawContent = fmt.Sprintf("ProductHunt unavailable (status=%d). Use training knowledge about %s community reception.", statusCode, plan.CompetitorName)
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
		SourceType:  "social",
		SourceURL:   sourceURL,
		RawContent:  rawContent,
		CapturedAt:  time.Now().UTC(),
		CollectorID: s.GenerateID(plan.TaskID),
	}
	s.RecordAudit(plan.TaskID, s.GenerateID(plan.TaskID),
		plan.CompetitorName, pack.SourceURL,
		"SocialCollector gathered community signals", confidence)
	return pack, nil
}

func (s *SocialCollector) HealthCheck(ctx context.Context) error { return nil }

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

// SocialCollector gathers social media and community signals.
type SocialCollector struct {
	agent.BaseAgent
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

	url := fmt.Sprintf("https://www.producthunt.com/products/%s", strings.ToLower(plan.CompetitorName))
	body, statusCode, err := fetchText(ctx, url)
	var rawContent string
	var confidence float64
	if err != nil || statusCode != 200 {
		rawContent = fmt.Sprintf("Stub social content for %s (fetch failed: status=%d err=%v)", plan.CompetitorName, statusCode, err)
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
		SourceType:  "social",
		SourceURL:   url,
		RawContent:  rawContent,
		StatusCode:  statusCode,
		CapturedAt:  time.Now().UTC(),
		CollectorID: s.GenerateID(plan.TaskID),
	}
	s.RecordAudit(plan.TaskID, s.GenerateID(plan.TaskID),
		plan.CompetitorName, pack.SourceURL,
		"SocialCollector gathered community signals", confidence)
	return pack, nil
}

func (s *SocialCollector) HealthCheck(ctx context.Context) error { return nil }

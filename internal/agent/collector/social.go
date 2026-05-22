package collector

import (
	"context"
	"fmt"
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
	pack := &schema.RawDataPack{
		TaskID:      plan.TaskID,
		SourceType:  "social",
		SourceURL:   fmt.Sprintf("https://twitter.com/%s", plan.CompetitorName),
		RawContent:  fmt.Sprintf("Stub social content for %s", plan.CompetitorName),
		StatusCode:  200,
		CapturedAt:  time.Now().UTC(),
		CollectorID: s.GenerateID(plan.TaskID),
	}
	s.RecordAudit(plan.TaskID, s.GenerateID(plan.TaskID),
		plan.CompetitorName, pack.SourceURL,
		"SocialCollector gathered community signals", 0.65)
	return pack, nil
}

func (s *SocialCollector) HealthCheck(ctx context.Context) error { return nil }

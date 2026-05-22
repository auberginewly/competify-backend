// Package collector contains all data collection agents.
package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/competify-ai/competify-backend/internal/agent"
	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
)

// WebCollector scrapes official websites and Product Hunt.
type WebCollector struct {
	agent.BaseAgent
}

// NewWebCollector creates a WebCollector with an optional audit chain.
func NewWebCollector(auditChain ...*provenance.AuditChain) *WebCollector {
	ba := agent.BaseAgent{Role: "collector_web"}
	if len(auditChain) > 0 {
		ba.AuditChain = auditChain[0]
	}
	return &WebCollector{BaseAgent: ba}
}

func (w *WebCollector) Name() string { return "collector_web" }

// Execute returns a stub RawDataPack so the pipeline can run end-to-end.
func (w *WebCollector) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	plan, ok := input.(*schema.TaskDAGPlan)
	if !ok {
		return nil, fmt.Errorf("web collector: expected *schema.TaskDAGPlan, got %T", input)
	}

	pack := &schema.RawDataPack{
		TaskID:      plan.TaskID,
		SourceType:  "web",
		SourceURL:   fmt.Sprintf("https://%s.com", plan.CompetitorName),
		RawContent:  fmt.Sprintf("Stub web content for %s", plan.CompetitorName),
		StatusCode:  200,
		CapturedAt:  time.Now().UTC(),
		CollectorID: w.GenerateID(plan.TaskID),
	}

	w.RecordAudit(plan.TaskID, w.GenerateID(plan.TaskID),
		plan.CompetitorName, pack.SourceURL,
		"WebCollector fetched homepage", 0.80)

	return pack, nil
}

func (w *WebCollector) HealthCheck(ctx context.Context) error { return nil }

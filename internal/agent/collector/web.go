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

// Execute scrapes the competitor's homepage and extracts title + meta description.
func (w *WebCollector) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	plan, ok := input.(*schema.TaskDAGPlan)
	if !ok {
		return nil, fmt.Errorf("web collector: expected *schema.TaskDAGPlan, got %T", input)
	}

	targetURL := fmt.Sprintf("https://%s.com", plan.CompetitorName)

	html, statusCode, err := fetchText(ctx, targetURL)
	var rawContent string
	var confidence float64
	if err != nil || statusCode != 200 {
		rawContent = fmt.Sprintf("Failed to fetch %s: status=%d err=%v", targetURL, statusCode, err)
		statusCode = 0
		confidence = 0.30
	} else {
		title := extractTitle(html)
		desc := extractMetaDescription(html)
		rawContent = fmt.Sprintf("Title: %s\nDescription: %s\nPreview: %s", title, desc, truncate(html, 800))
		confidence = 0.80

		// Emit a NEW_FEATURE event so the Reactive Watcher can update Dgraph.
		// Treat a successful page scrape as "possible new feature" detection.
		w.EmitEvent(ctx, plan.CompetitorName, "NEW_FEATURE",
			fmt.Sprintf("web: %s", title), targetURL)
	}

	pack := &schema.RawDataPack{
		TaskID:      plan.TaskID,
		SourceType:  "web",
		SourceURL:   targetURL,
		RawContent:  rawContent,
		StatusCode:  statusCode,
		CapturedAt:  time.Now().UTC(),
		CollectorID: w.GenerateID(plan.TaskID),
	}

	w.RecordAudit(plan.TaskID, w.GenerateID(plan.TaskID),
		plan.CompetitorName, pack.SourceURL,
		"WebCollector fetched homepage", confidence)

	return pack, nil
}

func (w *WebCollector) HealthCheck(ctx context.Context) error { return nil }

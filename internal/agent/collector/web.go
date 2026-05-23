// Package collector contains all data collection agents.
package collector

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/competify-ai/competify-backend/internal/agent"
	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
	"github.com/competify-ai/competify-backend/internal/storage/tavily"
)

// WebCollector collects official website and product positioning data.
type WebCollector struct {
	agent.BaseAgent
	TavilyClient *tavily.Client // nil = fall back to direct HTTP
}

func NewWebCollector(auditChain ...*provenance.AuditChain) *WebCollector {
	ba := agent.BaseAgent{Role: "collector_web"}
	if len(auditChain) > 0 {
		ba.AuditChain = auditChain[0]
	}
	return &WebCollector{BaseAgent: ba}
}

func (w *WebCollector) Name() string { return "collector_web" }

func (w *WebCollector) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	plan, ok := input.(*schema.TaskDAGPlan)
	if !ok {
		return nil, fmt.Errorf("web collector: expected *schema.TaskDAGPlan, got %T", input)
	}

	var rawContent string
	var confidence float64
	sourceURL := fmt.Sprintf("https://%s.com", plan.CompetitorName)

	if w.TavilyClient != nil {
		// Primary: Tavily search for product positioning and features.
		query := fmt.Sprintf("%s product features official website positioning 2024", plan.CompetitorName)
		result, err := w.TavilyClient.Search(ctx, query)
		if err != nil {
			log.Printf("[WebCollector] Tavily search failed for %s: %v", plan.CompetitorName, err)
			rawContent = fmt.Sprintf("Search unavailable. Use training knowledge about %s product positioning.", plan.CompetitorName)
			confidence = 0.40
		} else {
			rawContent = result
			confidence = 0.85
			w.EmitEvent(ctx, plan.CompetitorName, "NEW_FEATURE",
				fmt.Sprintf("tavily web search: %d chars", len(result)), sourceURL)
		}
	} else {
		// Fallback: direct HTTP scrape.
		html, statusCode, err := fetchText(ctx, sourceURL)
		if err != nil || statusCode != 200 {
			rawContent = fmt.Sprintf("Website unavailable (status=%d). Use training knowledge about %s product positioning.", statusCode, plan.CompetitorName)
			confidence = 0.40
		} else {
			title := extractTitle(html)
			desc := extractMetaDescription(html)
			rawContent = fmt.Sprintf("Title: %s\nDescription: %s\nPreview: %s", title, desc, truncate(html, 800))
			confidence = 0.80
			w.EmitEvent(ctx, plan.CompetitorName, "NEW_FEATURE", fmt.Sprintf("web: %s", title), sourceURL)
		}
	}

	pack := &schema.RawDataPack{
		TaskID:      plan.TaskID,
		SourceType:  "web",
		SourceURL:   sourceURL,
		RawContent:  rawContent,
		CapturedAt:  time.Now().UTC(),
		CollectorID: w.GenerateID(plan.TaskID),
	}
	w.RecordAudit(plan.TaskID, w.GenerateID(plan.TaskID),
		plan.CompetitorName, pack.SourceURL,
		"WebCollector collected product data", confidence)
	return pack, nil
}

func (w *WebCollector) HealthCheck(ctx context.Context) error { return nil }

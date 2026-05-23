package handler

import (
	"context"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/competify-ai/competify-backend/internal/schema"
)

// ListCompetitors handles GET /api/v1/ontology/competitors.
func ListCompetitors(ctx context.Context, c *app.RequestContext) {
	c.JSON(200, []schema.Competitor{
		{
			UID:          "0x1",
			CompanyName:  "Cursor",
			Website:      "https://cursor.sh",
			FundingStage: "series_b",
			TeamSize:     30,
			ThreatLevel:  5,
			Headquarters: "San Francisco, CA",
		},
		{
			UID:          "0x2",
			CompanyName:  "Windsurf",
			Website:      "https://windsurf.com",
			FundingStage: "series_a",
			TeamSize:     50,
			ThreatLevel:  4,
			Headquarters: "San Francisco, CA",
		},
	})
}

// GetCompetitor handles GET /api/v1/ontology/competitors/:name.
func GetCompetitor(ctx context.Context, c *app.RequestContext) {
	name := c.Param("name")
	c.JSON(200, schema.Competitor{
		UID:          "0x1",
		CompanyName:  name,
		Website:      "https://example.com",
		FundingStage: "series_a",
		TeamSize:     20,
		ThreatLevel:  3,
		Headquarters: "Beijing, CN",
	})
}

// GetOntologyGraph handles GET /api/v1/ontology/graph.
func GetOntologyGraph(ctx context.Context, c *app.RequestContext) {
	c.JSON(200, map[string]any{
		"nodes": []map[string]any{
			{"data": map[string]any{"id": "cursor", "label": "Cursor", "type": "competitor"}},
			{"data": map[string]any{"id": "windsurf", "label": "Windsurf", "type": "competitor"}},
			{"data": map[string]any{"id": "ai_editor", "label": "AI Editor", "type": "product"}},
		},
		"edges": []map[string]any{
			{"data": map[string]any{"source": "cursor", "target": "ai_editor", "relation": "develops"}},
			{"data": map[string]any{"source": "windsurf", "target": "ai_editor", "relation": "develops"}},
		},
	})
}

// GetTimeline handles GET /api/v1/ontology/timeline.
func GetTimeline(ctx context.Context, c *app.RequestContext) {
	eventDate, _ := time.Parse("2006-01-02", "2026-03-15")
	c.JSON(200, []schema.MarketEvent{
		{
			UID:         "0x10",
			EventType:   "funding",
			Description: "Cursor 宣布 Series B 融资 $50M",
			EventDate:   &eventDate,
			Impact:      "high",
			Confidence:  0.95,
			SourceURL:   "https://cursor.sh/blog/funding",
		},
	})
}

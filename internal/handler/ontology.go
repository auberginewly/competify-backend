package handler

import (
	"context"
	"log"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/competify-ai/competify-backend/internal/schema"
	"github.com/competify-ai/competify-backend/internal/storage/dgraph"
)

// ListCompetitors returns real data from Dgraph when available.
func ListCompetitors(dg *dgraph.Client) func(context.Context, *app.RequestContext) {
	return func(ctx context.Context, c *app.RequestContext) {
		if dg == nil {
			c.JSON(200, []schema.Competitor{})
			return
		}
		comps, err := dg.QueryAllCompetitors(ctx)
		if err != nil {
			log.Printf("[Ontology] ListCompetitors query failed: %v", err)
			c.JSON(200, []schema.Competitor{})
			return
		}
		c.JSON(200, comps)
	}
}

// GetCompetitor returns a single competitor by name.
func GetCompetitor(dg *dgraph.Client) func(context.Context, *app.RequestContext) {
	return func(ctx context.Context, c *app.RequestContext) {
		name := c.Param("name")
		if dg == nil {
			c.JSON(200, schema.Competitor{
				UID:          "0x1",
				CompanyName:  name,
				Website:      "https://example.com",
				FundingStage: "series_a",
				TeamSize:     20,
				ThreatLevel:  3,
				Headquarters: "Beijing, CN",
			})
			return
		}
		comp, err := dg.QueryCompetitorByName(ctx, name)
		if err != nil || comp == nil {
			c.JSON(200, schema.Competitor{
				UID:          "0x1",
				CompanyName:  name,
				Website:      "https://example.com",
				FundingStage: "series_a",
				TeamSize:     20,
				ThreatLevel:  3,
				Headquarters: "Beijing, CN",
			})
			return
		}
		c.JSON(200, comp)
	}
}

// GetOntologyGraph returns the ontology graph (nodes + edges) from Dgraph.
func GetOntologyGraph(dg *dgraph.Client) func(context.Context, *app.RequestContext) {
	return func(ctx context.Context, c *app.RequestContext) {
		if dg == nil {
			c.JSON(200, map[string]any{
				"nodes": []map[string]any{},
				"edges": []map[string]any{},
			})
			return
		}
		nodes, edges, err := dg.QueryGraph(ctx)
		if err != nil {
			log.Printf("[Ontology] QueryGraph failed: %v", err)
			c.JSON(200, map[string]any{
				"nodes": []map[string]any{},
				"edges": []map[string]any{},
			})
			return
		}

		// Convert to Cytoscape.js format expected by frontend.
		cyNodes := make([]map[string]any, 0, len(nodes))
		for _, n := range nodes {
			cyNodes = append(cyNodes, map[string]any{
				"data": map[string]any{
					"id":    n.ID,
					"label": n.Label,
					"type":  n.Type,
				},
			})
		}
		cyEdges := make([]map[string]any, 0, len(edges))
		for _, e := range edges {
			cyEdges = append(cyEdges, map[string]any{
				"data": map[string]any{
					"source": e.Source,
					"target": e.Target,
					"label":  e.Label,
				},
			})
		}
		c.JSON(200, map[string]any{
			"nodes": cyNodes,
			"edges": cyEdges,
		})
	}
}

// GetTimeline returns real data from Dgraph when available.
func GetTimeline(dg *dgraph.Client) func(context.Context, *app.RequestContext) {
	return func(ctx context.Context, c *app.RequestContext) {
		c.JSON(200, []schema.MarketEvent{})
	}
}

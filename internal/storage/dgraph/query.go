package dgraph

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/competify-ai/competify-backend/internal/schema"
	"github.com/dgraph-io/dgo/v240/protos/api"
)

// UpsertCompetitor inserts a Competitor or updates an existing one by company_name.
// Returns the entity's UID after the operation. Idempotent on company_name.
func (c *Client) UpsertCompetitor(ctx context.Context, comp *schema.Competitor) (string, error) {
	if comp.CompanyName == "" {
		return "", fmt.Errorf("dgraph.UpsertCompetitor: company_name required")
	}

	// Build payload with Dgraph type tag (dgraph.type) and namespaced predicates.
	payload := map[string]any{
		"uid":                        "uid(c)",
		"dgraph.type":                "Competitor",
		"Competitor.company_name":    comp.CompanyName,
		"Competitor.website":         comp.Website,
		"Competitor.funding_stage":   comp.FundingStage,
		"Competitor.team_size":       comp.TeamSize,
		"Competitor.threat_level":    comp.ThreatLevel,
		"Competitor.headquarters":    comp.Headquarters,
	}
	if comp.FoundedDate != nil {
		payload["Competitor.founded_date"] = comp.FoundedDate
	}

	setJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("dgraph.UpsertCompetitor: marshal: %w", err)
	}

	query := fmt.Sprintf(`{
		c as var(func: eq(Competitor.company_name, %q))
	}`, comp.CompanyName)

	req := &api.Request{
		Query:     query,
		Mutations: []*api.Mutation{{SetJson: setJSON}},
		CommitNow: true,
	}
	resp, err := c.dg.NewTxn().Do(ctx, req)
	if err != nil {
		return "", fmt.Errorf("dgraph.UpsertCompetitor: do: %w", err)
	}

	// Newly created entity → resp.Uids["uid(c)"]. Existing → resp.Uids is empty.
	if uid, ok := resp.Uids["uid(c)"]; ok {
		return uid, nil
	}
	// Re-query to get the existing UID.
	return c.queryCompetitorUID(ctx, comp.CompanyName)
}

// QueryCompetitorByName fetches a Competitor by company_name. Returns nil if not found.
func (c *Client) QueryCompetitorByName(ctx context.Context, name string) (*schema.Competitor, error) {
	q := fmt.Sprintf(`{
		q(func: eq(Competitor.company_name, %q)) {
			uid
			company_name:     Competitor.company_name
			website:          Competitor.website
			funding_stage:    Competitor.funding_stage
			team_size:        Competitor.team_size
			threat_level:     Competitor.threat_level
			headquarters:     Competitor.headquarters
			founded_date:     Competitor.founded_date
		}
	}`, name)

	resp, err := c.dg.NewReadOnlyTxn().Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("dgraph.QueryCompetitorByName: %w", err)
	}

	var result struct {
		Q []struct {
			UID          string  `json:"uid"`
			CompanyName  string  `json:"company_name"`
			Website      string  `json:"website"`
			FundingStage string  `json:"funding_stage"`
			TeamSize     int     `json:"team_size"`
			ThreatLevel  int     `json:"threat_level"`
			Headquarters string  `json:"headquarters"`
			FoundedDate  *string `json:"founded_date"`
		} `json:"q"`
	}
	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return nil, fmt.Errorf("dgraph.QueryCompetitorByName: unmarshal: %w", err)
	}
	if len(result.Q) == 0 {
		return nil, nil
	}
	r := result.Q[0]
	return &schema.Competitor{
		UID:          r.UID,
		CompanyName:  r.CompanyName,
		Website:      r.Website,
		FundingStage: r.FundingStage,
		TeamSize:     r.TeamSize,
		ThreatLevel:  r.ThreatLevel,
		Headquarters: r.Headquarters,
	}, nil
}

func (c *Client) queryCompetitorUID(ctx context.Context, name string) (string, error) {
	q := fmt.Sprintf(`{ q(func: eq(Competitor.company_name, %q)) { uid } }`, name)
	resp, err := c.dg.NewReadOnlyTxn().Query(ctx, q)
	if err != nil {
		return "", fmt.Errorf("dgraph.queryCompetitorUID: %w", err)
	}
	var result struct {
		Q []struct {
			UID string `json:"uid"`
		} `json:"q"`
	}
	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return "", fmt.Errorf("dgraph.queryCompetitorUID: unmarshal: %w", err)
	}
	if len(result.Q) == 0 {
		return "", fmt.Errorf("dgraph.queryCompetitorUID: not found")
	}
	return result.Q[0].UID, nil
}

// AddFeature creates a Feature node linked to the competitor's first Product.
// Phase 5 stub: creates the Feature and connects it via Product.offers_feature.
func (c *Client) AddFeature(ctx context.Context, competitorName, featureName string) error {
	// Find competitor UID and its first product UID.
	q := fmt.Sprintf(`{
		c as var(func: eq(Competitor.company_name, %q))
		p as var(func: uid(c)) { develops { uid } }
	}`, competitorName)

	feat := map[string]any{
		"uid":                   "_:feature",
		"dgraph.type":           "Feature",
		"Feature.feature_name":  featureName,
		"Feature.availability":  true,
		"Feature.maturity":      1,
		"Feature.category":      "new",
		"Feature.last_updated":  time.Now().UTC().Format(time.RFC3339),
		"Feature.part_of":       map[string]string{"uid": "uid(p)"},
	}
	setJSON, _ := json.Marshal(feat)

	req := &api.Request{
		Query:     q,
		Mutations: []*api.Mutation{{SetJson: setJSON}},
		CommitNow: true,
	}
	if _, err := c.dg.NewTxn().Do(ctx, req); err != nil {
		return fmt.Errorf("dgraph.AddFeature: %w", err)
	}
	return nil
}

// UpdatePricing creates or updates a PricingTier for the competitor.
// Phase 5 stub: adds a new PricingTier node.
func (c *Client) UpdatePricing(ctx context.Context, competitorName, payload string) error {
	q := fmt.Sprintf(`{
		c as var(func: eq(Competitor.company_name, %q))
		p as var(func: uid(c)) { develops { uid } }
	}`, competitorName)

	pricing := map[string]any{
		"uid":                     "_:pricing",
		"dgraph.type":             "PricingTier",
		"PricingTier.tier_name":   payload,
		"PricingTier.price":       0.0,
		"PricingTier.currency":    "USD",
		"PricingTier.last_updated": time.Now().UTC().Format(time.RFC3339),
	}
	setJSON, _ := json.Marshal(pricing)

	req := &api.Request{
		Query:     q,
		Mutations: []*api.Mutation{{SetJson: setJSON}},
		CommitNow: true,
	}
	if _, err := c.dg.NewTxn().Do(ctx, req); err != nil {
		return fmt.Errorf("dgraph.UpdatePricing: %w", err)
	}
	return nil
}

// RecordFunding updates the competitor's funding_stage and creates a MarketEvent.
// Phase 5 stub: updates funding_stage only.
func (c *Client) RecordFunding(ctx context.Context, competitorName, payload string) error {
	q := fmt.Sprintf(`{ c as var(func: eq(Competitor.company_name, %q)) }`, competitorName)

	update := map[string]any{
		"uid":                     "uid(c)",
		"Competitor.funding_stage": payload,
	}
	setJSON, _ := json.Marshal(update)

	req := &api.Request{
		Query:     q,
		Mutations: []*api.Mutation{{SetJson: setJSON}},
		CommitNow: true,
	}
	if _, err := c.dg.NewTxn().Do(ctx, req); err != nil {
		return fmt.Errorf("dgraph.RecordFunding: %w", err)
	}
	return nil
}

// QueryAllCompetitors returns every Competitor node in Dgraph.
func (c *Client) QueryAllCompetitors(ctx context.Context) ([]schema.Competitor, error) {
	q := `{
		q(func: type(Competitor)) {
			uid
			company_name:     Competitor.company_name
			website:          Competitor.website
			funding_stage:    Competitor.funding_stage
			team_size:        Competitor.team_size
			threat_level:     Competitor.threat_level
			headquarters:     Competitor.headquarters
			founded_date:     Competitor.founded_date
		}
	}`

	resp, err := c.dg.NewReadOnlyTxn().Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("dgraph.QueryAllCompetitors: %w", err)
	}

	var result struct {
		Q []struct {
			UID          string  `json:"uid"`
			CompanyName  string  `json:"company_name"`
			Website      string  `json:"website"`
			FundingStage string  `json:"funding_stage"`
			TeamSize     int     `json:"team_size"`
			ThreatLevel  int     `json:"threat_level"`
			Headquarters string  `json:"headquarters"`
			FoundedDate  *string `json:"founded_date"`
		} `json:"q"`
	}
	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return nil, fmt.Errorf("dgraph.QueryAllCompetitors: unmarshal: %w", err)
	}

	comps := make([]schema.Competitor, 0, len(result.Q))
	for _, r := range result.Q {
		comps = append(comps, schema.Competitor{
			UID:          r.UID,
			CompanyName:  r.CompanyName,
			Website:      r.Website,
			FundingStage: r.FundingStage,
			TeamSize:     r.TeamSize,
			ThreatLevel:  r.ThreatLevel,
			Headquarters: r.Headquarters,
		})
	}
	return comps, nil
}

// GraphNode is a Cytoscape-style node returned by QueryGraph.
type GraphNode struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"type"`
}

// GraphEdge is a Cytoscape-style edge returned by QueryGraph.
type GraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label,omitempty"`
}

// QueryGraph returns all Competitor nodes and their competes_with relationships.
// Returns empty slices when Dgraph has no data.
func (c *Client) QueryGraph(ctx context.Context) ([]GraphNode, []GraphEdge, error) {
	q := `{
		competitors(func: type(Competitor)) {
			uid
			company_name: Competitor.company_name
			competes_with { uid }
		}
	}`

	resp, err := c.dg.NewReadOnlyTxn().Query(ctx, q)
	if err != nil {
		return nil, nil, fmt.Errorf("dgraph.QueryGraph: %w", err)
	}

	var result struct {
		Competitors []struct {
			UID          string `json:"uid"`
			CompanyName  string `json:"company_name"`
			CompetesWith []struct {
				UID string `json:"uid"`
			} `json:"competes_with"`
		} `json:"competitors"`
	}
	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return nil, nil, fmt.Errorf("dgraph.QueryGraph: unmarshal: %w", err)
	}

	nodeMap := make(map[string]GraphNode, len(result.Competitors))
	var edges []GraphEdge

	for _, c := range result.Competitors {
		nodeMap[c.UID] = GraphNode{
			ID:    c.UID,
			Label: c.CompanyName,
			Type:  "competitor",
		}
		for _, rel := range c.CompetesWith {
			edges = append(edges, GraphEdge{
				Source: c.UID,
				Target: rel.UID,
				Label:  "competes_with",
			})
		}
	}

	nodes := make([]GraphNode, 0, len(nodeMap))
	for _, n := range nodeMap {
		nodes = append(nodes, n)
	}
	return nodes, edges, nil
}

package dgraph

import (
	"context"
	"encoding/json"
	"fmt"

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

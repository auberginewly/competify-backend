package viking

import (
	"context"
	"fmt"
)

// SearchByLevel wraps Find with a specific L0/L1/L2 filter.
// L0 = raw data, L1 = semantic chunks, L2 = summaries / embeddings.
func (c *Client) SearchByLevel(ctx context.Context, query, path, level string, topK int) ([]FindResult, error) {
	if level != "L0" && level != "L1" && level != "L2" && level != "all" {
		return nil, fmt.Errorf("viking.SearchByLevel: invalid level %q", level)
	}
	return c.Find(ctx, &FindRequest{
		Query: query,
		Path:  path,
		Level: level,
		TopK:  topK,
	})
}

// BatchGrep runs multiple Grep queries concurrently and merges deduplicated results.
func (c *Client) BatchGrep(ctx context.Context, patterns []string, path string) ([]FindResult, error) {
	results := make([]FindResult, 0)
	seen := make(map[string]struct{})
	for _, p := range patterns {
		hits, err := c.Grep(ctx, p, path)
		if err != nil {
			return nil, fmt.Errorf("viking.BatchGrep pattern %q: %w", p, err)
		}
		for _, h := range hits {
			if _, ok := seen[h.URI]; ok {
				continue
			}
			seen[h.URI] = struct{}{}
			results = append(results, h)
		}
	}
	return results, nil
}

// generateOppositeTerms creates inverse search phrases for Devil's Advocate.
// Used when Viking Grep finds no contradictory evidence.
func generateOppositeTerms(payload string) []string {
	terms := []string{
		"not " + payload,
		"discontinued " + payload,
		"deprecated " + payload,
	}
	return terms
}

// SearchForContradictions runs Viking Grep with reverse keywords.
// Returns non-empty results when contradictory evidence is found.
func (c *Client) SearchForContradictions(ctx context.Context, payload, searchPath string) ([]FindResult, error) {
	patterns := generateOppositeTerms(payload)
	return c.BatchGrep(ctx, patterns, searchPath)
}

// Package reviewer implements multi-agent cross validation with adversarial debate.
package reviewer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/competify-ai/competify-backend/internal/agent"
	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
	"github.com/competify-ai/competify-backend/internal/storage/viking"
)

// marketingHypeWords are used as a deterministic fallback when Viking Grep
// is unavailable or returns no results.
var marketingHypeWords = []string{
	"best", "leader", "number one", "undisputed", "unparalleled", "revolutionary",
}

// DevilsAdvocate challenges conclusions via 3 strategies:
//  1. Reverse-evidence search via Viking Grep (primary) / marketing-hype fallback.
//  2. Low-confidence detection (Score < 0.75).
//  3. Single-source warning (len(SourceURIs) < 2).
type DevilsAdvocate struct {
	agent.BaseAgent
	MaxRounds    int
	VikingClient *viking.Client
}

// NewDevilsAdvocate creates a DevilsAdvocate with optional audit chain and Viking client.
func NewDevilsAdvocate(vc *viking.Client, auditChain ...*provenance.AuditChain) *DevilsAdvocate {
	ba := agent.BaseAgent{Role: "devils_advocate"}
	if len(auditChain) > 0 {
		ba.AuditChain = auditChain[0]
	}
	return &DevilsAdvocate{BaseAgent: ba, MaxRounds: 3, VikingClient: vc}
}

func (d *DevilsAdvocate) Name() string { return "devils_advocate" }

// Execute runs the 3 challenge strategies against a single AnalysisResult.
// It returns APPROVE if no contradictions are found, otherwise RETRY_AUTO
// (or REJECT_HUMAN for the most severe cases).
func (d *DevilsAdvocate) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	result, ok := input.(*schema.AnalysisResult)
	if !ok {
		return nil, fmt.Errorf("devils advocate: expected *schema.AnalysisResult, got %T", input)
	}

	conflicts := d.challenge(ctx, result)

	var nextAction string
	var isApproved bool
	if len(conflicts) == 0 {
		nextAction = "APPROVE"
		isApproved = true
	} else if hasHighSeverity(conflicts) {
		nextAction = "REJECT_HUMAN"
		isApproved = false
	} else {
		nextAction = "RETRY_AUTO"
		isApproved = false
	}

	report := &schema.ReviewReport{
		TaskID:       result.TaskID,
		IsApproved:   isApproved,
		Conflicts:    conflicts,
		NextAction:   nextAction,
		ReviewedAt:   time.Now().UTC(),
		ReviewerID:   d.GenerateID(result.TaskID),
		DebateRounds: 1,
	}

	d.RecordAudit(result.TaskID, d.GenerateID(result.TaskID),
		result.Payload, nextAction,
		fmt.Sprintf("DevilsAdvocate found %d conflicts", len(conflicts)), result.Score)
	return report, nil
}

func (d *DevilsAdvocate) HealthCheck(ctx context.Context) error { return nil }

// challenge applies the 3 strategies and returns all detected conflicts.
// Strategy 1 prefers Viking Grep; falls back to marketing-hype detection.
func (d *DevilsAdvocate) challenge(ctx context.Context, result *schema.AnalysisResult) []schema.Conflict {
	var conflicts []schema.Conflict

	// Strategy 1: reverse-evidence via Viking Grep, then deterministic fallback.
	if grepConflicts := d.challengeWithViking(ctx, result); len(grepConflicts) > 0 {
		conflicts = append(conflicts, grepConflicts...)
	} else if hype := findHypeWords(result.Payload); len(hype) > 0 {
		conflicts = append(conflicts, schema.Conflict{
			Dimension:    result.Dimension,
			AgentIDs:     []string{result.AnalyzerID},
			Description:  fmt.Sprintf("Payload contains unverified marketing language: %v", hype),
			Severity:     "medium",
			SuggestedFix: "Replace with objective, third-party benchmarks or citations.",
		})
	}

	// Strategy 2: low confidence threshold.
	if result.Score < 0.75 {
		conflicts = append(conflicts, schema.Conflict{
			Dimension:    result.Dimension,
			AgentIDs:     []string{result.AnalyzerID},
			Description:  fmt.Sprintf("Low confidence score (%.2f < 0.75)", result.Score),
			Severity:     "medium",
			SuggestedFix: "Collect additional evidence or lower confidence threshold.",
		})
	}

	// Strategy 3: insufficient source diversity.
	if len(result.SourceURIs) < 2 {
		conflicts = append(conflicts, schema.Conflict{
			Dimension:    result.Dimension,
			AgentIDs:     []string{result.AnalyzerID},
			Description:  fmt.Sprintf("Only %d source(s) cited; minimum recommended is 2", len(result.SourceURIs)),
			Severity:     "low",
			SuggestedFix: "Collect from additional independent sources.",
		})
	}

	return conflicts
}

// challengeWithViking uses Viking Grep to search for contradictory evidence.
// Returns conflicts when contradictions are found; empty slice when Viking is
// unavailable or returns no hits.
func (d *DevilsAdvocate) challengeWithViking(ctx context.Context, result *schema.AnalysisResult) []schema.Conflict {
	if d.VikingClient == nil {
		return nil
	}

	searchPath := "viking://competify/"
	if len(result.SourceURIs) > 0 {
		searchPath = result.SourceURIs[0]
	}

	hits, err := d.VikingClient.SearchForContradictions(ctx, result.Payload, searchPath)
	if err != nil || len(hits) == 0 {
		return nil
	}

	return []schema.Conflict{{
		Dimension:    result.Dimension,
		AgentIDs:     []string{result.AnalyzerID},
		Description:  fmt.Sprintf("Viking Grep found contradictory evidence: %s", hits[0].Content),
		Severity:     "medium",
		SuggestedFix: fmt.Sprintf("Consider: %s", hits[0].URI),
	}}
}

func findHypeWords(payload string) []string {
	lower := strings.ToLower(payload)
	var found []string
	for _, w := range marketingHypeWords {
		if strings.Contains(lower, w) {
			found = append(found, w)
		}
	}
	return found
}

func hasHighSeverity(conflicts []schema.Conflict) bool {
	for _, c := range conflicts {
		if c.Severity == "high" {
			return true
		}
	}
	return false
}

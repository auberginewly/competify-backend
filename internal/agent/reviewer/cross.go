// Package reviewer implements multi-agent cross validation with adversarial debate.
package reviewer

import (
	"context"
	"fmt"
	"time"

	"github.com/competify-ai/competify-backend/internal/agent"
	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
)

// CrossReviewer aggregates analysis results and runs Devil's Advocate rounds.
type CrossReviewer struct {
	agent.BaseAgent
	Devil     *DevilsAdvocate
	MaxRounds int
}

// NewCrossReviewer creates a CrossReviewer with an injected Devil's Advocate.
func NewCrossReviewer(devil *DevilsAdvocate, auditChain ...*provenance.AuditChain) *CrossReviewer {
	ba := agent.BaseAgent{Role: "cross_reviewer"}
	if len(auditChain) > 0 {
		ba.AuditChain = auditChain[0]
	}
	return &CrossReviewer{BaseAgent: ba, Devil: devil, MaxRounds: 3}
}

func (c *CrossReviewer) Name() string { return "cross_reviewer" }

// Execute accepts one or more *schema.AnalysisResult and returns a ReviewReport.
// It delegates each result to the Devil's Advocate and aggregates conflicts.
func (c *CrossReviewer) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	results, ok := input.([]*schema.AnalysisResult)
	if !ok {
		if m, ok := input.(map[string]*schema.AnalysisResult); ok {
			for _, r := range m {
				results = append(results, r)
			}
		} else {
			single, sok := input.(*schema.AnalysisResult)
			if sok {
				results = []*schema.AnalysisResult{single}
			} else {
				return nil, fmt.Errorf("cross reviewer: expected []*schema.AnalysisResult, map[string]*schema.AnalysisResult, or *schema.AnalysisResult, got %T", input)
			}
		}
	}

	var allConflicts []schema.Conflict
	var taskID string
	if len(results) > 0 {
		taskID = results[0].TaskID
	}

	// Run Devil's Advocate against each analysis result.
	for _, result := range results {
		review, err := c.Devil.Execute(ctx, result)
		if err != nil {
			continue
		}
		report := review.(*schema.ReviewReport)
		if !report.IsApproved {
			allConflicts = append(allConflicts, report.Conflicts...)
		}
	}

	// Determine routing based on aggregated conflicts.
	var nextAction string
	var isApproved bool
	if len(allConflicts) == 0 {
		nextAction = "APPROVE"
		isApproved = true
	} else if hasHighSeverity(allConflicts) {
		nextAction = "REJECT_HUMAN"
		isApproved = false
	} else {
		nextAction = "RETRY_AUTO"
		isApproved = false
	}

	report := &schema.ReviewReport{
		TaskID:       taskID,
		IsApproved:   isApproved,
		Conflicts:    allConflicts,
		Analyses:     results,
		NextAction:   nextAction,
		ReviewedAt:   time.Now().UTC(),
		ReviewerID:   c.GenerateID(taskID),
		DebateRounds: 1,
	}

	c.RecordAudit(taskID, c.GenerateID(taskID),
		fmt.Sprintf("%d results", len(results)), nextAction,
		fmt.Sprintf("CrossReviewer aggregated %d conflicts", len(allConflicts)), 0.90)
	return report, nil
}

func (c *CrossReviewer) HealthCheck(ctx context.Context) error { return nil }

package dag

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/competify-ai/competify-backend/internal/schema"
)

// reviewCondition routes Cross-Reviewer output to the next node.
// Three branches: APPROVE → writer / RETRY_AUTO → retry / REJECT_HUMAN → human_intervention.
func reviewCondition(ctx context.Context, report *schema.ReviewReport) (string, error) {
	if report == nil {
		return "writer", fmt.Errorf("dag.reviewCondition: nil report")
	}
	switch report.NextAction {
	case "APPROVE":
		return "writer", nil
	case "RETRY_AUTO":
		return "retry", nil
	case "REJECT_HUMAN":
		return "human_intervention", nil
	default:
		return "writer", fmt.Errorf("dag.reviewCondition: unknown next_action %q", report.NextAction)
	}
}

// NewReviewBranch builds the GraphBranch used after cross_reviewer.
func NewReviewBranch() *compose.GraphBranch {
	return compose.NewGraphBranch(reviewCondition, map[string]bool{
		"writer":             true,
		"retry":              true,
		"human_intervention": true,
	})
}

// AwaitHumanIntervention is the human-in-the-loop node.
// Phase 4 stub: immediately returns the report so the pipeline can continue in tests.
func AwaitHumanIntervention(ctx context.Context, report *schema.ReviewReport) (*schema.ReviewReport, error) {
	// TODO Phase 7+: WebSocket notify + Redis pub/sub wait.
	report.NextAction = "APPROVE"
	return report, nil
}

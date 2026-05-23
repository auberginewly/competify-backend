package dag

import (
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/competify-ai/competify-backend/internal/schema"
)

// BuildCleanerWorkflow creates a field-level Workflow for deduplication and normalization.
// Currently it acts as a pass-through placeholder; fine-grained SimHash / normalize nodes
// will be added when the internal complexity justifies workflow-level decomposition.
func BuildCleanerWorkflow() (compose.Runnable[map[string]*schema.RawDataPack, *schema.NormalizedDataset], error) {
	wf := compose.NewWorkflow[map[string]*schema.RawDataPack, *schema.NormalizedDataset]()
	return wf.Compile(context.Background())
}

// BuildReviewerWorkflow creates a field-level Workflow for cross-review aggregation.
// Currently it acts as a pass-through placeholder; conflict-detect / severity-score nodes
// will be added when the review logic grows beyond the CrossReviewer agent.
func BuildReviewerWorkflow() (compose.Runnable[map[string]*schema.AnalysisResult, *schema.ReviewReport], error) {
	wf := compose.NewWorkflow[map[string]*schema.AnalysisResult, *schema.ReviewReport]()
	return wf.Compile(context.Background())
}

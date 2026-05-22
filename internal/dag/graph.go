// Package dag wires Agents into an Eino Graph.
package dag

import (
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/competify-ai/competify-backend/internal/agent"
	"github.com/competify-ai/competify-backend/internal/schema"
)

// BuildCompetifyGraph assembles the full agent pipeline with adversarial review branch.
//
// Pipeline:
//   orchestrator → collector → cleaner → analyzer → cross_reviewer → [Branch]
//     APPROVE       → writer → final_reviewer → END
//     RETRY_AUTO    → retry  → writer → final_reviewer → END
//     REJECT_HUMAN  → human_intervention → writer → final_reviewer → END
func BuildCompetifyGraph(
	orchestrator agent.Agent,
	collector agent.Agent,
	cleaner agent.Agent,
	analyzer agent.Agent,
	crossReviewer agent.Agent,
	writer agent.Agent,
	finalReviewer agent.Agent,
) (compose.Runnable[schema.UserQuery, schema.FinalReviewOutput], error) {
	graph := compose.NewGraph[schema.UserQuery, schema.FinalReviewOutput]()

	// 1. Orchestrator
	_ = graph.AddLambdaNode("orchestrator", wrapAgent[schema.UserQuery, schema.TaskDAGPlan](orchestrator))
	_ = graph.AddEdge(compose.START, "orchestrator")

	// 2. Collector
	_ = graph.AddLambdaNode("collector", wrapAgent[schema.TaskDAGPlan, schema.RawDataPack](collector))
	_ = graph.AddEdge("orchestrator", "collector")

	// 3. Cleaner
	_ = graph.AddLambdaNode("cleaner", wrapAgent[schema.RawDataPack, schema.NormalizedDataset](cleaner))
	_ = graph.AddEdge("collector", "cleaner")

	// 4. Analyzer
	_ = graph.AddLambdaNode("analyzer", wrapAgent[schema.NormalizedDataset, schema.AnalysisResult](analyzer))
	_ = graph.AddEdge("cleaner", "analyzer")

	// 5. Cross-Reviewer
	_ = graph.AddLambdaNode("cross_reviewer", wrapAgent[schema.AnalysisResult, schema.ReviewReport](crossReviewer))
	_ = graph.AddEdge("analyzer", "cross_reviewer")

	// 6. Branch after cross_reviewer
	_ = graph.AddBranch("cross_reviewer", NewReviewBranch())

	// 7. Writer
	_ = graph.AddLambdaNode("writer", wrapAgent[schema.ReviewReport, schema.DraftReport](writer))

	// 8. Retry stub (pass-through)
	_ = graph.AddLambdaNode("retry", compose.InvokableLambda(func(ctx context.Context, r *schema.ReviewReport) (*schema.ReviewReport, error) {
		return r, nil
	}))

	// 9. Human intervention stub (pass-through)
	_ = graph.AddLambdaNode("human_intervention", compose.InvokableLambda(func(ctx context.Context, r *schema.ReviewReport) (*schema.ReviewReport, error) {
		return AwaitHumanIntervention(ctx, r)
	}))

	// Branch edges
	_ = graph.AddEdge("cross_reviewer", "writer")
	_ = graph.AddEdge("cross_reviewer", "retry")
	_ = graph.AddEdge("cross_reviewer", "human_intervention")
	_ = graph.AddEdge("retry", "writer")
	_ = graph.AddEdge("human_intervention", "writer")

	// 10. Final-Reviewer
	_ = graph.AddLambdaNode("final_reviewer", wrapAgent[schema.DraftReport, schema.FinalReviewOutput](finalReviewer))
	_ = graph.AddEdge("writer", "final_reviewer")
	_ = graph.AddEdge("final_reviewer", compose.END)

	return graph.Compile(context.Background())
}

// wrapAgent adapts the generic Agent interface to a typed Eino Lambda.
func wrapAgent[I, O any](a agent.Agent) *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, in I) (O, error) {
		out, err := a.Execute(ctx, in)
		if err != nil {
			var zero O
			return zero, err
		}
		return out.(O), nil
	})
}

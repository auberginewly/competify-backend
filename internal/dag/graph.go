// Package dag wires Agents into an Eino Graph.
package dag

import (
	"context"
	"reflect"

	"github.com/cloudwego/eino/compose"
	"github.com/competify-ai/competify-backend/internal/agent"
	"github.com/competify-ai/competify-backend/internal/observability"
	"github.com/competify-ai/competify-backend/internal/schema"
)

func init() {
	// Register merge function for *ReviewReport so GraphBranch can fan-in to writer.
	// Only one branch is active at a time, so we just return the first value.
	compose.RegisterValuesMergeFunc(func(vs []*schema.ReviewReport) (*schema.ReviewReport, error) {
		if len(vs) > 0 {
			return vs[0], nil
		}
		return nil, nil
	})
}

// BuildCompetifyGraph assembles the full agent pipeline with adversarial review branch.
//
// Pipeline:
//   orchestrator → [5 Collectors parallel] → cleaner → [4 Analyzers parallel] → cross_reviewer → [Branch]
//     APPROVE       → writer → final_reviewer → END
//     RETRY_AUTO    → retry  → writer → final_reviewer → END
//     REJECT_HUMAN  → human_intervention → writer → final_reviewer → END
func BuildCompetifyGraph(
	orchestrator agent.Agent,
	collectorWeb agent.Agent,
	collectorAPI agent.Agent,
	collectorFin agent.Agent,
	collectorRev agent.Agent,
	collectorSoc agent.Agent,
	cleaner agent.Agent,
	analyzerFeat agent.Agent,
	analyzerPrice agent.Agent,
	analyzerTech agent.Agent,
	analyzerMkt agent.Agent,
	crossReviewer agent.Agent,
	writer agent.Agent,
	finalReviewer agent.Agent,
) (compose.Runnable[schema.UserQuery, *schema.FinalReviewOutput], error) {
	graph := compose.NewGraph[schema.UserQuery, *schema.FinalReviewOutput]()

	// 1. Orchestrator
	_ = graph.AddLambdaNode("orchestrator", wrapAgent[schema.UserQuery, *schema.TaskDAGPlan](orchestrator))
	_ = graph.AddEdge(compose.START, "orchestrator")

	// 2. Collectors (parallel) — each outputs a single-key map so Eino can merge them.
	_ = graph.AddLambdaNode("collector_web", collectorLambda(collectorWeb, "web"))
	_ = graph.AddLambdaNode("collector_api", collectorLambda(collectorAPI, "api"))
	_ = graph.AddLambdaNode("collector_fin", collectorLambda(collectorFin, "fin"))
	_ = graph.AddLambdaNode("collector_rev", collectorLambda(collectorRev, "rev"))
	_ = graph.AddLambdaNode("collector_soc", collectorLambda(collectorSoc, "soc"))
	_ = graph.AddEdge("orchestrator", "collector_web")
	_ = graph.AddEdge("orchestrator", "collector_api")
	_ = graph.AddEdge("orchestrator", "collector_fin")
	_ = graph.AddEdge("orchestrator", "collector_rev")
	_ = graph.AddEdge("orchestrator", "collector_soc")

	// 3. Cleaner (fan-in from all collectors via map merge)
	_ = graph.AddLambdaNode("cleaner", cleanerLambda(cleaner))
	_ = graph.AddEdge("collector_web", "cleaner")
	_ = graph.AddEdge("collector_api", "cleaner")
	_ = graph.AddEdge("collector_fin", "cleaner")
	_ = graph.AddEdge("collector_rev", "cleaner")
	_ = graph.AddEdge("collector_soc", "cleaner")

	// 4. Analyzers (parallel)
	_ = graph.AddLambdaNode("analyzer_feat", analyzerLambda(analyzerFeat, "feat"))
	_ = graph.AddLambdaNode("analyzer_price", analyzerLambda(analyzerPrice, "price"))
	_ = graph.AddLambdaNode("analyzer_tech", analyzerLambda(analyzerTech, "tech"))
	_ = graph.AddLambdaNode("analyzer_mkt", analyzerLambda(analyzerMkt, "mkt"))
	_ = graph.AddEdge("cleaner", "analyzer_feat")
	_ = graph.AddEdge("cleaner", "analyzer_price")
	_ = graph.AddEdge("cleaner", "analyzer_tech")
	_ = graph.AddEdge("cleaner", "analyzer_mkt")

	// 5. Cross-Reviewer (fan-in from all analyzers via map merge)
	_ = graph.AddLambdaNode("cross_reviewer", crossReviewerLambda(crossReviewer))
	_ = graph.AddEdge("analyzer_feat", "cross_reviewer")
	_ = graph.AddEdge("analyzer_price", "cross_reviewer")
	_ = graph.AddEdge("analyzer_tech", "cross_reviewer")
	_ = graph.AddEdge("analyzer_mkt", "cross_reviewer")

	// 6. Writer, Retry, HumanIntervention — must be added before Branch.
	_ = graph.AddLambdaNode("writer", wrapAgent[*schema.ReviewReport, *schema.DraftReport](writer))
	_ = graph.AddLambdaNode("retry", compose.InvokableLambda(func(ctx context.Context, r *schema.ReviewReport) (*schema.ReviewReport, error) {
		return r, nil
	}))
	_ = graph.AddLambdaNode("human_intervention", compose.InvokableLambda(func(ctx context.Context, r *schema.ReviewReport) (*schema.ReviewReport, error) {
		ptr, err := AwaitHumanIntervention(ctx, r)
		if err != nil {
			return nil, err
		}
		return ptr, nil
	}))

	// 7. Branch after cross_reviewer
	_ = graph.AddBranch("cross_reviewer", NewReviewBranch())

	// Branch edges
	_ = graph.AddEdge("cross_reviewer", "writer")
	_ = graph.AddEdge("cross_reviewer", "retry")
	_ = graph.AddEdge("cross_reviewer", "human_intervention")
	_ = graph.AddEdge("retry", "writer")
	_ = graph.AddEdge("human_intervention", "writer")

	// 10. Final-Reviewer
	_ = graph.AddLambdaNode("final_reviewer", wrapAgent[*schema.DraftReport, *schema.FinalReviewOutput](finalReviewer))
	_ = graph.AddEdge("writer", "final_reviewer")
	_ = graph.AddEdge("final_reviewer", compose.END)

	return graph.Compile(context.Background())
}

// collectorLambda wraps a collector agent to output a single-key map for Eino map-merge.
func collectorLambda(a agent.Agent, key string) *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, plan *schema.TaskDAGPlan) (map[string]*schema.RawDataPack, error) {
		var pack *schema.RawDataPack
		err := observability.TraceAgentExecution(ctx, a.Name(), plan.TaskID, func(ctx context.Context) error {
			out, err := a.Execute(ctx, plan)
			if err != nil {
				return err
			}
			pack = out.(*schema.RawDataPack)
			return nil
		})
		if err != nil {
			return nil, err
		}
		return map[string]*schema.RawDataPack{key: pack}, nil
	})
}

// cleanerLambda wraps the cleaner agent to accept a merged map from all collectors.
func cleanerLambda(a agent.Agent) *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, packs map[string]*schema.RawDataPack) (*schema.NormalizedDataset, error) {
		taskID := anyTaskID(packs)
		var ds *schema.NormalizedDataset
		err := observability.TraceAgentExecution(ctx, a.Name(), taskID, func(ctx context.Context) error {
			out, err := a.Execute(ctx, packs)
			if err != nil {
				return err
			}
			ds = out.(*schema.NormalizedDataset)
			return nil
		})
		return ds, err
	})
}

// analyzerLambda wraps an analyzer agent to output a single-key map for Eino map-merge.
func analyzerLambda(a agent.Agent, key string) *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, ds *schema.NormalizedDataset) (map[string]*schema.AnalysisResult, error) {
		var result *schema.AnalysisResult
		err := observability.TraceAgentExecution(ctx, a.Name(), ds.TaskID, func(ctx context.Context) error {
			out, err := a.Execute(ctx, ds)
			if err != nil {
				return err
			}
			result = out.(*schema.AnalysisResult)
			return nil
		})
		if err != nil {
			return nil, err
		}
		return map[string]*schema.AnalysisResult{key: result}, nil
	})
}

// crossReviewerLambda wraps the cross-reviewer agent to accept a merged map from all analyzers.
func crossReviewerLambda(a agent.Agent) *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, results map[string]*schema.AnalysisResult) (*schema.ReviewReport, error) {
		taskID := anyAnalysisTaskID(results)
		var report *schema.ReviewReport
		err := observability.TraceAgentExecution(ctx, a.Name(), taskID, func(ctx context.Context) error {
			out, err := a.Execute(ctx, results)
			if err != nil {
				return err
			}
			report = out.(*schema.ReviewReport)
			return nil
		})
		return report, err
	})
}

// anyTaskID returns the task ID from the first pack in the map (order undefined).
func anyTaskID(packs map[string]*schema.RawDataPack) string {
	for _, p := range packs {
		if p != nil {
			return p.TaskID
		}
	}
	return "unknown"
}

// anyAnalysisTaskID returns the task ID from the first AnalysisResult in the map.
func anyAnalysisTaskID(results map[string]*schema.AnalysisResult) string {
	for _, r := range results {
		if r != nil {
			return r.TaskID
		}
	}
	return "unknown"
}

// wrapAgent adapts the generic Agent interface to a typed Eino Lambda.
// If I is a pointer type, it is passed directly to Execute.
// If I is a value type, &I is passed so agents can assert *schema.XXX.
// If out is a pointer but O is a value type, it is automatically dereferenced.
func wrapAgent[I, O any](a agent.Agent) *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, in I) (O, error) {
		var input interface{}
		val := reflect.ValueOf(in)
		if val.Kind() == reflect.Ptr {
			input = in
		} else {
			input = &in
		}
		out, err := a.Execute(ctx, input)
		if err != nil {
			var zero O
			return zero, err
		}

		// Auto-dereference if out is pointer but O is value type.
		outVal := reflect.ValueOf(out)
		if outVal.Kind() == reflect.Ptr {
			var zero O
			zeroVal := reflect.ValueOf(&zero).Elem()
			if zeroVal.Kind() != reflect.Ptr {
				if !outVal.IsNil() {
					out = outVal.Elem().Interface()
				}
			}
		}
		return out.(O), nil
	})
}

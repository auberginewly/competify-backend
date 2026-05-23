// Package dag wires Agents into an Eino Graph.
package dag

import (
	"context"
	"reflect"

	"github.com/cloudwego/eino/compose"
	nats "github.com/nats-io/nats.go"
	"github.com/competify-ai/competify-backend/internal/agent"
	"github.com/competify-ai/competify-backend/internal/messaging"
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
	nc *nats.Conn,
) (compose.Runnable[schema.UserQuery, *schema.FinalReviewOutput], error) {
	graph := compose.NewGraph[schema.UserQuery, *schema.FinalReviewOutput]()

	// 1. Orchestrator
	_ = graph.AddLambdaNode("orchestrator", orchestratorLambda(orchestrator, nc))
	_ = graph.AddEdge(compose.START, "orchestrator")

	// 2. Collectors (parallel) — each outputs a single-key map so Eino can merge them.
	_ = graph.AddLambdaNode("collector_web", collectorLambda(collectorWeb, "web", nc))
	_ = graph.AddLambdaNode("collector_api", collectorLambda(collectorAPI, "api", nc))
	_ = graph.AddLambdaNode("collector_fin", collectorLambda(collectorFin, "fin", nc))
	_ = graph.AddLambdaNode("collector_rev", collectorLambda(collectorRev, "rev", nc))
	_ = graph.AddLambdaNode("collector_soc", collectorLambda(collectorSoc, "soc", nc))
	_ = graph.AddEdge("orchestrator", "collector_web")
	_ = graph.AddEdge("orchestrator", "collector_api")
	_ = graph.AddEdge("orchestrator", "collector_fin")
	_ = graph.AddEdge("orchestrator", "collector_rev")
	_ = graph.AddEdge("orchestrator", "collector_soc")

	// 3. Cleaner (fan-in from all collectors via map merge)
	_ = graph.AddLambdaNode("cleaner", cleanerLambda(cleaner, nc))
	_ = graph.AddEdge("collector_web", "cleaner")
	_ = graph.AddEdge("collector_api", "cleaner")
	_ = graph.AddEdge("collector_fin", "cleaner")
	_ = graph.AddEdge("collector_rev", "cleaner")
	_ = graph.AddEdge("collector_soc", "cleaner")

	// 4. Analyzers (parallel)
	_ = graph.AddLambdaNode("analyzer_feat", analyzerLambda(analyzerFeat, "feat", nc))
	_ = graph.AddLambdaNode("analyzer_price", analyzerLambda(analyzerPrice, "price", nc))
	_ = graph.AddLambdaNode("analyzer_tech", analyzerLambda(analyzerTech, "tech", nc))
	_ = graph.AddLambdaNode("analyzer_mkt", analyzerLambda(analyzerMkt, "mkt", nc))
	_ = graph.AddEdge("cleaner", "analyzer_feat")
	_ = graph.AddEdge("cleaner", "analyzer_price")
	_ = graph.AddEdge("cleaner", "analyzer_tech")
	_ = graph.AddEdge("cleaner", "analyzer_mkt")

	// 5. Cross-Reviewer (fan-in from all analyzers via map merge)
	_ = graph.AddLambdaNode("cross_reviewer", crossReviewerLambda(crossReviewer, nc))
	_ = graph.AddEdge("analyzer_feat", "cross_reviewer")
	_ = graph.AddEdge("analyzer_price", "cross_reviewer")
	_ = graph.AddEdge("analyzer_tech", "cross_reviewer")
	_ = graph.AddEdge("analyzer_mkt", "cross_reviewer")

	// 6. Writer, Retry, HumanIntervention — must be added before Branch.
	_ = graph.AddLambdaNode("writer", writerLambda(writer, nc))
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
	_ = graph.AddLambdaNode("final_reviewer", finalReviewerLambda(finalReviewer, nc))
	_ = graph.AddEdge("writer", "final_reviewer")
	_ = graph.AddEdge("final_reviewer", compose.END)

	return graph.Compile(context.Background())
}

func publishDAG(nc *nats.Conn, taskID, name, status, log string) {
	if nc == nil || taskID == "" {
		return
	}
	_ = messaging.PublishDAGEvent(nc, taskID, messaging.NewDAGEvent(name, status, 1.0, log))
}

func orchestratorLambda(a agent.Agent, nc *nats.Conn) *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, query schema.UserQuery) (*schema.TaskDAGPlan, error) {
		out, err := a.Execute(ctx, &query)
		if err != nil {
			return nil, err
		}
		plan := out.(*schema.TaskDAGPlan)
		publishDAG(nc, plan.TaskID, a.Name(), "done", a.Name()+" 完成")
		return plan, nil
	})
}

// collectorLambda wraps a collector agent to output a single-key map for Eino map-merge.
func collectorLambda(a agent.Agent, key string, nc *nats.Conn) *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, plan *schema.TaskDAGPlan) (map[string]*schema.RawDataPack, error) {
		publishDAG(nc, plan.TaskID, a.Name(), "running", a.Name()+" 开始")
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
			publishDAG(nc, plan.TaskID, a.Name(), "error", err.Error())
			return nil, err
		}
		publishDAG(nc, plan.TaskID, a.Name(), "done", a.Name()+" 完成")
		return map[string]*schema.RawDataPack{key: pack}, nil
	})
}

// cleanerLambda wraps the cleaner agent to accept a merged map from all collectors.
func cleanerLambda(a agent.Agent, nc *nats.Conn) *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, packs map[string]*schema.RawDataPack) (*schema.NormalizedDataset, error) {
		taskID := anyTaskID(packs)
		publishDAG(nc, taskID, a.Name(), "running", a.Name()+" 开始")
		var ds *schema.NormalizedDataset
		err := observability.TraceAgentExecution(ctx, a.Name(), taskID, func(ctx context.Context) error {
			out, err := a.Execute(ctx, packs)
			if err != nil {
				return err
			}
			ds = out.(*schema.NormalizedDataset)
			return nil
		})
		if err != nil {
			publishDAG(nc, taskID, a.Name(), "error", err.Error())
			return nil, err
		}
		publishDAG(nc, taskID, a.Name(), "done", a.Name()+" 完成")
		return ds, nil
	})
}

// analyzerLambda wraps an analyzer agent to output a single-key map for Eino map-merge.
func analyzerLambda(a agent.Agent, key string, nc *nats.Conn) *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, ds *schema.NormalizedDataset) (map[string]*schema.AnalysisResult, error) {
		publishDAG(nc, ds.TaskID, a.Name(), "running", a.Name()+" 开始")
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
			publishDAG(nc, ds.TaskID, a.Name(), "error", err.Error())
			return nil, err
		}
		publishDAG(nc, ds.TaskID, a.Name(), "done", a.Name()+" 完成")
		return map[string]*schema.AnalysisResult{key: result}, nil
	})
}

// crossReviewerLambda wraps the cross-reviewer agent to accept a merged map from all analyzers.
func crossReviewerLambda(a agent.Agent, nc *nats.Conn) *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, results map[string]*schema.AnalysisResult) (*schema.ReviewReport, error) {
		taskID := anyAnalysisTaskID(results)
		publishDAG(nc, taskID, a.Name(), "running", a.Name()+" 开始")
		var report *schema.ReviewReport
		err := observability.TraceAgentExecution(ctx, a.Name(), taskID, func(ctx context.Context) error {
			out, err := a.Execute(ctx, results)
			if err != nil {
				return err
			}
			report = out.(*schema.ReviewReport)
			return nil
		})
		if err != nil {
			publishDAG(nc, taskID, a.Name(), "error", err.Error())
			return nil, err
		}
		publishDAG(nc, taskID, a.Name(), "done", a.Name()+" 完成")
		return report, nil
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

func writerLambda(a agent.Agent, nc *nats.Conn) *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, r *schema.ReviewReport) (*schema.DraftReport, error) {
		publishDAG(nc, r.TaskID, a.Name(), "running", "Writer 正在生成报告")
		out, err := a.Execute(ctx, r)
		if err != nil {
			publishDAG(nc, r.TaskID, a.Name(), "error", err.Error())
			return nil, err
		}
		draft := out.(*schema.DraftReport)
		publishDAG(nc, r.TaskID, a.Name(), "done", "报告生成完成")
		return draft, nil
	})
}

func finalReviewerLambda(a agent.Agent, nc *nats.Conn) *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, draft *schema.DraftReport) (*schema.FinalReviewOutput, error) {
		publishDAG(nc, draft.TaskID, a.Name(), "running", "FinalReviewer 终审中")
		out, err := a.Execute(ctx, draft)
		if err != nil {
			publishDAG(nc, draft.TaskID, a.Name(), "error", err.Error())
			return nil, err
		}
		output := out.(*schema.FinalReviewOutput)
		publishDAG(nc, draft.TaskID, a.Name(), "done", "终审完成")
		return output, nil
	})
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

package dag

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/competify-ai/competify-backend/internal/agent"
	"github.com/competify-ai/competify-backend/internal/agent/analyzer"
	"github.com/competify-ai/competify-backend/internal/agent/collector"
	"github.com/competify-ai/competify-backend/internal/agent/reviewer"
	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
)

// AgentSet holds all initialized agents for pipeline construction.
type AgentSet struct {
	Orchestrator  agent.Agent
	Collector     agent.Agent
	Cleaner       agent.Agent
	Analyzer      agent.Agent
	CrossReviewer agent.Agent
	Writer        agent.Agent
	FinalReviewer agent.Agent
}

// BuildAllAgents initializes every agent role with a shared audit chain.
func BuildAllAgents(auditChain *provenance.AuditChain) (*AgentSet, error) {
	// Devil's Advocate for CrossReviewer (Viking client can be nil in Phase 5).
	devil := reviewer.NewDevilsAdvocate(nil, auditChain)

	set := &AgentSet{
		Orchestrator:  agent.NewOrchestrator(auditChain),
		Collector:     collector.NewWebCollector(auditChain),
		Cleaner:       agent.NewCleaner(auditChain),
		Analyzer:      analyzer.NewFeatureAnalyzer(auditChain),
		CrossReviewer: reviewer.NewCrossReviewer(devil, auditChain),
		Writer:        agent.NewWriter(auditChain),
		FinalReviewer: reviewer.NewFinalReviewer([]byte("competify-secret-key"), auditChain),
	}

	return set, nil
}

// BuildRunner assembles the full Eino Graph from an AgentSet.
func BuildRunner(set *AgentSet) (compose.Runnable[schema.UserQuery, *schema.FinalReviewOutput], error) {
	return BuildCompetifyGraph(
		set.Orchestrator,
		set.Collector,
		set.Cleaner,
		set.Analyzer,
		set.CrossReviewer,
		set.Writer,
		set.FinalReviewer,
	)
}

// ExecuteDAG runs the complete pipeline for a single task.
// It is a convenience wrapper used by the Worker.
func ExecuteDAG(ctx context.Context, runnable compose.Runnable[schema.UserQuery, *schema.FinalReviewOutput], query schema.UserQuery) (*schema.FinalReviewOutput, error) {
	out, err := runnable.Invoke(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("dag.ExecuteDAG: %w", err)
	}
	return out, nil
}

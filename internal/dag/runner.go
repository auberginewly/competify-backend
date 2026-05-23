package dag

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/compose"
	"github.com/competify-ai/competify-backend/internal/agent"
	"github.com/competify-ai/competify-backend/internal/agent/analyzer"
	"github.com/competify-ai/competify-backend/internal/agent/collector"
	"github.com/competify-ai/competify-backend/internal/agent/reviewer"
	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
	"github.com/competify-ai/competify-backend/internal/storage/viking"
)

// AgentSet holds all initialized agents for pipeline construction.
type AgentSet struct {
	Orchestrator   agent.Agent
	CollectorWeb   agent.Agent
	CollectorAPI   agent.Agent
	CollectorFin   agent.Agent
	CollectorRev   agent.Agent
	CollectorSoc   agent.Agent
	Cleaner        agent.Agent
	AnalyzerFeat   agent.Agent
	AnalyzerPrice  agent.Agent
	AnalyzerTech   agent.Agent
	AnalyzerMkt    agent.Agent
	CrossReviewer  agent.Agent
	Writer         agent.Agent
	FinalReviewer  agent.Agent
}

// BuildAllAgents initializes every agent role with a shared audit chain.
// model may be nil (analyzers fall back to stub); vc may be nil (DevilsAdvocate falls back to heuristics).
// notifyFn may be nil — Collectors call it on data detection for reactive ontology updates.
func BuildAllAgents(model *openai.ChatModel, auditChain *provenance.AuditChain, vc *viking.Client, notifyFn ...agent.EventNotifier) (*AgentSet, error) {
	devil := reviewer.NewDevilsAdvocate(vc, auditChain)

	var notify agent.EventNotifier
	if len(notifyFn) > 0 {
		notify = notifyFn[0]
	}

	webCol := collector.NewWebCollector(auditChain)
	webCol.Notify = notify
	apiCol := collector.NewAPICollector(auditChain)
	apiCol.Notify = notify

	set := &AgentSet{
		Orchestrator:  agent.NewOrchestrator(auditChain),
		CollectorWeb:  webCol,
		CollectorAPI:  apiCol,
		CollectorFin:  collector.NewFinancialCollector(auditChain),
		CollectorRev:  collector.NewReviewCollector(auditChain),
		CollectorSoc:  collector.NewSocialCollector(auditChain),
		Cleaner:       agent.NewCleaner(auditChain),
		AnalyzerFeat:  analyzer.NewFeatureAnalyzer(model, auditChain),
		AnalyzerPrice: analyzer.NewPricingAnalyzer(model, auditChain),
		AnalyzerTech:  analyzer.NewTechAnalyzer(model, auditChain),
		AnalyzerMkt:   analyzer.NewMarketAnalyzer(model, auditChain),
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
		set.CollectorWeb,
		set.CollectorAPI,
		set.CollectorFin,
		set.CollectorRev,
		set.CollectorSoc,
		set.Cleaner,
		set.AnalyzerFeat,
		set.AnalyzerPrice,
		set.AnalyzerTech,
		set.AnalyzerMkt,
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

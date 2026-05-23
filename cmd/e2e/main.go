package main

import (
	"context"
	"fmt"
	"github.com/competify-ai/competify-backend/internal/agent"
	"github.com/competify-ai/competify-backend/internal/agent/analyzer"
	"github.com/competify-ai/competify-backend/internal/agent/collector"
	"github.com/competify-ai/competify-backend/internal/agent/reviewer"
	"github.com/competify-ai/competify-backend/internal/dag"
	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
)

func main() {
	ac := provenance.NewAuditChain()
	
	orch := agent.NewOrchestrator(ac)
	coll := collector.NewWebCollector(ac)
	clean := agent.NewCleaner(ac)
	anal := analyzer.NewFeatureAnalyzer(ac)
	devil := reviewer.NewDevilsAdvocate(nil, ac)
	cross := reviewer.NewCrossReviewer(devil, ac)
	writer := agent.NewWriter(ac)
	final := reviewer.NewFinalReviewer([]byte("test-key"), ac)
	
	runnable, err := dag.BuildCompetifyGraph(orch, coll, clean, anal, cross, writer, final)
	if err != nil {
		fmt.Println("Compile error:", err)
		return
	}
	
	result, err := runnable.Invoke(context.Background(), schema.UserQuery{
		CompetitorName: "Cursor",
		Dimensions:     []string{"feature", "pricing"},
	})
	if err != nil {
		fmt.Println("Invoke error:", err)
		return
	}
	
	fmt.Println("SUCCESS!")
	fmt.Printf("Report Status: %s\n", result.Status)
	fmt.Printf("Signature: %s\n", result.Signature)
	fmt.Printf("Audit logs: %d\n", len(ac.Logs()))
	fmt.Printf("Merkle Root: %s\n", ac.GetMerkleRoot())
}

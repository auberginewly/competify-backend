package main

import (
	"context"
	"fmt"

	"github.com/competify-ai/competify-backend/internal/dag"
	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
)

func main() {
	ac := provenance.NewAuditChain()

	agents, err := dag.BuildAllAgents(nil, ac, nil)
	if err != nil {
		fmt.Println("Build agents error:", err)
		return
	}

	runnable, err := dag.BuildRunner(agents)
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

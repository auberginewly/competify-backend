// Command worker — Agent worker entrypoint (for distributed deployments).
// Phase 5: subscribes to NATS task queue, executes long-running Agent jobs.
package main

import (
	"context"
	"log"
	"os"

	"github.com/competify-ai/competify-backend/internal/dag"
	"github.com/competify-ai/competify-backend/internal/handler"
	"github.com/competify-ai/competify-backend/internal/messaging"
	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
	"github.com/nats-io/nats.go"
)

func main() {
	natsURL := getenv("NATS_ADDR", "nats://localhost:4222")
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("nats connect: %v", err)
	}
	defer nc.Close()

	ctx := context.Background()

	// Initialize Agents + DAG runnable.
	auditChain := provenance.NewAuditChain()
	agents, err := dag.BuildAllAgents(auditChain)
	if err != nil {
		log.Fatalf("build agents: %v", err)
	}
	runnable, err := dag.BuildRunner(agents)
	if err != nil {
		log.Fatalf("build runner: %v", err)
	}

	taskSub, err := messaging.NewTaskSubscriber(nc)
	if err != nil {
		log.Fatalf("task subscriber: %v", err)
	}

	log.Println("[Worker] waiting for tasks...")
	if err := taskSub.SubscribeTask(ctx, func(taskID string, query schema.UserQuery) {
		log.Printf("[Worker] received task %s", taskID)

		// Drive frontend progress bar via NATS while the real DAG runs.
		go handler.SimulateProgress(nc, taskID)

		// Execute the real DAG.
		output, err := dag.ExecuteDAG(ctx, runnable, query)
		if err != nil {
			log.Printf("[Worker] task %s failed: %v", taskID, err)
			return
		}

		log.Printf("[Worker] task %s completed, report length=%d", taskID, len(output.Content))
	}); err != nil {
		log.Fatalf("subscribe task: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

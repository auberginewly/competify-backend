// Command worker — Agent worker entrypoint (for distributed deployments).
// Phase 5: subscribes to NATS task queue, executes long-running Agent jobs.
package main

import (
	"context"
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/competify-ai/competify-backend/internal/dag"
	"github.com/competify-ai/competify-backend/internal/messaging"
	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
	"github.com/competify-ai/competify-backend/internal/storage/viking"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func main() {
	natsURL := getenv("NATS_ADDR", "nats://localhost:4222")
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("nats connect: %v", err)
	}
	defer nc.Close()

	ctx := context.Background()

	// Initialize LLM model (nil is okay — analyzers fall back to stub data).
	model := initChatModel(ctx)

	// Initialize Agents + DAG runnable.
	auditChain := provenance.NewAuditChain()
	vikingClient := initVikingClient()
	agents, err := dag.BuildAllAgents(model, auditChain, vikingClient, nil)
	if err != nil {
		log.Fatalf("build agents: %v", err)
	}
	runnable, err := dag.BuildRunner(agents, nil)
	if err != nil {
		log.Fatalf("build runner: %v", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatalf("jetstream: %v", err)
	}

	taskSub, err := messaging.NewTaskSubscriber(js)
	if err != nil {
		log.Fatalf("task subscriber: %v", err)
	}

	log.Println("[Worker] waiting for tasks...")
	if err := taskSub.SubscribeTask(ctx, func(taskID string, query schema.UserQuery) {
		log.Printf("[Worker] received task %s", taskID)

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

// initChatModel creates the shared OpenAI chat model from environment variables.
// Returns nil if MOCK_LLM is set or API key is missing — analyzers will fall back to stub data.
func initChatModel(ctx context.Context) *openai.ChatModel {
	if os.Getenv("MOCK_LLM") == "true" {
		log.Println("[LLM] MOCK_LLM=true, running in stub mode")
		return nil
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Println("[LLM] OPENAI_API_KEY not set, running in stub mode")
		return nil
	}
	baseURL := getenv("OPENAI_BASE_URL", "https://api.deepseek.com/v1")
	modelName := getenv("OPENAI_MODEL_NAME", "deepseek-chat")

	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   modelName,
	})
	if err != nil {
		log.Printf("[LLM] failed to init chat model: %v, falling back to stub mode", err)
		return nil
	}
	log.Printf("[LLM] initialized model %s via %s", modelName, baseURL)
	return model
}

// initVikingClient creates an OpenViking client from env vars.
// Returns nil when VIKING_URL is not set — DevilsAdvocate falls back to heuristics.
func initVikingClient() *viking.Client {
	url := os.Getenv("VIKING_URL")
	if url == "" {
		return nil
	}
	return viking.NewClient(url, os.Getenv("VIKING_API_KEY"))
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

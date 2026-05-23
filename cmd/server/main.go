// Command server — HTTP API entrypoint.
//
// Phase 5: NATS task queue + embedded Worker goroutine for async DAG execution.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/competify-ai/competify-backend/internal/dag"
	"github.com/competify-ai/competify-backend/internal/handler"
	"github.com/competify-ai/competify-backend/internal/messaging"
	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
	"github.com/competify-ai/competify-backend/internal/storage/dgraph"
	"github.com/competify-ai/competify-backend/internal/storage/memory"
	"github.com/competify-ai/competify-backend/internal/storage/viking"
	"github.com/cloudwego/eino/compose"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

func main() {
	initSchema := flag.Bool("init-schema", false, "Inject Dgraph schema then exit")
	smokeTest := flag.Bool("smoke-test", false, "End-to-end Dgraph upsert+query verification")
	flag.Parse()

	dgraphAddr := getenv("DGRAPH_ADDR", "localhost:9080")

	if *initSchema {
		runInitSchema(dgraphAddr)
		return
	}
	if *smokeTest {
		runSmokeTest(dgraphAddr)
		return
	}

	port := getenv("SERVER_PORT", "8080")
	natsURL := getenv("NATS_ADDR", "nats://localhost:4222")

	ctx := context.Background()

	// 1. Connect to NATS (external or embedded fallback).
	nc, embeddedNS, err := connectNATS(natsURL)
	if err != nil {
		log.Fatalf("nats connect: %v", err)
	}
	defer nc.Close()
	if embeddedNS != nil {
		defer embeddedNS.Shutdown()
	}

	// 2. Initialize in-memory stores (shared between server and worker goroutine).
	taskStore := memory.NewTaskStore()
	reportStore := memory.NewReportStore()

	// 3. Initialize messaging.
	taskPublisher, err := messaging.NewTaskPublisher(nc)
	if err != nil {
		log.Fatalf("task publisher: %v", err)
	}
	if err := taskPublisher.EnsureTaskStream(ctx); err != nil {
		log.Fatalf("ensure task stream: %v", err)
	}

	// 3.5 Initialize LLM model (nil is okay — analyzers fall back to stub data).
	model := initChatModel(ctx)

	// 4. Initialize Agents + DAG runnable.
	auditChain := provenance.NewAuditChain()
	vikingClient := initVikingClient()
	agents, err := dag.BuildAllAgents(model, auditChain, vikingClient)
	if err != nil {
		log.Fatalf("build agents: %v", err)
	}
	runnable, err := dag.BuildRunner(agents)
	if err != nil {
		log.Fatalf("build runner: %v", err)
	}

	// 5. Start Worker goroutine (same process, shares memory stores).
	go runWorker(ctx, nc, taskStore, reportStore, auditChain, agents, runnable)

	// 6. Register Hertz routes with injected dependencies.
	h := server.Default(server.WithHostPorts(":" + port))
	deps := &handler.Deps{
		TaskStore:     taskStore,
		ReportStore:   reportStore,
		TaskPublisher: taskPublisher,
		NATSConn:      nc,
	}
	handler.Register(h, deps)

	log.Printf("CompetifyAI server listening on :%s (NATS: %s)", port, natsURL)
	if err := h.Run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// connectNATS tries external NATS first; falls back to embedded NATS server.
func connectNATS(natsURL string) (*nats.Conn, *natsserver.Server, error) {
	nc, err := nats.Connect(natsURL, nats.Timeout(3*time.Second))
	if err == nil {
		return nc, nil, nil
	}
	log.Printf("external NATS unavailable (%v), starting embedded server...", err)

	ns, err := natsserver.NewServer(&natsserver.Options{
		Port:      -1, // random free port
		JetStream: true,
		StoreDir:  os.TempDir(),
	})
	if err != nil {
		return nil, nil, err
	}
	go ns.Start()
	if !ns.ReadyForConnections(5 * time.Second) {
		ns.Shutdown()
		return nil, nil, err
	}

	embeddedURL := ns.ClientURL()
	nc, err = nats.Connect(embeddedURL, nats.Timeout(3*time.Second))
	if err != nil {
		ns.Shutdown()
		return nil, nil, err
	}
	return nc, ns, nil
}

// runWorker starts the inline Worker goroutine that consumes NATS tasks.
func runWorker(
	ctx context.Context,
	nc *nats.Conn,
	taskStore *memory.TaskStore,
	reportStore *memory.ReportStore,
	auditChain *provenance.AuditChain,
	agents *dag.AgentSet,
	runnable compose.Runnable[schema.UserQuery, *schema.FinalReviewOutput],
) {
	log.Println("[Worker] starting inline worker goroutine...")

	taskSub, err := messaging.NewTaskSubscriber(nc)
	if err != nil {
		log.Fatalf("[Worker] task subscriber: %v", err)
	}

	if err := taskSub.SubscribeTask(ctx, func(taskID string, query schema.UserQuery) {
		log.Printf("[Worker] received task %s", taskID)
		taskStore.Update(taskID, "running")

		// Drive frontend progress bar via NATS while the real DAG runs.
		go handler.SimulateProgress(nc, taskID)

		// Execute the real DAG.
		output, err := dag.ExecuteDAG(ctx, runnable, query)
		if err != nil {
			log.Printf("[Worker] task %s failed: %v", taskID, err)
			taskStore.Update(taskID, "error")
			return
		}

		// Store the final report.
		merkleRoot := ""
		if auditChain != nil {
			merkleRoot = auditChain.GetMerkleRoot()
		}
		report := schema.FinalReport{
			TaskID:     taskID,
			ReportID:   output.ReportID,
			Content:    output.Content,
			Status:     output.Status,
			Signature:  output.Signature,
			ApprovedBy: output.ApprovedBy,
			ApprovedAt: output.ApprovedAt,
			MerkleRoot: merkleRoot,
		}
		reportStore.Save(report)
		taskStore.Update(taskID, "done")
		log.Printf("[Worker] task %s completed", taskID)
	}); err != nil {
		log.Fatalf("[Worker] subscribe task: %v", err)
	}
}

func runInitSchema(addr string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c, err := dgraph.NewClient(addr)
	if err != nil {
		log.Fatalf("connect %s: %v", addr, err)
	}
	defer c.Close()

	log.Printf("➡️  Injecting schema into %s ...", addr)
	if err := c.InitializeSchema(ctx); err != nil {
		log.Fatalf("init schema: %v", err)
	}
	log.Println("✅ schema injected")
}

func runSmokeTest(addr string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c, err := dgraph.NewClient(addr)
	if err != nil {
		log.Fatalf("connect %s: %v", addr, err)
	}
	defer c.Close()

	log.Printf("[1/3] InitializeSchema → %s", addr)
	if err := c.InitializeSchema(ctx); err != nil {
		log.Fatalf("init schema: %v", err)
	}

	founded := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	comp := &schema.Competitor{
		CompanyName:  "Cursor",
		Website:      "https://cursor.sh",
		FundingStage: "series_b",
		TeamSize:     30,
		ThreatLevel:  5,
		Headquarters: "San Francisco, CA",
		FoundedDate:  &founded,
	}

	log.Printf("[2/3] UpsertCompetitor → %s", comp.CompanyName)
	uid, err := c.UpsertCompetitor(ctx, comp)
	if err != nil {
		log.Fatalf("upsert: %v", err)
	}
	log.Printf("       got uid = %s", uid)

	log.Printf("[3/3] QueryCompetitorByName → %s", comp.CompanyName)
	got, err := c.QueryCompetitorByName(ctx, comp.CompanyName)
	if err != nil {
		log.Fatalf("query: %v", err)
	}
	if got == nil {
		log.Fatal("query returned nil — round-trip failed")
	}
	log.Printf("       uid=%s  company_name=%s  website=%s  threat_level=%d",
		got.UID, got.CompanyName, got.Website, got.ThreatLevel)
	log.Println("✅ smoke test passed")
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
		log.Println("[Viking] VIKING_URL not set, DevilsAdvocate will use heuristic fallback")
		return nil
	}
	apiKey := os.Getenv("VIKING_API_KEY")
	log.Printf("[Viking] client initialized → %s", url)
	return viking.NewClient(url, apiKey)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

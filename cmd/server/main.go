// Command server — HTTP API entrypoint.
//
// Phase 1 flags:
//
//	-init-schema   只跑 Dgraph InitializeSchema 后退出
//	-smoke-test    端到端验证：init schema + upsert + query
//
// Phase 2: 接 Hertz / DAG / handlers / observability。
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/competify-ai/competify-backend/internal/handler"
	"github.com/competify-ai/competify-backend/internal/schema"
	"github.com/competify-ai/competify-backend/internal/storage/dgraph"
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

	// Phase 2: start Hertz HTTP server with stub handlers.
	h := server.Default(server.WithHostPorts(":" + port))
	handler.Register(h)

	log.Printf("CompetifyAI server listening on :%s", port)
	if err := h.Run(); err != nil {
		log.Fatalf("server error: %v", err)
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

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

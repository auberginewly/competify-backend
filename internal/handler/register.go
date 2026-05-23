package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/competify-ai/competify-backend/internal/messaging"
	"github.com/competify-ai/competify-backend/internal/storage/dgraph"
	"github.com/competify-ai/competify-backend/internal/storage/memory"
	"github.com/nats-io/nats.go"
)

// Deps holds all dependencies injected into HTTP handlers.
type Deps struct {
	TaskStore     *memory.TaskStore
	ReportStore   *memory.ReportStore
	TaskPublisher *messaging.TaskPublisher
	NATSConn      *nats.Conn
	DgraphClient  *dgraph.Client
}

// Register mounts all HTTP and WebSocket routes onto the Hertz server.
func Register(h *server.Hertz, deps *Deps) {
	api := h.Group("/api/v1")

	// Tasks
	api.POST("/tasks", CreateTaskHandler(deps))
	api.GET("/tasks/:id", GetTaskHandler(deps))
	api.GET("/tasks/:id/dag", DAGWebSocketHandler(deps))

	// Reports
	api.GET("/reports/:id", GetReportHandler(deps))
	api.GET("/reports/:id/provenance", GetProvenance)
	api.POST("/reports/:id/approve", ApproveReport)

	// Ontology
	api.GET("/ontology/competitors", ListCompetitors(deps.DgraphClient))
	api.GET("/ontology/competitors/:name", GetCompetitor(deps.DgraphClient))
	api.GET("/ontology/graph", GetOntologyGraph(deps.DgraphClient))
	api.GET("/ontology/timeline", GetTimeline(deps.DgraphClient))

	// Audit
	api.GET("/audit/:provenance_id", GetAuditLog)
	api.GET("/audit/:provenance_id/verify", VerifyMerkle)

	// Observability
	h.GET("/healthz", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(200, map[string]string{"status": "ok"})
	})
}

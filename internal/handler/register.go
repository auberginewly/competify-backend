package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
)

// Register mounts all HTTP and WebSocket routes onto the Hertz server.
func Register(h *server.Hertz) {
	api := h.Group("/api/v1")

	// Tasks
	api.POST("/tasks", CreateTask)
	api.GET("/tasks/:id", GetTask)
	api.GET("/tasks/:id/dag", DAGWebSocket)

	// Reports
	api.GET("/reports/:id", GetReport)
	api.GET("/reports/:id/provenance", GetProvenance)
	api.POST("/reports/:id/approve", ApproveReport)

	// Ontology
	api.GET("/ontology/competitors", ListCompetitors)
	api.GET("/ontology/competitors/:name", GetCompetitor)
	api.GET("/ontology/graph", GetOntologyGraph)
	api.GET("/ontology/timeline", GetTimeline)

	// Audit
	api.GET("/audit/:provenance_id", GetAuditLog)
	api.GET("/audit/:provenance_id/verify", VerifyMerkle)

	// Observability
	h.GET("/healthz", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(200, map[string]string{"status": "ok"})
	})
}

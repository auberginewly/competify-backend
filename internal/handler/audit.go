package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

// GetAuditLog handles GET /api/v1/audit/:provenance_id.
func GetAuditLog(ctx context.Context, c *app.RequestContext) {
	provenanceID := c.Param("provenance_id")
	c.JSON(200, map[string]any{
		"provenance_id": provenanceID,
		"events": []map[string]any{
			{"agent": "collector_web", "action": "scrape", "timestamp": "2026-05-22T12:00:00Z", "merkle_hash": "0xabc..."},
			{"agent": "analyzer_feat", "action": "analyze", "timestamp": "2026-05-22T12:05:00Z", "merkle_hash": "0xdef..."},
		},
	})
}

// VerifyMerkle handles GET /api/v1/audit/:provenance_id/verify.
func VerifyMerkle(ctx context.Context, c *app.RequestContext) {
	c.JSON(200, map[string]any{
		"valid":     true,
		"root_hash": "0x7a3f9e2b1c8d4e5f6a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f",
	})
}

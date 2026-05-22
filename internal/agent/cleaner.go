// Cleaner deduplicates (SimHash) and normalizes RawDataPack into NormalizedDataset.
package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
)

// Cleaner removes noise and standardizes raw data.
type Cleaner struct {
	BaseAgent
}

// NewCleaner creates a Cleaner with an optional audit chain.
func NewCleaner(auditChain ...*provenance.AuditChain) *Cleaner {
	ba := BaseAgent{Role: "cleaner"}
	if len(auditChain) > 0 {
		ba.AuditChain = auditChain[0]
	}
	return &Cleaner{BaseAgent: ba}
}

func (c *Cleaner) Name() string { return "cleaner" }

// Execute transforms RawDataPack into NormalizedDataset.
// Phase 2 stub: returns a minimal normalized record so the pipeline continues.
func (c *Cleaner) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	pack, ok := input.(*schema.RawDataPack)
	if !ok {
		return nil, fmt.Errorf("cleaner: expected *schema.RawDataPack, got %T", input)
	}

	out := &schema.NormalizedDataset{
		TaskID:      pack.TaskID,
		VikingURI:   fmt.Sprintf("viking://competify/tasks/%s/collectors/%s", pack.TaskID, pack.SourceType),
		SourceType:  pack.SourceType,
		CleanedText: pack.RawContent,
		Structured:  map[string]interface{}{"source_url": pack.SourceURL},
		Confidence:  0.85,
		CleanedAt:   time.Now().UTC(),
		CleanerID:   c.GenerateID(pack.TaskID),
	}

	c.RecordAudit(pack.TaskID, c.GenerateID(pack.TaskID),
		pack.SourceURL, out.VikingURI,
		"Cleaner normalized raw data", out.Confidence)

	return out, nil
}

func (c *Cleaner) HealthCheck(ctx context.Context) error { return nil }

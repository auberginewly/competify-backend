// Cleaner deduplicates (SimHash) and normalizes RawDataPack into NormalizedDataset.
package agent

import (
	"context"
	"fmt"
	"strings"
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

// Execute transforms RawDataPack(s) into NormalizedDataset.
// It deduplicates parallel collector outputs using 64-bit SimHash.
func (c *Cleaner) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	var packs map[string]*schema.RawDataPack

	switch v := input.(type) {
	case map[string]*schema.RawDataPack:
		packs = v
	case *schema.RawDataPack:
		packs = map[string]*schema.RawDataPack{"single": v}
	default:
		return nil, fmt.Errorf("cleaner: expected map[string]*schema.RawDataPack or *schema.RawDataPack, got %T", input)
	}

	if len(packs) == 0 {
		return nil, fmt.Errorf("cleaner: empty collector map")
	}

	// SimHash deduplication across all collector outputs.
	var deduped []string
	var seen []uint64
	var taskID string
	for _, p := range packs {
		if p == nil || p.RawContent == "" {
			continue
		}
		if taskID == "" {
			taskID = p.TaskID
		}
		h := simhash(p.RawContent)
		duplicate := false
		for _, s := range seen {
			if hammingDistance(h, s) < 3 {
				duplicate = true
				break
			}
		}
		if !duplicate {
			seen = append(seen, h)
			deduped = append(deduped, p.RawContent)
		}
	}

	cleanedText := strings.Join(deduped, "\n---\n")
	if cleanedText == "" {
		cleanedText = "No content collected."
	}

	combinedHash := simhash(cleanedText)
	confidence := 0.85
	if len(deduped) < len(packs) {
		confidence = 0.70 // lowered because some sources were duplicates
	}

	out := &schema.NormalizedDataset{
		TaskID:       taskID,
		VikingURI:    fmt.Sprintf("viking://competify/tasks/%s/cleaner", taskID),
		SourceType:   "aggregated",
		CleanedText:  cleanedText,
		Structured:   map[string]interface{}{"unique_sources": len(deduped), "total_sources": len(packs)},
		SimHashValue: combinedHash,
		Confidence:   confidence,
		CleanedAt:    time.Now().UTC(),
		CleanerID:    c.GenerateID(taskID),
	}

	c.RecordAudit(taskID, c.GenerateID(taskID),
		fmt.Sprintf("%d sources", len(packs)), out.VikingURI,
		fmt.Sprintf("Cleaner deduplicated to %d unique sources via SimHash", len(deduped)), out.Confidence)

	return out, nil
}

func (c *Cleaner) HealthCheck(ctx context.Context) error { return nil }

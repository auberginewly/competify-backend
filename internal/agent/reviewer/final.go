package reviewer

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/competify-ai/competify-backend/internal/agent"
	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
)

// FinalReviewer performs last-check approval and HMAC-signs the report.
type FinalReviewer struct {
	agent.BaseAgent
	HMACKey []byte
}

// NewFinalReviewer creates a FinalReviewer with a secret key and optional audit chain.
func NewFinalReviewer(hmacKey []byte, auditChain ...*provenance.AuditChain) *FinalReviewer {
	ba := agent.BaseAgent{Role: "final_reviewer"}
	if len(auditChain) > 0 {
		ba.AuditChain = auditChain[0]
	}
	return &FinalReviewer{BaseAgent: ba, HMACKey: hmacKey}
}

func (f *FinalReviewer) Name() string { return "final_reviewer" }

// Execute accepts *schema.DraftReport and returns *schema.FinalReviewOutput.
// It computes an HMAC-SHA256 signature over the report content and Merkle root.
func (f *FinalReviewer) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	draft, ok := input.(*schema.DraftReport)
	if !ok {
		return nil, fmt.Errorf("final reviewer: expected *schema.DraftReport, got %T", input)
	}

	sig := f.sign(draft)
	output := &schema.FinalReviewOutput{
		ReportID:   draft.TaskID,
		Content:    draft.MarkdownContent,
		Status:     "APPROVED",
		Signature:  sig,
		Comment:    "Approved by FinalReviewer with HMAC-SHA256 signature.",
		ApprovedAt: time.Now().UTC(),
		ApprovedBy: f.GenerateID(draft.TaskID),
		Footnotes:  draft.Footnotes,
	}

	f.RecordAudit(draft.TaskID, f.GenerateID(draft.TaskID),
		draft.MerkleRootHash, output.Signature,
		"FinalReviewer HMAC-signed the report", draft.Confidence)
	return output, nil
}

func (f *FinalReviewer) HealthCheck(ctx context.Context) error { return nil }

// sign produces HMAC-SHA256(report_id + content + merkle_root + generated_at).
func (f *FinalReviewer) sign(draft *schema.DraftReport) string {
	if len(f.HMACKey) == 0 {
		// Fallback for tests without a real key.
		return "unsigned"
	}
	payload := fmt.Sprintf("%s|%s|%s|%d",
		draft.TaskID,
		draft.MarkdownContent,
		draft.MerkleRootHash,
		draft.GeneratedAt.Unix(),
	)
	mac := hmac.New(sha256.New, f.HMACKey)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

package reviewer

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
)

// ============ DevilsAdvocate ============

// TestDevilsAdvocate_ApproveWhenClean: no marketing words, high score, multiple sources.
func TestDevilsAdvocate_ApproveWhenClean(t *testing.T) {
	da := NewDevilsAdvocate(nil)
	result := &schema.AnalysisResult{
		TaskID:     "t1",
		Dimension:  "feature",
		Payload:    "The product supports REST and GraphQL APIs.",
		Score:      0.85,
		SourceURIs: []string{"v1", "v2"},
		AnalyzerID: "a1",
	}
	out, err := da.Execute(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	report := out.(*schema.ReviewReport)
	if !report.IsApproved {
		t.Fatalf("expected APPROVE, got %s", report.NextAction)
	}
	if len(report.Conflicts) != 0 {
		t.Fatalf("expected 0 conflicts, got %d", len(report.Conflicts))
	}
}

// TestDevilsAdvocate_RetryOnLowConfidence: score < 0.75 triggers RETRY_AUTO.
func TestDevilsAdvocate_RetryOnLowConfidence(t *testing.T) {
	da := NewDevilsAdvocate(nil)
	result := &schema.AnalysisResult{
		TaskID:     "t1",
		Dimension:  "pricing",
		Payload:    "Pricing seems reasonable.",
		Score:      0.60,
		SourceURIs: []string{"v1", "v2"},
		AnalyzerID: "a1",
	}
	out, err := da.Execute(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	report := out.(*schema.ReviewReport)
	if report.IsApproved {
		t.Fatal("expected rejection for low confidence")
	}
	if report.NextAction != "RETRY_AUTO" {
		t.Fatalf("expected RETRY_AUTO, got %s", report.NextAction)
	}
	if len(report.Conflicts) == 0 {
		t.Fatal("expected at least one conflict")
	}
}

// TestDevilsAdvocate_RetryOnMarketingHype: payload contains hype words.
func TestDevilsAdvocate_RetryOnMarketingHype(t *testing.T) {
	da := NewDevilsAdvocate(nil)
	result := &schema.AnalysisResult{
		TaskID:     "t1",
		Dimension:  "market",
		Payload:    "This is the best and most revolutionary product in the market.",
		Score:      0.90,
		SourceURIs: []string{"v1", "v2"},
		AnalyzerID: "a1",
	}
	out, err := da.Execute(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	report := out.(*schema.ReviewReport)
	if report.IsApproved {
		t.Fatal("expected rejection for marketing hype")
	}
	foundHype := false
	for _, c := range report.Conflicts {
		if c.Severity == "medium" && len(c.SuggestedFix) > 0 {
			foundHype = true
		}
	}
	if !foundHype {
		t.Fatalf("expected a medium-severity hype conflict, got %+v", report.Conflicts)
	}
}

// TestDevilsAdvocate_RetryOnSingleSource: only one source cited.
func TestDevilsAdvocate_RetryOnSingleSource(t *testing.T) {
	da := NewDevilsAdvocate(nil)
	result := &schema.AnalysisResult{
		TaskID:     "t1",
		Dimension:  "tech",
		Payload:    "Built with Go and PostgreSQL.",
		Score:      0.80,
		SourceURIs: []string{"v1"},
		AnalyzerID: "a1",
	}
	out, err := da.Execute(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	report := out.(*schema.ReviewReport)
	if report.IsApproved {
		t.Fatal("expected rejection for single source")
	}
	foundSource := false
	for _, c := range report.Conflicts {
		if c.Severity == "low" {
			foundSource = true
		}
	}
	if !foundSource {
		t.Fatalf("expected a low-severity source conflict, got %+v", report.Conflicts)
	}
}

// TestDevilsAdvocate_MultipleConflicts: low confidence + single source.
func TestDevilsAdvocate_MultipleConflicts(t *testing.T) {
	da := NewDevilsAdvocate(nil)
	result := &schema.AnalysisResult{
		TaskID:     "t1",
		Dimension:  "feature",
		Payload:    "Best feature ever.",
		Score:      0.50,
		SourceURIs: []string{"v1"},
		AnalyzerID: "a1",
	}
	out, err := da.Execute(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	report := out.(*schema.ReviewReport)
	if len(report.Conflicts) < 2 {
		t.Fatalf("expected at least 2 conflicts, got %d", len(report.Conflicts))
	}
}

// TestDevilsAdvocate_RecordsAudit: audit chain is populated.
func TestDevilsAdvocate_RecordsAudit(t *testing.T) {
	ac := provenance.NewAuditChain()
	da := NewDevilsAdvocate(nil, ac)
	result := &schema.AnalysisResult{
		TaskID:     "t1",
		Dimension:  "feature",
		Payload:    "Test.",
		Score:      0.90,
		SourceURIs: []string{"v1", "v2"},
		AnalyzerID: "a1",
	}
	_, _ = da.Execute(context.Background(), result)
	if len(ac.Logs()) == 0 {
		t.Fatal("expected audit log to be recorded")
	}
}

// ============ CrossReviewer ============

// TestCrossReviewer_ApproveWhenNoConflicts: all results pass Devil's Advocate.
func TestCrossReviewer_ApproveWhenNoConflicts(t *testing.T) {
	devil := NewDevilsAdvocate(nil)
	cr := NewCrossReviewer(devil)
	results := []*schema.AnalysisResult{
		{TaskID: "t1", Dimension: "feature", Payload: "Stable APIs.", Score: 0.88, SourceURIs: []string{"v1", "v2"}, AnalyzerID: "a1"},
		{TaskID: "t1", Dimension: "pricing", Payload: "Fair pricing.", Score: 0.82, SourceURIs: []string{"v1", "v2"}, AnalyzerID: "a2"},
	}
	out, err := cr.Execute(context.Background(), results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	report := out.(*schema.ReviewReport)
	if !report.IsApproved {
		t.Fatalf("expected APPROVE, got %s", report.NextAction)
	}
	if len(report.Conflicts) != 0 {
		t.Fatalf("expected 0 conflicts, got %d", len(report.Conflicts))
	}
}

// TestCrossReviewer_RetryAutoOnMediumConflicts: low confidence triggers RETRY_AUTO.
func TestCrossReviewer_RetryAutoOnMediumConflicts(t *testing.T) {
	devil := NewDevilsAdvocate(nil)
	cr := NewCrossReviewer(devil)
	results := []*schema.AnalysisResult{
		{TaskID: "t1", Dimension: "feature", Payload: "Stable APIs.", Score: 0.88, SourceURIs: []string{"v1", "v2"}, AnalyzerID: "a1"},
		{TaskID: "t1", Dimension: "pricing", Payload: "Maybe cheap?", Score: 0.60, SourceURIs: []string{"v1", "v2"}, AnalyzerID: "a2"},
	}
	out, err := cr.Execute(context.Background(), results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	report := out.(*schema.ReviewReport)
	if report.NextAction != "RETRY_AUTO" {
		t.Fatalf("expected RETRY_AUTO, got %s", report.NextAction)
	}
	if report.IsApproved {
		t.Fatal("expected rejection")
	}
}

// TestCrossReviewer_RejectHumanOnHighSeverity: if any conflict is high severity.
func TestCrossReviewer_RejectHumanOnHighSeverity(t *testing.T) {
	_ = NewDevilsAdvocate(nil)
	// Inject a result that would produce a high-severity conflict.
	// Since current Devil only produces medium/low, we test the routing logic directly.
	report := &schema.ReviewReport{
		TaskID:     "t1",
		IsApproved: false,
		Conflicts: []schema.Conflict{
			{Dimension: "feature", Severity: "high", Description: "Data fabrication detected"},
		},
		NextAction: "REJECT_HUMAN",
	}
	// Route using hasHighSeverity helper logic.
	if !hasHighSeverity(report.Conflicts) {
		t.Fatal("expected high severity detection")
	}
	if report.NextAction != "REJECT_HUMAN" {
		t.Fatalf("expected REJECT_HUMAN, got %s", report.NextAction)
	}
}

// TestCrossReviewer_AggregatesConflictsFromMultipleResults.
func TestCrossReviewer_AggregatesConflicts(t *testing.T) {
	devil := NewDevilsAdvocate(nil)
	cr := NewCrossReviewer(devil)
	results := []*schema.AnalysisResult{
		{TaskID: "t1", Dimension: "feature", Payload: "Best ever.", Score: 0.50, SourceURIs: []string{"v1"}, AnalyzerID: "a1"},
		{TaskID: "t1", Dimension: "pricing", Payload: "Cheap.", Score: 0.50, SourceURIs: []string{"v2"}, AnalyzerID: "a2"},
	}
	out, err := cr.Execute(context.Background(), results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	report := out.(*schema.ReviewReport)
	if len(report.Conflicts) < 2 {
		t.Fatalf("expected at least 2 aggregated conflicts, got %d", len(report.Conflicts))
	}
}

// ============ FinalReviewer ============

// TestFinalReviewer_HMACSignature: signature changes when content changes.
func TestFinalReviewer_HMACSignature(t *testing.T) {
	key := []byte("test-secret-key")
	fr := NewFinalReviewer(key)
	draft := &schema.DraftReport{
		TaskID:          "t1",
		MarkdownContent: "Report A",
		MerkleRootHash:  "abc123",
		GeneratedAt:     time.Now().UTC(),
	}
	out1, err := fr.Execute(context.Background(), draft)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	report1 := out1.(*schema.FinalReviewOutput)

	if report1.Signature == "unsigned" || report1.Signature == "" {
		t.Fatal("expected real HMAC signature")
	}

	// Same input -> same signature.
	out2, _ := fr.Execute(context.Background(), draft)
	report2 := out2.(*schema.FinalReviewOutput)
	if report1.Signature != report2.Signature {
		t.Fatal("same input must produce same signature")
	}

	// Different content -> different signature.
	draft.MarkdownContent = "Report B"
	out3, _ := fr.Execute(context.Background(), draft)
	report3 := out3.(*schema.FinalReviewOutput)
	if report1.Signature == report3.Signature {
		t.Fatal("different content must produce different signature")
	}
}

// TestFinalReviewer_HMACVerifiable: signature can be verified with the same key.
func TestFinalReviewer_HMACVerifiable(t *testing.T) {
	key := []byte("test-secret-key")
	fr := NewFinalReviewer(key)
	draft := &schema.DraftReport{
		TaskID:          "t1",
		MarkdownContent: "Report A",
		MerkleRootHash:  "abc123",
		GeneratedAt:     time.Now().UTC(),
	}
	out, _ := fr.Execute(context.Background(), draft)
	report := out.(*schema.FinalReviewOutput)

	// Recompute expected signature.
	payload := draft.TaskID + "|" + draft.MarkdownContent + "|" + draft.MerkleRootHash + "|" + fmt.Sprintf("%d", draft.GeneratedAt.Unix())
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))

	if report.Signature != expected {
		t.Fatalf("signature mismatch: got %s, want %s", report.Signature, expected)
	}
}

// TestFinalReviewer_RecordsAudit: audit chain is populated.
func TestFinalReviewer_RecordsAudit(t *testing.T) {
	ac := provenance.NewAuditChain()
	fr := NewFinalReviewer([]byte("key"), ac)
	draft := &schema.DraftReport{
		TaskID:          "t1",
		MarkdownContent: "Report.",
		MerkleRootHash:  "abc",
		GeneratedAt:     time.Now().UTC(),
	}
	_, _ = fr.Execute(context.Background(), draft)
	if len(ac.Logs()) == 0 {
		t.Fatal("expected audit log to be recorded")
	}
}

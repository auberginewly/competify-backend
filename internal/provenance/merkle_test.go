package provenance

import "testing"

// TestMerkleTree_RootStable: same input -> same root.
func TestMerkleTree_RootStable(t *testing.T) {
	leaves := []string{"a", "b", "c", "d"}
	t1 := BuildMerkleTree(leaves)
	t2 := BuildMerkleTree(leaves)
	if t1.GetRootHash() != t2.GetRootHash() {
		t.Fatalf("expected stable root, got %s vs %s", t1.GetRootHash(), t2.GetRootHash())
	}
}

// TestMerkleTree_TamperDetected: changing one leaf must change the root.
func TestMerkleTree_TamperDetected(t *testing.T) {
	original := []string{"a", "b", "c", "d"}
	tampered := []string{"a", "X", "c", "d"}
	t1 := BuildMerkleTree(original)
	t2 := BuildMerkleTree(tampered)
	if t1.GetRootHash() == t2.GetRootHash() {
		t.Fatal("tampered tree must produce different root")
	}
}

// TestVerifyTraceability_DetectsCorruption: explicit verify call returns false.
func TestVerifyTraceability_DetectsCorruption(t *testing.T) {
	original := []string{"a", "b", "c"}
	root := BuildMerkleTree(original).GetRootHash()

	ok, err := VerifyTraceability(root, []string{"a", "b", "c"})
	if err != nil || !ok {
		t.Fatalf("verify legitimate chain should succeed, got ok=%v err=%v", ok, err)
	}

	ok, _ = VerifyTraceability(root, []string{"a", "X", "c"})
	if ok {
		t.Fatal("verify tampered chain must return false")
	}
}

// TestCalculateConfidence_Bounded: clamp to [0,1].
func TestCalculateConfidence_Bounded(t *testing.T) {
	f := ConfidenceFactors{SourceScore: 2.0, CleanScore: 2.0, LLMScore: 2.0, ReviewScore: 2.0}
	c := CalculateConfidence(f)
	if c != 1.0 {
		t.Fatalf("expected clamp to 1.0, got %f", c)
	}
}

// TestBuildMerkleTree_EmptyInput returns nil for empty leaves.
func TestBuildMerkleTree_EmptyInput(t *testing.T) {
	root := BuildMerkleTree([]string{})
	if root != nil {
		t.Fatal("expected nil root for empty input")
	}
}

// TestBuildMerkleTree_SingleLeaf promotes a lone leaf to root.
func TestBuildMerkleTree_SingleLeaf(t *testing.T) {
	root := BuildMerkleTree([]string{"only"})
	if root == nil {
		t.Fatal("expected non-nil root")
	}
	if root.Left != nil || root.Right != nil {
		t.Fatal("single-leaf root must have no children")
	}
	expected := sha256Hex("only")
	if root.Hash != expected {
		t.Fatalf("expected leaf hash %s, got %s", expected, root.Hash)
	}
}

// TestBuildMerkleTree_OddCount handles odd-numbered leaves by promoting the last node.
func TestBuildMerkleTree_OddCount(t *testing.T) {
	leaves := []string{"a", "b", "c"}
	root := BuildMerkleTree(leaves)
	if root == nil {
		t.Fatal("expected non-nil root")
	}
	// Ensure the tree can still be verified.
	ok, err := VerifyTraceability(root.GetRootHash(), leaves)
	if err != nil || !ok {
		t.Fatalf("odd-count tree must verify, got ok=%v err=%v", ok, err)
	}
}

// TestGetProofPath returns sibling hashes for a valid leaf.
func TestGetProofPath(t *testing.T) {
	leaves := []string{"a", "b", "c", "d"}
	root := BuildMerkleTree(leaves)
	target := sha256Hex("a")
	path, err := root.GetProofPath(target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(path) == 0 {
		t.Fatal("expected non-empty proof path")
	}
}

// TestGetProofPath_NotFound errors for an unknown hash.
func TestGetProofPath_NotFound(t *testing.T) {
	leaves := []string{"a", "b", "c", "d"}
	root := BuildMerkleTree(leaves)
	_, err := root.GetProofPath("unknown_hash")
	if err == nil {
		t.Fatal("expected error for unknown hash")
	}
}

// TestCalculateConfidence_Zero returns 0 when all factors are zero.
func TestCalculateConfidence_Zero(t *testing.T) {
	f := ConfidenceFactors{}
	c := CalculateConfidence(f)
	if c != 0.0 {
		t.Fatalf("expected 0.0, got %f", c)
	}
}

// TestCalculateConfidence_CustomWeights uses provided weights.
func TestCalculateConfidence_CustomWeights(t *testing.T) {
	f := ConfidenceFactors{SourceScore: 1.0, CleanScore: 1.0, LLMScore: 1.0, ReviewScore: 1.0}
	w := Weights{Source: 0.25, Clean: 0.25, LLM: 0.25, Review: 0.25}
	c := CalculateConfidence(f, w)
	if c != 1.0 {
		t.Fatalf("expected 1.0 with custom weights, got %f", c)
	}
}

// TestLabel maps scores correctly.
func TestLabel(t *testing.T) {
	cases := []struct {
		score float64
		want  string
	}{
		{0.95, "HIGH"},
		{0.80, "MEDIUM"},
		{0.65, "LOW"},
		{0.50, "SUSPICIOUS"},
	}
	for _, tc := range cases {
		got := Label(tc.score)
		if got != tc.want {
			t.Fatalf("Label(%.2f) = %s, want %s", tc.score, got, tc.want)
		}
	}
}

// TestAnchorConfidence binds score to Merkle root.
func TestAnchorConfidence(t *testing.T) {
	f := ConfidenceFactors{SourceScore: 1.0, CleanScore: 1.0, LLMScore: 1.0, ReviewScore: 1.0}
	ac := AnchorConfidence(f, "abc123", "prov-1")
	if ac.MerkleRoot != "abc123" {
		t.Fatalf("expected MerkleRoot abc123, got %s", ac.MerkleRoot)
	}
	if ac.ProvenanceID != "prov-1" {
		t.Fatalf("expected ProvenanceID prov-1, got %s", ac.ProvenanceID)
	}
	if ac.Label != "HIGH" {
		t.Fatalf("expected label HIGH, got %s", ac.Label)
	}
}

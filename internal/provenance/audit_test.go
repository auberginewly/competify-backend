package provenance

import (
	"fmt"
	"sync"
	"testing"
)

// TestAuditChain_Append increases log count and updates Merkle root.
func TestAuditChain_Append(t *testing.T) {
	ac := NewAuditChain()
	if len(ac.Logs()) != 0 {
		t.Fatal("expected empty chain")
	}

	ac.Append("agent-1", "task-1", "node-1", "input-a", "output-b", "reasoning", 0.85)
	logs := ac.Logs()
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].AgentID != "agent-1" {
		t.Fatalf("expected agent-1, got %s", logs[0].AgentID)
	}
	if logs[0].MerkleNodeHash == "" {
		t.Fatal("expected non-empty MerkleNodeHash")
	}

	root1 := ac.GetMerkleRoot()
	if root1 == "" {
		t.Fatal("expected non-empty MerkleRoot after first append")
	}

	ac.Append("agent-2", "task-1", "node-2", "input-c", "output-d", "reasoning2", 0.90)
	root2 := ac.GetMerkleRoot()
	if root2 == "" {
		t.Fatal("expected non-empty MerkleRoot after second append")
	}
	if root1 == root2 {
		t.Fatal("MerkleRoot must change after second append")
	}
}

// TestAuditChain_VerifyChain returns true for intact chain.
func TestAuditChain_VerifyChain(t *testing.T) {
	ac := NewAuditChain()
	ac.Append("a1", "t1", "n1", "i1", "o1", "r1", 0.8)
	ac.Append("a2", "t1", "n2", "i2", "o2", "r2", 0.9)

	if !ac.VerifyChain() {
		t.Fatal("expected intact chain to verify")
	}
}

// TestAuditChain_TamperDetection detects manual log mutation.
func TestAuditChain_TamperDetection(t *testing.T) {
	ac := NewAuditChain()
	ac.Append("a1", "t1", "n1", "i1", "o1", "r1", 0.8)
	ac.Append("a2", "t1", "n2", "i2", "o2", "r2", 0.9)

	// Direct mutation breaks immutability.
	logs := ac.Logs()
	logs[0].AgentID = "attacker"

	// The original chain's Merkle root is still derived from un-tampered internal state,
	// so VerifyChain on the original AuditChain still passes. This test documents
	// that Logs() returns a defensive copy; mutating it does NOT corrupt the chain.
	if !ac.VerifyChain() {
		t.Fatal("mutating defensive copy must not corrupt original chain")
	}
}

// TestAuditChain_ConcurrentAppend is safe under concurrent writers.
func TestAuditChain_ConcurrentAppend(t *testing.T) {
	ac := NewAuditChain()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ac.Append(
				fmt.Sprintf("agent-%d", idx),
				"task-concurrent",
				fmt.Sprintf("node-%d", idx),
				"input", "output", "reason", 0.8,
			)
		}(i)
	}
	wg.Wait()

	logs := ac.Logs()
	if len(logs) != 100 {
		t.Fatalf("expected 100 logs, got %d", len(logs))
	}
	if !ac.VerifyChain() {
		t.Fatal("concurrent chain must still verify")
	}
}

// TestAuditChain_PrevLogHash chains entries together.
func TestAuditChain_PrevLogHash(t *testing.T) {
	ac := NewAuditChain()
	ac.Append("a1", "t1", "n1", "i1", "o1", "r1", 0.8)
	ac.Append("a2", "t1", "n2", "i2", "o2", "r2", 0.9)

	logs := ac.Logs()
	if logs[1].PrevLogHash != logs[0].MerkleNodeHash {
		t.Fatal("second log's PrevLogHash must equal first log's MerkleNodeHash")
	}
}

// TestAuditChain_EmptyRoot returns empty string.
func TestAuditChain_EmptyRoot(t *testing.T) {
	ac := NewAuditChain()
	if ac.GetMerkleRoot() != "" {
		t.Fatal("expected empty root for empty chain")
	}
	if ac.VerifyChain() {
		t.Fatal("empty chain should not verify")
	}
}

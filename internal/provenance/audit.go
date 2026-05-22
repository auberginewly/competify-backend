package provenance

import (
	"sync"
	"time"
)

// AuditLog is a single append-only entry capturing one Agent's execution.
// All AuditLogs in a task chain together via PrevLogHash, forming a hash chain.
type AuditLog struct {
	Timestamp      string  `json:"timestamp"`
	AgentID        string  `json:"agent_id"`
	TaskID         string  `json:"task_id"`
	NodeID         string  `json:"node_id"`
	InputHash      string  `json:"input_hash"`
	OutputHash     string  `json:"output_hash"`
	Reasoning      string  `json:"reasoning"`
	Confidence     float64 `json:"confidence"`
	MerkleNodeHash string  `json:"merkle_node_hash"` // this entry's hash, used as leaf in Merkle Tree
	PrevLogHash    string  `json:"prev_log_hash"`    // chain link
}

// AuditChain manages append-only audit logs synced with a Merkle Tree.
// Thread-safe for concurrent appends.
type AuditChain struct {
	mu         sync.RWMutex
	logs       []AuditLog
	lastHash   string
	merkleTree *MerkleNode
}

// NewAuditChain creates an empty chain.
func NewAuditChain() *AuditChain {
	return &AuditChain{logs: make([]AuditLog, 0)}
}

// Append adds a new log entry. The MerkleNodeHash is computed from the entry's
// content plus the previous hash, ensuring tampering is detectable.
func (ac *AuditChain) Append(agentID, taskID, nodeID, inputData, outputData, reasoning string, confidence float64) *AuditLog {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	log := AuditLog{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		AgentID:     agentID,
		TaskID:      taskID,
		NodeID:      nodeID,
		InputHash:   sha256Hex(inputData),
		OutputHash:  sha256Hex(outputData),
		Reasoning:   reasoning,
		Confidence:  confidence,
		PrevLogHash: ac.lastHash,
	}

	// This entry's hash chains in the previous, making history immutable.
	chainData := log.Timestamp + log.AgentID + log.TaskID + log.NodeID +
		log.InputHash + log.OutputHash + log.PrevLogHash
	log.MerkleNodeHash = sha256Hex(chainData)
	ac.lastHash = log.MerkleNodeHash

	ac.logs = append(ac.logs, log)

	// Rebuild Merkle Tree. For large chains, switch to incremental updates.
	leafData := make([]string, len(ac.logs))
	for i, l := range ac.logs {
		leafData[i] = l.MerkleNodeHash
	}
	ac.merkleTree = BuildMerkleTree(leafData)

	return &log
}

// GetMerkleRoot returns the current Merkle Root, "" if empty.
func (ac *AuditChain) GetMerkleRoot() string {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	if ac.merkleTree == nil {
		return ""
	}
	return ac.merkleTree.GetRootHash()
}

// VerifyChain re-derives the root from current logs and checks integrity.
func (ac *AuditChain) VerifyChain() bool {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	leafData := make([]string, len(ac.logs))
	for i, l := range ac.logs {
		leafData[i] = l.MerkleNodeHash
	}
	ok, _ := VerifyTraceability(ac.merkleTree.GetRootHash(), leafData)
	return ok
}

// Logs returns a snapshot of all log entries (defensive copy).
func (ac *AuditChain) Logs() []AuditLog {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	out := make([]AuditLog, len(ac.logs))
	copy(out, ac.logs)
	return out
}

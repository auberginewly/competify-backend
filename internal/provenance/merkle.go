// Package provenance implements the Merkle Tree audit chain and confidence scoring.
// This is one of the project's signature features — every conclusion in the final
// report is anchored to a Merkle Root, making tampering cryptographically detectable.
package provenance

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// MerkleNode is a node in the Merkle Tree. Leaves hash raw data; internal nodes
// hash the concatenation of their children's hashes plus an agent signature.
type MerkleNode struct {
	Hash  string      `json:"hash"`
	Left  *MerkleNode `json:"left,omitempty"`
	Right *MerkleNode `json:"right,omitempty"`
	Data  string      `json:"data,omitempty"` // raw data for leaves
	Level int         `json:"level"`
}

// NewMerkleNode constructs a node. Leaves: pass nil for left+right, raw data for data.
// Internal nodes: pass child nodes and an agent metadata signature for data.
func NewMerkleNode(left, right *MerkleNode, data string) *MerkleNode {
	node := &MerkleNode{Left: left, Right: right, Data: data}

	if left == nil && right == nil {
		// Leaf node: hash raw data directly.
		h := sha256.Sum256([]byte(data))
		node.Hash = hex.EncodeToString(h[:])
		node.Level = 0
		return node
	}

	// Internal node: Hash(Left.Hash + Right.Hash + AgentSignature)
	var input []byte
	if left != nil {
		input = append(input, []byte(left.Hash)...)
	}
	if right != nil {
		input = append(input, []byte(right.Hash)...)
	}
	input = append(input, []byte(data)...)
	h := sha256.Sum256(input)
	node.Hash = hex.EncodeToString(h[:])
	if left != nil {
		node.Level = left.Level + 1
	}
	return node
}

// BuildMerkleTree constructs a full Merkle Tree from leaf data, returning the root.
func BuildMerkleTree(leafData []string) *MerkleNode {
	if len(leafData) == 0 {
		return nil
	}

	nodes := make([]*MerkleNode, 0, len(leafData))
	for _, d := range leafData {
		nodes = append(nodes, NewMerkleNode(nil, nil, d))
	}

	// Build upward level by level.
	for len(nodes) > 1 {
		next := make([]*MerkleNode, 0, (len(nodes)+1)/2)
		for i := 0; i < len(nodes); i += 2 {
			if i+1 < len(nodes) {
				next = append(next, NewMerkleNode(nodes[i], nodes[i+1], "agent_process_sign"))
			} else {
				// Odd count: promote the last node alone.
				next = append(next, NewMerkleNode(nodes[i], nil, "agent_process_sign"))
			}
		}
		nodes = next
	}

	return nodes[0]
}

// GetRootHash returns the root hash, "" if tree is nil.
func (n *MerkleNode) GetRootHash() string {
	if n == nil {
		return ""
	}
	return n.Hash
}

// VerifyTraceability rebuilds the tree from leaves and checks the root matches.
// Returns (false, nil) if tampered, (true, nil) if intact.
func VerifyTraceability(expectedRoot string, leafData []string) (bool, error) {
	if len(leafData) == 0 {
		return false, fmt.Errorf("provenance.VerifyTraceability: empty leaf data")
	}
	rebuilt := BuildMerkleTree(leafData)
	return rebuilt.GetRootHash() == expectedRoot, nil
}

// GetProofPath returns the sibling hashes needed to verify a leaf, root-ward.
// Used for lightweight verification without rebuilding the whole tree.
func (n *MerkleNode) GetProofPath(targetHash string) ([]string, error) {
	if n == nil {
		return nil, fmt.Errorf("provenance.GetProofPath: empty tree")
	}
	var path []string
	if !findPath(n, targetHash, &path) {
		return nil, fmt.Errorf("provenance.GetProofPath: hash not found")
	}
	return path, nil
}

func findPath(node *MerkleNode, target string, path *[]string) bool {
	if node == nil {
		return false
	}
	if node.Hash == target && node.Left == nil && node.Right == nil {
		return true
	}
	if findPath(node.Left, target, path) {
		if node.Right != nil {
			*path = append(*path, node.Right.Hash)
		}
		return true
	}
	if findPath(node.Right, target, path) {
		if node.Left != nil {
			*path = append(*path, node.Left.Hash)
		}
		return true
	}
	return false
}

// sha256Hex is a helper for callers that need to hash arbitrary strings.
func sha256Hex(data string) string {
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

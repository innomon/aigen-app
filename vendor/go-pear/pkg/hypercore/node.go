package hypercore

import "fmt"

// Node represents a node in the Hypercore flat Merkle tree.
type Node struct {
	Index uint64   `json:"index"`
	Size  uint64   `json:"size"`
	Hash  [32]byte `json:"hash"`
}

// String returns human-readable node representation.
func (n *Node) String() string {
	return fmt.Sprintf("Node(index=%d, size=%d, hash=%x)", n.Index, n.Size, n.Hash[:4])
}

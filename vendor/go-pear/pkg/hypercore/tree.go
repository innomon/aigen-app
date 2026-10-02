package hypercore

import (
	"errors"
	"fmt"
	"sync"

	"go-pear/pkg/flattree"
)

var (
	// ErrBlockNotFound is returned when querying a block index exceeding current length.
	ErrBlockNotFound = errors.New("hypercore: block not found")
	// ErrInvalidProof is returned when a Merkle proof fails validation.
	ErrInvalidProof = errors.New("hypercore: invalid Merkle proof")
)

// Tree represents an in-memory append-only Merkle tree.
type Tree struct {
	mu         sync.RWMutex
	nodes      map[uint64]*Node
	length     uint64
	byteLength uint64
	roots      []*Node
}

// NewTree creates an empty append-only Merkle tree.
func NewTree() *Tree {
	return &Tree{
		nodes: make(map[uint64]*Node),
		roots: make([]*Node, 0),
	}
}

// Length returns the total number of blocks (leaves) in the tree.
func (t *Tree) Length() uint64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.length
}

// ByteLength returns the total byte size of all appended blocks.
func (t *Tree) ByteLength() uint64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.byteLength
}

// Roots returns the current peak root nodes for the tree.
func (t *Tree) Roots() []*Node {
	t.mu.RLock()
	defer t.mu.RUnlock()
	copied := make([]*Node, len(t.roots))
	copy(copied, t.roots)
	return copied
}

// SetRoots updates the tree's peak roots and length after verifying an upgrade.
func (t *Tree) SetRoots(roots []*Node, length uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.roots = make([]*Node, len(roots))
	copy(t.roots, roots)
	for _, r := range roots {
		t.nodes[r.Index] = r
	}
	if length > t.length {
		t.length = length
	}
}

// GetNode retrieves a tree node by its flat index.
func (t *Tree) GetNode(index uint64) (*Node, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	n, ok := t.nodes[index]
	return n, ok
}

// Append adds a new data block to the tree, computing leaf and parent hashes up to peak roots.
func (t *Tree) Append(data []byte) (*Node, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	blockIndex := t.length
	leafFlatIndex := flattree.Index(0, blockIndex)
	blockSize := uint64(len(data))
	leafHash := LeafHash(data)

	leafNode := &Node{
		Index: leafFlatIndex,
		Size:  blockSize,
		Hash:  leafHash,
	}

	t.nodes[leafFlatIndex] = leafNode
	t.length++
	t.byteLength += blockSize

	// Update parents and peak roots
	curr := leafNode
	for curr.Index%2 == 0 || flattree.Offset(curr.Index)%2 != 0 {
		// Check if sibling exists
		sibIndex := flattree.Sibling(curr.Index)
		sib, exists := t.nodes[sibIndex]
		if !exists {
			break
		}

		var left, right *Node
		if curr.Index < sibIndex {
			left, right = curr, sib
		} else {
			left, right = sib, curr
		}

		parentIndex := flattree.Parent(curr.Index)
		parentSize := left.Size + right.Size
		parentHash := ParentHash(parentSize, left.Hash, right.Hash)

		parentNode := &Node{
			Index: parentIndex,
			Size:  parentSize,
			Hash:  parentHash,
		}

		t.nodes[parentIndex] = parentNode
		curr = parentNode
	}

	// Recompute peak roots using flattree.FullRoots
	rootIndices := flattree.FullRoots(t.length)
	t.roots = make([]*Node, 0, len(rootIndices))
	for _, idx := range rootIndices {
		if node, ok := t.nodes[idx]; ok {
			t.roots = append(t.roots, node)
		}
	}

	return leafNode, nil
}

// Proof generates the sibling Merkle proof path for a leaf node.
func (t *Tree) Proof(blockIndex uint64) ([]*Node, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if blockIndex >= t.length {
		return nil, ErrBlockNotFound
	}

	var proof []*Node
	currIndex := flattree.Index(0, blockIndex)

	for {
		// Stop if currIndex is one of the peak roots
		isRoot := false
		for _, r := range t.roots {
			if r.Index == currIndex {
				isRoot = true
				break
			}
		}
		if isRoot {
			break
		}

		sibIndex := flattree.Sibling(currIndex)
		sib, exists := t.nodes[sibIndex]
		if !exists {
			return nil, fmt.Errorf("missing proof node at index %d", sibIndex)
		}
		proof = append(proof, sib)
		currIndex = flattree.Parent(currIndex)
	}

	return proof, nil
}

// Verify verifies that a leaf node and its Merkle proof reconstruct a valid peak root.
func (t *Tree) Verify(leaf *Node, proof []*Node) bool {
	curr := *leaf

	for _, sib := range proof {
		var left, right Node
		if curr.Index < sib.Index {
			left = curr
			right = *sib
		} else {
			left = *sib
			right = curr
		}

		parentIndex := flattree.Parent(curr.Index)
		parentSize := left.Size + right.Size
		parentHash := ParentHash(parentSize, left.Hash, right.Hash)

		curr = Node{
			Index: parentIndex,
			Size:  parentSize,
			Hash:  parentHash,
		}
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	// Check if reconstructed curr matches one of the tree's peak roots
	for _, root := range t.roots {
		if root.Index == curr.Index && root.Hash == curr.Hash && root.Size == curr.Size {
			return true
		}
	}

	return false
}

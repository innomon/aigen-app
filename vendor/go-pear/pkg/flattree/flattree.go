// Package flattree implements 1D binary flat-tree mapping arithmetic for Hypercore and Merkle tree engines.
package flattree

import (
	"errors"
	"math/bits"
)

var (
	// ErrLeafHasNoChildren is returned when attempting to access children of a leaf node.
	ErrLeafHasNoChildren = errors.New("flattree: leaf node has no children")
)

// IsLeaf returns true if index is a leaf node (even numbers).
func IsLeaf(index uint64) bool {
	return index%2 == 0
}

// IsParent returns true if index is an internal parent node (odd numbers).
func IsParent(index uint64) bool {
	return index%2 != 0
}

// Depth returns the tree depth for a given flat index (number of trailing ones).
func Depth(index uint64) uint64 {
	return uint64(bits.TrailingZeros64(^index))
}

// Offset returns the relative horizontal offset at the index's depth.
func Offset(index uint64) uint64 {
	d := Depth(index)
	return index >> (d + 1)
}

// Index computes the 1D flat-tree index from a given depth and horizontal offset.
func Index(depth, offset uint64) uint64 {
	return (offset << (depth + 1)) | ((1 << depth) - 1)
}

// Parent returns the parent flat index for the given index.
func Parent(index uint64) uint64 {
	d := Depth(index)
	return Index(d+1, Offset(index)/2)
}

// Sibling returns the sibling flat index at the same depth.
func Sibling(index uint64) uint64 {
	d := Depth(index)
	offset := Offset(index)
	if offset%2 == 0 {
		return Index(d, offset+1)
	}
	return Index(d, offset-1)
}

// LeftChild returns the left child index of an internal parent node.
func LeftChild(index uint64) (uint64, error) {
	if IsLeaf(index) {
		return 0, ErrLeafHasNoChildren
	}
	d := Depth(index)
	offset := Offset(index)
	return Index(d-1, offset*2), nil
}

// RightChild returns the right child index of an internal parent node.
func RightChild(index uint64) (uint64, error) {
	if IsLeaf(index) {
		return 0, ErrLeafHasNoChildren
	}
	d := Depth(index)
	offset := Offset(index)
	return Index(d-1, offset*2+1), nil
}

// Children returns both left and right children for an internal node.
func Children(index uint64) (uint64, uint64, error) {
	left, err := LeftChild(index)
	if err != nil {
		return 0, 0, err
	}
	right, _ := RightChild(index)
	return left, right, nil
}

// Spans returns the leftmost and rightmost leaf indices covered by this node.
func Spans(index uint64) (uint64, uint64) {
	if IsLeaf(index) {
		return index, index
	}
	d := Depth(index)
	offset := Offset(index)
	width := uint64(1) << d
	leftLeaf := Index(0, offset*width)
	rightLeaf := Index(0, (offset+1)*width-1)
	return leftLeaf, rightLeaf
}

// FullRoots returns the list of peak root flat indices for a given block count (number of leaves).
func FullRoots(blockCount uint64) []uint64 {
	if blockCount == 0 {
		return nil
	}
	var roots []uint64
	var offset uint64
	var factor uint64 = 1

	for blockCount > 0 {
		factor = 1
		depth := uint64(0)
		for (factor * 2) <= blockCount {
			factor *= 2
			depth++
		}

		actualRoot := Index(depth, offset/factor)
		roots = append(roots, actualRoot)

		offset += factor
		blockCount -= factor
	}

	return roots
}

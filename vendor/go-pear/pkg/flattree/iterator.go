package flattree

// Iterator provides stateful navigation across a flat-tree.
type Iterator struct {
	index  uint64
	offset uint64
	depth  uint64
}

// NewIterator creates an iterator initialized at index 0 (the first leaf).
func NewIterator() *Iterator {
	return NewIteratorAt(0)
}

// NewIteratorAt creates an iterator positioned at the given flat index.
func NewIteratorAt(index uint64) *Iterator {
	it := &Iterator{}
	it.Seek(index)
	return it
}

// Seek moves the iterator position to target index and recalculates depth/offset.
func (it *Iterator) Seek(index uint64) {
	it.index = index
	it.depth = Depth(index)
	it.offset = Offset(index)
}

// Index returns current flat index.
func (it *Iterator) Index() uint64 {
	return it.index
}

// Depth returns depth of the current position.
func (it *Iterator) Depth() uint64 {
	return it.depth
}

// Offset returns horizontal offset of current position.
func (it *Iterator) Offset() uint64 {
	return it.offset
}

// IsLeaf returns true if current position is a leaf node.
func (it *Iterator) IsLeaf() bool {
	return IsLeaf(it.index)
}

// Parent moves iterator to parent node.
func (it *Iterator) Parent() uint64 {
	it.index = Parent(it.index)
	it.depth++
	it.offset /= 2
	return it.index
}

// LeftChild moves iterator to left child. Returns error if currently at a leaf.
func (it *Iterator) LeftChild() (uint64, error) {
	if it.IsLeaf() {
		return it.index, ErrLeafHasNoChildren
	}
	it.index, _ = LeftChild(it.index)
	it.depth--
	it.offset *= 2
	return it.index, nil
}

// RightChild moves iterator to right child. Returns error if currently at a leaf.
func (it *Iterator) RightChild() (uint64, error) {
	if it.IsLeaf() {
		return it.index, ErrLeafHasNoChildren
	}
	it.index, _ = RightChild(it.index)
	it.depth--
	it.offset = it.offset*2 + 1
	return it.index, nil
}

// Sibling moves iterator to sibling node.
func (it *Iterator) Sibling() uint64 {
	it.index = Sibling(it.index)
	if it.offset%2 == 0 {
		it.offset++
	} else {
		it.offset--
	}
	return it.index
}

// NextLeaf advances to the next leaf index (index + 2).
func (it *Iterator) NextLeaf() uint64 {
	if !it.IsLeaf() {
		_, right := Spans(it.index)
		it.Seek(right + 2)
	} else {
		it.Seek(it.index + 2)
	}
	return it.index
}

// PrevLeaf moves to the previous leaf index (index - 2).
func (it *Iterator) PrevLeaf() (uint64, error) {
	if it.index < 2 {
		return it.index, ErrLeafHasNoChildren
	}
	if !it.IsLeaf() {
		left, _ := Spans(it.index)
		if left < 2 {
			return it.index, ErrLeafHasNoChildren
		}
		it.Seek(left - 2)
	} else {
		it.Seek(it.index - 2)
	}
	return it.index, nil
}

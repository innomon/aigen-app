package hypercore

import (
	"sync"
)

const (
	// PageBits is the number of bits in one bitfield page: 26624 bits (832 uint32 words * 32).
	PageWords = 832
	PageBits  = PageWords * 32
)

// Bitfield manages local block availability in pages of 832 32-bit words.
type Bitfield struct {
	mu    sync.RWMutex
	pages map[uint64][]uint32
}

// NewBitfield creates an empty local bitfield.
func NewBitfield() *Bitfield {
	return &Bitfield{
		pages: make(map[uint64][]uint32),
	}
}

// Get returns true if the block at index is downloaded and verified.
func (b *Bitfield) Get(index uint64) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	pageIndex := index / PageBits
	page, ok := b.pages[pageIndex]
	if !ok {
		return false
	}

	bitOffset := index % PageBits
	wordIndex := bitOffset / 32
	bitIndex := bitOffset % 32

	return (page[wordIndex] & (1 << bitIndex)) != 0
}

// Set marks the block at index as present or absent. Returns true if bit value changed.
func (b *Bitfield) Set(index uint64, val bool) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	pageIndex := index / PageBits
	page, ok := b.pages[pageIndex]
	if !ok {
		if !val {
			return false
		}
		page = make([]uint32, PageWords)
		b.pages[pageIndex] = page
	}

	bitOffset := index % PageBits
	wordIndex := bitOffset / 32
	bitIndex := bitOffset % 32

	mask := uint32(1 << bitIndex)
	curr := (page[wordIndex] & mask) != 0
	if curr == val {
		return false
	}

	if val {
		page[wordIndex] |= mask
	} else {
		page[wordIndex] &^= mask
	}
	return true
}

// SetRange sets a contiguous range of bits [start, start+length).
func (b *Bitfield) SetRange(start, length uint64, val bool) bool {
	changed := false
	for i := uint64(0); i < length; i++ {
		if b.Set(start+i, val) {
			changed = true
		}
	}
	return changed
}

// ContiguousLength returns the number of contiguous blocks present starting from index 0.
func (b *Bitfield) ContiguousLength() uint64 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var length uint64
	for {
		pageIndex := length / PageBits
		page, ok := b.pages[pageIndex]
		if !ok {
			return length
		}

		bitOffset := length % PageBits
		wordIndex := bitOffset / 32

		// Scan full 32-bit words if aligned
		if bitOffset%32 == 0 && wordIndex < PageWords && page[wordIndex] == 0xffffffff {
			length += 32
			continue
		}

		bitIndex := bitOffset % 32
		if (page[wordIndex] & (1 << bitIndex)) == 0 {
			return length
		}
		length++
	}
}

// GetWords returns the uint32 bitfield slice for a given word offset and count.
func (b *Bitfield) GetWords(wordStart, count uint64) []uint32 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	res := make([]uint32, count)
	for i := uint64(0); i < count; i++ {
		wordIdx := wordStart + i
		pageIdx := wordIdx / PageWords
		pWordIdx := wordIdx % PageWords
		if page, ok := b.pages[pageIdx]; ok {
			res[i] = page[pWordIdx]
		}
	}
	return res
}

// RemoteBitfield tracks a remote peer available block bitfield and range announcements.
type RemoteBitfield struct {
	bf *Bitfield
}

// NewRemoteBitfield creates an empty remote bitfield tracker.
func NewRemoteBitfield() *RemoteBitfield {
	return &RemoteBitfield{
		bf: NewBitfield(),
	}
}

func (r *RemoteBitfield) Get(index uint64) bool {
	return r.bf.Get(index)
}

func (r *RemoteBitfield) Set(index uint64, val bool) bool {
	return r.bf.Set(index, val)
}

func (r *RemoteBitfield) SetRange(start, length uint64, val bool) bool {
	return r.bf.SetRange(start, length, val)
}

func (r *RemoteBitfield) ContiguousLength() uint64 {
	return r.bf.ContiguousLength()
}

// Insert updates remote bitfield words starting from word index.
func (r *RemoteBitfield) Insert(wordStart uint64, words []uint32) {
	r.bf.mu.Lock()
	defer r.bf.mu.Unlock()

	for i, w := range words {
		wordIdx := wordStart + uint64(i)
		pageIdx := wordIdx / PageWords
		pWordIdx := wordIdx % PageWords

		page, ok := r.bf.pages[pageIdx]
		if !ok {
			if w == 0 {
				continue
			}
			page = make([]uint32, PageWords)
			r.bf.pages[pageIdx] = page
		}
		page[pWordIdx] = w
	}
}

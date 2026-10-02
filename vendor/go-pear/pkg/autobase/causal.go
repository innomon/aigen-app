package autobase

import (
	"bytes"
	"encoding/hex"
	"errors"
	"sort"

	"go-pear/pkg/compactenc"
)

var (
	ErrInvalidCausalEncoding = errors.New("autobase: invalid causal message encoding")
	ErrWriterNotFound        = errors.New("autobase: writer not found")
	ErrNoLocalWriter         = errors.New("autobase: no local writer configured")
)

// ClockVector maps a 32-byte writer public key to the highest sequence index observed from that writer.
type ClockVector map[[32]byte]uint64

// Clone creates a deep copy of the ClockVector.
func (cv ClockVector) Clone() ClockVector {
	res := make(ClockVector, len(cv))
	for k, v := range cv {
		res[k] = v
	}
	return res
}

// TotalTicks computes the sum of all observed clock ticks for tie-breaking.
func (cv ClockVector) TotalTicks() uint64 {
	var total uint64
	for _, v := range cv {
		total += v
	}
	return total
}

// CausalNode represents a linearized entry in the multi-writer causal DAG.
type CausalNode struct {
	Writer [32]byte
	Seq    uint64
	Clock  ClockVector
	Value  []byte
}

// PreencodeCausalNode calculates the byte size required to encode a CausalNode.
func PreencodeCausalNode(n *CausalNode) int {
	size := 32 // Writer key (fixed 32 bytes)
	size += compactenc.PreencodeUint64(n.Seq)
	size += compactenc.PreencodeUint(uint64(len(n.Clock)))

	for _, v := range n.Clock {
		size += 32 // Key
		size += compactenc.PreencodeUint64(v)
	}

	size += compactenc.PreencodeBuffer(n.Value)
	return size
}

// EncodeCausalNode serializes a CausalNode using Compact-Encoding.
func EncodeCausalNode(st *compactenc.State, n *CausalNode) {
	_ = compactenc.EncodeFixed32(st, n.Writer)
	_ = compactenc.EncodeUint64(st, n.Seq)
	_ = compactenc.EncodeUint(st, uint64(len(n.Clock)))

	// Sort keys for deterministic encoding
	keys := make([][32]byte, 0, len(n.Clock))
	for k := range n.Clock {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(keys[i][:], keys[j][:]) < 0
	})

	for _, k := range keys {
		_ = compactenc.EncodeFixed32(st, k)
		_ = compactenc.EncodeUint64(st, n.Clock[k])
	}

	_ = compactenc.EncodeBuffer(st, n.Value)
}

// DecodeCausalNode deserializes a binary slice into a CausalNode using Compact-Encoding.
func DecodeCausalNode(data []byte) (*CausalNode, error) {
	st := compactenc.NewState(data)
	writer, err := compactenc.DecodeFixed32(st)
	if err != nil {
		return nil, ErrInvalidCausalEncoding
	}

	seq, err := compactenc.DecodeUint64(st)
	if err != nil {
		return nil, ErrInvalidCausalEncoding
	}

	clockLen, err := compactenc.DecodeUint(st)
	if err != nil {
		return nil, ErrInvalidCausalEncoding
	}

	clock := make(ClockVector, clockLen)
	for i := uint64(0); i < clockLen; i++ {
		wKey, err := compactenc.DecodeFixed32(st)
		if err != nil {
			return nil, ErrInvalidCausalEncoding
		}
		cSeq, err := compactenc.DecodeUint64(st)
		if err != nil {
			return nil, ErrInvalidCausalEncoding
		}
		clock[wKey] = cSeq
	}

	val, err := compactenc.DecodeBuffer(st)
	if err != nil {
		return nil, ErrInvalidCausalEncoding
	}

	return &CausalNode{
		Writer: writer,
		Seq:    seq,
		Clock:  clock,
		Value:  val,
	}, nil
}

// CausalCompare determines deterministic partial ordering between two CausalNodes:
// 1. Causal dependency (if a.Clock dominates b.Clock or vice versa)
// 2. Vector clock sum (lower total ticks comes first)
// 3. Deterministic tie-breaking on Writer Key and Sequence index.
func CausalCompare(a, b *CausalNode) int {
	aDominates := clockDominates(a.Clock, b.Clock, b.Writer, b.Seq)
	bDominates := clockDominates(b.Clock, a.Clock, a.Writer, a.Seq)

	if aDominates && !bDominates {
		return 1 // a is newer (comes after b)
	}
	if bDominates && !aDominates {
		return -1 // b is newer (a comes before b)
	}

	// Concurrent / Tie-break on clock sum
	aTicks := a.Clock.TotalTicks()
	bTicks := b.Clock.TotalTicks()
	if aTicks != bTicks {
		if aTicks < bTicks {
			return -1
		}
		return 1
	}

	// Tie-break on Writer Public Key
	cmp := bytes.Compare(a.Writer[:], b.Writer[:])
	if cmp != 0 {
		return cmp
	}

	// Tie-break on sequence index
	if a.Seq < b.Seq {
		return -1
	} else if a.Seq > b.Seq {
		return 1
	}
	return 0
}

func clockDominates(a, b ClockVector, bWriter [32]byte, bSeq uint64) bool {
	if a == nil {
		return false
	}
	seenSeq, ok := a[bWriter]
	return ok && seenSeq >= bSeq
}

func formatKey(k [32]byte) string {
	return hex.EncodeToString(k[:8])
}

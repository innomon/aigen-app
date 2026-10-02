package autobase

import (
	"sort"
	"sync"

	"go-pear/pkg/compactenc"
	"go-pear/pkg/hypercore"
)

// Autobase coordinates multi-writer append-only hypercores and linearizes their history deterministically.
type Autobase struct {
	mu          sync.RWMutex
	localWriter *hypercore.Hypercore
	writers     map[[32]byte]*hypercore.Hypercore
	clock       ClockVector

	// Incremental DAG linearization state
	confirmedPrefix []*CausalNode
	checkpointSeq   map[[32]byte]uint64
	nodeCache       map[[32]byte]map[uint64]*CausalNode
}

// New creates a new Autobase instance.
func New() *Autobase {
	return &Autobase{
		writers:       make(map[[32]byte]*hypercore.Hypercore),
		clock:         make(ClockVector),
		checkpointSeq: make(map[[32]byte]uint64),
		nodeCache:     make(map[[32]byte]map[uint64]*CausalNode),
	}
}

// AddWriter registers a Hypercore writer to the Autobase.
func (ab *Autobase) AddWriter(core *hypercore.Hypercore, isLocal bool) {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	ab.writers[core.Key] = core
	if isLocal {
		ab.localWriter = core
	}
}

// LocalWriter returns the active local writer core.
func (ab *Autobase) LocalWriter() *hypercore.Hypercore {
	ab.mu.RLock()
	defer ab.mu.RUnlock()
	return ab.localWriter
}

// Append creates a new causal message with current clock vector and appends it to the local writer.
func (ab *Autobase) Append(value []byte) (*CausalNode, error) {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	if ab.localWriter == nil {
		return nil, ErrNoLocalWriter
	}

	// Update local clock vector
	ab.updateClocksLocked()

	seq := ab.localWriter.Length()
	node := &CausalNode{
		Writer: ab.localWriter.Key,
		Seq:    seq,
		Clock:  ab.clock.Clone(),
		Value:  value,
	}

	sz := PreencodeCausalNode(node)
	st := compactenc.NewAllocatedState(sz)
	EncodeCausalNode(st, node)

	if _, err := ab.localWriter.Append(st.Buffer); err != nil {
		return nil, err
	}

	ab.clock[ab.localWriter.Key] = seq
	if ab.nodeCache[ab.localWriter.Key] == nil {
		ab.nodeCache[ab.localWriter.Key] = make(map[uint64]*CausalNode)
	}
	ab.nodeCache[ab.localWriter.Key][seq] = node
	return node, nil
}

func (ab *Autobase) updateClocksLocked() {
	for k, w := range ab.writers {
		lenW := w.Length()
		if lenW > 0 {
			seq := lenW - 1
			if cur, exists := ab.clock[k]; !exists || seq > cur {
				ab.clock[k] = seq
			}
		}
	}
}

// mergeClock transitively incorporates a remote node's clock vector into local clocks.
func (ab *Autobase) mergeClock(clock ClockVector) {
	for writerKey, seq := range clock {
		if currentSeq, exists := ab.clock[writerKey]; !exists || seq > currentSeq {
			ab.clock[writerKey] = seq
		}
	}
}

// Linearize collects causal nodes across all registered writers and returns a deterministically ordered slice,
// using an incremental checkpoint frontier and partial topological sort to avoid O(N²) full-history rescans.
func (ab *Autobase) Linearize() ([]*CausalNode, error) {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	// Check if any writer has new blocks beyond the current checkpoint
	hasNew := false
	for k, w := range ab.writers {
		if w.Length() > ab.checkpointSeq[k] {
			hasNew = true
			break
		}
	}

	// If no writer changed, return the already linearized sequence
	if !hasNew && len(ab.confirmedPrefix) > 0 {
		out := make([]*CausalNode, len(ab.confirmedPrefix))
		copy(out, ab.confirmedPrefix)
		return out, nil
	}

	// Read newly arrived blocks beyond the checkpoint frontier
	var frontierNodes []*CausalNode
	for k, w := range ab.writers {
		start := ab.checkpointSeq[k]
		lenW := w.Length()
		if ab.nodeCache[k] == nil {
			ab.nodeCache[k] = make(map[uint64]*CausalNode)
		}

		for i := start; i < lenW; i++ {
			node, cached := ab.nodeCache[k][i]
			if !cached {
				block, err := w.Get(i)
				if err != nil {
					continue
				}
				var errDec error
				node, errDec = DecodeCausalNode(block)
				if errDec != nil {
					continue
				}
				ab.nodeCache[k][i] = node
			}
			ab.mergeClock(node.Clock)
			frontierNodes = append(frontierNodes, node)
		}
	}

	if len(frontierNodes) == 0 {
		out := make([]*CausalNode, len(ab.confirmedPrefix))
		copy(out, ab.confirmedPrefix)
		return out, nil
	}

	// Sort frontier nodes using deterministic causal topological comparison
	sort.SliceStable(frontierNodes, func(i, j int) bool {
		return CausalCompare(frontierNodes[i], frontierNodes[j]) < 0
	})

	// Check if the confirmed prefix is empty: everything in frontier becomes new base
	if len(ab.confirmedPrefix) == 0 {
		ab.confirmedPrefix = frontierNodes
		for k, w := range ab.writers {
			ab.checkpointSeq[k] = w.Length()
		}
		out := make([]*CausalNode, len(ab.confirmedPrefix))
		copy(out, ab.confirmedPrefix)
		return out, nil
	}

	// Merge new frontier into confirmed sequence preserving causal sort
	merged := make([]*CausalNode, 0, len(ab.confirmedPrefix)+len(frontierNodes))
	i, j := 0, 0
	for i < len(ab.confirmedPrefix) && j < len(frontierNodes) {
		if CausalCompare(ab.confirmedPrefix[i], frontierNodes[j]) <= 0 {
			merged = append(merged, ab.confirmedPrefix[i])
			i++
		} else {
			merged = append(merged, frontierNodes[j])
			j++
		}
	}
	for i < len(ab.confirmedPrefix) {
		merged = append(merged, ab.confirmedPrefix[i])
		i++
	}
	for j < len(frontierNodes) {
		merged = append(merged, frontierNodes[j])
		j++
	}

	ab.confirmedPrefix = merged
	for k, w := range ab.writers {
		ab.checkpointSeq[k] = w.Length()
	}

	out := make([]*CausalNode, len(ab.confirmedPrefix))
	copy(out, ab.confirmedPrefix)
	return out, nil
}

// LinearizedValues returns all linearized message payloads.
func (ab *Autobase) LinearizedValues() ([][]byte, error) {
	nodes, err := ab.Linearize()
	if err != nil {
		return nil, err
	}

	values := make([][]byte, len(nodes))
	for i, n := range nodes {
		values[i] = n.Value
	}
	return values, nil
}

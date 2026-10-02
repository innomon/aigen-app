package p2p

import (
	"fmt"
	"sync"

	"sqlite-p2p/internal/store"

	"go-pear/pkg/autobase"
	"go-pear/pkg/hypercore"
)

// AutobaseManager orchestrates causal multi-writer linearization of changesets.
type AutobaseManager struct {
	mu       sync.RWMutex
	base     *autobase.Autobase
	local    *ChangesetFeed
	writers  map[[32]byte]*hypercore.Hypercore
}

// NewAutobaseManager creates an AutobaseManager backed by a local ChangesetFeed.
func NewAutobaseManager(localFeed *ChangesetFeed) *AutobaseManager {
	ab := autobase.New()
	ab.AddWriter(localFeed.Core(), true)

	mgr := &AutobaseManager{
		base:    ab,
		local:   localFeed,
		writers: make(map[[32]byte]*hypercore.Hypercore),
	}
	mgr.writers[localFeed.Key()] = localFeed.Core()
	return mgr
}

// AddRemoteWriter attaches a remote peer Hypercore writer to the consensus group.
func (m *AutobaseManager) AddRemoteWriter(core *hypercore.Hypercore) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.base.AddWriter(core, false)
	m.writers[core.Key] = core
}

// Append wraps and causally commits a changeset via the local Autobase writer.
func (m *AutobaseManager) Append(cs *store.Changeset) (*autobase.CausalNode, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	raw, err := cs.Encode()
	if err != nil {
		return nil, fmt.Errorf("encode changeset: %w", err)
	}

	return m.base.Append(raw)
}

// LinearizeChangesets linearizes all causal nodes across writers and decodes the changesets.
func (m *AutobaseManager) LinearizeChangesets() ([]*store.Changeset, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	values, err := m.base.LinearizedValues()
	if err != nil {
		return nil, fmt.Errorf("autobase linearization failed: %w", err)
	}

	result := make([]*store.Changeset, 0, len(values))
	for _, val := range values {
		cs, err := store.DecodeChangeset(val)
		if err != nil {
			// Skip or continue if raw node is non-changeset payload
			continue
		}
		result = append(result, cs)
	}

	return result, nil
}

// WriterCount returns the number of active writers in the Autobase.
func (m *AutobaseManager) WriterCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.writers)
}

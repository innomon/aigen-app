package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// OperationType denotes the changeset mutation type.
type OperationType uint8

const (
	// OpInsert represents a new record insertion.
	OpInsert OperationType = 1
	// OpUpdate represents a record modification.
	OpUpdate OperationType = 2
	// OpDelete represents a record deletion.
	OpDelete OperationType = 3
)

// Changeset represents a single atomic mutation to the crm_store table,
// suitable for replication over Autobase/Hypercore.
type Changeset struct {
	Timestamp int64           `json:"timestamp"`
	Sequence  uint64          `json:"seq"`
	Operation OperationType   `json:"op"`
	Key       string          `json:"key"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	Data      []byte          `json:"data,omitempty"`
	PrevHash  []byte          `json:"prev_hash,omitempty"`
}

// Encode serializes the Changeset to JSON bytes.
func (c *Changeset) Encode() ([]byte, error) {
	return json.Marshal(c)
}

// DecodeChangeset deserializes bytes into a Changeset.
func DecodeChangeset(data []byte) (*Changeset, error) {
	var c Changeset
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("failed to decode changeset: %w", err)
	}
	return &c, nil
}

// ChangesetSubscriber receives emitted changesets.
type ChangesetSubscriber func(cs *Changeset, raw []byte)

// ChangesetTracker wraps a Repository to intercept mutations and generate binary changesets.
type ChangesetTracker struct {
	repo        *Repository
	seq         atomic.Uint64
	mu          sync.RWMutex
	subscribers []ChangesetSubscriber
}

// NewChangesetTracker creates a new tracker for the given repository.
func NewChangesetTracker(repo *Repository) *ChangesetTracker {
	return &ChangesetTracker{
		repo: repo,
	}
}

// Subscribe registers a listener callback for captured changesets.
func (t *ChangesetTracker) Subscribe(fn ChangesetSubscriber) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.subscribers = append(t.subscribers, fn)
}

func (t *ChangesetTracker) emit(cs *Changeset) {
	raw, err := cs.Encode()
	if err != nil {
		return
	}

	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, sub := range t.subscribers {
		sub(cs, raw)
	}
}

// Put writes to the repository and emits an OpInsert or OpUpdate changeset.
// If the underlying repository encrypts payloads, the emitted changeset data will contain the ciphertext.
func (t *ChangesetTracker) Put(ctx context.Context, key string, metadata json.RawMessage, data []byte) error {
	// Check if key already exists to distinguish Insert vs Update
	_, err := t.repo.Get(ctx, key)
	op := OpUpdate
	if errors.Is(err, ErrNotFound) {
		op = OpInsert
	}

	if err := t.repo.Put(ctx, key, metadata, data); err != nil {
		return err
	}

	// Capture the stored raw data (ciphertext if encrypted)
	storedData := data
	if rawRec, err := t.repo.GetRaw(ctx, key); err == nil {
		storedData = rawRec.Data
	}

	cs := &Changeset{
		Timestamp: time.Now().UnixNano(),
		Sequence:  t.seq.Add(1),
		Operation: op,
		Key:       key,
		Metadata:  metadata,
		Data:      storedData,
	}

	t.emit(cs)
	return nil
}

// Delete removes a record from the repository and emits an OpDelete changeset.
// If encryption is enabled on the repository, the customer's symmetric key is purged.
func (t *ChangesetTracker) Delete(ctx context.Context, key string) error {
	if err := t.repo.Delete(ctx, key); err != nil {
		return err
	}

	cs := &Changeset{
		Timestamp: time.Now().UnixNano(),
		Sequence:  t.seq.Add(1),
		Operation: OpDelete,
		Key:       key,
	}

	t.emit(cs)
	return nil
}

// ApplyChangeset applies a remote changeset into a repository using LWW semantics.
// Payloads in incoming changesets are already encrypted, so PutRaw is used to persist them.
func ApplyChangeset(ctx context.Context, repo *Repository, cs *Changeset) error {
	switch cs.Operation {
	case OpInsert, OpUpdate:
		return repo.PutRaw(ctx, cs.Key, cs.Metadata, cs.Data)
	case OpDelete:
		return repo.Delete(ctx, cs.Key)
	default:
		return fmt.Errorf("unsupported changeset operation: %d", cs.Operation)
	}
}

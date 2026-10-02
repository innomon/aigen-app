package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"sqlite-p2p/internal/store"
)

// ReplicationEngine integrates SQLite storage, Hypercore changeset persistence,
// Autobase consensus, and P2P network replication with LWW conflict resolution.
type ReplicationEngine struct {
	mu         sync.RWMutex
	repo       *store.Repository
	feed       *ChangesetFeed
	replicator *Replicator
	autobase   *AutobaseManager
	timestamps map[string]int64 // Track latest known timestamp per key for LWW resolution
	logger     *slog.Logger
}

// SetLogger attaches a structured logger to the ReplicationEngine.
func (e *ReplicationEngine) SetLogger(l *slog.Logger) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.logger = l
}

// NewReplicationEngine creates and initializes a ReplicationEngine.
func NewReplicationEngine(repo *store.Repository, feed *ChangesetFeed, replicator *Replicator) *ReplicationEngine {
	var ab *AutobaseManager
	if feed != nil {
		ab = NewAutobaseManager(feed)
	}

	return &ReplicationEngine{
		repo:       repo,
		feed:       feed,
		replicator: replicator,
		autobase:   ab,
		timestamps: make(map[string]int64),
	}
}


// PutLocal performs a local write to SQLite and appends the resulting changeset to Hypercore and broadcasts it.
func (e *ReplicationEngine) PutLocal(ctx context.Context, key string, metadata json.RawMessage, data []byte) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check insert vs update
	_, err := e.repo.Get(ctx, key)
	op := store.OpUpdate
	if err == store.ErrNotFound {
		op = store.OpInsert
	}

	if err := e.repo.Put(ctx, key, metadata, data); err != nil {
		return fmt.Errorf("local put failed: %w", err)
	}

	// Capture the stored raw data (ciphertext if repository is encryption-aware)
	storedData := data
	if rawRec, err := e.repo.GetRaw(ctx, key); err == nil {
		storedData = rawRec.Data
	}

	now := time.Now().UnixNano()
	e.timestamps[key] = now

	cs := &store.Changeset{
		Timestamp: now,
		Sequence:  uint64(now),
		Operation: op,
		Key:       key,
		Metadata:  metadata,
		Data:      storedData,
	}

	if e.autobase != nil {
		_, _ = e.autobase.Append(cs)
	} else if e.feed != nil {
		_, _ = e.feed.Append(cs)
	}
	if e.replicator != nil {
		_ = e.replicator.BroadcastChangeset(cs)
	}

	if e.logger != nil {
		e.logger.Info("local record stored and changeset appended",
			"event", "local_mutation_applied",
			"key", key,
			"operation", op,
			"timestamp", now,
		)
	}

	return nil
}

// DeleteLocal removes a record locally and broadcasts an OpDelete changeset.
func (e *ReplicationEngine) DeleteLocal(ctx context.Context, key string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if err := e.repo.Delete(ctx, key); err != nil {
		return fmt.Errorf("local delete failed: %w", err)
	}

	now := time.Now().UnixNano()
	e.timestamps[key] = now

	cs := &store.Changeset{
		Timestamp: now,
		Sequence:  uint64(now),
		Operation: store.OpDelete,
		Key:       key,
	}

	if e.autobase != nil {
		_, _ = e.autobase.Append(cs)
	} else if e.feed != nil {
		_, _ = e.feed.Append(cs)
	}
	if e.replicator != nil {
		_ = e.replicator.BroadcastChangeset(cs)
	}

	if e.logger != nil {
		e.logger.Info("local record deleted and changeset appended",
			"event", "local_delete_applied",
			"key", key,
			"timestamp", now,
		)
	}

	return nil
}

// ApplyRemoteChangeset applies a remote changeset using Last-Writer-Wins (LWW) conflict resolution.
func (e *ReplicationEngine) ApplyRemoteChangeset(ctx context.Context, cs *store.Changeset) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check if local timestamp is newer (LWW conflict resolution)
	if existingTS, ok := e.timestamps[cs.Key]; ok {
		if existingTS >= cs.Timestamp {
			// Local state is newer or equal; skip applying obsolete changeset
			if e.logger != nil {
				e.logger.Info("obsolete changeset dropped by LWW conflict resolution",
					"event", "lww_changeset_skipped",
					"key", cs.Key,
					"local_timestamp", existingTS,
					"remote_timestamp", cs.Timestamp,
				)
			}
			return nil
		}
	}

	// Apply mutation to SQLite repository
	switch cs.Operation {
	case store.OpInsert, store.OpUpdate:
		if err := e.repo.PutRaw(ctx, cs.Key, cs.Metadata, cs.Data); err != nil {
			return fmt.Errorf("apply remote changeset failed: %w", err)
		}
	case store.OpDelete:
		if err := e.repo.Delete(ctx, cs.Key); err != nil && err != store.ErrNotFound {
			return fmt.Errorf("apply remote delete failed: %w", err)
		}
	default:
		return fmt.Errorf("unknown changeset operation: %v", cs.Operation)
	}

	// Update latest known timestamp
	e.timestamps[cs.Key] = cs.Timestamp

	// Append to local feed history
	if e.feed != nil {
		_, _ = e.feed.Append(cs)
	}

	if e.logger != nil {
		e.logger.Info("remote changeset applied to local state",
			"event", "remote_changeset_applied",
			"key", cs.Key,
			"operation", cs.Operation,
			"timestamp", cs.Timestamp,
		)
	}

	return nil
}


// Feed returns the underlying ChangesetFeed.
func (e *ReplicationEngine) Feed() *ChangesetFeed {
	return e.feed
}

// Autobase returns the AutobaseManager instance.
func (e *ReplicationEngine) Autobase() *AutobaseManager {
	return e.autobase
}

// SyncPeers replays the entire local feed from sequence 0 and broadcasts each changeset
// to all active peer connections. Returns the count of changesets broadcasted.
func (e *ReplicationEngine) SyncPeers(ctx context.Context) (int, error) {
	if e.feed == nil || e.replicator == nil {
		return 0, nil
	}

	count := 0
	err := e.feed.Replay(0, func(cs *store.Changeset) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := e.replicator.BroadcastChangeset(cs); err != nil {
			return err
		}
		count++
		return nil
	})

	if err != nil {
		if e.logger != nil {
			e.logger.Error("peer sync failed during feed replay broadcast",
				"event", "sync_peers_failed",
				"error", err,
				"broadcast_count", count,
			)
		}
		return count, err
	}

	if e.logger != nil {
		e.logger.Info("peer sync sweep completed",
			"event", "sync_peers_completed",
			"broadcast_count", count,
			"peer_count", e.replicator.PeerCount(),
		)
	}

	return count, nil
}


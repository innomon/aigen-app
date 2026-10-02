package p2p

import (
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"

	"sqlite-p2p/internal/store"
)

// ChangesetHandler is invoked when a remote changeset is received over replication.
type ChangesetHandler func(cs *store.Changeset) error

// Replicator manages streaming changesets over peer connections.
type Replicator struct {
	mu       sync.RWMutex
	feed     *ChangesetFeed
	handler  ChangesetHandler
	peers    map[net.Conn]struct{}
	stopChan chan struct{}
	logger   *slog.Logger
}

// SetLogger attaches a structured logger to the Replicator.
func (r *Replicator) SetLogger(l *slog.Logger) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logger = l
}

// NewReplicator creates a new Replicator instance.
func NewReplicator(feed *ChangesetFeed, handler ChangesetHandler) *Replicator {
	return &Replicator{
		feed:     feed,
		handler:  handler,
		peers:    make(map[net.Conn]struct{}),
		stopChan: make(chan struct{}),
	}
}


// SetHandler updates the incoming changeset handler callback.
func (r *Replicator) SetHandler(handler ChangesetHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handler = handler
}

// AddPeer registers an active peer connection for bidirectional replication.
func (r *Replicator) AddPeer(conn net.Conn) {
	r.mu.Lock()
	r.peers[conn] = struct{}{}
	r.mu.Unlock()
}

// RemovePeer unregisters an active peer connection.
func (r *Replicator) RemovePeer(conn net.Conn) {
	r.mu.Lock()
	delete(r.peers, conn)
	r.mu.Unlock()
}

// HandleConnection takes ownership of a peer connection and reads incoming changesets.
func (r *Replicator) HandleConnection(conn net.Conn) {
	r.AddPeer(conn)
	defer func() {
		r.RemovePeer(conn)
		_ = conn.Close()
	}()

	for {
		select {
		case <-r.stopChan:
			return
		default:
		}

		// Read length-prefixed payload (4-byte big endian)
		var length uint32
		if err := binary.Read(conn, binary.BigEndian, &length); err != nil {
			return
		}

		if length == 0 || length > 10*1024*1024 { // max 10MB per changeset
			return
		}

		buf := make([]byte, length)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return
		}

		cs, err := store.DecodeChangeset(buf)
		if err != nil {
			if r.logger != nil {
				r.logger.Warn("failed to decode replication changeset",
					"event", "changeset_decode_failed",
					"error", err,
					"payload_len", len(buf),
				)
			}
			continue
		}

		if r.logger != nil {
			r.logger.Info("changeset received over replication",
				"event", "changeset_received",
				"key", cs.Key,
				"operation", cs.Operation,
				"timestamp", cs.Timestamp,
			)
		}

		// Record to local feed if not already present
		if r.feed != nil {
			_, _ = r.feed.Append(cs)
		}

		// Dispatch to handler (e.g. Autobase / SQLite apply)
		if r.handler != nil {
			if err := r.handler(cs); err != nil && r.logger != nil {
				r.logger.Warn("changeset handler error",
					"event", "changeset_handler_error",
					"key", cs.Key,
					"error", err,
				)
			}
		}
	}
}

// BroadcastChangeset sends a changeset to all active peer connections.
func (r *Replicator) BroadcastChangeset(cs *store.Changeset) error {
	raw, err := cs.Encode()
	if err != nil {
		if r.logger != nil {
			r.logger.Error("failed to encode changeset for broadcast",
				"event", "changeset_encode_failed",
				"key", cs.Key,
				"error", err,
			)
		}
		return fmt.Errorf("encode changeset: %w", err)
	}

	length := uint32(len(raw))
	msg := make([]byte, 4+len(raw))
	binary.BigEndian.PutUint32(msg[:4], length)
	copy(msg[4:], raw)

	r.mu.RLock()
	peers := make([]net.Conn, 0, len(r.peers))
	for p := range r.peers {
		peers = append(peers, p)
	}
	l := r.logger
	r.mu.RUnlock()

	for _, p := range peers {
		_, _ = p.Write(msg)
	}

	if l != nil {
		l.Info("changeset broadcast to peers",
			"event", "changeset_broadcast",
			"key", cs.Key,
			"operation", cs.Operation,
			"peer_count", len(peers),
		)
	}

	return nil
}


// PeerCount returns the count of active replication channels.
func (r *Replicator) PeerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.peers)
}

// Close stops replication and closes all connections.
func (r *Replicator) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	select {
	case <-r.stopChan:
		return nil
	default:
		close(r.stopChan)
	}

	for p := range r.peers {
		_ = p.Close()
	}
	r.peers = make(map[net.Conn]struct{})
	return nil
}

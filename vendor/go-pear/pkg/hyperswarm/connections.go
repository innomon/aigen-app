package hyperswarm

import (
	"errors"
	"net"
	"sync"
	"time"
)

var (
	ErrDuplicateConnection = errors.New("hyperswarm: duplicate connection to peer")
)

// PeerConnection represents an active encrypted peer stream in the swarm.
type PeerConnection struct {
	RemotePublicKey [32]byte
	Conn            net.Conn
	IsInitiator     bool
	Topic           [32]byte
	CreatedAt       time.Time
}

// ConnectionSet tracks all active connections in the swarm, preventing duplicates.
type ConnectionSet struct {
	mu          sync.RWMutex
	connections map[[32]byte]*PeerConnection
}

// NewConnectionSet creates an empty connection pool.
func NewConnectionSet() *ConnectionSet {
	return &ConnectionSet{
		connections: make(map[[32]byte]*PeerConnection),
	}
}

// Add adds a connection to the set. Returns error if already connected.
func (cs *ConnectionSet) Add(conn *PeerConnection) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if _, exists := cs.connections[conn.RemotePublicKey]; exists {
		return ErrDuplicateConnection
	}

	cs.connections[conn.RemotePublicKey] = conn
	return nil
}

// Remove removes a connection by peer public key.
func (cs *ConnectionSet) Remove(remotePK [32]byte) {
	cs.mu.Lock()
	delete(cs.connections, remotePK)
	cs.mu.Unlock()
}

// Get returns the active connection for a peer public key if present.
func (cs *ConnectionSet) Get(remotePK [32]byte) (*PeerConnection, bool) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	conn, ok := cs.connections[remotePK]
	return conn, ok
}

// List returns a snapshot of all active connections.
func (cs *ConnectionSet) List() []*PeerConnection {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	res := make([]*PeerConnection, 0, len(cs.connections))
	for _, c := range cs.connections {
		res = append(res, c)
	}
	return res
}

// Size returns total count of active connections.
func (cs *ConnectionSet) Size() int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return len(cs.connections)
}

// Close closes all active connections in the set.
func (cs *ConnectionSet) Close() {
	cs.mu.Lock()
	for pk, c := range cs.connections {
		_ = c.Conn.Close()
		delete(cs.connections, pk)
	}
	cs.mu.Unlock()
}

package p2p

import (
	"context"
	"database/sql"
	"time"

	"sqlite-p2p/internal/crypto"
	internalp2p "sqlite-p2p/internal/p2p"
	"sqlite-p2p/internal/store"
)

// ErrNotFound indicates a requested key does not exist.
var ErrNotFound = store.ErrNotFound

// Record aliases the underlying storage record.
type Record = store.Record

// Repository aliases the underlying generic KV repository.
type Repository = store.Repository

// ChangesetTracker aliases the mutation capture tracker.
type ChangesetTracker = store.ChangesetTracker

// Changeset aliases the binary SQLite changeset structure.
type Changeset = store.Changeset

// OperationType aliases the operation type.
type OperationType = store.OperationType

const (
	OpInsert = store.OpInsert
	OpUpdate = store.OpUpdate
	OpDelete = store.OpDelete
)

// DecodeChangeset deserializes bytes into a Changeset.
func DecodeChangeset(data []byte) (*Changeset, error) {
	return store.DecodeChangeset(data)
}

// ApplyChangeset applies a remote changeset to the repository.
func ApplyChangeset(ctx context.Context, repo *Repository, cs *Changeset) error {
	return store.ApplyChangeset(ctx, repo, cs)
}

// KeyRegistry aliases the per-key AES-256-GCM registry.
type KeyRegistry = crypto.KeyRegistry

// ReplicationEngine aliases the P2P synchronization coordinator.
type ReplicationEngine = internalp2p.ReplicationEngine

// OpenDB opens a pure Go SQLite database with WAL and foreign key PRAGMAs.
func OpenDB(dbPath string, enableWAL bool) (*sql.DB, error) {
	return store.OpenDB(dbPath, enableWAL)
}

// NewRepository initializes a Repository on the provided sql.DB.
func NewRepository(db *sql.DB) *Repository {
	return store.NewRepository(db)
}

// NewChangesetTracker initializes a ChangesetTracker for the repository.
func NewChangesetTracker(repo *Repository) *ChangesetTracker {
	return store.NewChangesetTracker(repo)
}

// NewKeyRegistry initializes a cryptographic KeyRegistry on the provided sql.DB.
func NewKeyRegistry(db *sql.DB) (*KeyRegistry, error) {
	return crypto.NewKeyRegistry(db)
}

// SwarmManager aliases the peer discovery and connection coordinator.
type SwarmManager = internalp2p.SwarmManager

// SwarmManagerOptions aliases the swarm configuration options.
type SwarmManagerOptions = internalp2p.SwarmManagerOptions

// NewSwarmManager initializes a SwarmManager with the provided options.
func NewSwarmManager(opts SwarmManagerOptions) (*SwarmManager, error) {
	return internalp2p.NewSwarmManager(opts)
}

// ChangesetFeed aliases the append-only changeset feed.
type ChangesetFeed = internalp2p.ChangesetFeed

// NewChangesetFeed initializes or opens a ChangesetFeed at dir.
func NewChangesetFeed(dir string) (*ChangesetFeed, error) {
	return internalp2p.NewChangesetFeed(dir)
}

// Replicator aliases the replication state machine.
type Replicator = internalp2p.Replicator

// ChangesetHandler aliases the callback for incoming remote changesets.
type ChangesetHandler = internalp2p.ChangesetHandler

// NewReplicator initializes a Replicator.
func NewReplicator(feed *ChangesetFeed, handler ChangesetHandler) *Replicator {
	return internalp2p.NewReplicator(feed, handler)
}

// NewReplicationEngine initializes a ReplicationEngine.
func NewReplicationEngine(repo *Repository, feed *ChangesetFeed, replicator *Replicator) *ReplicationEngine {
	return internalp2p.NewReplicationEngine(repo, feed, replicator)
}

// DefaultBeaconPort is the standard UDP broadcast port for local cluster discovery.
const DefaultBeaconPort = internalp2p.DefaultBeaconPort

// BeaconMessage aliases the local broadcast beacon message.
type BeaconMessage = internalp2p.BeaconMessage

// StartBeaconBroadcaster starts periodic LAN discovery broadcast.
func StartBeaconBroadcaster(ctx context.Context, msg BeaconMessage, port int, interval time.Duration) error {
	return internalp2p.StartBeaconBroadcaster(ctx, msg, port, interval)
}

// DiscoverLANBootstrap listens for a LAN beacon matching cluster/topic.
func DiscoverLANBootstrap(ctx context.Context, cluster, topic string, port int, timeout time.Duration) (string, error) {
	return internalp2p.DiscoverLANBootstrap(ctx, cluster, topic, port, timeout)
}

// NormalizeDHTAddr resolves an unspecified IP to a routable LAN IP.
func NormalizeDHTAddr(addr string) string {
	return internalp2p.NormalizeDHTAddr(addr)
}


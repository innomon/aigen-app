package hypercore

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"go-pear/pkg/protomux"
)

// KeyPair represents Ed25519 public and secret keys for a Hypercore.
type KeyPair struct {
	PublicKey [32]byte
	SecretKey [64]byte
}

// GenerateKeyPair generates a new random Ed25519 keypair.
func GenerateKeyPair() (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	var kp KeyPair
	copy(kp.PublicKey[:], pub)
	copy(kp.SecretKey[:], priv)
	return &kp, nil
}

// Hypercore represents an append-only log core instance with Merkle validation and replication.
type Hypercore struct {
	mu           sync.RWMutex
	Key          [32]byte
	DiscoveryKey [32]byte
	KeyPair      *KeyPair
	Tree         *Tree
	Bitfield     *Bitfield
	Manifest     *Manifest
	StorageDir   string
	blocks       map[uint64][]byte
	peers        []*ReplicatorPeer
}

// New creates a new Hypercore with keypair and manifest.
func New(kp *KeyPair, m *Manifest) *Hypercore {
	return NewWithStorage(kp, m, "")
}

// NewWithStorage creates a new Hypercore with keypair, manifest, and optional persistence directory.
func NewWithStorage(kp *KeyPair, m *Manifest, storageDir string) *Hypercore {
	if kp == nil {
		kp, _ = GenerateKeyPair()
	}
	var key [32]byte
	if kp != nil {
		key = kp.PublicKey
	}

	dk := DiscoveryKey(key)

	if m == nil {
		m = &Manifest{
			Version: 0,
			Hash:    "blake2b",
			Quorum:  1,
			Signers: []Signer{
				{
					Signature: "ed25519",
					Namespace: DefaultNamespace,
					PublicKey: key,
				},
			},
		}
	}

	core := &Hypercore{
		Key:          key,
		DiscoveryKey: dk,
		KeyPair:      kp,
		Tree:         NewTree(),
		Bitfield:     NewBitfield(),
		Manifest:     m,
		StorageDir:   storageDir,
		blocks:       make(map[uint64][]byte),
		peers:        make([]*ReplicatorPeer, 0),
	}

	if storageDir != "" {
		core.loadStorage()
	}

	return core
}

func (h *Hypercore) loadStorage() {
	files, err := os.ReadDir(h.StorageDir)
	if err != nil {
		return
	}
	type bFile struct {
		idx  int
		path string
	}
	var blks []bFile
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".block") {
			var idx int
			if _, err := fmt.Sscanf(f.Name(), "%d.block", &idx); err == nil {
				blks = append(blks, bFile{idx: idx, path: filepath.Join(h.StorageDir, f.Name())})
			}
		}
	}
	sort.Slice(blks, func(i, j int) bool { return blks[i].idx < blks[j].idx })
	for _, b := range blks {
		data, err := os.ReadFile(b.path)
		if err == nil {
			_, _ = h.Append(data)
		}
	}
}

// Length returns the total block count.
func (h *Hypercore) Length() uint64 {
	return h.Tree.Length()
}

// ByteLength returns total byte size.
func (h *Hypercore) ByteLength() uint64 {
	return h.Tree.ByteLength()
}

// Append appends a data block, updates Merkle tree and bitfield, and notifies peers.
func (h *Hypercore) Append(data []byte) (*Node, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	idx := h.Tree.Length()
	node, err := h.Tree.Append(data)
	if err != nil {
		return nil, err
	}

	h.blocks[idx] = append([]byte(nil), data...)
	h.Bitfield.Set(idx, true)

	if h.StorageDir != "" {
		_ = os.MkdirAll(h.StorageDir, 0755)
		_ = os.WriteFile(filepath.Join(h.StorageDir, fmt.Sprintf("%d.block", idx)), data, 0644)
	}

	// Broadcast range announcement to active peers
	for _, p := range h.peers {
		_ = p.SendRange(idx, 1, false)
	}

	return node, nil
}

// Get retrieves a block by leaf index.
func (h *Hypercore) Get(index uint64) ([]byte, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data, ok := h.blocks[index]
	if !ok {
		return nil, ErrBlockNotFound
	}
	res := make([]byte, len(data))
	copy(res, data)
	return res, nil
}

// Put saves a verified data block and updates the tree if not already present.
func (h *Hypercore) Put(index uint64, data []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.blocks[index] = append([]byte(nil), data...)
	h.Bitfield.Set(index, true)
}

// AttachReplicator binds the core to a Protomux channel multiplexer.
func (h *Hypercore) AttachReplicator(mux *protomux.Multiplexer, isInitiator bool, handshakeHash [32]byte) (*ReplicatorPeer, error) {
	peer, err := NewReplicatorPeer(h, mux, isInitiator, handshakeHash)
	if err != nil {
		return nil, err
	}

	h.mu.Lock()
	h.peers = append(h.peers, peer)
	h.mu.Unlock()

	return peer, nil
}

package hyperdht

import (
	"bytes"
	"math/bits"
	"net"
	"sort"
	"sync"
	"time"
)

const (
	// K is the Kademlia bucket capacity.
	K = 20
	// NumBuckets is the number of buckets in the routing table (256 bits).
	NumBuckets = 256
)

// PeerEndpoint represents a known node on the HyperDHT network.
type PeerEndpoint struct {
	ID        [32]byte
	Addr      *net.UDPAddr
	PublicKey [32]byte
	LastSeen  time.Time
}

// Distance computes the XOR metric distance between two 32-byte IDs.
func Distance(a, b [32]byte) [32]byte {
	var dist [32]byte
	for i := 0; i < 32; i++ {
		dist[i] = a[i] ^ b[i]
	}
	return dist
}

// CompareDistance compares distance from a to target vs distance from b to target.
// Returns -1 if a is closer to target, 1 if b is closer, 0 if equal.
func CompareDistance(a, b, target [32]byte) int {
	da := Distance(a, target)
	db := Distance(b, target)
	return bytes.Compare(da[:], db[:])
}

// BucketIndex returns the k-bucket index (0..255) for target relative to localID.
func BucketIndex(localID, targetID [32]byte) int {
	dist := Distance(localID, targetID)
	for i := 0; i < 32; i++ {
		b := dist[i]
		if b != 0 {
			lz := bits.LeadingZeros8(b)
			return (i * 8) + lz
		}
	}
	return NumBuckets - 1
}

// RoutingTable manages Kademlia k-buckets indexed by XOR distance.
type RoutingTable struct {
	mu      sync.RWMutex
	localID [32]byte
	buckets [NumBuckets][]*PeerEndpoint
}

// NewRoutingTable creates an empty routing table for the given local ID.
func NewRoutingTable(localID [32]byte) *RoutingTable {
	return &RoutingTable{
		localID: localID,
	}
}

// Add inserts or updates a peer in the appropriate k-bucket.
func (rt *RoutingTable) Add(peer *PeerEndpoint) bool {
	if peer == nil || peer.ID == rt.localID {
		return false
	}

	rt.mu.Lock()
	defer rt.mu.Unlock()

	idx := BucketIndex(rt.localID, peer.ID)
	bucket := rt.buckets[idx]

	for i, existing := range bucket {
		if existing.ID == peer.ID {
			// Update address and timestamp, move to tail
			existing.Addr = peer.Addr
			existing.LastSeen = time.Now()
			rt.buckets[idx] = append(append(bucket[:i], bucket[i+1:]...), existing)
			return true
		}
	}

	if len(bucket) < K {
		peer.LastSeen = time.Now()
		rt.buckets[idx] = append(bucket, peer)
		return true
	}

	// Bucket is full
	return false
}

// Remove removes a peer from the routing table.
func (rt *RoutingTable) Remove(id [32]byte) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	idx := BucketIndex(rt.localID, id)
	bucket := rt.buckets[idx]
	for i, peer := range bucket {
		if peer.ID == id {
			rt.buckets[idx] = append(bucket[:i], bucket[i+1:]...)
			return
		}
	}
}

// Closest returns the count closest known peers to target.
func (rt *RoutingTable) Closest(target [32]byte, count int) []*PeerEndpoint {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	var all []*PeerEndpoint
	for _, bucket := range rt.buckets {
		for _, peer := range bucket {
			all = append(all, peer)
		}
	}

	sort.Slice(all, func(i, j int) bool {
		return CompareDistance(all[i].ID, all[j].ID, target) < 0
	})

	if len(all) > count {
		all = all[:count]
	}
	return all
}

// Size returns total count of known peers.
func (rt *RoutingTable) Size() int {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	total := 0
	for _, bucket := range rt.buckets {
		total += len(bucket)
	}
	return total
}

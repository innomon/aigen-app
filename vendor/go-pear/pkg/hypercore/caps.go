package hypercore

import (
	"encoding/binary"

	"golang.org/x/crypto/blake2b"
)

var (
	// HypercoreNamespaces are the standard 6 namespaces derived for "hypercore".
	hypercoreNamespaces = Namespace("hypercore", 6)

	// TreeNS is the namespace used for tree root signing.
	TreeNS = hypercoreNamespaces[0]
	// ReplicateInitiatorNS is the replication capability namespace for initiator peers.
	ReplicateInitiatorNS = hypercoreNamespaces[1]
	// ReplicateResponderNS is the replication capability namespace for responder peers.
	ReplicateResponderNS = hypercoreNamespaces[2]
	// ManifestNS is the namespace used for manifest hashing.
	ManifestNS = hypercoreNamespaces[3]
	// DefaultNamespace is the default namespace for signers.
	DefaultNamespace = hypercoreNamespaces[4]
	// DefaultEncryptionNS is the default encryption namespace.
	DefaultEncryptionNS = hypercoreNamespaces[5]
)

// Namespace generates an array of 32-byte namespaced hashes for a given name string.
// Formula: base = BLAKE2b(name) -> for each i: Keyed-BLAKE2b(key=base, input=[byte(i)]).
func Namespace(name string, count int) [][32]byte {
	hBase, _ := blake2b.New256(nil)
	hBase.Write([]byte(name))
	base := hBase.Sum(nil)

	res := make([][32]byte, count)
	for i := 0; i < count; i++ {
		h, _ := blake2b.New256(base)
		h.Write([]byte{byte(i)})
		copy(res[i][:], h.Sum(nil))
	}
	return res
}

// ReplicateCapability calculates the replication capability verification hash:
// BLAKE2b-256(key=handshakeHash, input=[ReplicateInitiatorNS/ReplicateResponderNS || coreKey]).
func ReplicateCapability(isInitiator bool, coreKey [32]byte, handshakeHash [32]byte) [32]byte {
	h, _ := blake2b.New256(handshakeHash[:])
	if isInitiator {
		h.Write(ReplicateInitiatorNS[:])
	} else {
		h.Write(ReplicateResponderNS[:])
	}
	h.Write(coreKey[:])

	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// TreeSignable constructs the 112-byte buffer signed by Hypercore signers:
// TreeNS (32B) || manifestHash (32B) || treeHash (32B) || uint64_le(length) (8B) || uint64_le(fork) (8B).
func TreeSignable(manifestHash, treeHash [32]byte, length, fork uint64) []byte {
	buf := make([]byte, 112)
	copy(buf[0:32], TreeNS[:])
	copy(buf[32:64], manifestHash[:])
	copy(buf[64:96], treeHash[:])
	binary.LittleEndian.PutUint64(buf[96:104], length)
	binary.LittleEndian.PutUint64(buf[104:112], fork)
	return buf
}

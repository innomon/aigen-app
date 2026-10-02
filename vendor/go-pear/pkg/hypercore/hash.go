package hypercore

import (
	"encoding/binary"

	"golang.org/x/crypto/blake2b"
)

const (
	// LeafPrefix is the 1-byte domain separator prefix for Merkle leaf hashes.
	LeafPrefix byte = 0x00
	// ParentPrefix is the 1-byte domain separator prefix for Merkle parent hashes.
	ParentPrefix byte = 0x01
	// RootPrefix is the 1-byte domain separator prefix for Merkle peak roots hash.
	RootPrefix byte = 0x02
)

// HypercoreKey is the BLAKE2b key used to derive discovery keys.
var HypercoreKey = []byte("hypercore")

// LeafHash calculates the BLAKE2b-256 hash of a leaf data block:
// BLAKE2b(0x00 || uint64_le(len(data)) || data).
func LeafHash(data []byte) [32]byte {
	h, _ := blake2b.New256(nil)
	h.Write([]byte{LeafPrefix})

	var lenBuf [8]byte
	binary.LittleEndian.PutUint64(lenBuf[:], uint64(len(data)))
	h.Write(lenBuf[:])

	h.Write(data)
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// ParentHash calculates the BLAKE2b-256 hash of a parent node:
// BLAKE2b(0x01 || uint64_le(size) || left_hash || right_hash).
func ParentHash(size uint64, leftHash, rightHash [32]byte) [32]byte {
	h, _ := blake2b.New256(nil)
	h.Write([]byte{ParentPrefix})

	var szBuf [8]byte
	binary.LittleEndian.PutUint64(szBuf[:], size)
	h.Write(szBuf[:])

	h.Write(leftHash[:])
	h.Write(rightHash[:])

	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// TreeHash calculates the BLAKE2b-256 hash of the peak Merkle roots:
// BLAKE2b(0x02 || for each root: [root.hash || uint64_le(root.index) || uint64_le(root.size)]).
func TreeHash(roots []*Node) [32]byte {
	h, _ := blake2b.New256(nil)
	h.Write([]byte{RootPrefix})

	var numBuf [8]byte
	for _, r := range roots {
		h.Write(r.Hash[:])

		binary.LittleEndian.PutUint64(numBuf[:], r.Index)
		h.Write(numBuf[:])

		binary.LittleEndian.PutUint64(numBuf[:], r.Size)
		h.Write(numBuf[:])
	}

	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// DiscoveryKey derives the public 32-byte discovery key from a 32-byte public key:
// BLAKE2b-256(key="hypercore", input=publicKey).
func DiscoveryKey(publicKey [32]byte) [32]byte {
	h, _ := blake2b.New256(HypercoreKey)
	h.Write(publicKey[:])
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

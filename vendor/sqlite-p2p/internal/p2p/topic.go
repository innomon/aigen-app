package p2p

import (
	"crypto/sha256"
)

// DeriveTopic hashes a topic string into a 32-byte cryptographic cluster key.
func DeriveTopic(topic string) [32]byte {
	return sha256.Sum256([]byte(topic))
}

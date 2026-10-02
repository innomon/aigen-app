// Package secretstream provides Noise XX authenticated encrypted transport streams.
package secretstream

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
)

const (
	// ProtocolName is the Noise protocol string for Holepunch secret streams.
	ProtocolName = "Noise_XX_25519_ChaChaPoly_BLAKE2b"
	// KeySize is the Curve25519 / ChaCha20 key size in bytes.
	KeySize = 32
	// TagSize is the Poly1305 authentication tag size in bytes.
	TagSize = 16
	// MaxFrameSize is the maximum unencrypted payload size per frame (64KB - 1).
	MaxFrameSize = 65535 - TagSize
)

var (
	// ErrHandshakeFailed is returned when handshake validation or decryption fails.
	ErrHandshakeFailed = errors.New("secretstream: handshake failed")
	// ErrConnectionClosed is returned when operating on a closed connection.
	ErrConnectionClosed = errors.New("secretstream: connection closed")
)

// KeyPair represents a Curve25519 public/private keypair.
type KeyPair struct {
	Public  [KeySize]byte
	Private [KeySize]byte
}

// GenerateKeyPair generates a random Curve25519 keypair.
func GenerateKeyPair() (*KeyPair, error) {
	var priv [KeySize]byte
	if _, err := io.ReadFull(rand.Reader, priv[:]); err != nil {
		return nil, fmt.Errorf("failed to read random bytes: %w", err)
	}
	var pub [KeySize]byte
	curve25519.ScalarBaseMult(&pub, &priv)
	return &KeyPair{Public: pub, Private: priv}, nil
}

// blake2bHash returns the 32-byte BLAKE2b-256 hash of data.
func blake2bHash(data []byte) [32]byte {
	return blake2b.Sum256(data)
}

// blake2bKeyed returns the 32-byte keyed BLAKE2b hash.
func blake2bKeyed(key, data []byte) [32]byte {
	h, _ := blake2b.New256(key)
	h.Write(data)
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// HKDF2 derives two 32-byte keys from chaining key and input key material.
// Per RFC 5869 Section 2.3 and Noise Specification Section 4.1:
// T(1) = HMAC-Hash(PRK, info || 0x01)
// T(2) = HMAC-Hash(PRK, T(1) || info || 0x02)
func HKDF2(ck [32]byte, ikm []byte) ([32]byte, [32]byte) {
	temp := blake2bKeyed(ck[:], ikm)
	out1 := blake2bKeyed(temp[:], []byte{0x01})
	in2 := append(out1[:], 0x02)
	out2 := blake2bKeyed(temp[:], in2)
	return out1, out2
}

// CipherState tracks AEAD encryption key and auto-incrementing nonce.
type CipherState struct {
	aead   cipher.AEAD
	nonce  uint64
	hasKey bool
}

// NewCipherState creates a CipherState from a 32-byte key.
func NewCipherState(key []byte) (*CipherState, error) {
	if len(key) == 0 {
		return &CipherState{hasKey: false}, nil
	}
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, err
	}
	return &CipherState{aead: aead, nonce: 0, hasKey: true}, nil
}

func (cs *CipherState) NonceBytes() []byte {
	var n [12]byte
	binary.LittleEndian.PutUint64(n[4:], cs.nonce)
	return n[:]
}

// EncryptWithAd encrypts plaintext with authenticating additional data.
func (cs *CipherState) EncryptWithAd(ad, plaintext []byte) []byte {
	if !cs.hasKey {
		return plaintext
	}
	if cs.nonce == math.MaxUint64 {
		panic("secretstream: nonce overflow, rekey required")
	}
	nonce := cs.NonceBytes()
	out := cs.aead.Seal(nil, nonce, plaintext, ad)
	cs.nonce++
	return out
}

// DecryptWithAd decrypts ciphertext with authenticating additional data.
func (cs *CipherState) DecryptWithAd(ad, ciphertext []byte) ([]byte, error) {
	if !cs.hasKey {
		return ciphertext, nil
	}
	if cs.nonce == math.MaxUint64 {
		return nil, errors.New("secretstream: nonce overflow, rekey required")
	}
	nonce := cs.NonceBytes()
	out, err := cs.aead.Open(nil, nonce, ciphertext, ad)
	if err != nil {
		return nil, err
	}
	cs.nonce++
	return out, nil
}

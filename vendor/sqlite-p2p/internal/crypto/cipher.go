// Package crypto provides symmetric encryption and key management primitives
// for securing customer data payloads and enabling cryptographic shredding.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

const (
	// KeySize is the expected byte length of an AES-256 key (32 bytes).
	KeySize = 32

	// NonceSize is the standard GCM nonce size in bytes (12 bytes).
	NonceSize = 12
)

var (
	// ErrInvalidKeySize is returned when an AES key does not match KeySize.
	ErrInvalidKeySize = errors.New("crypto: invalid key size, must be 32 bytes for AES-256")

	// ErrCiphertextTooShort is returned when ciphertext is shorter than NonceSize.
	ErrCiphertextTooShort = errors.New("crypto: ciphertext too short to contain nonce")

	// ErrDecryptionFailed is returned when authentication or decryption fails.
	ErrDecryptionFailed = errors.New("crypto: authenticated decryption failed")
)

// GenerateKey generates a cryptographically secure 256-bit (32-byte) symmetric key.
func GenerateKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}
	return key, nil
}

// Encrypt encrypts plaintext using AES-256-GCM with a freshly generated 12-byte nonce.
// The nonce is prefixed to the resulting ciphertext: [nonce (12 bytes) || ciphertext || tag (16 bytes)].
func Encrypt(key []byte, plaintext []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM AEAD: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to read random nonce: %w", err)
	}

	// Seal appends the ciphertext and tag to nonce
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt authenticates and decrypts an AES-256-GCM ciphertext created by Encrypt.
// It extracts the 12-byte nonce from the prefix and decrypts the remaining payload.
func Decrypt(key []byte, ciphertext []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}

	if len(ciphertext) < NonceSize {
		return nil, ErrCiphertextTooShort
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM AEAD: %w", err)
	}

	nonce := ciphertext[:gcm.NonceSize()]
	payload := ciphertext[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, payload, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

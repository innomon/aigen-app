package ids

import (
	"crypto/rand"
	"encoding/base32"
	"time"
)

// encoding matches the length of ULID (26 chars) when encoding 16 bytes with NoPadding
var encoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// NewID generates a sortable UUID v7 encoded in Base32.
// It is a drop-in replacement for ULID.
func NewID() string {
	uuid := make([]byte, 16)
	_, _ = rand.Read(uuid)

	// UUID v7: 48-bit timestamp
	now := uint64(time.Now().UnixMilli())
	uuid[0] = byte(now >> 40)
	uuid[1] = byte(now >> 32)
	uuid[2] = byte(now >> 24)
	uuid[3] = byte(now >> 16)
	uuid[4] = byte(now >> 8)
	uuid[5] = byte(now)

	// Version 7 (0111) in bits 4-7 of byte 6
	uuid[6] = (uuid[6] & 0x0f) | 0x70
	// Variant 10xxxxxx in byte 8
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return encoding.EncodeToString(uuid)
}

// NewRandomID generates a random UUID v4 encoded in Base32.
// It is a drop-in replacement for NanoID.
func NewRandomID() string {
	uuid := make([]byte, 16)
	_, _ = rand.Read(uuid)

	// Version 4 (0100) in bits 4-7 of byte 6
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	// Variant 10xxxxxx in byte 8
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return encoding.EncodeToString(uuid)
}

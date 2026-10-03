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

// NewRandomInt64ID generates a positive random int64 id within the safe 53-bit JSON integer range (up to 2^53-1).
func NewRandomInt64ID() int64 {
	var b [8]byte
	_, _ = rand.Read(b[:])
	// Mask to 53 bits (0x001F_FFFF_FFFF_FFFF) for IEEE-754 float64 safety
	val := int64(b[0]&0x1f)<<48 | int64(b[1])<<40 | int64(b[2])<<32 |
		int64(b[3])<<24 | int64(b[4])<<16 | int64(b[5])<<8 | int64(b[6])
	if val <= 0 {
		return time.Now().UnixNano() & 0x001FFFFFFFFFFFFF
	}
	return val
}


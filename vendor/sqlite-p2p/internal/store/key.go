package store

import (
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// CustomerNamespace is the standardized prefix for customer records.
const CustomerNamespace = "in.qzip.crm.customer"

var base32NoPadding = base32.StdEncoding.WithPadding(base32.NoPadding)

// NormalizePhoneNumber strips formatting characters and country code prefix (e.g. +91)
// returning the canonical 10-digit national number or digits.
func NormalizePhoneNumber(phone string) (string, error) {
	var digits strings.Builder
	for _, r := range phone {
		if unicode.IsDigit(r) {
			digits.WriteRune(r)
		}
	}

	d := digits.String()
	if len(d) == 0 {
		return "", errors.New("empty phone number")
	}

	// If phone starts with Indian country code 91 and has 12 digits, strip 91
	if len(d) == 12 && strings.HasPrefix(d, "91") {
		d = d[2:]
	} else if len(d) > 10 && strings.HasPrefix(d, "0") {
		d = strings.TrimLeft(d, "0")
	}

	if len(d) < 7 {
		return "", fmt.Errorf("phone number too short after normalization (length %d)", len(d))
	}

	return d, nil
}

// HashIdentifier computes the SHA-256 hash of an identifier and encodes it to Base32 (no padding).
func HashIdentifier(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return base32NoPadding.EncodeToString(h[:])
}

// FormatCustomerKey creates a privacy-preserving customer primary key:
// in.qzip.crm.customer:<BASE32(SHA256(normalized_phone))>
func FormatCustomerKey(phone string) (string, error) {
	norm, err := NormalizePhoneNumber(phone)
	if err != nil {
		return "", fmt.Errorf("invalid phone number: %w", err)
	}

	encodedHash := HashIdentifier(norm)
	return fmt.Sprintf("%s:%s", CustomerNamespace, encodedHash), nil
}

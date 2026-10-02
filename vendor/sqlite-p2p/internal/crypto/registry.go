package crypto

import (
	"database/sql"
	"errors"
	"fmt"
)

var (
	// ErrKeyNotFound is returned when a requested key ID does not exist in the registry.
	ErrKeyNotFound = errors.New("crypto: key not found")

	// ErrInvalidKeyID is returned when an empty or invalid key ID is supplied.
	ErrInvalidKeyID = errors.New("crypto: key ID cannot be empty")
)

const keySchemaDDL = `
CREATE TABLE IF NOT EXISTS crm_keys (
    key_id TEXT PRIMARY KEY,
    key_bytes BLOB NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
) WITHOUT ROWID;
`

// KeyRegistry manages persistent symmetric keys stored in SQLite for per-customer payload encryption and crypto-shredding.
type KeyRegistry struct {
	db *sql.DB
}

// NewKeyRegistry initializes the crm_keys table in SQLite if it does not exist and returns a new KeyRegistry.
func NewKeyRegistry(db *sql.DB) (*KeyRegistry, error) {
	if db == nil {
		return nil, errors.New("crypto: db connection cannot be nil")
	}

	if _, err := db.Exec(keySchemaDDL); err != nil {
		return nil, fmt.Errorf("failed to initialize crm_keys table: %w", err)
	}

	return &KeyRegistry{db: db}, nil
}

// GetKey retrieves the 32-byte symmetric key associated with keyID.
// If the key does not exist, ErrKeyNotFound is returned.
func (r *KeyRegistry) GetKey(keyID string) ([]byte, error) {
	if keyID == "" {
		return nil, ErrInvalidKeyID
	}

	query := `SELECT key_bytes FROM crm_keys WHERE key_id = ?;`
	row := r.db.QueryRow(query, keyID)

	var keyBytes []byte
	if err := row.Scan(&keyBytes); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to query key for %s: %w", keyID, err)
	}

	return keyBytes, nil
}

// SetKey stores or updates the symmetric key for keyID.
func (r *KeyRegistry) SetKey(keyID string, key []byte) error {
	if keyID == "" {
		return ErrInvalidKeyID
	}
	if len(key) != KeySize {
		return ErrInvalidKeySize
	}

	query := `
INSERT INTO crm_keys (key_id, key_bytes, created_at)
VALUES (?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(key_id) DO UPDATE SET
    key_bytes = excluded.key_bytes,
    created_at = excluded.created_at;
`
	if _, err := r.db.Exec(query, keyID, key); err != nil {
		return fmt.Errorf("failed to store key for %s: %w", keyID, err)
	}

	return nil
}

// GetOrCreateKey retrieves the key for keyID if it exists, or generates and stores a new key.
func (r *KeyRegistry) GetOrCreateKey(keyID string) ([]byte, error) {
	if keyID == "" {
		return nil, ErrInvalidKeyID
	}

	existing, err := r.GetKey(keyID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrKeyNotFound) {
		return nil, err
	}

	newKey, err := GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key for %s: %w", keyID, err)
	}

	if err := r.SetKey(keyID, newKey); err != nil {
		return nil, err
	}

	return newKey, nil
}

// PurgeKey permanently deletes the symmetric key associated with keyID, executing crypto-shredding.
// All historical ciphertexts encrypted with this key will become permanently undecipherable.
func (r *KeyRegistry) PurgeKey(keyID string) error {
	if keyID == "" {
		return ErrInvalidKeyID
	}

	query := `DELETE FROM crm_keys WHERE key_id = ?;`
	if _, err := r.db.Exec(query, keyID); err != nil {
		return fmt.Errorf("failed to purge key %s: %w", keyID, err)
	}

	return nil
}

// HasKey returns true if a key exists for keyID.
func (r *KeyRegistry) HasKey(keyID string) (bool, error) {
	if keyID == "" {
		return false, ErrInvalidKeyID
	}

	query := `SELECT 1 FROM crm_keys WHERE key_id = ?;`
	row := r.db.QueryRow(query, keyID)

	var dummy int
	if err := row.Scan(&dummy); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check existence of key %s: %w", keyID, err)
	}

	return true, nil
}

// ListKeys returns all registered key IDs.
func (r *KeyRegistry) ListKeys() ([]string, error) {
	query := `SELECT key_id FROM crm_keys ORDER BY created_at ASC;`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query keys: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, fmt.Errorf("failed to scan key_id: %w", err)
		}
		keys = append(keys, k)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return keys, nil
}

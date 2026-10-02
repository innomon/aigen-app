package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"sqlite-p2p/internal/crypto"
)

// ErrNotFound is returned when a requested key does not exist in crm_store.
var ErrNotFound = errors.New("record not found")

// Record represents a single entry in crm_store.
type Record struct {
	Key      string          `json:"key"`
	Metadata json.RawMessage `json:"metadata"`
	Data     []byte          `json:"data"`
}

// Repository handles CRUD operations for the unified crm_store table,
// optionally encrypting customer payloads and managing cryptographic shredding.
type Repository struct {
	db   *sql.DB
	keys *crypto.KeyRegistry
}

// NewRepository creates a new Repository backed by the given SQL database.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// SetKeyRegistry configures a KeyRegistry for payload encryption and crypto-shredding.
func (r *Repository) SetKeyRegistry(keys *crypto.KeyRegistry) {
	r.keys = keys
}

// KeyRegistry returns the configured KeyRegistry, if any.
func (r *Repository) KeyRegistry() *crypto.KeyRegistry {
	return r.keys
}

// PutRaw writes or upserts a record directly into crm_store without attempting encryption.
func (r *Repository) PutRaw(ctx context.Context, key string, metadata json.RawMessage, data []byte) error {
	if len(metadata) == 0 || !json.Valid(metadata) {
		return fmt.Errorf("invalid json metadata for key %s", key)
	}

	query := `
	INSERT INTO crm_store (key, metadata, data)
	VALUES (?, ?, ?)
	ON CONFLICT(key) DO UPDATE SET
		metadata = excluded.metadata,
		data = excluded.data;
	`
	_, err := r.db.ExecContext(ctx, query, key, string(metadata), data)
	if err != nil {
		return fmt.Errorf("failed to put record %s: %w", key, err)
	}
	return nil
}

// Put writes or upserts a record into crm_store. If a KeyRegistry is configured,
// data is encrypted with AES-256-GCM prior to storage.
func (r *Repository) Put(ctx context.Context, key string, metadata json.RawMessage, data []byte) error {
	payload := data
	if r.keys != nil && len(data) > 0 {
		symKey, err := r.keys.GetOrCreateKey(key)
		if err != nil {
			return fmt.Errorf("failed to obtain encryption key for %s: %w", key, err)
		}
		ciphertext, err := crypto.Encrypt(symKey, data)
		if err != nil {
			return fmt.Errorf("failed to encrypt payload for %s: %w", key, err)
		}
		payload = ciphertext
	}

	return r.PutRaw(ctx, key, metadata, payload)
}

// GetRaw retrieves a record from crm_store by its primary key without attempting decryption.
func (r *Repository) GetRaw(ctx context.Context, key string) (*Record, error) {
	query := `SELECT key, metadata, data FROM crm_store WHERE key = ?;`
	row := r.db.QueryRowContext(ctx, query, key)

	var rec Record
	var metaStr string
	err := row.Scan(&rec.Key, &metaStr, &rec.Data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get record %s: %w", key, err)
	}

	rec.Metadata = json.RawMessage(metaStr)
	return &rec, nil
}

// Get retrieves a record from crm_store by its primary key. If a KeyRegistry is configured,
// the payload is authenticated and decrypted with the per-customer symmetric key.
// If the key has been purged (crypto-shredded), decryption fails.
func (r *Repository) Get(ctx context.Context, key string) (*Record, error) {
	rec, err := r.GetRaw(ctx, key)
	if err != nil {
		return nil, err
	}

	if r.keys != nil && len(rec.Data) > 0 {
		symKey, err := r.keys.GetKey(key)
		if err != nil {
			return nil, fmt.Errorf("decryption failed, key unavailable (crypto-shredded): %w", err)
		}
		plaintext, err := crypto.Decrypt(symKey, rec.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt payload for %s: %w", key, err)
		}
		rec.Data = plaintext
	}

	return rec, nil
}

// Delete removes a record by key from crm_store. If a KeyRegistry is configured,
// the customer's symmetric key is permanently purged (crypto-shredding).
func (r *Repository) Delete(ctx context.Context, key string) error {
	query := `DELETE FROM crm_store WHERE key = ?;`
	_, err := r.db.ExecContext(ctx, query, key)
	if err != nil {
		return fmt.Errorf("failed to delete record %s: %w", key, err)
	}

	if r.keys != nil {
		_ = r.keys.PurgeKey(key)
	}

	return nil
}

// List returns records matching a key prefix with pagination, decrypting payloads when keys are available.
func (r *Repository) List(ctx context.Context, prefix string, limit, offset int) ([]Record, error) {
	query := `
	SELECT key, metadata, data
	FROM crm_store
	WHERE key LIKE ?
	ORDER BY key ASC
	LIMIT ? OFFSET ?;
	`
	rows, err := r.db.QueryContext(ctx, query, prefix+"%", limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list records: %w", err)
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var rec Record
		var metaStr string
		if err := rows.Scan(&rec.Key, &metaStr, &rec.Data); err != nil {
			return nil, fmt.Errorf("failed to scan record: %w", err)
		}
		rec.Metadata = json.RawMessage(metaStr)

		if r.keys != nil && len(rec.Data) > 0 {
			if symKey, err := r.keys.GetKey(rec.Key); err == nil {
				if plaintext, err := crypto.Decrypt(symKey, rec.Data); err == nil {
					rec.Data = plaintext
				}
			}
		}

		records = append(records, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during iteration: %w", err)
	}

	return records, nil
}

// Count returns the number of records with keys starting with the given prefix.
func (r *Repository) Count(ctx context.Context, prefix string) (int, error) {
	query := `SELECT count(*) FROM crm_store WHERE key LIKE ?;`
	var count int
	err := r.db.QueryRowContext(ctx, query, prefix+"%").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count records: %w", err)
	}
	return count, nil
}

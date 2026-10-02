package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schemaDDL = `
CREATE TABLE IF NOT EXISTS crm_store (
    key TEXT PRIMARY KEY,
    metadata TEXT CHECK(json_valid(metadata)),
    data BLOB
) WITHOUT ROWID;
`

// OpenDB opens a pure Go SQLite database, applies required PRAGMAs, and creates the crm_store table.
func OpenDB(dbPath string, enableWAL bool) (*sql.DB, error) {
	if dbPath != ":memory:" {
		dir := filepath.Dir(dbPath)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create db directory %s: %w", dir, err)
			}
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Always enforce foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	if enableWAL {
		if _, err := db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("failed to set WAL mode: %w", err)
		}
		if _, err := db.Exec("PRAGMA synchronous = NORMAL;"); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("failed to set synchronous normal: %w", err)
		}
	}

	// Create unified crm_store table
	if _, err := db.Exec(schemaDDL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize crm_store schema: %w", err)
	}

	return db, nil
}

package filestore

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"time"

	_ "github.com/lib/pq"
)

type PostgresFileStore struct {
	db *sql.DB
}

func NewPostgresFileStore(url string) (*PostgresFileStore, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	s := &PostgresFileStore{db: db}
	if err := s.init(); err != nil {
		return nil, fmt.Errorf("failed to initialize postgres storage schema: %w", err)
	}

	return s, nil
}

func (s *PostgresFileStore) init() error {
	query := `
		CREATE TABLE IF NOT EXISTS filesys (
			path TEXT PRIMARY KEY,
			metadata JSONB,
			content BYTEA,
			tmstamp TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_filesys_metadata ON filesys USING GIN (metadata);
	`
	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresFileStore) Upload(ctx context.Context, path string, reader io.Reader) error {
	content, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read object data: %w", err)
	}

	metadata := map[string]any{
		"size":          len(content),
		"last_modified": time.Now().UTC().Format(time.RFC3339),
	}
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO filesys (path, metadata, content)
		VALUES ($1, $2, $3)
		ON CONFLICT (path) DO UPDATE
		SET metadata = EXCLUDED.metadata,
		    content = EXCLUDED.content,
		    tmstamp = CURRENT_TIMESTAMP
	`
	_, err = s.db.ExecContext(ctx, query, path, metaJSON, content)
	return err
}

func (s *PostgresFileStore) GetMetadata(ctx context.Context, path string) (*FileMetadata, error) {
	var metaJSON []byte
	var createdAt time.Time
	query := `SELECT metadata, tmstamp FROM filesys WHERE path = $1`
	err := s.db.QueryRowContext(ctx, query, path).Scan(&metaJSON, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	var metadata struct {
		Size        int64  `json:"size"`
		ContentType string `json:"content_type"`
	}
	if err := json.Unmarshal(metaJSON, &metadata); err != nil {
		return nil, err
	}

	return &FileMetadata{
		Size:        metadata.Size,
		ContentType: metadata.ContentType,
		CreatedAt:   createdAt,
	}, nil
}

func (s *PostgresFileStore) GetUrl(path string) string {
	return fmt.Sprintf("/api/files/pg/%s", path)
}

func (s *PostgresFileStore) Download(ctx context.Context, path string, writer io.Writer) error {
	var content []byte
	query := `SELECT content FROM filesys WHERE path = $1`
	err := s.db.QueryRowContext(ctx, query, path).Scan(&content)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, bytes.NewReader(content))
	return err
}

func (s *PostgresFileStore) Delete(ctx context.Context, path string) error {
	query := `DELETE FROM filesys WHERE path = $1`
	_, err := s.db.ExecContext(ctx, query, path)
	return err
}

func (s *PostgresFileStore) DeleteByPrefix(ctx context.Context, prefix string) error {
	query := `DELETE FROM filesys WHERE path LIKE $1`
	_, err := s.db.ExecContext(ctx, query, prefix+"%")
	return err
}

func (s *PostgresFileStore) List(ctx context.Context, prefix string) ([]string, error) {
	query := `SELECT path FROM filesys WHERE path LIKE $1`
	rows, err := s.db.QueryContext(ctx, query, prefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		files = append(files, path)
	}
	return files, nil
}

func (s *PostgresFileStore) PurgeExpired(ctx context.Context, prefix string, ttlSeconds int) (int, error) {
	query := `
		DELETE FROM filesys 
		WHERE path LIKE $1 
		AND tmstamp < NOW() - (INTERVAL '1 second' * $2)
	`
	result, err := s.db.ExecContext(ctx, query, prefix+"%", ttlSeconds)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(rowsAffected), nil
}

func (s *PostgresFileStore) GetUploadedChunks(ctx context.Context, path string) ([]string, error) {
	return nil, fmt.Errorf("Chunked upload not implemented for Postgres")
}

func (s *PostgresFileStore) UploadChunk(ctx context.Context, path string, chunkNumber int, reader io.Reader) (string, error) {
	return "", fmt.Errorf("Chunked upload not implemented for Postgres")
}

func (s *PostgresFileStore) CommitChunks(ctx context.Context, path string) error {
	return fmt.Errorf("Chunked upload not implemented for Postgres")
}

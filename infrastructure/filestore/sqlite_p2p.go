package filestore

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	p2ppkg "sqlite-p2p/pkg/p2p"
)

type SqliteP2PFileStore struct {
	engine    *p2ppkg.Engine
	db        *sql.DB
	urlPrefix string

	// In-memory chunk buffer: uploadKey (path) -> chunkNumber -> data
	chunkMu sync.Mutex
	chunks  map[string]map[int][]byte
}

func parseSqliteP2PFileStoreURL(connStr string) (p2ppkg.EngineOptions, error) {
	opts := p2ppkg.EngineOptions{
		EnableWAL: true,
	}

	raw := connStr
	if raw == "" || raw == "sqlite-p2p://" || raw == "p2p://" || raw == "default" {
		opts.DBPath = "data/filestore.db"
		return opts, nil
	}

	if raw == ":memory:" || raw == "sqlite-p2p://:memory:" || raw == "p2p://:memory:" {
		opts.DBPath = ":memory:"
		opts.EnableWAL = false
		return opts, nil
	}

	var parsedQuery url.Values
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return opts, fmt.Errorf("failed to parse sqlite-p2p filestore url: %w", err)
		}
		dbPath := u.Path
		if u.Host != "" && u.Host != "localhost" && u.Host != "." {
			dbPath = u.Host + u.Path
		}
		if dbPath == "" {
			dbPath = "data/filestore.db"
		}
		opts.DBPath = dbPath
		parsedQuery = u.Query()
	} else if strings.Contains(raw, "?") {
		parts := strings.SplitN(raw, "?", 2)
		opts.DBPath = parts[0]
		var err error
		parsedQuery, err = url.ParseQuery(parts[1])
		if err != nil {
			return opts, fmt.Errorf("failed to parse query string: %w", err)
		}
	} else {
		opts.DBPath = raw
	}

	if parsedQuery != nil {
		if walStr := parsedQuery.Get("wal"); walStr != "" {
			opts.EnableWAL = walStr == "1" || strings.ToLower(walStr) == "true"
		}
		if cryptoStr := parsedQuery.Get("crypto"); cryptoStr != "" {
			opts.EnableCrypto = cryptoStr == "1" || strings.ToLower(cryptoStr) == "true"
		}
		if topicStr := parsedQuery.Get("topic"); topicStr != "" {
			var topic [32]byte
			copy(topic[:], []byte(topicStr))
			opts.SwarmTopic = topic
		}
		if portStr := parsedQuery.Get("port"); portStr != "" {
			if p, err := strconv.Atoi(portStr); err == nil {
				opts.SwarmPort = p
			}
		}
		if bootstrapStr := parsedQuery.Get("bootstrap"); bootstrapStr != "" {
			opts.Bootstrap = strings.Split(bootstrapStr, ",")
		}
	}

	return opts, nil
}

// NewSqliteP2PFileStore initializes a new SqliteP2PFileStore.
func NewSqliteP2PFileStore(connStr string, urlPrefix string) (*SqliteP2PFileStore, error) {
	opts, err := parseSqliteP2PFileStoreURL(connStr)
	if err != nil {
		return nil, err
	}

	engine, err := p2ppkg.OpenEngine(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite-p2p engine for filestore: %w", err)
	}

	if urlPrefix == "" {
		urlPrefix = "/api/files/p2p"
	}

	s := &SqliteP2PFileStore{
		engine:    engine,
		db:        engine.DB(),
		urlPrefix: strings.TrimSuffix(urlPrefix, "/"),
		chunks:    make(map[string]map[int][]byte),
	}

	if err := s.init(); err != nil {
		_ = engine.Close()
		return nil, fmt.Errorf("failed to initialize filesys schema: %w", err)
	}

	return s, nil
}

func (s *SqliteP2PFileStore) init() error {
	query := `
		CREATE TABLE IF NOT EXISTS filesys (
			path TEXT PRIMARY KEY,
			metadata TEXT CHECK(json_valid(metadata)),
			content BLOB,
			tmstamp DATETIME DEFAULT CURRENT_TIMESTAMP
		) WITHOUT ROWID;
		CREATE INDEX IF NOT EXISTS idx_filesys_tmstamp ON filesys (tmstamp);
	`
	_, err := s.db.Exec(query)
	return err
}

func (s *SqliteP2PFileStore) Upload(ctx context.Context, path string, reader io.Reader) error {
	content, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read object data: %w", err)
	}

	metadata := map[string]any{
		"size":          len(content),
		"last_modified": time.Now().UTC().Format(time.RFC3339),
	}
	metaBytes, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to serialize metadata: %w", err)
	}

	query := `
		INSERT INTO filesys (path, metadata, content, tmstamp)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(path) DO UPDATE SET
			metadata = excluded.metadata,
			content = excluded.content,
			tmstamp = excluded.tmstamp;
	`
	_, err = s.db.ExecContext(ctx, query, path, string(metaBytes), content)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

func (s *SqliteP2PFileStore) GetMetadata(ctx context.Context, path string) (*FileMetadata, error) {
	query := `SELECT metadata, tmstamp FROM filesys WHERE path = ?`
	row := s.db.QueryRowContext(ctx, query, path)

	var metaStr string
	var tm time.Time
	if err := row.Scan(&metaStr, &tm); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	var metaMap map[string]any
	if err := json.Unmarshal([]byte(metaStr), &metaMap); err != nil {
		return nil, fmt.Errorf("failed to parse metadata json: %w", err)
	}

	var size int64
	if sVal, ok := metaMap["size"].(float64); ok {
		size = int64(sVal)
	}

	return &FileMetadata{
		Size:      size,
		CreatedAt: tm,
	}, nil
}

func (s *SqliteP2PFileStore) GetUrl(path string) string {
	return fmt.Sprintf("%s/%s", s.urlPrefix, strings.TrimPrefix(path, "/"))
}

func (s *SqliteP2PFileStore) Download(ctx context.Context, path string, writer io.Writer) error {
	query := `SELECT content FROM filesys WHERE path = ?`
	row := s.db.QueryRowContext(ctx, query, path)

	var content []byte
	if err := row.Scan(&content); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("file not found: %s", path)
		}
		return fmt.Errorf("failed to read content: %w", err)
	}

	_, err := io.Copy(writer, bytes.NewReader(content))
	return err
}

func (s *SqliteP2PFileStore) Delete(ctx context.Context, path string) error {
	query := `DELETE FROM filesys WHERE path = ?`
	_, err := s.db.ExecContext(ctx, query, path)
	return err
}

func (s *SqliteP2PFileStore) DeleteByPrefix(ctx context.Context, prefix string) error {
	query := `DELETE FROM filesys WHERE path LIKE ?`
	_, err := s.db.ExecContext(ctx, query, prefix+"%")
	return err
}

func (s *SqliteP2PFileStore) List(ctx context.Context, prefix string) ([]string, error) {
	query := `SELECT path FROM filesys WHERE path LIKE ? ORDER BY path ASC`
	rows, err := s.db.QueryContext(ctx, query, prefix+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return paths, nil
}

func (s *SqliteP2PFileStore) PurgeExpired(ctx context.Context, prefix string, ttlSeconds int) (int, error) {
	cutoff := time.Now().UTC().Add(-time.Duration(ttlSeconds) * time.Second).Format("2006-01-02 15:04:05")
	query := `DELETE FROM filesys WHERE path LIKE ? AND tmstamp < ?`
	res, err := s.db.ExecContext(ctx, query, prefix+"%", cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to purge expired files: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(rows), nil
}

// GetUploadedChunks returns a list of chunk IDs already uploaded for a multi-part file upload.
func (s *SqliteP2PFileStore) GetUploadedChunks(ctx context.Context, path string) ([]string, error) {
	s.chunkMu.Lock()
	defer s.chunkMu.Unlock()

	chunkMap, exists := s.chunks[path]
	if !exists {
		return []string{}, nil
	}

	result := make([]string, 0, len(chunkMap))
	for num := range chunkMap {
		result = append(result, strconv.Itoa(num))
	}
	sort.Strings(result)
	return result, nil
}

// UploadChunk stores an individual chunk of a multi-part file upload.
func (s *SqliteP2PFileStore) UploadChunk(ctx context.Context, path string, chunkNumber int, reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read chunk data: %w", err)
	}

	s.chunkMu.Lock()
	defer s.chunkMu.Unlock()

	if _, exists := s.chunks[path]; !exists {
		s.chunks[path] = make(map[int][]byte)
	}
	s.chunks[path][chunkNumber] = data
	return strconv.Itoa(chunkNumber), nil
}

// CommitChunks joins all uploaded chunks into the final destination path in filesys.
func (s *SqliteP2PFileStore) CommitChunks(ctx context.Context, path string) error {
	s.chunkMu.Lock()
	chunkMap, exists := s.chunks[path]
	if !exists || len(chunkMap) == 0 {
		s.chunkMu.Unlock()
		return fmt.Errorf("no chunks found to commit for path: %s", path)
	}

	keys := make([]int, 0, len(chunkMap))
	for k := range chunkMap {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	var assembled bytes.Buffer
	for _, k := range keys {
		assembled.Write(chunkMap[k])
	}
	s.chunkMu.Unlock()

	if err := s.Upload(ctx, path, &assembled); err != nil {
		return fmt.Errorf("failed to upload committed chunks: %w", err)
	}

	s.chunkMu.Lock()
	delete(s.chunks, path)
	s.chunkMu.Unlock()

	return nil
}

// Close closes the underlying p2p engine.
func (s *SqliteP2PFileStore) Close() error {
	if s.engine != nil {
		return s.engine.Close()
	}
	return nil
}

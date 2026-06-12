package filestore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	surrealdb "github.com/surrealdb/surrealdb.go"
)

type SurrealFileDoc struct {
	Path     string    `json:"path"`
	Metadata string    `json:"metadata"`
	Content  []byte    `json:"content"`
	Tmstamp  time.Time `json:"tmstamp"`
}

type SurrealFileStore struct {
	client *surrealdb.DB
}

func parseSurrealDBConnString(connStr string) (endpoint, username, password, ns, db string, err error) {
	rawStr := connStr
	if strings.HasPrefix(rawStr, "surreal://") {
		rawStr = "ws://" + strings.TrimPrefix(rawStr, "surreal://")
	} else if strings.HasPrefix(rawStr, "surrealdb://") {
		rawStr = "ws://" + strings.TrimPrefix(rawStr, "surrealdb://")
	}

	u, err := url.Parse(rawStr)
	if err != nil {
		return "", "", "", "", "", err
	}

	endpoint = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) >= 1 && parts[0] != "" {
		ns = parts[0]
	}
	if len(parts) >= 2 && parts[1] != "" {
		db = parts[1]
	}

	if ns == "" {
		ns = "aigen"
	}
	if db == "" {
		db = "aigen"
	}
	if username == "" {
		username = "root"
	}
	if password == "" {
		password = "root"
	}

	return endpoint, username, password, ns, db, nil
}

func NewSurrealFileStore(ctx context.Context, connectionString string) (*SurrealFileStore, error) {
	endpoint, username, password, ns, db, err := parseSurrealDBConnString(connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	client, err := surrealdb.FromEndpointURLString(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to surrealdb: %w", err)
	}

	_, err = client.SignIn(ctx, surrealdb.Auth{
		Username: username,
		Password: password,
	})
	if err != nil {
		client.Close(ctx)
		return nil, fmt.Errorf("failed to sign in to surrealdb: %w", err)
	}

	// Define namespace if it doesn't exist
	_, _ = surrealdb.Query[any](ctx, client, fmt.Sprintf("DEFINE NAMESPACE `%s`", ns), nil)

	// Use namespace
	if err := client.Use(ctx, ns, ""); err != nil {
		client.Close(ctx)
		return nil, fmt.Errorf("failed to use namespace: %w", err)
	}

	// Define database if it doesn't exist
	_, _ = surrealdb.Query[any](ctx, client, fmt.Sprintf("DEFINE DATABASE `%s`", db), nil)

	// Use both namespace and database
	if err := client.Use(ctx, ns, db); err != nil {
		client.Close(ctx)
		return nil, fmt.Errorf("failed to use database: %w", err)
	}

	return &SurrealFileStore{
		client: client,
	}, nil
}

func (s *SurrealFileStore) Upload(ctx context.Context, path string, reader io.Reader) error {
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

	doc := SurrealFileDoc{
		Path:     path,
		Metadata: string(metaJSON),
		Content:  content,
		Tmstamp:  time.Now().UTC(),
	}

	idStr := fmt.Sprintf("filesys:`%s`", path)
	query := fmt.Sprintf("UPSERT %s CONTENT $content", idStr)
	_, err = surrealdb.Query[any](ctx, s.client, query, map[string]any{"content": doc})
	return err
}

func (s *SurrealFileStore) GetMetadata(ctx context.Context, path string) (*FileMetadata, error) {
	idStr := fmt.Sprintf("filesys:`%s`", path)
	query := fmt.Sprintf("SELECT metadata, tmstamp FROM %s", idStr)

	results, err := surrealdb.Query[[]struct {
		Metadata string    `json:"metadata"`
		Tmstamp  time.Time `json:"tmstamp"`
	}](ctx, s.client, query, nil)
	if err != nil {
		return nil, err
	}
	if len(*results) == 0 {
		return nil, nil
	}
	qRes := (*results)[0]
	if qRes.Status != "OK" || len(qRes.Result) == 0 {
		return nil, nil
	}

	doc := qRes.Result[0]
	var meta struct {
		Size        int64  `json:"size"`
		ContentType string `json:"content_type"`
	}
	if err := json.Unmarshal([]byte(doc.Metadata), &meta); err != nil {
		return nil, err
	}

	return &FileMetadata{
		Size:        meta.Size,
		ContentType: meta.ContentType,
		CreatedAt:   doc.Tmstamp,
	}, nil
}

func (s *SurrealFileStore) GetUrl(path string) string {
	return fmt.Sprintf("/api/files/surreal/%s", path)
}

func (s *SurrealFileStore) Download(ctx context.Context, path string, writer io.Writer) error {
	idStr := fmt.Sprintf("filesys:`%s`", path)
	query := fmt.Sprintf("SELECT content FROM %s", idStr)

	results, err := surrealdb.Query[[]struct {
		Content []byte `json:"content"`
	}](ctx, s.client, query, nil)
	if err != nil {
		return err
	}
	if len(*results) == 0 {
		return fmt.Errorf("file not found: %s", path)
	}
	qRes := (*results)[0]
	if qRes.Status != "OK" || len(qRes.Result) == 0 {
		return fmt.Errorf("file not found: %s", path)
	}

	_, err = io.Copy(writer, bytes.NewReader(qRes.Result[0].Content))
	return err
}

func (s *SurrealFileStore) Delete(ctx context.Context, path string) error {
	idStr := fmt.Sprintf("filesys:`%s`", path)
	query := fmt.Sprintf("DELETE %s", idStr)
	_, err := surrealdb.Query[any](ctx, s.client, query, nil)
	return err
}

func (s *SurrealFileStore) DeleteByPrefix(ctx context.Context, prefix string) error {
	query := "DELETE FROM filesys WHERE string::starts_with(path, $prefix)"
	_, err := surrealdb.Query[any](ctx, s.client, query, map[string]any{
		"prefix": prefix,
	})
	return err
}

func (s *SurrealFileStore) List(ctx context.Context, prefix string) ([]string, error) {
	query := "SELECT path FROM filesys WHERE string::starts_with(path, $prefix)"
	results, err := surrealdb.Query[[]struct {
		Path string `json:"path"`
	}](ctx, s.client, query, map[string]any{
		"prefix": prefix,
	})
	if err != nil {
		return nil, err
	}
	if len(*results) == 0 {
		return nil, nil
	}
	qRes := (*results)[0]
	if qRes.Status != "OK" {
		return nil, fmt.Errorf("query status not OK: %s", qRes.Status)
	}

	var files []string
	for _, doc := range qRes.Result {
		files = append(files, doc.Path)
	}
	return files, nil
}

func (s *SurrealFileStore) PurgeExpired(ctx context.Context, prefix string, ttlSeconds int) (int, error) {
	cutoff := time.Now().UTC().Add(-time.Duration(ttlSeconds) * time.Second)
	query := "DELETE FROM filesys WHERE string::starts_with(path, $prefix) AND tmstamp < $cutoff"

	results, err := surrealdb.Query[[]struct {
		Path string `json:"path"`
	}](ctx, s.client, query, map[string]any{
		"prefix": prefix,
		"cutoff": cutoff,
	})
	if err != nil {
		return 0, err
	}
	if len(*results) == 0 {
		return 0, nil
	}
	qRes := (*results)[0]
	if qRes.Status != "OK" {
		return 0, fmt.Errorf("query status not OK: %s", qRes.Status)
	}
	return len(qRes.Result), nil
}

func (s *SurrealFileStore) GetUploadedChunks(ctx context.Context, path string) ([]string, error) {
	return nil, fmt.Errorf("Chunked upload not implemented for SurrealDB")
}

func (s *SurrealFileStore) UploadChunk(ctx context.Context, path string, chunkNumber int, reader io.Reader) (string, error) {
	return "", fmt.Errorf("Chunked upload not implemented for SurrealDB")
}

func (s *SurrealFileStore) CommitChunks(ctx context.Context, path string) error {
	return fmt.Errorf("Chunked upload not implemented for SurrealDB")
}

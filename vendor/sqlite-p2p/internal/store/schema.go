package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// SchemaNamespacePrefix defines the URI prefix used for schema keys in crm_store.
const SchemaNamespacePrefix = "org.schema:"

// SchemaRecord represents a stored schema contract and its operational documentation.
type SchemaRecord struct {
	URI         string          `json:"uri"`
	Key         string          `json:"key"`
	JSONSchema  json.RawMessage `json:"schema"`
	DocMarkdown string          `json:"doc_markdown"`
}

// FormatSchemaKey formats a schema URI into its canonical crm_store primary key.
func FormatSchemaKey(uri string) string {
	if strings.HasPrefix(uri, SchemaNamespacePrefix) {
		return uri
	}
	return SchemaNamespacePrefix + uri
}

// ExtractSchemaURI extracts the raw URI from a schema key.
func ExtractSchemaURI(key string) string {
	return strings.TrimPrefix(key, SchemaNamespacePrefix)
}

// PutSchema stores or updates a schema definition with JSON schema metadata and Markdown doc payload.
func (r *Repository) PutSchema(ctx context.Context, uri string, schemaJSON json.RawMessage, docMarkdown string) (*SchemaRecord, error) {
	if strings.TrimSpace(uri) == "" {
		return nil, errors.New("schema uri cannot be empty")
	}
	if len(schemaJSON) == 0 || !json.Valid(schemaJSON) {
		return nil, errors.New("invalid json schema")
	}

	key := FormatSchemaKey(uri)
	err := r.PutRaw(ctx, key, schemaJSON, []byte(docMarkdown))
	if err != nil {
		return nil, fmt.Errorf("failed to put schema %s: %w", uri, err)
	}

	return &SchemaRecord{
		URI:         ExtractSchemaURI(key),
		Key:         key,
		JSONSchema:  schemaJSON,
		DocMarkdown: docMarkdown,
	}, nil
}

// GetSchema fetches a schema definition from crm_store by URI or canonical key.
func (r *Repository) GetSchema(ctx context.Context, uri string) (*SchemaRecord, error) {
	key := FormatSchemaKey(uri)
	rec, err := r.GetRaw(ctx, key)
	if err != nil {
		return nil, err
	}

	return &SchemaRecord{
		URI:         ExtractSchemaURI(rec.Key),
		Key:         rec.Key,
		JSONSchema:  rec.Metadata,
		DocMarkdown: string(rec.Data),
	}, nil
}

// ListSchemas lists all stored schemas matching the org.schema:* namespace with pagination.
func (r *Repository) ListSchemas(ctx context.Context, limit, offset int) ([]SchemaRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	query := `
	SELECT key, metadata, data
	FROM crm_store
	WHERE key LIKE ?
	ORDER BY key ASC
	LIMIT ? OFFSET ?;
	`
	rows, err := r.db.QueryContext(ctx, query, SchemaNamespacePrefix+"%", limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list schemas: %w", err)
	}
	defer rows.Close()

	var list []SchemaRecord
	for rows.Next() {
		var key string
		var metaStr string
		var data []byte
		if err := rows.Scan(&key, &metaStr, &data); err != nil {
			return nil, fmt.Errorf("failed to scan schema row: %w", err)
		}

		list = append(list, SchemaRecord{
			URI:         ExtractSchemaURI(key),
			Key:         key,
			JSONSchema:  json.RawMessage(metaStr),
			DocMarkdown: string(data),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading schema rows: %w", err)
	}

	return list, nil
}

// DeleteSchema deletes a schema record from crm_store by URI or canonical key.
func (r *Repository) DeleteSchema(ctx context.Context, uri string) error {
	key := FormatSchemaKey(uri)
	query := `DELETE FROM crm_store WHERE key = ?;`
	res, err := r.db.ExecContext(ctx, query, key)
	if err != nil {
		return fmt.Errorf("failed to delete schema %s: %w", uri, err)
	}
	rowsAffected, err := res.RowsAffected()
	if err == nil && rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

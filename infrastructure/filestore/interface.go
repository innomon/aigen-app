package filestore

import (
	"context"
	"io"
	"time"
)

type FileMetadata struct {
	Size        int64
	ContentType string
	CreatedAt   time.Time
}

type IFileStore interface {
	Upload(ctx context.Context, path string, reader io.Reader) error
	GetMetadata(ctx context.Context, path string) (*FileMetadata, error)
	GetUrl(path string) string
	Download(ctx context.Context, path string, writer io.Writer) error
	Delete(ctx context.Context, path string) error
	DeleteByPrefix(ctx context.Context, prefix string) error
	List(ctx context.Context, prefix string) ([]string, error)
	PurgeExpired(ctx context.Context, prefix string, ttlSeconds int) (int, error)

	// Chunked upload
	GetUploadedChunks(ctx context.Context, path string) ([]string, error)
	UploadChunk(ctx context.Context, path string, chunkNumber int, reader io.Reader) (string, error)
	CommitChunks(ctx context.Context, path string) error
}

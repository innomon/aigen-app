package filestore

import (
	"context"
	"io"
)

type FileMetadata struct {
	Size        int64
	ContentType string
}

type IFileStore interface {
	Upload(ctx context.Context, path string, reader io.Reader) error
	GetMetadata(ctx context.Context, path string) (*FileMetadata, error)
	GetUrl(path string) string
	Download(ctx context.Context, path string, writer io.Writer) error
	Delete(ctx context.Context, path string) error
	DeleteByPrefix(ctx context.Context, prefix string) error

	// Chunked upload
	GetUploadedChunks(ctx context.Context, path string) ([]string, error)
	UploadChunk(ctx context.Context, path string, chunkNumber int, reader io.Reader) (string, error)
	CommitChunks(ctx context.Context, path string) error
}

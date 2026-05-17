package filestore

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

type GCSFileStore struct {
	client *storage.Client
	bucket string
}

func NewGCSFileStore(ctx context.Context, bucket string, credentialsFile string) (*GCSFileStore, error) {
	var opts []option.ClientOption
	if credentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(credentialsFile))
	}

	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return &GCSFileStore{
		client: client,
		bucket: bucket,
	}, nil
}

func (g *GCSFileStore) Upload(ctx context.Context, path string, reader io.Reader) error {
	obj := g.client.Bucket(g.bucket).Object(path)
	w := obj.NewWriter(ctx)
	if _, err := io.Copy(w, reader); err != nil {
		w.Close()
		return err
	}
	return w.Close()
}

func (g *GCSFileStore) GetMetadata(ctx context.Context, path string) (*FileMetadata, error) {
	attrs, err := g.client.Bucket(g.bucket).Object(path).Attrs(ctx)
	if err != nil {
		return nil, err
	}

	return &FileMetadata{
		Size:        attrs.Size,
		ContentType: attrs.ContentType,
	}, nil
}

func (g *GCSFileStore) GetUrl(path string) string {
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", g.bucket, path)
}

func (g *GCSFileStore) Download(ctx context.Context, path string, writer io.Writer) error {
	r, err := g.client.Bucket(g.bucket).Object(path).NewReader(ctx)
	if err != nil {
		return err
	}
	defer r.Close()

	_, err = io.Copy(writer, r)
	return err
}

func (g *GCSFileStore) Delete(ctx context.Context, path string) error {
	return g.client.Bucket(g.bucket).Object(path).Delete(ctx)
}

func (g *GCSFileStore) DeleteByPrefix(ctx context.Context, prefix string) error {
	return fmt.Errorf("DeleteByPrefix not implemented for GCS")
}

func (g *GCSFileStore) GetUploadedChunks(ctx context.Context, path string) ([]string, error) {
	return nil, fmt.Errorf("Chunked upload not implemented for GCS")
}

func (g *GCSFileStore) UploadChunk(ctx context.Context, path string, chunkNumber int, reader io.Reader) (string, error) {
	return "", fmt.Errorf("Chunked upload not implemented for GCS")
}

func (g *GCSFileStore) CommitChunks(ctx context.Context, path string) error {
	return fmt.Errorf("Chunked upload not implemented for GCS")
}

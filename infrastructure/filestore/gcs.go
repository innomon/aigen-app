package filestore

import (
	"context"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
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
		CreatedAt:   attrs.Created,
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

func (g *GCSFileStore) List(ctx context.Context, prefix string) ([]string, error) {
	var files []string
	it := g.client.Bucket(g.bucket).Objects(ctx, &storage.Query{Prefix: prefix})
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		files = append(files, attrs.Name)
	}
	return files, nil
}

func (g *GCSFileStore) PurgeExpired(ctx context.Context, prefix string, ttlSeconds int) (int, error) {
	now := time.Now()
	expiryDuration := time.Duration(ttlSeconds) * time.Second
	count := 0

	it := g.client.Bucket(g.bucket).Objects(ctx, &storage.Query{Prefix: prefix})
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return count, err
		}

		if now.Sub(attrs.Created) > expiryDuration {
			err = g.client.Bucket(g.bucket).Object(attrs.Name).Delete(ctx)
			if err == nil {
				count++
			}
		}
	}
	return count, nil
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

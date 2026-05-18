package filestore

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type LocalFileStore struct {
	pathPrefix string
	urlPrefix  string
}

func NewLocalFileStore(pathPrefix, urlPrefix string) *LocalFileStore {
	// Ensure path prefix exists
	if _, err := os.Stat(pathPrefix); os.IsNotExist(err) {
		os.MkdirAll(pathPrefix, 0755)
	}
	return &LocalFileStore{
		pathPrefix: pathPrefix,
		urlPrefix:  urlPrefix,
	}
}

func (s *LocalFileStore) Upload(ctx context.Context, path string, reader io.Reader) error {
	fullPath := filepath.Join(s.pathPrefix, path)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	return err
}

func (s *LocalFileStore) GetMetadata(ctx context.Context, path string) (*FileMetadata, error) {
	fullPath := filepath.Join(s.pathPrefix, path)
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	contentType := mime.TypeByExtension(filepath.Ext(path))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return &FileMetadata{
		Size:        info.Size(),
		ContentType: contentType,
		CreatedAt:   info.ModTime(),
	}, nil
}

func (s *LocalFileStore) GetUrl(path string) string {
	return filepath.Join(s.urlPrefix, path)
}

func (s *LocalFileStore) Download(ctx context.Context, path string, writer io.Writer) error {
	fullPath := filepath.Join(s.pathPrefix, path)
	file, err := os.Open(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(writer, file)
	return err
}

func (s *LocalFileStore) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(s.pathPrefix, path)
	return os.Remove(fullPath)
}

func (s *LocalFileStore) DeleteByPrefix(ctx context.Context, prefix string) error {
	fullPath := filepath.Join(s.pathPrefix, prefix)
	return os.RemoveAll(fullPath)
}

func (s *LocalFileStore) List(ctx context.Context, prefix string) ([]string, error) {
	fullPath := filepath.Join(s.pathPrefix, prefix)
	var files []string
	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, err := filepath.Rel(s.pathPrefix, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
		}
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	return files, nil
}

func (s *LocalFileStore) PurgeExpired(ctx context.Context, prefix string, ttlSeconds int) (int, error) {
	fullPath := filepath.Join(s.pathPrefix, prefix)
	now := time.Now()
	expiryDuration := time.Duration(ttlSeconds) * time.Second
	count := 0

	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walk even if error on one file
		}
		if !info.IsDir() && now.Sub(info.ModTime()) > expiryDuration {
			if err := os.Remove(path); err == nil {
				count++
			}
		}
		return nil
	})

	return count, err
}

func (s *LocalFileStore) GetUploadedChunks(ctx context.Context, path string) ([]string, error) {
	chunkDir := filepath.Join(s.pathPrefix, "chunks", path)
	if _, err := os.Stat(chunkDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(chunkDir)
	if err != nil {
		return nil, err
	}

	var chunks []string
	for _, entry := range entries {
		if !entry.IsDir() {
			chunks = append(chunks, entry.Name())
		}
	}
	sort.Strings(chunks)
	return chunks, nil
}

func (s *LocalFileStore) UploadChunk(ctx context.Context, path string, chunkNumber int, reader io.Reader) (string, error) {
	chunkDir := filepath.Join(s.pathPrefix, "chunks", path)
	if err := os.MkdirAll(chunkDir, 0755); err != nil {
		return "", err
	}

	chunkName := fmt.Sprintf("%06d", chunkNumber)
	chunkPath := filepath.Join(chunkDir, chunkName)

	file, err := os.Create(chunkPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return "", err
	}

	return chunkName, nil
}

func (s *LocalFileStore) CommitChunks(ctx context.Context, path string) error {
	chunks, err := s.GetUploadedChunks(ctx, path)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(s.pathPrefix, path)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	destFile, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	chunkDir := filepath.Join(s.pathPrefix, "chunks", path)
	for _, chunkName := range chunks {
		chunkPath := filepath.Join(chunkDir, chunkName)
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return err
		}
		_, err = io.Copy(destFile, chunkFile)
		chunkFile.Close()
		if err != nil {
			return err
		}
	}

	return os.RemoveAll(chunkDir)
}

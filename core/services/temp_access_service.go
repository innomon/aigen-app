package services

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/infrastructure/filestore"
)

type TempAccessService struct {
	configs   []descriptors.TemporaryAccessConfig
	filestore filestore.IFileStore
}

func NewTempAccessService(configs []descriptors.TemporaryAccessConfig, filestore filestore.IFileStore) *TempAccessService {
	return &TempAccessService{
		configs:   configs,
		filestore: filestore,
	}
}

func (s *TempAccessService) GetConfig(path string) *descriptors.TemporaryAccessConfig {
	for _, c := range s.configs {
		if c.Path == path {
			return &c
		}
	}
	return nil
}

func (s *TempAccessService) IsExpired(ctx context.Context, path string, filename string) (bool, error) {
	config := s.GetConfig(path)
	if config == nil {
		return false, fmt.Errorf("configuration not found for path: %s", path)
	}

	fullPath := filepath.Join(path, filename)
	meta, err := s.filestore.GetMetadata(ctx, fullPath)
	if err != nil {
		return false, err
	}
	if meta == nil {
		return false, fmt.Errorf("file not found: %s", fullPath)
	}

	expiryTime := meta.CreatedAt.Add(time.Duration(config.TTL) * time.Second)
	return time.Now().After(expiryTime), nil
}

func (s *TempAccessService) CleanupExpired(ctx context.Context, path string) (int, error) {
	config := s.GetConfig(path)
	if config == nil {
		return 0, fmt.Errorf("configuration not found for path: %s", path)
	}

	return s.filestore.PurgeExpired(ctx, path, config.TTL)
}

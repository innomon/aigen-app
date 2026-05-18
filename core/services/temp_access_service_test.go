package services

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/infrastructure/filestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockFileStore struct {
	mock.Mock
}

func (m *MockFileStore) Upload(ctx context.Context, path string, reader io.Reader) error {
	args := m.Called(ctx, path, reader)
	return args.Error(0)
}

func (m *MockFileStore) GetMetadata(ctx context.Context, path string) (*filestore.FileMetadata, error) {
	args := m.Called(ctx, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*filestore.FileMetadata), args.Error(1)
}

func (m *MockFileStore) GetUrl(path string) string {
	args := m.Called(path)
	return args.String(0)
}

func (m *MockFileStore) Download(ctx context.Context, path string, writer io.Writer) error {
	args := m.Called(ctx, path, writer)
	return args.Error(0)
}

func (m *MockFileStore) Delete(ctx context.Context, path string) error {
	args := m.Called(ctx, path)
	return args.Error(0)
}

func (m *MockFileStore) DeleteByPrefix(ctx context.Context, prefix string) error {
	args := m.Called(ctx, prefix)
	return args.Error(0)
}

func (m *MockFileStore) List(ctx context.Context, prefix string) ([]string, error) {
	args := m.Called(ctx, prefix)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockFileStore) PurgeExpired(ctx context.Context, prefix string, ttlSeconds int) (int, error) {
	args := m.Called(ctx, prefix, ttlSeconds)
	return args.Int(0), args.Error(1)
}

func (m *MockFileStore) GetUploadedChunks(ctx context.Context, path string) ([]string, error) {
	args := m.Called(ctx, path)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockFileStore) UploadChunk(ctx context.Context, path string, chunkNumber int, reader io.Reader) (string, error) {
	args := m.Called(ctx, path, chunkNumber, reader)
	return args.String(0), args.Error(1)
}

func (m *MockFileStore) CommitChunks(ctx context.Context, path string) error {
	args := m.Called(ctx, path)
	return args.Error(0)
}

func TestTempAccessService_IsExpired(t *testing.T) {
	configs := []descriptors.TemporaryAccessConfig{
		{Path: "tmp", TTL: 300, Role: "admin"},
	}
	mockFS := new(MockFileStore)
	svc := NewTempAccessService(configs, mockFS)
	ctx := context.Background()

	t.Run("File Not Expired", func(t *testing.T) {
		createdAt := time.Now().Add(-100 * time.Second)
		mockFS.On("GetMetadata", ctx, "tmp/test.txt").Return(&filestore.FileMetadata{
			CreatedAt: createdAt,
		}, nil).Once()

		expired, err := svc.IsExpired(ctx, "tmp", "test.txt")
		assert.NoError(t, err)
		assert.False(t, expired)
	})

	t.Run("File Expired", func(t *testing.T) {
		createdAt := time.Now().Add(-400 * time.Second)
		mockFS.On("GetMetadata", ctx, "tmp/expired.txt").Return(&filestore.FileMetadata{
			CreatedAt: createdAt,
		}, nil).Once()

		expired, err := svc.IsExpired(ctx, "tmp", "expired.txt")
		assert.NoError(t, err)
		assert.True(t, expired)
	})

	t.Run("File Not Found", func(t *testing.T) {
		mockFS.On("GetMetadata", ctx, "tmp/missing.txt").Return(nil, nil).Once()

		expired, err := svc.IsExpired(ctx, "tmp", "missing.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file not found")
		assert.False(t, expired)
	})

	t.Run("Invalid Path Config", func(t *testing.T) {
		expired, err := svc.IsExpired(ctx, "invalid", "test.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "configuration not found")
		assert.False(t, expired)
	})
}

func TestTempAccessService_CleanupExpired(t *testing.T) {
	configs := []descriptors.TemporaryAccessConfig{
		{Path: "tmp", TTL: 300, Role: "admin"},
	}
	mockFS := new(MockFileStore)
	svc := NewTempAccessService(configs, mockFS)
	ctx := context.Background()

	t.Run("Cleanup Successful", func(t *testing.T) {
		mockFS.On("PurgeExpired", ctx, "tmp", 300).Return(1, nil).Once()

		count, err := svc.CleanupExpired(ctx, "tmp")
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
		mockFS.AssertExpectations(t)
	})
}

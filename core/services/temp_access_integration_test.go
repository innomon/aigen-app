package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/infrastructure/filestore"
	"github.com/stretchr/testify/assert"
)

func TestTempAccessIntegration(t *testing.T) {
	tmpDir := "test_tmp_storage"
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	fs := filestore.NewLocalFileStore(tmpDir, "/files")
	configs := []descriptors.TemporaryAccessConfig{
		{Path: "tmp", TTL: 1, Role: "admin"}, // 1 second TTL
	}
	svc := NewTempAccessService(configs, fs)
	ctx := context.Background()

	t.Run("Full Lifecycle", func(t *testing.T) {
		filename := "test.txt"
		path := "tmp"
		fullPath := filepath.Join(path, filename)

		// 1. Upload
		err := fs.Upload(ctx, fullPath, strings.NewReader("hello integration"))
		assert.NoError(t, err)

		// 2. Check if valid immediately
		expired, err := svc.IsExpired(ctx, path, filename)
		assert.NoError(t, err)
		assert.False(t, expired)

		// 3. Wait for TTL
		time.Sleep(2 * time.Second)

		// 4. Check if expired
		expired, err = svc.IsExpired(ctx, path, filename)
		assert.NoError(t, err)
		assert.True(t, expired)

		// 5. Delete
		err = fs.Delete(ctx, fullPath)
		assert.NoError(t, err)

		// 6. Verify Gone
		_, err = svc.IsExpired(ctx, path, filename)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file not found")
	})

	t.Run("Batch Cleanup", func(t *testing.T) {
		path := "tmp"
		// 1. Upload two files
		fs.Upload(ctx, filepath.Join(path, "file1.txt"), strings.NewReader("content1"))
		fs.Upload(ctx, filepath.Join(path, "file2.txt"), strings.NewReader("content2"))

		// 2. Wait for TTL
		time.Sleep(2 * time.Second)

		// 3. Run cleanup
		count, err := svc.CleanupExpired(ctx, path)
		assert.NoError(t, err)
		assert.Equal(t, 2, count)

		// 4. Verify they are gone
		files, _ := fs.List(ctx, path)
		assert.Len(t, files, 0)
	})
}

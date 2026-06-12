package filestore

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSurrealFileStore(t *testing.T) {
	connStr := "surreal://root:root@127.0.0.1:8920/testns/testdb"
	
	ctx := context.Background()
	store, err := CreateFileStore(ctx, Config{
		Driver: "surrealdb",
		SurrealDB: struct {
			URL string
		}{
			URL: connStr,
		},
	})
	if err != nil {
		t.Skipf("SurrealDB not running or failed to connect: %v", err)
		return
	}

	// 1. Upload
	content1 := []byte("hello surrealdb filestore")
	err = store.Upload(ctx, "docs/hello.txt", bytes.NewReader(content1))
	assert.NoError(t, err)

	content2 := []byte("another file content")
	err = store.Upload(ctx, "docs/sub/another.txt", bytes.NewReader(content2))
	assert.NoError(t, err)

	// 2. GetMetadata
	meta, err := store.GetMetadata(ctx, "docs/hello.txt")
	assert.NoError(t, err)
	assert.NotNil(t, meta)
	assert.Equal(t, int64(len(content1)), meta.Size)
	assert.True(t, time.Since(meta.CreatedAt) < 5*time.Second)

	// 3. GetUrl
	urlStr := store.GetUrl("docs/hello.txt")
	assert.Contains(t, urlStr, "/api/files/surreal/docs/hello.txt")

	// 4. Download
	var buf bytes.Buffer
	err = store.Download(ctx, "docs/hello.txt", &buf)
	assert.NoError(t, err)
	assert.Equal(t, content1, buf.Bytes())

	// 5. List
	files, err := store.List(ctx, "docs/")
	assert.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Contains(t, files, "docs/hello.txt")
	assert.Contains(t, files, "docs/sub/another.txt")

	// 6. PurgeExpired / DeleteByPrefix
	// Test DeleteByPrefix
	err = store.DeleteByPrefix(ctx, "docs/sub")
	assert.NoError(t, err)

	files, err = store.List(ctx, "docs/")
	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Contains(t, files, "docs/hello.txt")

	// 7. Delete
	err = store.Delete(ctx, "docs/hello.txt")
	assert.NoError(t, err)

	files, err = store.List(ctx, "docs/")
	assert.NoError(t, err)
	assert.Len(t, files, 0)
}

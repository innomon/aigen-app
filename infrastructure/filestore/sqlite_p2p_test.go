package filestore

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSqliteP2PFileStore(t *testing.T) {
	ctx := context.Background()
	store, err := CreateFileStore(ctx, Config{
		Driver: "sqlite-p2p",
		SqliteP2P: struct {
			URL       string
			UrlPrefix string
		}{
			URL:       "sqlite-p2p://:memory:",
			UrlPrefix: "/api/files/custom",
		},
	})
	require.NoError(t, err)

	// 1. Upload
	content1 := []byte("hello sqlite-p2p filestore")
	err = store.Upload(ctx, "docs/hello.txt", bytes.NewReader(content1))
	assert.NoError(t, err)

	content2 := []byte("another file content")
	err = store.Upload(ctx, "docs/sub/another.txt", bytes.NewReader(content2))
	assert.NoError(t, err)

	// 2. GetMetadata
	meta, err := store.GetMetadata(ctx, "docs/hello.txt")
	assert.NoError(t, err)
	require.NotNil(t, meta)
	assert.Equal(t, int64(len(content1)), meta.Size)

	// 3. GetUrl
	urlStr := store.GetUrl("docs/hello.txt")
	assert.Equal(t, "/api/files/custom/docs/hello.txt", urlStr)

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

	// 6. Chunked upload
	chunk1 := []byte("chunk 1 - ")
	chunk2 := []byte("chunk 2")
	_, err = store.UploadChunk(ctx, "docs/chunked.txt", 1, bytes.NewReader(chunk1))
	assert.NoError(t, err)
	_, err = store.UploadChunk(ctx, "docs/chunked.txt", 2, bytes.NewReader(chunk2))
	assert.NoError(t, err)

	chunks, err := store.GetUploadedChunks(ctx, "docs/chunked.txt")
	assert.NoError(t, err)
	assert.Len(t, chunks, 2)

	err = store.CommitChunks(ctx, "docs/chunked.txt")
	assert.NoError(t, err)

	var chunkBuf bytes.Buffer
	err = store.Download(ctx, "docs/chunked.txt", &chunkBuf)
	assert.NoError(t, err)
	assert.Equal(t, "chunk 1 - chunk 2", chunkBuf.String())

	// 7. DeleteByPrefix
	err = store.DeleteByPrefix(ctx, "docs/sub")
	assert.NoError(t, err)

	files, err = store.List(ctx, "docs/")
	assert.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Contains(t, files, "docs/hello.txt")
	assert.Contains(t, files, "docs/chunked.txt")

	// 8. Delete
	err = store.Delete(ctx, "docs/hello.txt")
	assert.NoError(t, err)

	files, err = store.List(ctx, "docs/")
	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Contains(t, files, "docs/chunked.txt")
}

package api

import (
	"io"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func TestOverlayFS_Open(t *testing.T) {
	// Higher priority
	fs1 := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("custom index")},
		"about.html": &fstest.MapFile{Data: []byte("custom about")},
		"dir/file1.txt": &fstest.MapFile{Data: []byte("custom file1")},
	}

	// Lower priority
	fs2 := fstest.MapFS{
		"index.html":   &fstest.MapFile{Data: []byte("embedded index")},
		"contact.html": &fstest.MapFile{Data: []byte("embedded contact")},
		"dir/file1.txt": &fstest.MapFile{Data: []byte("embedded file1")},
		"dir/file2.txt": &fstest.MapFile{Data: []byte("embedded file2")},
	}

	overlay := NewOverlayFS(fs1, fs2)

	tests := []struct {
		name           string
		path           string
		expectedData   string
		expectError    bool
	}{
		{
			name:         "Overridden file (should use fs1)",
			path:         "index.html",
			expectedData: "custom index",
			expectError:  false,
		},
		{
			name:         "Fallback file (only in fs2)",
			path:         "contact.html",
			expectedData: "embedded contact",
			expectError:  false,
		},
		{
			name:         "File only in fs1",
			path:         "about.html",
			expectedData: "custom about",
			expectError:  false,
		},
		{
			name:         "Overridden nested file (should use fs1)",
			path:         "dir/file1.txt",
			expectedData: "custom file1",
			expectError:  false,
		},
		{
			name:         "Nested fallback file (only in fs2)",
			path:         "dir/file2.txt",
			expectedData: "embedded file2",
			expectError:  false,
		},
		{
			name:         "Non-existent file",
			path:         "missing.html",
			expectedData: "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := overlay.Open(tt.path)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			defer f.Close()

			data, err := io.ReadAll(f)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedData, string(data))
		})
	}
}

func TestOverlayFS_Stat(t *testing.T) {
	fs1 := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("custom index")},
	}
	fs2 := fstest.MapFS{
		"index.html":   &fstest.MapFile{Data: []byte("embedded index")},
		"contact.html": &fstest.MapFile{Data: []byte("embedded contact")},
	}

	overlay := NewOverlayFS(fs1, fs2)

	// Stat overridden file (should be from fs1)
	info, err := overlay.Stat("index.html")
	assert.NoError(t, err)
	assert.Equal(t, int64(len("custom index")), info.Size())

	// Stat fallback file (should be from fs2)
	info, err = overlay.Stat("contact.html")
	assert.NoError(t, err)
	assert.Equal(t, int64(len("embedded contact")), info.Size())

	// Stat non-existent file
	_, err = overlay.Stat("missing.html")
	assert.Error(t, err)
}

func TestOverlayFS_ReadDir(t *testing.T) {
	fs1 := fstest.MapFS{
		"dir/file1.txt": &fstest.MapFile{Data: []byte("custom file1")},
		"dir/file3.txt": &fstest.MapFile{Data: []byte("custom file3")},
	}
	fs2 := fstest.MapFS{
		"dir/file1.txt": &fstest.MapFile{Data: []byte("embedded file1")},
		"dir/file2.txt": &fstest.MapFile{Data: []byte("embedded file2")},
	}

	overlay := NewOverlayFS(fs1, fs2)

	entries, err := overlay.ReadDir("dir")
	assert.NoError(t, err)

	// Should contain: file1.txt, file2.txt, file3.txt
	assert.Len(t, entries, 3)

	names := make(map[string]bool)
	for _, entry := range entries {
		names[entry.Name()] = true
	}

	assert.True(t, names["file1.txt"])
	assert.True(t, names["file2.txt"])
	assert.True(t, names["file3.txt"])
}

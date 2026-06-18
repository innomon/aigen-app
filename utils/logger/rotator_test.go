package logger

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestLogRotator(t *testing.T) {
	t.Run("basic_write_and_close", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "log_rotator_test")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		filename := "app.log"
		rotator, err := NewLogRotator(tempDir, filename, 1, 3)
		if err != nil {
			t.Fatalf("failed to create rotator: %v", err)
		}
		defer rotator.Close()

		data := []byte("hello world\n")
		n, err := rotator.Write(data)
		if err != nil {
			t.Fatalf("write failed: %v", err)
		}
		if n != len(data) {
			t.Errorf("expected to write %d bytes, wrote %d", len(data), n)
		}

		filePath := filepath.Join(tempDir, filename)
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("read file failed: %v", err)
		}
		if !bytes.Equal(content, data) {
			t.Errorf("expected content %q, got %q", string(data), string(content))
		}
	})

	t.Run("rotation_and_pruning", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "log_rotator_test")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		filename := "app.log"
		// Set maxSizeMB to 1 (which translates to 1MB = 1048576 bytes)
		// Set maxBackups to 2
		rotator, err := NewLogRotator(tempDir, filename, 1, 2)
		if err != nil {
			t.Fatalf("failed to create rotator: %v", err)
		}
		defer rotator.Close()

		// Write slightly more than 1MB to trigger rotation
		chunk := make([]byte, 1024)
		for i := range chunk {
			chunk[i] = 'a'
		}

		// Write 1025 chunks = 1025 KB > 1MB
		for i := 0; i < 1025; i++ {
			_, err = rotator.Write(chunk)
			if err != nil {
				t.Fatalf("write chunk %d failed: %v", i, err)
			}
		}

		// Check that we rotated (meaning backup files exist)
		files, err := os.ReadDir(tempDir)
		if err != nil {
			t.Fatalf("read dir failed: %v", err)
		}

		var backupCount int
		for _, f := range files {
			if strings.HasPrefix(f.Name(), "app.") && f.Name() != "app.log" {
				backupCount++
			}
		}

		if backupCount == 0 {
			t.Error("expected at least one backup file due to rotation")
		}

		// Write more to trigger another rotation and verify pruning
		// We'll write another 1025 chunks to trigger second rotation
		for i := 0; i < 1025; i++ {
			_, err = rotator.Write(chunk)
			if err != nil {
				t.Fatalf("write chunk %d failed: %v", i, err)
			}
		}

		// Write another 1025 chunks to trigger third rotation (total backups created = 3, maxBackups = 2, so 1 should be pruned)
		for i := 0; i < 1025; i++ {
			_, err = rotator.Write(chunk)
			if err != nil {
				t.Fatalf("write chunk %d failed: %v", i, err)
			}
		}

		files, err = os.ReadDir(tempDir)
		if err != nil {
			t.Fatalf("read dir failed: %v", err)
		}

		backupCount = 0
		for _, f := range files {
			if strings.HasPrefix(f.Name(), "app.") && f.Name() != "app.log" {
				backupCount++
			}
		}

		if backupCount > 2 {
			t.Errorf("expected at most 2 backups, got %d", backupCount)
		}
	})

	t.Run("concurrency_safety", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "log_rotator_test")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		filename := "app.log"
		rotator, err := NewLogRotator(tempDir, filename, 1, 3)
		if err != nil {
			t.Fatalf("failed to create rotator: %v", err)
		}
		defer rotator.Close()

		var wg sync.WaitGroup
		numWorkers := 10
		numWritesPerWorker := 100
		data := []byte("data-chunk\n")

		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < numWritesPerWorker; i++ {
					_, err := rotator.Write(data)
					if err != nil {
						t.Errorf("concurrent write failed: %v", err)
					}
				}
			}()
		}

		wg.Wait()
	})
}

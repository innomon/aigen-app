package logger

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

type captureHandler struct {
	records []slog.Record
	mu      sync.Mutex
}

func (c *captureHandler) Enabled(ctx context.Context, l slog.Level) bool { return true }
func (c *captureHandler) Handle(ctx context.Context, r slog.Record) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.records = append(c.records, r)
	return nil
}
func (c *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return c }
func (c *captureHandler) WithGroup(name string) slog.Handler      { return c }

func TestLogger(t *testing.T) {
	t.Run("init_and_config", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "logger_init_test")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		cfg := Config{
			Level:          "DEBUG",
			ConsoleEnabled: true,
			FileEnabled:    true,
			Dir:            tempDir,
			FileName:       "test_app.log",
			MaxSizeMB:      1,
			MaxBackups:     3,
		}

		logger, err := Init(cfg)
		if err != nil {
			t.Fatalf("failed to initialize logger: %v", err)
		}
		if logger == nil {
			t.Fatal("expected non-nil logger")
		}

		// Verify file was created
		filePath := filepath.Join(tempDir, "test_app.log")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Error("expected log file to be created, but it does not exist")
		}
	})

	t.Run("custom_logger_source_preservation", func(t *testing.T) {
		capturer := &captureHandler{}
		originalLog := Log
		defer func() { Log = originalLog }()

		Log = slog.New(capturer)
		cl := NewCustomLogger("test-module")

		cl.Infof("test log message with format: %s", "hello")

		if len(capturer.records) != 1 {
			t.Fatalf("expected 1 record, got %d", len(capturer.records))
		}

		r := capturer.records[0]
		if r.Level != slog.LevelInfo {
			t.Errorf("expected level INFO, got %v", r.Level)
		}
		if !strings.Contains(r.Message, "[test-module] test log message with format: hello") {
			t.Errorf("unexpected message: %q", r.Message)
		}

		// Verify caller function is this test function, not logger wrapper functions
		frame, _ := runtime.CallersFrames([]uintptr{r.PC}).Next()
		funcName := frame.Function
		if strings.Contains(funcName, "CustomLogger") || strings.Contains(funcName, "logger.log") {
			t.Errorf("caller preservation failed: resolved to wrapper function %q", funcName)
		}
		if !strings.Contains(funcName, "TestLogger") {
			t.Errorf("expected caller function to be TestLogger, got %q", funcName)
		}
	})

	t.Run("package_level_helpers_source_preservation", func(t *testing.T) {
		capturer := &captureHandler{}
		originalLog := Log
		defer func() { Log = originalLog }()

		Log = slog.New(capturer)

		Infof("package level info log %d", 42)
		Errorf("package level error log")

		if len(capturer.records) != 2 {
			t.Fatalf("expected 2 records, got %d", len(capturer.records))
		}

		for _, r := range capturer.records {
			frame, _ := runtime.CallersFrames([]uintptr{r.PC}).Next()
			funcName := frame.Function
			if strings.Contains(funcName, "logger.log") || strings.Contains(funcName, "logger.Infof") || strings.Contains(funcName, "logger.Errorf") {
				t.Errorf("caller preservation failed: resolved to wrapper function %q", funcName)
			}
			if !strings.Contains(funcName, "TestLogger") {
				t.Errorf("expected caller function to be TestLogger, got %q", funcName)
			}
		}
	})
}

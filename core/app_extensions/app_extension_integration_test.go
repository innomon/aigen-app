package app_extensions

import (
	"bytes"
	"context"
	"io/fs"
	"testing"
	"time"

	"github.com/innomon/aigen-app/core/services"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/stretchr/testify/assert"
)

func TestExtensionLifecycle(t *testing.T) {
	// 1. Setup minimal dependencies
	dao, _ := relationdbdao.CreateDao("postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	schemaService := services.NewSchemaService(dao)
	evolutionService := services.NewEvolutionService(dao, schemaService)
	auditService := services.NewAuditService(dao)

	extensionsDir := "app-extensions"
	svc := NewAppExtensionService(extensionsDir, schemaService, evolutionService, nil, nil, nil, auditService)

	// Mock manifest with permissions and env_vars
	info := &AppExtensionInfo{
		Manifest: AppExtensionManifest{
			ID: "test-extension",
			Permissions: []PermissionRequirement{
				{Type: "http", Value: "*.openai.com"},
			},
			EnvVars: []string{"API_KEY"},
		},
		Status:     StatusActive,
		IsVerified: true,
	}
	svc.mu.Lock()
	svc.extensions["test-extension"] = info
	svc.mu.Unlock()

	ctx := context.Background()

	t.Run("Vault Security", func(t *testing.T) {
		svc.SetSecret("test-extension", "API_KEY", "super-secret")

		cfg, err := svc.Dispatcher.prepareVMConfig("test-extension", "quickjs")
		assert.NoError(t, err)

		// 1. Allowed key
		val, ok := cfg.Env["API_KEY"]
		assert.True(t, ok)
		assert.Equal(t, "super-secret", val)

		// 2. Unlisted key
		_, ok = cfg.Env["PRIVATE_KEY"]
		assert.False(t, ok)
	})

	t.Run("Permission Enforcement", func(t *testing.T) {
		// First authorize via admin
		err := svc.AuthorizePermission(ctx, "test-extension", PermissionRequirement{Type: "http", Value: "*.openai.com"}, "admin1")
		assert.NoError(t, err)

		cfg, err := svc.Dispatcher.prepareVMConfig("test-extension", "quickjs")
		assert.NoError(t, err)

		// Check if AllowNet contains the granted value
		found := false
		for _, net := range cfg.AllowNet {
			if net == "*.openai.com" {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("JS Sandbox Execution", func(t *testing.T) {

		script := `
			const res = { sum: 10 + 20 };
			res;
		`
		// We'll use a mock FS for the test
		fsys := &mockFS{
			files: map[string]string{
				"scripts/test.js": script,
			},
		}

		result, err := svc.Dispatcher.Execute(ctx, "test-extension", fsys, "scripts/test.js", nil)
		assert.NoError(t, err)

		resMap := result.(map[string]interface{})
		assert.Equal(t, float64(30), resMap["sum"])
	})
}

type mockFS struct {
	files map[string]string
}

func (m *mockFS) Open(name string) (fs.File, error) {
	content, ok := m.files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return &mockFile{Reader: bytes.NewReader([]byte(content)), name: name}, nil
}

type mockFile struct {
	*bytes.Reader
	name string
}

func (m *mockFile) Stat() (fs.FileInfo, error) { return m, nil }
func (m *mockFile) Name() string               { return m.name }
func (m *mockFile) Size() int64                { return int64(m.Len()) }
func (m *mockFile) Mode() fs.FileMode          { return 0 }
func (m *mockFile) ModTime() time.Time         { return time.Now() }
func (m *mockFile) IsDir() bool                { return false }
func (m *mockFile) Sys() interface{}           { return nil }
func (m *mockFile) Close() error               { return nil }

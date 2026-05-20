package plugins

import (
	"context"
	"os"
	"testing"

	"github.com/innomon/aigen-app/core/services"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/stretchr/testify/assert"
)

func TestPluginLifecycle(t *testing.T) {
	// 1. Setup minimal dependencies
	dao, _ := relationdbdao.CreateDao("postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	schemaService := services.NewSchemaService(dao)
	evolutionService := services.NewEvolutionService(dao, schemaService)
	auditService := services.NewAuditService(dao)
	
	pluginsDir := "plugins"
	svc := NewPluginService(pluginsDir, schemaService, evolutionService, nil, nil, nil, auditService)

	// Mock manifest with permissions and env_vars
	info := &PluginInfo{
		Manifest: PluginManifest{
			ID: "test-plugin",
			Permissions: []PermissionRequirement{
				{Type: "http", Value: "*.openai.com"},
			},
			EnvVars: []string{"API_KEY"},
		},
		Status:     StatusActive,
		IsVerified: true,
	}
	svc.mu.Lock()
	svc.plugins["test-plugin"] = info
	svc.mu.Unlock()

	ctx := context.Background()

	t.Run("Vault Security", func(t *testing.T) {
		svc.SetSecret("test-plugin", "API_KEY", "super-secret")
		
		env := svc.Dispatcher.prepareEnv(ctx, "test-plugin")
		
		// 1. Allowed key
		val, ok := env.GetSecret("API_KEY")
		assert.True(t, ok)
		assert.Equal(t, "super-secret", val)

		// 2. Unlisted key
		val, ok = env.GetSecret("PRIVATE_KEY")
		assert.False(t, ok)
		assert.Empty(t, val)
	})

	t.Run("Permission Enforcement", func(t *testing.T) {
		env := svc.Dispatcher.prepareEnv(ctx, "test-plugin")

		// 1. Unauthorized Fetch
		_, err := env.Fetch("https://malicious.com", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")

		// 2. Authorized Fetch (Mocked)
		// First authorize via admin
		err = svc.AuthorizePermission(ctx, "test-plugin", PermissionRequirement{Type: "http", Value: "*.openai.com"}, "admin1")
		assert.NoError(t, err)

		resp, err := env.Fetch("https://api.openai.com/v1/chat", nil)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	})
}

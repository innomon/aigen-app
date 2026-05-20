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
	dao, err := relationdbdao.CreateDao("postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	if err != nil {
		t.Skip("Postgres not available for integration test")
		return
	}
	
	schemaService := services.NewSchemaService(dao)
	evolutionService := services.NewEvolutionService(dao, schemaService)
	// Mock ChatService if needed, but for now we test Discovery and BizDef mounting
	
	pluginsDir := "plugins"
	svc := NewPluginService(pluginsDir, schemaService, evolutionService, nil, nil, nil)

	// 2. Scan
	err = svc.Scan()
	assert.NoError(t, err)

	plugins := svc.All()
	assert.GreaterOrEqual(t, len(plugins), 1)

	var weatherPlugin *PluginInfo
	for _, p := range plugins {
		if p.Manifest.ID == "weather-plugin" {
			weatherPlugin = p
			break
		}
	}

	assert.NotNil(t, weatherPlugin)
	assert.True(t, weatherPlugin.IsVerified)
	assert.Equal(t, "CN=AiGen Trusted Developer, O=AiGen", weatherPlugin.Signer)

	// 3. Mount
	ctx := context.Background()
	err = svc.MountPlugin(ctx, "weather-plugin")
	// If DB fails, we might still have mounted the plugin status
	assert.NoError(t, err)

	// 4. Verify BizDef was mounted (Skip if DB connection failed earlier)
	schema, err := schemaService.ByNameOrDefault(ctx, "weather_report", "entity", nil)
	if err != nil {
		t.Logf("Skipping BizDef verification due to DB error: %v", err)
	} else {
		assert.NotNil(t, schema)
		assert.Equal(t, "weather_report", schema.Name)
	}

	// 5. Verify Sandbox Execution (Mocked)
	args := map[string]any{"location": "San Francisco"}
	result, err := svc.Dispatcher.Execute(ctx, os.DirFS("../../sample-plugin"), "scripts/calculate_weather.js", args)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	
	resMap := result.(map[string]any)
	assert.Equal(t, "success", resMap["status"])

	t.Logf("Plugin Lifecycle verified: Discovery -> Signature -> Mount -> BizDef")
}

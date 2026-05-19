package services

import (
	"testing"
	"time"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/innomon/aigen-app/utils/datamodels"
	"github.com/stretchr/testify/assert"
)

func TestEvolutionService_EvolveRecord(t *testing.T) {
	dao := relationdbdao.NewMemoryDao()
	schemaService := NewSchemaService(dao)
	evolutionService := NewEvolutionService(dao, schemaService)

	dateV2 := time.Now().Add(-24 * time.Hour).Truncate(time.Second)
	dateV3 := time.Now().Add(-12 * time.Hour).Truncate(time.Second)

	manifest := descriptors.EvolutionManifest{
		"crm_lead": {
			"v2": {
				Date:        dateV2,
				Description: "Rename and Add",
				Actions: []descriptors.EvolutionAction{
					{Action: "rename", From: "old_status", To: "status"},
					{Action: "add", Field: "lead_score", Default: 50},
				},
			},
			"v3": {
				Date:        dateV3,
				Description: "Drop",
				Actions: []descriptors.EvolutionAction{
					{Action: "drop", Field: "temp_data"},
				},
			},
		},
	}

	evolutionService.RegisterManifest("crm", manifest)

	t.Run("Upgrade from v1 (empty version)", func(t *testing.T) {
		rec := map[string]interface{}{
			"old_status": "open",
			"temp_data":  "secret",
			"name":       "John Doe",
		}
		meta := &datamodels.MetaData{}

		evolved, modified, err := evolutionService.EvolveRecord("crm_lead", rec, meta)
		assert.NoError(t, err)
		assert.True(t, modified)
		assert.Equal(t, "v3", meta.SchemaVersion)
		assert.Equal(t, dateV3.Format(time.RFC3339), meta.SchemaVersionDate)

		assert.Equal(t, "open", evolved["status"])
		assert.Equal(t, 50, evolved["lead_score"])
		assert.NotContains(t, evolved, "old_status")
		assert.NotContains(t, evolved, "temp_data")
		assert.Equal(t, "John Doe", evolved["name"])
	})

	t.Run("Upgrade from v2", func(t *testing.T) {
		rec := map[string]interface{}{
			"status":     "contacted",
			"lead_score": 100,
			"temp_data":  "trash",
		}
		meta := &datamodels.MetaData{
			SchemaVersion:     "v2",
			SchemaVersionDate: dateV2.Format(time.RFC3339),
		}

		evolved, modified, err := evolutionService.EvolveRecord("crm_lead", rec, meta)
		assert.NoError(t, err)
		assert.True(t, modified)
		assert.Equal(t, "v3", meta.SchemaVersion)

		assert.NotContains(t, evolved, "temp_data")
		assert.Equal(t, "contacted", evolved["status"])
		assert.Equal(t, 100, evolved["lead_score"])
	})

	t.Run("Already at latest version", func(t *testing.T) {
		rec := map[string]interface{}{
			"status":     "qualified",
			"lead_score": 75,
		}
		meta := &datamodels.MetaData{
			SchemaVersion:     "v3",
			SchemaVersionDate: dateV3.Format(time.RFC3339),
		}

		_, modified, err := evolutionService.EvolveRecord("crm_lead", rec, meta)
		assert.NoError(t, err)
		assert.False(t, modified)
	})
}

package bizdefs

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/innomon/aigen-app/core/services"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/stretchr/testify/assert"
)

func TestSetupBizDefFromFS_OverrideAndMerge(t *testing.T) {
	ctx := context.WithValue(context.Background(), "roles", []string{"sa"})

	// Core filesystem representation
	coreFS := fstest.MapFS{
		"schemas/crm_lead.json": &fstest.MapFile{Data: []byte(`{
			"name": "crm_lead",
			"table_name": "crm_lead",
			"primary_key": "id",
			"label_attribute_name": "lead_name",
			"attributes": [
				{"field": "id", "header": "ID", "data_type": "integer"},
				{"field": "lead_name", "header": "Core Lead Name", "data_type": "string"}
			]
		}`)},
		"schemas/crm_contact.json": &fstest.MapFile{Data: []byte(`{
			"name": "crm_contact",
			"table_name": "crm_contact",
			"primary_key": "id",
			"label_attribute_name": "contact_name",
			"attributes": [
				{"field": "id", "header": "ID", "data_type": "integer"},
				{"field": "contact_name", "header": "Core Contact Name", "data_type": "string"}
			]
		}`)},
	}

	// App Extension overriding filesystem representation
	extensionFS := fstest.MapFS{
		"schemas/crm_lead.json": &fstest.MapFile{Data: []byte(`{
			"name": "crm_lead",
			"table_name": "crm_lead",
			"primary_key": "id",
			"label_attribute_name": "lead_name",
			"attributes": [
				{"field": "id", "header": "ID", "data_type": "integer"},
				{"field": "lead_name", "header": "Overridden Lead Name", "data_type": "string"},
				{"field": "custom_field", "header": "Custom Field", "data_type": "string"}
			]
		}`)},
	}

	// Initialize database & services
	dao, err := relationdbdao.CreateDao("memory://")
	assert.NoError(t, err)
	defer dao.Close()

	err = dao.EnsureTable(ctx)
	assert.NoError(t, err)

	schemaService := services.NewSchemaService(dao)

	// 1. Setup core schemas
	err = SetupBizDefFromFS(ctx, coreFS, "crm", schemaService)
	assert.NoError(t, err)

	// Verify core lead schema
	leadSchema, err := schemaService.LoadEntity(ctx, "crm_lead")
	assert.NoError(t, err)
	assert.NotNil(t, leadSchema)
	assert.Equal(t, "Core Lead Name", leadSchema.Attributes[1].Header)
	assert.Len(t, leadSchema.Attributes, 2)

	// 2. Setup app extension overrides
	err = SetupBizDefFromFS(ctx, extensionFS, "crm", schemaService)
	assert.NoError(t, err)

	// Verify overridden lead schema
	leadSchema, err = schemaService.LoadEntity(ctx, "crm_lead")
	assert.NoError(t, err)
	assert.NotNil(t, leadSchema)
	assert.Equal(t, "Overridden Lead Name", leadSchema.Attributes[1].Header)
	assert.Len(t, leadSchema.Attributes, 3)
	assert.Equal(t, "custom_field", leadSchema.Attributes[2].Field)

	// Verify that contact schema is still intact
	contactSchema, err := schemaService.LoadEntity(ctx, "crm_contact")
	assert.NoError(t, err)
	assert.NotNil(t, contactSchema)
	assert.Equal(t, "Core Contact Name", contactSchema.Attributes[1].Header)

	// 3. Re-run identical app extension schema setup (should be idempotent and skip updating)
	err = SetupBizDefFromFS(ctx, extensionFS, "crm", schemaService)
	assert.NoError(t, err)

	// 4. Modify overriding schema and setup again (should update in-place)
	extensionFS["schemas/crm_lead.json"] = &fstest.MapFile{Data: []byte(`{
		"name": "crm_lead",
		"table_name": "crm_lead",
		"primary_key": "id",
		"label_attribute_name": "lead_name",
		"attributes": [
			{"field": "id", "header": "ID", "data_type": "integer"},
			{"field": "lead_name", "header": "Fully Updated Overridden Name", "data_type": "string"},
			{"field": "custom_field", "header": "Custom Field", "data_type": "string"}
		]
	}`)}

	err = SetupBizDefFromFS(ctx, extensionFS, "crm", schemaService)
	assert.NoError(t, err)

	// Verify fully updated lead schema
	leadSchema, err = schemaService.LoadEntity(ctx, "crm_lead")
	assert.NoError(t, err)
	assert.Equal(t, "Fully Updated Overridden Name", leadSchema.Attributes[1].Header)
}

package services

import (
	"context"
	"testing"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/stretchr/testify/assert"
)

func TestSchemaService(t *testing.T) {
	dao, err := relationdbdao.CreateDao("memory://")
	assert.NoError(t, err)
	defer dao.Close()

	ctx := context.Background()
	err = dao.EnsureTable(ctx)
	assert.NoError(t, err)

	svc := NewSchemaService(dao)

	t.Run("Save and Get All", func(t *testing.T) {
		s := &descriptors.Schema{
			Name: "test_entity",
			Type: descriptors.EntitySchema,
			Settings: &descriptors.SchemaSettings{
				Entity: &descriptors.Entity{
					Name: "test_entity",
				},
			},
		}

		saved, err := svc.Save(ctx, s, true)
		assert.NoError(t, err)
		assert.NotEmpty(t, saved.SchemaId)

		all, err := svc.All(ctx, nil, nil, nil)
		assert.NoError(t, err)
		assert.Len(t, all, 1)
		if len(all) > 0 {
			assert.Equal(t, "test_entity", all[0].Name)
		}
	})

	t.Run("BySchemaId", func(t *testing.T) {
		all, _ := svc.All(ctx, nil, nil, nil)
		if len(all) == 0 {
			t.Skip("No schemas found")
		}
		schemaId := all[0].SchemaId
		
		found, err := svc.BySchemaId(ctx, schemaId)
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, "test_entity", found.Name)
	})

	t.Run("Built-in Schemas ByNameOrDefault and BySchemaId", func(t *testing.T) {
		for _, name := range []string{"Role", "User", "UserRole", "DocPerm"} {
			schema, err := svc.ByNameOrDefault(ctx, name, descriptors.EntitySchema, nil)
			assert.NoError(t, err)
			assert.NotNil(t, schema)
			assert.Equal(t, name, schema.Name)
			assert.NotNil(t, schema.Settings)
			assert.NotNil(t, schema.Settings.Entity)
			assert.NotEmpty(t, schema.Settings.Entity.Attributes)

			bySchemaId, err := svc.BySchemaId(ctx, schema.SchemaId)
			assert.NoError(t, err)
			assert.NotNil(t, bySchemaId)
			assert.Equal(t, name, bySchemaId.Name)
		}
	})
}

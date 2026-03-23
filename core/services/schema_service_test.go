package services

import (
	"context"
	"os"
	"testing"

	"github.com/innomon/aigen-cms/core/descriptors"
	"github.com/innomon/aigen-cms/infrastructure/relationdbdao"
	"github.com/stretchr/testify/assert"
)

func TestSchemaService(t *testing.T) {
	dbFile := "test_schemas.db"
	os.Remove(dbFile)
	defer os.Remove(dbFile)

	dao, err := relationdbdao.CreateDao(descriptors.SQLite, dbFile)
	assert.NoError(t, err)
	defer dao.Close()

	// Manually create schema table for test
	_, err = dao.GetDb().Exec(`
		CREATE TABLE __schemas (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			schema_id TEXT,
			name TEXT,
			type TEXT,
			settings TEXT,
			description TEXT,
			is_latest BOOLEAN,
			publication_status TEXT,
			created_at DATETIME,
			created_by TEXT,
			deleted BOOLEAN
		)
	`)
	assert.NoError(t, err)

	svc := NewSchemaService(dao)
	ctx := context.Background()

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
		assert.Equal(t, "test_entity", all[0].Name)
	})
}

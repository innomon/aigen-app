package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/innomon/aigen-app/utils/datamodels"
	"github.com/stretchr/testify/assert"
)

func TestEvolutionIntegration(t *testing.T) {
	ctx := context.Background()
	dao, _ := relationdbdao.CreateDao("memory://")
	dao.EnsureTable(ctx)

	schemaSvc := NewSchemaService(dao)
	evolutionSvc := NewEvolutionService(dao, schemaSvc)
	permSvc := NewPermissionService(dao, schemaSvc)
	entitySvc := NewEntityService(schemaSvc, evolutionSvc, dao, permSvc)

	// 1. Setup Initial Schema
	entityName := "TestEntity"
	entitySchema := &descriptors.Entity{
		Name:       entityName,
		TableName:  "test_entities",
		PrimaryKey: "id",
		Attributes: []descriptors.Attribute{
			{Field: "id", DataType: "String"},
			{Field: "old_field", DataType: "String"},
		},
	}
	schemaSvc.Save(ctx, &descriptors.Schema{
		Name:     entityName,
		Type:     descriptors.EntitySchema,
		Settings: &descriptors.SchemaSettings{Entity: entitySchema},
	}, true)

	// 2. Insert v1 Records (manually so they don't have versioning yet)
	for i := 1; i <= 5; i++ {
		rec := datamodels.RecJSON{
			Namespace: fmt.Sprintf("aigen.bizdef.entities.%s", entityName),
			Key:       fmt.Sprintf("rec_%d", i),
			Rec: map[string]interface{}{
				"id":        fmt.Sprintf("rec_%d", i),
				"old_field": fmt.Sprintf("value_%d", i),
			},
			MetaData: datamodels.MetaData{
				Revision: 1,
			},
			Tmstamp: time.Now(),
		}
		// Use DAO directly to bypass EntityService JIT (which doesn't apply yet as there's no manifest)
		dao.Save(ctx, rec)
	}

	// 3. Register Evolution Manifest
	dateV2 := time.Now().Add(-1 * time.Hour).Truncate(time.Second)
	manifest := descriptors.EvolutionManifest{
		entityName: {
			"v2": {
				Date:        dateV2,
				Description: "V2 Upgrade",
				Actions: []descriptors.EvolutionAction{
					{Action: "rename", From: "old_field", To: "new_field"},
					{Action: "add", Field: "added_field", Default: "default_val"},
				},
			},
		},
	}
	evolutionSvc.RegisterManifest("test_bizdef", manifest)

	t.Run("JIT Upgrade on Single", func(t *testing.T) {
		record, err := entitySvc.Single(ctx, entityName, "rec_1")
		assert.NoError(t, err)
		assert.Equal(t, "value_1", record["new_field"])
		assert.Equal(t, "default_val", record["added_field"])
		assert.NotContains(t, record, "old_field")
	})

	t.Run("JIT Upgrade on List", func(t *testing.T) {
		records, _, err := entitySvc.List(ctx, entityName, datamodels.Pagination{}, nil, nil)
		assert.NoError(t, err)
		assert.Len(t, records, 5)
		for _, r := range records {
			assert.Contains(t, r, "new_field")
			assert.Contains(t, r, "added_field")
			assert.NotContains(t, r, "old_field")
		}
	})

	t.Run("Batch Migration (Scrubber)", func(t *testing.T) {
		upgraded, failed, err := evolutionSvc.ScrubEntity(ctx, entityName, 2)
		assert.NoError(t, err)
		assert.Equal(t, 5, upgraded)
		assert.Equal(t, 0, failed)

		// Verify records in DB are actually upgraded
		for i := 1; i <= 5; i++ {
			rec, err := dao.Get(ctx, fmt.Sprintf("aigen.bizdef.entities.%s", entityName), fmt.Sprintf("rec_%d", i))
			assert.NoError(t, err)
			assert.Equal(t, "v2", rec.MetaData.SchemaVersion)
			// Original was 1. dao.Save made it 2. ScrubEntity calls SaveConditional(expected=2), which sets it to 3.
			assert.Equal(t, int64(3), rec.MetaData.Revision)
			
			recData := rec.Rec.(map[string]interface{})
			assert.Contains(t, recData, "new_field")
			assert.NotContains(t, recData, "old_field")
		}
	})

	t.Run("Optimistic Concurrency Conflict", func(t *testing.T) {
		// 1. Create a record
		key := "conflict_rec"
		namespace := fmt.Sprintf("aigen.bizdef.entities.%s", entityName)
		dao.Save(ctx, datamodels.RecJSON{
			Namespace: namespace,
			Key:       key,
			Rec:       map[string]interface{}{"id": key},
			MetaData:  datamodels.MetaData{Revision: 0},
		})
		// DB Revision becomes 1

		// 2. Successful update
		rec, _ := dao.Get(ctx, namespace, key) // rec.MetaData.Revision is 1
		err := dao.SaveConditional(ctx, *rec, 1)
		assert.NoError(t, err)
		// Now revision is 2

		// 3. Original process tries to update with old revision (1)
		err = dao.SaveConditional(ctx, *rec, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "optimistic concurrency conflict")
	})
}

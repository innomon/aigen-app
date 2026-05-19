package bizdefs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/core/services"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/innomon/aigen-app/utils/datamodels"
)

type BizDefsConfig struct {
	EnabledBizDefs []string `json:"enabled_bizdefs"`
}

func LoadBizDefsConfig(bizdefsDir string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(bizdefsDir, "bizdefs.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cfg BizDefsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return cfg.EnabledBizDefs, nil
}

func SetupBizDef(ctx context.Context, bizdefsDir string, bizdefName string, schemaService *services.SchemaService, dao relationdbdao.IPrimaryDao) error {
	schemasDir := filepath.Join(bizdefsDir, bizdefName, "schemas")
	files, err := os.ReadDir(schemasDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read %s schemas directory: %w", bizdefName, err)
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(schemasDir, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read schema file %s: %w", filePath, err)
		}

		var entity descriptors.Entity
		if err := json.Unmarshal(data, &entity); err != nil {
			return fmt.Errorf("failed to parse schema file %s: %w", filePath, err)
		}

		existing, err := schemaService.ByNameOrDefault(ctx, entity.Name, descriptors.EntitySchema, nil)
		if err != nil {
			return fmt.Errorf("failed to check existing schema %s: %w", entity.Name, err)
		}

		if existing != nil {
			continue
		}

		schema := &descriptors.Schema{
			Name:              entity.Name,
			Type:              descriptors.EntitySchema,
			IsLatest:          true,
			PublicationStatus: descriptors.Published,
			Settings: &descriptors.SchemaSettings{
				Entity: &entity,
			},
		}

		_, err = schemaService.Save(ctx, schema, true)
		if err != nil {
			return fmt.Errorf("failed to save schema %s: %w", entity.Name, err)
		}

		fmt.Printf("Registered schema: %s (BizDef: %s)\n", entity.Name, bizdefName)
	}

	return nil
}

type TestDataEntry struct {
	Entity   string
	Ref      string
	Data     map[string]interface{}
	Children map[string][]map[string]interface{}
}

func SetupBizDefTestData(ctx context.Context, bizdefsDir string, bizdefName string, entityService services.IEntityService) error {
	filePath := filepath.Join(bizdefsDir, bizdefName, "data", "test_data.json")
	dataBytes, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read test_data.json for %s: %v", bizdefName, err)
	}

	fmt.Printf("Setting up test data for %s from JSON...\n", bizdefName)

	var entries []TestDataEntry
	if err := json.Unmarshal(dataBytes, &entries); err != nil {
		return fmt.Errorf("failed to unmarshal test data: %v", err)
	}

	if len(entries) > 0 {
		limit := "1"
		records, _, err := entityService.List(ctx, entries[0].Entity, datamodels.Pagination{Limit: &limit}, nil, nil)
		if err != nil {
			return fmt.Errorf("failed to list %s: %v", entries[0].Entity, err)
		}
		if len(records) > 0 {
			fmt.Printf("Test data already exists for %s, skipping.\n", bizdefName)
			return nil
		}
	}

	refMap := make(map[string]interface{})

	resolveRefs := func(data map[string]interface{}) {
		for k, v := range data {
			if strVal, ok := v.(string); ok && strings.HasPrefix(strVal, "$Ref:") {
				refKey := strings.TrimPrefix(strVal, "$Ref:")
				if resolvedVal, exists := refMap[refKey]; exists {
					data[k] = resolvedVal
				}
			}
		}
	}

	for _, entry := range entries {
		resolveRefs(entry.Data)

		rec, err := entityService.Insert(ctx, entry.Entity, entry.Data)
		if err != nil {
			return fmt.Errorf("failed to insert %s (Ref: %s): %v", entry.Entity, entry.Ref, err)
		}

		if entry.Ref != "" {
			refMap[entry.Ref] = rec["id"]
		}

		if entry.Children != nil {
			for childAttr, childrenArr := range entry.Children {
				for i, childData := range childrenArr {
					resolveRefs(childData)
					_, err = entityService.CollectionInsert(ctx, entry.Entity, fmt.Sprintf("%v", rec["id"]), childAttr, childData)
					if err != nil {
						return fmt.Errorf("failed to insert child %s for %s (Ref: %s) at index %d: %v", childAttr, entry.Entity, entry.Ref, i, err)
					}
				}
			}
		}
	}

	fmt.Printf("Test data successfully created for %s.\n", bizdefName)
	return nil
}

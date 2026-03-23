package apps

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/innomon/aigen-cms/core/descriptors"
	"github.com/innomon/aigen-cms/core/services"
	"github.com/innomon/aigen-cms/infrastructure/relationdbdao"
	"github.com/innomon/aigen-cms/utils/datamodels"
)

type AppsConfig struct {
	EnabledApps []string `json:"enabled_apps"`
}

func LoadAppsConfig(appsDir string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(appsDir, "apps.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No config, no apps
		}
		return nil, err
	}
	var cfg AppsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return cfg.EnabledApps, nil
}

func SetupApp(ctx context.Context, appsDir string, appName string, schemaService *services.SchemaService, dao relationdbdao.IPrimaryDao) error {
	schemasDir := filepath.Join(appsDir, appName, "schemas")
	files, err := os.ReadDir(schemasDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Nothing to do
		}
		return fmt.Errorf("failed to read %s schemas directory: %w", appName, err)
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

		// Check if schema already exists to avoid duplicates
		existing, err := schemaService.ByNameOrDefault(ctx, entity.Name, descriptors.EntitySchema, nil)
		if err != nil {
			return fmt.Errorf("failed to check existing schema %s: %w", entity.Name, err)
		}

		if existing != nil {
			continue // Schema already exists
		}

		// CREATE TABLE
		var cols []datamodels.Column
		// Default fields
		cols = append(cols, datamodels.Column{Name: "id", Type: datamodels.Id})

		for _, attr := range entity.Attributes {
			if !attr.DataType.IsLocal() {
				continue
			}

			var colType datamodels.ColumnType
			switch attr.DataType {
			case descriptors.Int:
				colType = datamodels.Int
			case descriptors.Float:
				colType = datamodels.Float
			case descriptors.Datetime:
				colType = datamodels.Datetime
			case descriptors.Boolean:
				colType = datamodels.Boolean
			case descriptors.Text:
				colType = datamodels.Text
			case descriptors.String, descriptors.DataTypeLookup:
				colType = datamodels.String // IDs are UUIDs (strings)
			default:
				colType = datamodels.String
			}

			cols = append(cols, datamodels.Column{Name: attr.Field, Type: colType})
		}

		// System fields
		cols = append(cols, datamodels.Column{Name: "created_at", Type: datamodels.CreatedTime})
		cols = append(cols, datamodels.Column{Name: "updated_at", Type: datamodels.UpdatedTime})
		cols = append(cols, datamodels.Column{Name: "deleted", Type: datamodels.Boolean})

		err = dao.CreateTable(ctx, entity.TableName, cols)
		if err != nil && !isTableExistsError(err) {
			return fmt.Errorf("failed to create table %s: %w", entity.TableName, err)
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

		fmt.Printf("Registered schema and created table: %s\n", entity.Name)
	}

	return nil
}

func isTableExistsError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "already exists") || strings.Contains(errStr, "exists")
}

type TestDataEntry struct {
	Entity   string
	Ref      string
	Data     map[string]interface{}
	Children map[string][]map[string]interface{}
}

func SetupAppTestData(ctx context.Context, appsDir string, appName string, entityService services.IEntityService) error {
	filePath := filepath.Join(appsDir, appName, "data", "test_data.json")
	dataBytes, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Nothing to do
		}
		return fmt.Errorf("failed to read test_data.json for %s: %v", appName, err)
	}

	fmt.Printf("Setting up test data for %s from JSON...\n", appName)

	var entries []TestDataEntry
	if err := json.Unmarshal(dataBytes, &entries); err != nil {
		return fmt.Errorf("failed to unmarshal test data: %v", err)
	}

	// For deduplication logic, we could check if the first entry entity exists
	if len(entries) > 0 {
		limit := "1"
		records, _, err := entityService.List(ctx, entries[0].Entity, datamodels.Pagination{Limit: &limit}, nil, nil)
		if err != nil {
			return fmt.Errorf("failed to list %s: %v", entries[0].Entity, err)
		}
		if len(records) > 0 {
			fmt.Printf("Test data already exists for %s, skipping.\n", appName)
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
				} else {
					fmt.Printf("Warning: Could not resolve reference %s\n", refKey)
				}
			}
		}
	}

	for _, entry := range entries {
		// Prepare data, resolve references
		resolveRefs(entry.Data)

		rec, err := entityService.Insert(ctx, entry.Entity, entry.Data)
		if err != nil {
			return fmt.Errorf("failed to insert %s (Ref: %s): %v", entry.Entity, entry.Ref, err)
		}

		if entry.Ref != "" {
			refMap[entry.Ref] = rec["id"]
		}

		// Insert children
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

	fmt.Printf("Test data successfully created for %s.\n", appName)
	return nil
}

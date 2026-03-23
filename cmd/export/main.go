package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/innomon/aigen-cms/core/descriptors"
	"github.com/innomon/aigen-cms/core/services"
	"github.com/innomon/aigen-cms/infrastructure/relationdbdao"
	"github.com/innomon/aigen-cms/utils/datamodels"
)

func main() {
	dbPath := flag.String("db", "formcms.db", "Path to SQLite database")
	outDir := flag.String("out", "exports", "Output directory for schemas and data")
	flag.Parse()

	log.Printf("Starting export from %s to %s", *dbPath, *outDir)

	dao, err := relationdbdao.CreateDao(descriptors.SQLite, *dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dao.Close()

	schemaService := services.NewSchemaService(dao)
	permissionService := services.NewPermissionService(dao, schemaService)
	entityService := services.NewEntityService(schemaService, dao, permissionService)

	ctx := context.Background()

	schemas, err := schemaService.All(ctx, nil, nil, nil)
	if err != nil {
		log.Fatalf("Failed to fetch schemas: %v", err)
	}

	for _, schema := range schemas {
		var content interface{}
		subDir := string(schema.Type)
		fileName := fmt.Sprintf("%s.json", schema.Name)

		if schema.Settings != nil {
			switch schema.Type {
			case descriptors.EntitySchema:
				content = schema.Settings.Entity
			case descriptors.MenuSchema:
				content = schema.Settings.Menu
			case descriptors.PageSchema:
				content = schema.Settings.Page
			case descriptors.QuerySchema:
				content = schema.Settings.Query
			default:
				content = schema.Settings
			}
		}

		if content == nil {
			log.Printf("Warning: schema %s has no settings to export", schema.Name)
			continue
		}

		schemaPath := filepath.Join(*outDir, "schemas", subDir, fileName)
		if err := saveJSON(schemaPath, content); err != nil {
			log.Printf("Error saving schema %s: %v", schema.Name, err)
		} else {
			log.Printf("Exported schema: %s", schemaPath)
		}

		// Export data if it's an entity schema
		if schema.Type == descriptors.EntitySchema {
			limit := "10000" // reasonable limit for export
			results, _, err := entityService.List(ctx, schema.Name, datamodels.Pagination{Limit: &limit}, nil, nil)
			if err != nil {
				log.Printf("Warning: failed to list data for %s: %v", schema.Name, err)
				continue
			}

			if len(results) > 0 {
				dataPath := filepath.Join(*outDir, "data", fileName)
				if err := saveJSON(dataPath, results); err != nil {
					log.Printf("Error saving data for %s: %v", schema.Name, err)
				} else {
					log.Printf("Exported %d records to %s", len(results), dataPath)
				}
			}
		}
	}

	log.Println("Export complete.")
}

func saveJSON(path string, data interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

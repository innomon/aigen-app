package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/innomon/aigen-cms/core/descriptors"
	"github.com/innomon/aigen-cms/core/services"
	"github.com/innomon/aigen-cms/infrastructure/relationdbdao"
	"github.com/innomon/aigen-cms/utils/datamodels"
)

func isTableExistsError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "already exists") || strings.Contains(errStr, "exists")
}

func main() {
	dbPath := flag.String("db", "formcms.db", "Path to target SQLite database")
	inDir := flag.String("in", "exports", "Input directory for schemas and data")
	flag.Parse()

	log.Printf("Starting import from %s to %s", *inDir, *dbPath)

	dao, err := relationdbdao.CreateDao(descriptors.SQLite, *dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dao.Close()

	// Ensure core tables exist
	_, err = dao.GetDb().ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS __schemas (
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
		);
		CREATE TABLE IF NOT EXISTS __users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL,
			avatar_path TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);
	`)
	if err != nil {
		log.Fatalf("Failed to create core tables: %v", err)
	}

	schemaService := services.NewSchemaService(dao)
	ctx := context.Background()

	// 1. Import Schemas
	schemasDir := filepath.Join(*inDir, "schemas")
	if _, err := os.Stat(schemasDir); err == nil {
		importSchemas(ctx, dao, schemaService, schemasDir)
	} else {
		log.Printf("Schemas directory not found: %s", schemasDir)
	}

	// 2. Import Data
	dataDir := filepath.Join(*inDir, "data")
	if _, err := os.Stat(dataDir); err == nil {
		importData(ctx, dao, schemaService, dataDir)
	} else {
		log.Printf("Data directory not found: %s", dataDir)
	}

	log.Println("Import complete.")
}

func importSchemas(ctx context.Context, dao relationdbdao.IPrimaryDao, schemaService *services.SchemaService, schemasDir string) {
	types := []descriptors.SchemaType{
		descriptors.EntitySchema,
		descriptors.PageSchema,
		descriptors.MenuSchema,
		descriptors.QuerySchema,
	}

	for _, schemaType := range types {
		typeDir := filepath.Join(schemasDir, string(schemaType))
		files, err := os.ReadDir(typeDir)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Printf("Warning: failed to read %s schemas directory: %v", schemaType, err)
			}
			continue
		}

		for _, file := range files {
			if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
				continue
			}

			filePath := filepath.Join(typeDir, file.Name())
			dataBytes, err := os.ReadFile(filePath)
			if err != nil {
				log.Printf("Error reading file %s: %v", filePath, err)
				continue
			}

			schemaName := strings.TrimSuffix(file.Name(), ".json")
			settings := &descriptors.SchemaSettings{}

			var entity *descriptors.Entity
			switch schemaType {
			case descriptors.EntitySchema:
				entity = &descriptors.Entity{}
				if err := json.Unmarshal(dataBytes, entity); err == nil {
					settings.Entity = entity
				}
			case descriptors.MenuSchema:
				menu := &descriptors.Menu{}
				if err := json.Unmarshal(dataBytes, menu); err == nil {
					settings.Menu = menu
				}
			case descriptors.PageSchema:
				page := &descriptors.Page{}
				if err := json.Unmarshal(dataBytes, page); err == nil {
					settings.Page = page
				}
			case descriptors.QuerySchema:
				query := &descriptors.Query{}
				if err := json.Unmarshal(dataBytes, query); err == nil {
					settings.Query = query
				}
			}

			// For EntitySchema, construct the table if not exists
			if schemaType == descriptors.EntitySchema && settings.Entity != nil {
				createTableForEntity(ctx, dao, settings.Entity)
			}

			existing, _ := schemaService.ByNameOrDefault(ctx, schemaName, schemaType, nil)
			if existing != nil {
				log.Printf("Schema %s (%s) already exists, skipping...", schemaName, schemaType)
				continue
			}

			schema := &descriptors.Schema{
				Name:              schemaName,
				Type:              schemaType,
				IsLatest:          true,
				PublicationStatus: descriptors.Published,
				Settings:          settings,
			}

			_, err = schemaService.Save(ctx, schema, true)
			if err != nil {
				log.Printf("Failed to save schema %s: %v", schemaName, err)
			} else {
				log.Printf("Imported schema: %s (%s)", schemaName, schemaType)
			}
		}
	}
}

func createTableForEntity(ctx context.Context, dao relationdbdao.IPrimaryDao, entity *descriptors.Entity) {
	var cols []datamodels.Column
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
			colType = datamodels.String
		default:
			colType = datamodels.String
		}
		cols = append(cols, datamodels.Column{Name: attr.Field, Type: colType})
	}

	cols = append(cols, datamodels.Column{Name: "created_at", Type: datamodels.CreatedTime})
	cols = append(cols, datamodels.Column{Name: "updated_at", Type: datamodels.UpdatedTime})
	cols = append(cols, datamodels.Column{Name: "deleted", Type: datamodels.Boolean})

	err := dao.CreateTable(ctx, entity.TableName, cols)
	if err != nil && !isTableExistsError(err) {
		log.Printf("Failed to create table %s: %v", entity.TableName, err)
	}
}

func importData(ctx context.Context, dao relationdbdao.IPrimaryDao, schemaService *services.SchemaService, dataDir string) {
	files, err := os.ReadDir(dataDir)
	if err != nil {
		log.Printf("Warning: failed to read data directory: %v", err)
		return
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		schemaName := strings.TrimSuffix(file.Name(), ".json")
		entity, err := schemaService.LoadEntity(ctx, schemaName)
		if err != nil {
			log.Printf("Warning: Cannot import data for %s because entity schema is missing.", schemaName)
			continue
		}

		filePath := filepath.Join(dataDir, file.Name())
		dataBytes, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("Error reading data file %s: %v", filePath, err)
			continue
		}

		var records []map[string]interface{}
		if err := json.Unmarshal(dataBytes, &records); err != nil {
			log.Printf("Error parsing JSON data in %s: %v", filePath, err)
			continue
		}

		importedCount := 0
		for _, record := range records {
			// Basic duplication check by id if it exists
			if idVal, ok := record["id"]; ok {
				query, args, _ := dao.GetBuilder().Select("count(*)").From(entity.TableName).Where(squirrel.Eq{"id": idVal}).ToSql()
				var count int
				if err := dao.GetDb().QueryRowContext(ctx, query, args...).Scan(&count); err == nil && count > 0 {
					continue // skip existing row
				}
			}

			// Clean fields and normalize types for cross-database compatibility (e.g. SQLite 0/1 to Postgres true/false)
			var cols []string
			var vals []interface{}
			for k, v := range record {
				// Normalize Booleans
				for _, attr := range entity.Attributes {
					if attr.Field == k && attr.DataType == descriptors.Boolean {
						if floatVal, ok := v.(float64); ok { // JSON unmarshals numbers to float64
							if floatVal == 1 {
								v = true
							} else if floatVal == 0 {
								v = false
							}
						}
					}
				}
				// Skip 'deleted' if it's nil
				if k == "deleted" && v == nil {
					v = false
				}

				cols = append(cols, k)
				vals = append(vals, v)
			}

			insertQuery, args, err := dao.GetBuilder().Insert(entity.TableName).Columns(cols...).Values(vals...).ToSql()
			if err != nil {
				log.Printf("Error building insert query for %s: %v", entity.TableName, err)
				continue
			}

			if _, err := dao.GetDb().ExecContext(ctx, insertQuery, args...); err != nil {
				log.Printf("Error inserting record into %s: %v", entity.TableName, err)
				continue
			}
			importedCount++
		}

		if importedCount > 0 {
			log.Printf("Imported %d new records into %s", importedCount, entity.TableName)
		} else {
			log.Printf("Checked %d records for %s (no new records imported)", len(records), entity.TableName)
		}
	}
}

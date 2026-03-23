package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/core/services"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/innomon/aigen-app/utils/datamodels"
)

func main() {
	dbPath := flag.String("db", "formcms.db", "Path to target SQLite database")
	inDir := flag.String("in", "exports", "Input directory for schemas and data")
	flag.Parse()

	log.Printf("Starting import from %s to %s", *inDir, *dbPath)

	dao, err := relationdbdao.CreateDao(*dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dao.Close()

	if err := dao.EnsureTable(context.Background()); err != nil {
		log.Fatalf("Failed to ensure records table: %v", err)
	}

	schemaService := services.NewSchemaService(dao)
	ctx := context.Background()

	// 1. Import Schemas
	schemasDir := filepath.Join(*inDir, "schemas")
	if _, err := os.Stat(schemasDir); err == nil {
		importSchemas(ctx, dao, schemaService, schemasDir)
	}

	// 2. Import Data
	dataDir := filepath.Join(*inDir, "data")
	if _, err := os.Stat(dataDir); err == nil {
		importData(ctx, dao, schemaService, dataDir)
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
			continue
		}

		for _, file := range files {
			if filepath.Ext(file.Name()) != ".json" {
				continue
			}

			filePath := filepath.Join(typeDir, file.Name())
			dataBytes, _ := os.ReadFile(filePath)
			schemaName := strings.TrimSuffix(file.Name(), ".json")
			settings := &descriptors.SchemaSettings{}

			switch schemaType {
			case descriptors.EntitySchema:
				entity := &descriptors.Entity{}
				json.Unmarshal(dataBytes, entity)
				settings.Entity = entity
			case descriptors.MenuSchema:
				menu := &descriptors.Menu{}
				json.Unmarshal(dataBytes, menu)
				settings.Menu = menu
			case descriptors.PageSchema:
				page := &descriptors.Page{}
				json.Unmarshal(dataBytes, page)
				settings.Page = page
			case descriptors.QuerySchema:
				query := &descriptors.Query{}
				json.Unmarshal(dataBytes, query)
				settings.Query = query
			}

			schema := &descriptors.Schema{
				Name:              schemaName,
				Type:              schemaType,
				IsLatest:          true,
				PublicationStatus: descriptors.Published,
				Settings:          settings,
			}

			schemaService.Save(ctx, schema, true)
			log.Printf("Imported schema: %s (%s)", schemaName, schemaType)
		}
	}
}

func importData(ctx context.Context, dao relationdbdao.IPrimaryDao, schemaService *services.SchemaService, dataDir string) {
	files, err := os.ReadDir(dataDir)
	if err != nil {
		return
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		schemaName := strings.TrimSuffix(file.Name(), ".json")
		filePath := filepath.Join(dataDir, file.Name())
		dataBytes, _ := os.ReadFile(filePath)

		var records []map[string]interface{}
		json.Unmarshal(dataBytes, &records)

		for _, record := range records {
			idVal := record["id"]
			if idVal == nil {
				idVal = time.Now().UnixNano()
			}

			rec := datamodels.RecJSON{
				Namespace: "aigen.app.entities." + schemaName,
				Key:       fmt.Sprintf("%v", idVal),
				Rec:       record,
				Tmstamp:   time.Now(),
			}
			dao.Save(ctx, rec)
		}
		log.Printf("Imported %d records into %s", len(records), schemaName)
	}
}

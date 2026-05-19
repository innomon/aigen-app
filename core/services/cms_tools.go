package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/innomon/aigen-app/utils/datamodels"
	"github.com/innomon/agentic/pkg/registry"
)

func RegisterCMSTools(entityService IEntityService, schemaService *SchemaService, a2uiService *A2UIService) {
	registry.RegisterToolHandler("cms_bizdef_list", func(ctx context.Context, args map[string]any) (any, error) {
		bizdefsFile := filepath.Join("bizdefs", "bizdefs.json")
		b, err := os.ReadFile(bizdefsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read bizdefs.json: %w", err)
		}
		var bizdefsConfig struct {
			EnabledBizDefs []string `json:"enabled_bizdefs"`
		}
		if err := json.Unmarshal(b, &bizdefsConfig); err != nil {
			return nil, fmt.Errorf("failed to parse bizdefs.json: %w", err)
		}

		type BizDefSummary struct {
			Name        string `json:"name"`
			DisplayName string `json:"display_name,omitempty"`
			Description string `json:"description,omitempty"`
		}
		var summaries []BizDefSummary

		for _, bizdefName := range bizdefsConfig.EnabledBizDefs {
			defFile := filepath.Join("bizdefs", bizdefName, "bizdef.json")
			defBytes, err := os.ReadFile(defFile)
			if err != nil {
				// bizdef.json is optional, just add the name if missing
				summaries = append(summaries, BizDefSummary{Name: bizdefName})
				continue
			}
			var def struct {
				Name        string `json:"name"`
				DisplayName string `json:"display_name"`
				Description string `json:"description"`
			}
			if err := json.Unmarshal(defBytes, &def); err == nil {
				if def.Name == "" {
					def.Name = bizdefName
				}
				summaries = append(summaries, BizDefSummary{
					Name:        def.Name,
					DisplayName: def.DisplayName,
					Description: def.Description,
				})
			} else {
				summaries = append(summaries, BizDefSummary{Name: bizdefName})
			}
		}

		return summaries, nil
	})

	registry.RegisterToolHandler("cms_bizdef_get", func(ctx context.Context, args map[string]any) (any, error) {
		name, ok := args["name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'name' argument")
		}

		defFile := filepath.Join("bizdefs", name, "bizdef.json")
		defBytes, err := os.ReadFile(defFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read bizdef definition for %s: %w", name, err)
		}

		var bizdefDef map[string]any
		if err := json.Unmarshal(defBytes, &bizdefDef); err != nil {
			return nil, fmt.Errorf("failed to parse bizdef definition for %s: %w", name, err)
		}

		// Try to read context files for entities
		if entities, ok := bizdefDef["entities"].(map[string]any); ok {
			for entityName, entityDataRaw := range entities {
				if entityData, ok := entityDataRaw.(map[string]any); ok {
					if contextFile, ok := entityData["context_file"].(string); ok && contextFile != "" {
						ctxFilePath := filepath.Join("bizdefs", name, contextFile)
						ctxBytes, err := os.ReadFile(ctxFilePath)
						if err == nil {
							entityData["context_content"] = string(ctxBytes)
						} else {
							entityData["context_content"] = fmt.Sprintf("failed to read context file: %v", err)
						}
					}
				}
				entities[entityName] = entityDataRaw
			}
			bizdefDef["entities"] = entities
		}

		return bizdefDef, nil
	})

	registry.RegisterToolHandler("cms_entity_list", func(ctx context.Context, args map[string]any) (any, error) {
		name, ok := args["name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'name' argument")
		}

		limit := "10"
		if l, ok := args["limit"].(string); ok {
			limit = l
		}

		records, _, err := entityService.List(ctx, name, datamodels.Pagination{Limit: &limit}, nil, nil)
		return records, err
	})

	registry.RegisterToolHandler("cms_entity_get", func(ctx context.Context, args map[string]any) (any, error) {
		name, ok := args["name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'name' argument")
		}
		id, ok := args["id"]
		if !ok {
			return nil, fmt.Errorf("missing 'id' argument")
		}

		record, err := entityService.Single(ctx, name, id)
		return record, err
	})

	registry.RegisterToolHandler("cms_entity_create", func(ctx context.Context, args map[string]any) (any, error) {
		name, ok := args["name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'name' argument")
		}
		data, ok := args["data"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'data' argument")
		}

		record, err := entityService.Insert(ctx, name, data)
		return record, err
	})

	registry.RegisterToolHandler("cms_schema_list", func(ctx context.Context, args map[string]any) (any, error) {
		schemas, err := schemaService.All(ctx, nil, nil, nil)
		return schemas, err
	})

	registry.RegisterToolHandler("cms_a2ui_update", func(ctx context.Context, args map[string]any) (any, error) {
		id, ok := args["id"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'id' argument")
		}
		typ, ok := args["type"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'type' argument")
		}

		var attrs map[string]interface{}
		if a, ok := args["attributes"].(map[string]interface{}); ok {
			attrs = a
		} else if a, ok := args["attributes"].(map[string]any); ok {
			attrs = make(map[string]interface{})
			for k, v := range a {
				attrs[k] = v
			}
		}

		var children []string
		if c, ok := args["children"].([]interface{}); ok {
			for _, child := range c {
				if str, ok := child.(string); ok {
					children = append(children, str)
				}
			}
		} else if c, ok := args["children"].([]string); ok {
			children = c
		}

		comp := A2UIComponent{
			ID:         id,
			Type:       typ,
			Attributes: attrs,
			Children:   children,
		}

		a2uiService.UpdateComponent(ctx, comp)
		return "Component updated successfully", nil
	})
}

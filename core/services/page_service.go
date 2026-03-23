package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/aymerick/raymond"
	"github.com/innomon/aigen-cms/core/descriptors"
	"github.com/innomon/aigen-cms/utils/datamodels"
)

type PageService struct {
	schemaService  ISchemaService
	graphqlService IGraphQLService
}

func NewPageService(schemaService ISchemaService, graphqlService IGraphQLService) *PageService {
	return &PageService{
		schemaService:  schemaService,
		graphqlService: graphqlService,
	}
}

func (s *PageService) Render(ctx context.Context, path string, strArgs datamodels.StrArgs) (string, error) {
	page, err := s.resolvePage(ctx, path)
	if err != nil {
		return "", err
	}

	data, err := s.GetPageData(ctx, page, strArgs)
	if err != nil {
		return "", err
	}

	tpl, err := raymond.Parse(page.Html)
	if err != nil {
		return "", err
	}

	return tpl.Exec(data)
}

func (s *PageService) resolvePage(ctx context.Context, path string) (*descriptors.Page, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	var name string
	matchPrefix := false

	if len(parts) == 0 || (len(parts) == 1 && parts[0] == "") {
		name = "home"
	} else {
		name = parts[0]
		matchPrefix = len(parts) > 1
	}

	var schema *descriptors.Schema
	var err error
	if matchPrefix {
		schema, err = s.schemaService.ByStartsOrDefault(ctx, name, descriptors.PageSchema, nil)
	} else {
		schema, err = s.schemaService.ByNameOrDefault(ctx, name, descriptors.PageSchema, nil)
	}

	if err != nil || schema == nil || schema.Settings == nil || schema.Settings.Page == nil {
		return nil, fmt.Errorf("page %s not found", path)
	}

	return schema.Settings.Page, nil
}

func (s *PageService) GetPageData(ctx context.Context, page *descriptors.Page, strArgs datamodels.StrArgs) (map[string]interface{}, error) {
	data := make(map[string]interface{})
	if page.Metadata == nil || page.Metadata.Architecture == nil {
		return data, nil
	}

	for _, sq := range page.Metadata.Architecture.SelectedQueries {
		variables := make(map[string]interface{})
		// Simple variable mapping for now
		for k, v := range strArgs {
			if len(v) > 0 {
				variables[k] = v[0]
			}
		}

		res, err := s.graphqlService.ExecuteStoredQuery(ctx, sq.QueryName, variables)
		if err != nil {
			// Log error but continue
			continue
		}
		data[sq.FieldName] = res
	}

	return data, nil
}

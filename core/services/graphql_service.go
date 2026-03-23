package services

import (
	"context"
	"fmt"

	"github.com/innomon/aigen-cms/core/descriptors"
	"github.com/innomon/aigen-cms/utils/datamodels"
	"github.com/graphql-go/graphql"
)

type GraphQLService struct {
	schemaService ISchemaService
	entityService IEntityService
}

func NewGraphQLService(schemaService ISchemaService, entityService IEntityService) *GraphQLService {
	return &GraphQLService{
		schemaService: schemaService,
		entityService: entityService,
	}
}

func (s *GraphQLService) Query(ctx context.Context, query string, variables map[string]interface{}) (interface{}, error) {
	schema, err := s.BuildSchema(ctx)
	if err != nil {
		return nil, err
	}

	params := graphql.Params{
		Schema:         schema,
		RequestString:  query,
		VariableValues: variables,
		Context:        ctx,
	}
	res := graphql.Do(params)
	if len(res.Errors) > 0 {
		return nil, fmt.Errorf("errors: %v", res.Errors)
	}
	return res.Data, nil
}

func (s *GraphQLService) ExecuteStoredQuery(ctx context.Context, name string, variables map[string]interface{}) (interface{}, error) {
	schema, err := s.schemaService.ByNameOrDefault(ctx, name, descriptors.QuerySchema, nil)
	if err != nil {
		return nil, err
	}
	if schema == nil || schema.Settings == nil || schema.Settings.Query == nil {
		return nil, fmt.Errorf("stored query %s not found", name)
	}

	return s.Query(ctx, schema.Settings.Query.Source, variables)
}

func (s *GraphQLService) BuildSchema(ctx context.Context) (graphql.Schema, error) {
	schemas, err := s.schemaService.All(ctx, nil, nil, nil)
	if err != nil {
		return graphql.Schema{}, err
	}

	entityMap := make(map[string]*descriptors.Entity)
	for _, sc := range schemas {
		if sc.Type == descriptors.EntitySchema && sc.Settings != nil && sc.Settings.Entity != nil {
			entityMap[sc.Name] = sc.Settings.Entity
		}
	}

	graphMap := make(map[string]*graphql.Object)

	// First pass: Create plain types
	for name, entity := range entityMap {
		fields := graphql.Fields{}
		for _, attr := range entity.Attributes {
			if !attr.DataType.IsCompound() {
				fields[attr.Field] = &graphql.Field{
					Type: s.getGraphQLType(attr),
				}
			}
		}
		graphMap[name] = graphql.NewObject(graphql.ObjectConfig{
			Name:   name,
			Fields: fields,
		})
	}

	// Second pass: Add compound types and root queries
	queryFields := graphql.Fields{}
	for name, entity := range entityMap {
		obj := graphMap[name]
		for _, attr := range entity.Attributes {
			if attr.DataType.IsCompound() {
				// Simplified: assume target entity is in entityMap
				// In a full implementation, we'd use Options to find the target
				// For now, let's just implement the root queries
			}
		}

		entityName := name
		queryFields[entityName] = &graphql.Field{
			Type: obj,
			Args: graphql.FieldConfigArgument{
				"id": &graphql.ArgumentConfig{Type: graphql.Int},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				id := p.Args["id"]
				return s.entityService.Single(p.Context, entityName, id)
			},
		}

		queryFields[entityName+"List"] = &graphql.Field{
			Type: graphql.NewList(obj),
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				records, _, err := s.entityService.List(p.Context, entityName, datamodels.Pagination{}, nil, nil)
				return records, err
			},
		}
	}

	rootQuery := graphql.NewObject(graphql.ObjectConfig{
		Name:   "Query",
		Fields: queryFields,
	})

	return graphql.NewSchema(graphql.SchemaConfig{
		Query: rootQuery,
	})
}

func (s *GraphQLService) getGraphQLType(attr descriptors.Attribute) graphql.Output {
	switch attr.DataType {
	case descriptors.Int:
		return graphql.Int
	case descriptors.Datetime:
		return graphql.DateTime
	case descriptors.Boolean:
		return graphql.Boolean
	default:
		return graphql.String
	}
}

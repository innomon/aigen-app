package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/innomon/aigen-app/utils/datamodels"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

const SchemaNamespace = "aigen.core.descriptors.Schema"

type SchemaService struct {
	dao relationdbdao.IPrimaryDao
}

func NewSchemaService(dao relationdbdao.IPrimaryDao) *SchemaService {
	return &SchemaService{dao: dao}
}

func (s *SchemaService) All(ctx context.Context, schemaType *descriptors.SchemaType, names []string, status *descriptors.PublicationStatus) ([]*descriptors.Schema, error) {
	filters := []datamodels.Filter{
		{
			FieldName: "deleted",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{false}},
			},
		},
	}

	if schemaType != nil {
		filters = append(filters, datamodels.Filter{
			FieldName: "type",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{*schemaType}},
			},
		})
	}
	if len(names) > 0 {
		vals := make([]interface{}, len(names))
		for i, n := range names {
			vals[i] = n
		}
		filters = append(filters, datamodels.Filter{
			FieldName: "name",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: vals},
			},
		})
	}
	if status != nil {
		filters = append(filters, datamodels.Filter{
			FieldName: "publication_status",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{*status}},
			},
		})
	} else {
		filters = append(filters, datamodels.Filter{
			FieldName: "is_latest",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{true}},
			},
		})
	}

	recs, _, err := s.dao.List(ctx, SchemaNamespace, filters, datamodels.Pagination{}, nil)
	if err != nil {
		return nil, err
	}

	var results []*descriptors.Schema
	for _, r := range recs {
		schema, err := descriptors.RecordToSchema(r.Rec.(map[string]interface{}))
		if err != nil {
			return nil, err
		}
		results = append(results, schema)
	}

	return results, nil
}

func (s *SchemaService) ById(ctx context.Context, id int64) (*descriptors.Schema, error) {
	// Key in SchemaNamespace is schemaId (string). For int64 ID, we might need a different lookup or key convention.
	// Actually, the previous implementation used auto-increment ID. 
	// In the new model, we should probably use schemaId as the key.
	
	filters := []datamodels.Filter{
		{
			FieldName: "id",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{id}},
			},
		},
		{
			FieldName: "deleted",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{false}},
			},
		},
	}
	recs, _, err := s.dao.List(ctx, SchemaNamespace, filters, datamodels.Pagination{}, nil)
	if err != nil || len(recs) == 0 {
		return nil, err
	}
	return descriptors.RecordToSchema(recs[0].Rec.(map[string]interface{}))
}

func (s *SchemaService) BySchemaId(ctx context.Context, schemaId string) (*descriptors.Schema, error) {
	rec, err := s.dao.Get(ctx, SchemaNamespace, schemaId)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, nil
	}
	return descriptors.RecordToSchema(rec.Rec.(map[string]interface{}))
}

func (s *SchemaService) ByNameOrDefault(ctx context.Context, name string, schemaType descriptors.SchemaType, status *descriptors.PublicationStatus) (*descriptors.Schema, error) {
	filters := []datamodels.Filter{
		{
			FieldName: "name",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{name}},
			},
		},
		{
			FieldName: "type",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{schemaType}},
			},
		},
		{
			FieldName: "deleted",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{false}},
			},
		},
	}

	if status != nil {
		filters = append(filters, datamodels.Filter{
			FieldName: "publication_status",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{*status}},
			},
		})
	} else {
		filters = append(filters, datamodels.Filter{
			FieldName: "is_latest",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{true}},
			},
		})
	}

	recs, _, err := s.dao.List(ctx, SchemaNamespace, filters, datamodels.Pagination{}, []datamodels.Sort{{Field: "id", Order: datamodels.SortOrderDesc}})
	if err != nil || len(recs) == 0 {
		return nil, err
	}
	return descriptors.RecordToSchema(recs[0].Rec.(map[string]interface{}))
}

func (s *SchemaService) ByStartsOrDefault(ctx context.Context, name string, schemaType descriptors.SchemaType, status *descriptors.PublicationStatus) (*descriptors.Schema, error) {
	// Simplified: Like is not yet implemented in DAO.List JSON filtering.
	// For now, I'll use Exact match or implement Like in DAO.
	return s.ByNameOrDefault(ctx, name, schemaType, status)
}

func (s *SchemaService) LoadEntity(ctx context.Context, name string) (*descriptors.Entity, error) {
	schema, err := s.ByNameOrDefault(ctx, name, descriptors.EntitySchema, nil)
	if err != nil {
		return nil, err
	}
	if schema == nil || schema.Settings == nil || schema.Settings.Entity == nil {
		return nil, fmt.Errorf("entity %s not found", name)
	}
	return schema.Settings.Entity, nil
}

func (s *SchemaService) LoadLoadedEntity(ctx context.Context, name string) (*descriptors.LoadedEntity, error) {
	return s.loadLoadedEntityInternal(ctx, name, make(map[string]*descriptors.LoadedEntity))
}

func (s *SchemaService) loadLoadedEntityInternal(ctx context.Context, name string, processed map[string]*descriptors.LoadedEntity) (*descriptors.LoadedEntity, error) {
	if le, ok := processed[name]; ok {
		return le, nil
	}

	entity, err := s.LoadEntity(ctx, name)
	if err != nil {
		return nil, err
	}

	le := entity.ToLoadedEntity()
	processed[name] = le

	if err := s.loadAttributes(ctx, le, processed); err != nil {
		return nil, err
	}

	return le, nil
}

func (s *SchemaService) loadAttributes(ctx context.Context, le *descriptors.LoadedEntity, processed map[string]*descriptors.LoadedEntity) error {
	for i := range le.LoadedAttributes {
		attr := &le.LoadedAttributes[i]
		var err error
		switch attr.DataType {
		case descriptors.DataTypeLookup:
			err = s.loadLookup(ctx, attr, processed)
		case descriptors.DataTypeJunction:
			err = s.loadJunction(ctx, le, attr, processed)
		case descriptors.DataTypeCollection:
			err = s.loadCollection(ctx, le, attr, processed)
		}
		if err != nil {
			return err
		}
	}
	// Re-assign special attributes to point to the instances in the LoadedAttributes slice
	for i := range le.LoadedAttributes {
		attr := le.LoadedAttributes[i]
		if attr.Field == le.PrimaryKey {
			le.PrimaryKeyAttribute = attr
		}
		if attr.Field == le.LabelAttributeName {
			le.LabelAttribute = attr
		}
		if attr.Field == "publicationStatus" {
			le.PublicationStatusAttribute = attr
		}
		if attr.Field == "updatedAt" {
			le.UpdatedAtAttribute = attr
		}
	}
	return nil
}

func (s *SchemaService) loadLookup(ctx context.Context, attr *descriptors.LoadedAttribute, processed map[string]*descriptors.LoadedEntity) error {
	targetName := attr.Options
	target, err := s.loadLoadedEntityInternal(ctx, targetName, processed)
	if err != nil {
		return err
	}
	attr.Lookup = &descriptors.Lookup{TargetEntity: target}
	return nil
}

func (s *SchemaService) loadJunction(ctx context.Context, sourceLe *descriptors.LoadedEntity, attr *descriptors.LoadedAttribute, processed map[string]*descriptors.LoadedEntity) error {
	parts := strings.Split(attr.Options, "|")
	if len(parts) != 4 {
		return fmt.Errorf("invalid junction options: %s", attr.Options)
	}
	junctionTableName, targetEntityName, sourceFieldName, targetFieldName := parts[0], parts[1], parts[2], parts[3]

	targetLe, err := s.loadLoadedEntityInternal(ctx, targetEntityName, processed)
	if err != nil {
		return err
	}

	junctionLe := &descriptors.LoadedEntity{
		Entity: descriptors.Entity{
			TableName: junctionTableName,
			Name:      junctionTableName,
		},
	}

	attr.Junction = &descriptors.Junction{
		SourceEntity:    sourceLe,
		TargetEntity:    targetLe,
		JunctionEntity:  junctionLe,
		SourceAttribute: &descriptors.LoadedAttribute{Attribute: descriptors.Attribute{Field: sourceFieldName}},
		TargetAttribute: &descriptors.LoadedAttribute{Attribute: descriptors.Attribute{Field: targetFieldName}},
	}
	return nil
}

func (s *SchemaService) loadCollection(ctx context.Context, sourceLe *descriptors.LoadedEntity, attr *descriptors.LoadedAttribute, processed map[string]*descriptors.LoadedEntity) error {
	parts := strings.Split(attr.Options, "|")
	if len(parts) != 2 {
		return fmt.Errorf("invalid collection options: %s", attr.Options)
	}
	targetEntityName, linkFieldName := parts[0], parts[1]

	targetLe, err := s.loadLoadedEntityInternal(ctx, targetEntityName, processed)
	if err != nil {
		return err
	}

	var linkAttr *descriptors.LoadedAttribute
	for i := range targetLe.LoadedAttributes {
		if targetLe.LoadedAttributes[i].Field == linkFieldName {
			linkAttr = &targetLe.LoadedAttributes[i]
			break
		}
	}

	attr.Collection = &descriptors.Collection{
		SourceEntity:  sourceLe,
		TargetEntity:  targetLe,
		LinkAttribute: linkAttr,
	}
	return nil
}

func (s *SchemaService) Save(ctx context.Context, schema *descriptors.Schema, asPublished bool) (*descriptors.Schema, error) {
	if schema.SchemaId == "" {
		id, _ := gonanoid.New(12)
		schema.SchemaId = id
	}

	if asPublished || schema.Id == 0 {
		schema.PublicationStatus = descriptors.Published
	} else {
		schema.PublicationStatus = descriptors.Draft
	}
	schema.IsLatest = true
	schema.CreatedAt = time.Now()

	// Handle versioning/latest flag
	if schema.IsLatest {
		filters := []datamodels.Filter{
			{
				FieldName: "schema_id",
				Constraints: []datamodels.Constraint{
					{Match: "equals", Values: []interface{}{schema.SchemaId}},
				},
			},
			{
				FieldName: "is_latest",
				Constraints: matchEqualityConstraint("equals", true),
			},
		}
		recs, _, _ := s.dao.List(ctx, SchemaNamespace, filters, datamodels.Pagination{}, nil)
		for _, r := range recs {
			data := r.Rec.(map[string]interface{})
			data["is_latest"] = false
			r.Rec = data
			s.dao.Save(ctx, r)
		}
	}

	if asPublished {
		filters := []datamodels.Filter{
			{
				FieldName: "schema_id",
				Constraints: []datamodels.Constraint{
					{Match: "equals", Values: []interface{}{schema.SchemaId}},
				},
			},
			{
				FieldName: "publication_status",
				Constraints: []datamodels.Constraint{
					{Match: "equals", Values: []interface{}{descriptors.Published}},
				},
			},
		}
		recs, _, _ := s.dao.List(ctx, SchemaNamespace, filters, datamodels.Pagination{}, nil)
		for _, r := range recs {
			data := r.Rec.(map[string]interface{})
			data["publication_status"] = descriptors.Draft
			r.Rec = data
			s.dao.Save(ctx, r)
		}
	}

	// For simple pivot, we use schemaId as key, but if we have multiple versions, 
	// we might need a composite key or just use schemaId as key for ONLY the latest.
	// However, the user said Key is unique ID. If we want history, we need a unique key per version.
	
	if schema.Id == 0 {
		// New record, need an ID. In JSON store, we might use ULID or NanoID.
		schema.Id = time.Now().UnixNano() // Temporary ID
	}

	rec := datamodels.RecJSON{
		Namespace: SchemaNamespace,
		Key:       schema.SchemaId, // Using SchemaId as key means we only keep ONE version as primary.
		Rec:       schema,
		Tmstamp:   time.Now(),
	}

	err := s.dao.Save(ctx, rec)
	return schema, err
}

func (s *SchemaService) Delete(ctx context.Context, schemaId string) error {
	return s.dao.Delete(ctx, SchemaNamespace, schemaId)
}

func matchEqualityConstraint(match string, val interface{}) []datamodels.Constraint {
	return []datamodels.Constraint{
		{Match: match, Values: []interface{}{val}},
	}
}

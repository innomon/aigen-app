package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/innomon/aigen-cms/core/descriptors"
	"github.com/innomon/aigen-cms/infrastructure/relationdbdao"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

const SchemaTableName = "__schemas"

type SchemaService struct {
	dao relationdbdao.IPrimaryDao
}

func NewSchemaService(dao relationdbdao.IPrimaryDao) *SchemaService {
	return &SchemaService{dao: dao}
}

func (s *SchemaService) All(ctx context.Context, schemaType *descriptors.SchemaType, names []string, status *descriptors.PublicationStatus) ([]*descriptors.Schema, error) {
	sb := s.dao.GetBuilder().Select("*").From(SchemaTableName).Where(squirrel.Eq{"deleted": false})

	if schemaType != nil {
		sb = sb.Where(squirrel.Eq{"type": *schemaType})
	}
	if len(names) > 0 {
		sb = sb.Where(squirrel.Eq{"name": names})
	}
	if status != nil {
		sb = sb.Where(squirrel.Eq{"publication_status": *status})
	} else {
		sb = sb.Where(squirrel.Eq{"is_latest": true})
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.dao.GetDb().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*descriptors.Schema
	for rows.Next() {
		schema, err := s.scanSchema(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, schema)
	}

	return results, nil
}

func (s *SchemaService) ById(ctx context.Context, id int64) (*descriptors.Schema, error) {
	query, args, err := s.dao.GetBuilder().Select("*").From(SchemaTableName).Where(squirrel.Eq{"id": id, "deleted": false}).ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.dao.GetDb().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	return s.scanSchema(rows)
}

func (s *SchemaService) BySchemaId(ctx context.Context, schemaId string) (*descriptors.Schema, error) {
	query, args, err := s.dao.GetBuilder().Select("*").From(SchemaTableName).
		Where(squirrel.Eq{"schema_id": schemaId, "deleted": false}).
		OrderBy("id DESC").Limit(1).ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.dao.GetDb().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	return s.scanSchema(rows)
}

func (s *SchemaService) ByNameOrDefault(ctx context.Context, name string, schemaType descriptors.SchemaType, status *descriptors.PublicationStatus) (*descriptors.Schema, error) {
	sb := s.dao.GetBuilder().Select("*").From(SchemaTableName).
		Where(squirrel.Eq{"name": name, "type": schemaType, "deleted": false})

	if status != nil {
		sb = sb.Where(squirrel.Eq{"publication_status": *status})
	} else {
		sb = sb.Where(squirrel.Eq{"is_latest": true})
	}

	query, args, err := sb.OrderBy("id DESC").Limit(1).ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.dao.GetDb().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	return s.scanSchema(rows)
}

func (s *SchemaService) ByStartsOrDefault(ctx context.Context, name string, schemaType descriptors.SchemaType, status *descriptors.PublicationStatus) (*descriptors.Schema, error) {
	sb := s.dao.GetBuilder().Select("*").From(SchemaTableName).
		Where(squirrel.Like{"name": name + "%"}).
		Where(squirrel.Eq{"type": schemaType, "deleted": false})

	if status != nil {
		sb = sb.Where(squirrel.Eq{"publication_status": *status})
	} else {
		sb = sb.Where(squirrel.Eq{"is_latest": true})
	}

	query, args, err := sb.OrderBy("id DESC").Limit(1).ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.dao.GetDb().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	return s.scanSchema(rows)
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

	updateQuery, updateArgs, err := s.dao.GetBuilder().Update(SchemaTableName).
		Set("is_latest", false).
		Where(squirrel.Eq{"schema_id": schema.SchemaId, "is_latest": true}).ToSql()
	if err != nil {
		return nil, err
	}

	tx, err := s.dao.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, updateQuery, updateArgs...); err != nil {
		return nil, err
	}

	if asPublished {
		pubQuery, pubArgs, err := s.dao.GetBuilder().Update(SchemaTableName).
			Set("publication_status", descriptors.Draft).
			Where(squirrel.Eq{"schema_id": schema.SchemaId, "publication_status": descriptors.Published}).ToSql()
		if err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, pubQuery, pubArgs...); err != nil {
			return nil, err
		}
	}

	settingsJSON, _ := json.Marshal(schema.Settings)
	insertQuery, insertArgs, err := s.dao.GetBuilder().Insert(SchemaTableName).
		Columns("schema_id", "name", "type", "settings", "description", "is_latest", "publication_status", "created_at", "created_by", "deleted").
		Values(schema.SchemaId, schema.Name, schema.Type, string(settingsJSON), schema.Description, schema.IsLatest, schema.PublicationStatus, schema.CreatedAt, schema.CreatedBy, false).
		ToSql()
	if err != nil {
		return nil, err
	}

	var newId int64
	if strings.Contains(insertQuery, "$1") {
		err = tx.QueryRowContext(ctx, insertQuery+" RETURNING id", insertArgs...).Scan(&newId)
	} else {
		res, err := tx.ExecContext(ctx, insertQuery, insertArgs...)
		if err != nil {
			return nil, err
		}
		newId, err = res.LastInsertId()
		if err != nil {
			return nil, err
		}
	}

	if err != nil {
		return nil, err
	}

	schema.Id = newId
	return schema, tx.Commit()
}

func (s *SchemaService) Delete(ctx context.Context, schemaId string) error {
	query, args, err := s.dao.GetBuilder().Update(SchemaTableName).
		Set("deleted", true).
		Where(squirrel.Eq{"schema_id": schemaId}).ToSql()
	if err != nil {
		return err
	}
	_, err = s.dao.GetDb().ExecContext(ctx, query, args...)
	return err
}

func (s *SchemaService) scanSchema(scanner interface {
	Scan(dest ...interface{}) error
	Columns() ([]string, error)
}) (*descriptors.Schema, error) {
	cols, _ := scanner.Columns()
	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range cols {
		valuePtrs[i] = &values[i]
	}

	if err := scanner.Scan(valuePtrs...); err != nil {
		return nil, err
	}

	record := make(map[string]interface{})
	for i, col := range cols {
		val := values[i]
		if b, ok := val.([]byte); ok {
			record[col] = string(b)
		} else {
			record[col] = val
		}
	}

	return descriptors.RecordToSchema(record)
}

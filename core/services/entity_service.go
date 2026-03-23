package services

import (
	"context"
	"fmt"
	"time"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/innomon/aigen-app/utils/datamodels"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"golang.org/x/crypto/bcrypt"
)

type EntityService struct {
	schemaService     ISchemaService
	dao               relationdbdao.IPrimaryDao
	permissionService IPermissionService
}

func NewEntityService(schemaService ISchemaService, dao relationdbdao.IPrimaryDao, permissionService IPermissionService) *EntityService {
	return &EntityService{
		schemaService:     schemaService,
		dao:               dao,
		permissionService: permissionService,
	}
}

func (s *EntityService) getNamespace(entityName string) string {
	return fmt.Sprintf("aigen.app.entities.%s", entityName)
}

func (s *EntityService) List(ctx context.Context, name string, pagination datamodels.Pagination, filters []datamodels.Filter, sorts []datamodels.Sort) ([]datamodels.Record, int64, error) {
	userId, _ := ctx.Value("userId").(int64)
	rowFilters, _ := s.permissionService.GetRowFilters(ctx, userId, name)
	filters = append(filters, rowFilters...)

	recs, total, err := s.dao.List(ctx, s.getNamespace(name), filters, pagination, sorts)
	if err != nil {
		return nil, 0, err
	}

	roles, _ := ctx.Value("roles").([]string)
	fieldPerms, _ := s.permissionService.GetFieldPermissions(ctx, name, roles)

	var results []datamodels.Record
	for _, r := range recs {
		recData := r.Rec.(map[string]interface{})
		filteredRec := make(datamodels.Record)
		for k, v := range recData {
			if p, ok := fieldPerms[k]; ok && !p["read"] {
				continue
			}
			filteredRec[k] = v
		}
		results = append(results, filteredRec)
	}

	return results, total, nil
}

func (s *EntityService) Single(ctx context.Context, name string, id interface{}) (datamodels.Record, error) {
	key := fmt.Sprintf("%v", id)
	rec, err := s.dao.Get(ctx, s.getNamespace(name), key)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("record not found")
	}

	roles, _ := ctx.Value("roles").([]string)
	fieldPerms, _ := s.permissionService.GetFieldPermissions(ctx, name, roles)

	recData := rec.Rec.(map[string]interface{})
	filteredRec := make(datamodels.Record)
	for k, v := range recData {
		if p, ok := fieldPerms[k]; ok && !p["read"] {
			continue
		}
		filteredRec[k] = v
	}

	return filteredRec, nil
}

func (s *EntityService) Insert(ctx context.Context, name string, data datamodels.Record) (datamodels.Record, error) {
	roles, _ := ctx.Value("roles").([]string)
	fieldPerms, _ := s.permissionService.GetFieldPermissions(ctx, name, roles)

	entity, err := s.schemaService.LoadEntity(ctx, name)
	if err != nil {
		return nil, err
	}

	id := data[entity.PrimaryKey]
	if id == nil || id == "" {
		newId, _ := gonanoid.New(12)
		id = newId
		data[entity.PrimaryKey] = id
	}

	cleanData := make(map[string]interface{})
	for k, v := range data {
		if p, ok := fieldPerms[k]; ok && !p["write"] {
			continue
		}
		
		val := v
		if name == "User" && k == "password_hash" {
			if str, ok := v.(string); ok && str != "" {
				hashed, _ := bcrypt.GenerateFromPassword([]byte(str), bcrypt.DefaultCost)
				val = string(hashed)
			}
		}
		cleanData[k] = val
	}

	cleanData["created_at"] = time.Now()
	cleanData["updated_at"] = time.Now()

	metadata := map[string]interface{}{
		"roles": roles,
		"owner": ctx.Value("userId"),
	}

	rec := datamodels.RecJSON{
		Namespace: s.getNamespace(name),
		Key:       fmt.Sprintf("%v", id),
		Rec:       cleanData,
		MetaData:  metadata,
		Tmstamp:   time.Now(),
	}

	if err := s.dao.Save(ctx, rec); err != nil {
		return nil, err
	}

	return s.Single(ctx, name, id)
}

func (s *EntityService) Update(ctx context.Context, name string, data datamodels.Record) (datamodels.Record, error) {
	entity, err := s.schemaService.LoadEntity(ctx, name)
	if err != nil {
		return nil, err
	}

	id := data[entity.PrimaryKey]
	existing, err := s.Single(ctx, name, id)
	if err != nil {
		return nil, err
	}

	roles, _ := ctx.Value("roles").([]string)
	fieldPerms, _ := s.permissionService.GetFieldPermissions(ctx, name, roles)

	for k, v := range data {
		if k == entity.PrimaryKey {
			continue
		}
		if p, ok := fieldPerms[k]; ok && !p["write"] {
			continue
		}
		
		val := v
		if name == "User" && k == "password_hash" {
			if str, ok := v.(string); ok && str != "" {
				hashed, _ := bcrypt.GenerateFromPassword([]byte(str), bcrypt.DefaultCost)
				val = string(hashed)
			} else {
				continue
			}
		}
		existing[k] = val
	}

	existing["updated_at"] = time.Now()

	metadata := map[string]interface{}{
		"roles": roles,
		"owner": ctx.Value("userId"),
	}

	rec := datamodels.RecJSON{
		Namespace: s.getNamespace(name),
		Key:       fmt.Sprintf("%v", id),
		Rec:       existing,
		MetaData:  metadata,
		Tmstamp:   time.Now(),
	}

	if err := s.dao.Save(ctx, rec); err != nil {
		return nil, err
	}

	return s.Single(ctx, name, id)
}

func (s *EntityService) Delete(ctx context.Context, name string, id interface{}) error {
	return s.dao.Delete(ctx, s.getNamespace(name), fmt.Sprintf("%v", id))
}

func (s *EntityService) CollectionList(ctx context.Context, name, id, attrName string, pagination datamodels.Pagination, filters []datamodels.Filter, sorts []datamodels.Sort) ([]datamodels.Record, int64, error) {
	le, err := s.schemaService.LoadLoadedEntity(ctx, name)
	if err != nil {
		return nil, 0, err
	}

	var collectionAttr *descriptors.LoadedAttribute
	for i := range le.LoadedAttributes {
		if le.LoadedAttributes[i].Field == attrName {
			collectionAttr = &le.LoadedAttributes[i]
			break
		}
	}
	if collectionAttr == nil || collectionAttr.Collection == nil {
		return nil, 0, fmt.Errorf("collection attribute %s not found", attrName)
	}

	collection := collectionAttr.Collection
	targetEntity := collection.TargetEntity

	filters = append(filters, datamodels.Filter{
		FieldName: collection.LinkAttribute.Field,
		Constraints: []datamodels.Constraint{
			{Match: "equals", Values: []interface{}{id}},
		},
	})

	return s.List(ctx, targetEntity.Name, pagination, filters, sorts)
}

func (s *EntityService) CollectionInsert(ctx context.Context, name, id, attrName string, data datamodels.Record) (datamodels.Record, error) {
	le, err := s.schemaService.LoadLoadedEntity(ctx, name)
	if err != nil {
		return nil, err
	}

	var collectionAttr *descriptors.LoadedAttribute
	for i := range le.LoadedAttributes {
		if le.LoadedAttributes[i].Field == attrName {
			collectionAttr = &le.LoadedAttributes[i]
			break
		}
	}
	if collectionAttr == nil || collectionAttr.Collection == nil {
		return nil, fmt.Errorf("collection attribute %s not found", attrName)
	}

	collection := collectionAttr.Collection
	data[collection.LinkAttribute.Field] = id
	return s.Insert(ctx, collection.TargetEntity.Name, data)
}

func (s *EntityService) JunctionList(ctx context.Context, name, id, attrName string, exclude bool, pagination datamodels.Pagination, filters []datamodels.Filter, sorts []datamodels.Sort) ([]datamodels.Record, int64, error) {
	// Junctions need special handling in a single-table JSON model.
	// For now, I'll return an error or a simplified implementation.
	return nil, 0, fmt.Errorf("junctions not yet supported in JSON store model")
}

func (s *EntityService) JunctionSave(ctx context.Context, name, id, attrName string, targetIds []interface{}) error {
	return fmt.Errorf("junctions not yet supported in JSON store model")
}

func (s *EntityService) JunctionDelete(ctx context.Context, name, id, attrName string, targetIds []interface{}) error {
	return fmt.Errorf("junctions not yet supported in JSON store model")
}

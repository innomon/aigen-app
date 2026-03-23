package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"strconv"

	"github.com/Masterminds/squirrel"
	"github.com/innomon/aigen-cms/core/descriptors"
	"github.com/innomon/aigen-cms/infrastructure/relationdbdao"
	"github.com/innomon/aigen-cms/utils/datamodels"
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

func (s *EntityService) List(ctx context.Context, name string, pagination datamodels.Pagination, filters []datamodels.Filter, sorts []datamodels.Sort) ([]datamodels.Record, int64, error) {
	entity, err := s.schemaService.LoadEntity(ctx, name)
	if err != nil {
		return nil, 0, err
	}

	userId, _ := ctx.Value("userId").(int64)
	rowFilters, _ := s.permissionService.GetRowFilters(ctx, userId, name)
	filters = append(filters, rowFilters...)

	sb := entity.SelectQuery(s.dao.GetBuilder())
	sb = s.applyFilters(sb, filters)
	sb = s.applySorts(sb, sorts)
	sb = s.applyPagination(sb, pagination, uint64(entity.DefaultPageSize))

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.dao.GetDb().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	roles, _ := ctx.Value("roles").([]string)
	fieldPerms, _ := s.permissionService.GetFieldPermissions(ctx, name, roles)

	results, err := s.scanRows(rows, fieldPerms)
	if err != nil {
		return nil, 0, err
	}

	return results, int64(len(results)), nil
}

func (s *EntityService) Single(ctx context.Context, name string, id interface{}) (datamodels.Record, error) {
	entity, err := s.schemaService.LoadEntity(ctx, name)
	if err != nil {
		return nil, err
	}

	userId, _ := ctx.Value("userId").(int64)
	rowFilters, _ := s.permissionService.GetRowFilters(ctx, userId, name)

	sb := entity.SelectQuery(s.dao.GetBuilder()).Where(squirrel.Eq{entity.PrimaryKey: id})
	sb = s.applyFilters(sb, rowFilters)

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.dao.GetDb().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roles, _ := ctx.Value("roles").([]string)
	fieldPerms, _ := s.permissionService.GetFieldPermissions(ctx, name, roles)

	results, err := s.scanRows(rows, fieldPerms)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("record not found or access denied")
	}

	return results[0], nil
}

func (s *EntityService) Insert(ctx context.Context, name string, data datamodels.Record) (datamodels.Record, error) {
	entity, err := s.schemaService.LoadEntity(ctx, name)
	if err != nil {
		return nil, err
	}

	roles, _ := ctx.Value("roles").([]string)
	fieldPerms, _ := s.permissionService.GetFieldPermissions(ctx, name, roles)

	var columns []string
	var values []interface{}
	for k, v := range data {
		if p, ok := fieldPerms[k]; ok && !p["write"] {
			continue // Skip unauthorized fields
		}

		val := v
		// Handle password hashing if it's the User entity and password_hash field
		if name == "User" && k == "password_hash" {
			if str, ok := v.(string); ok && str != "" {
				hashed, err := bcrypt.GenerateFromPassword([]byte(str), bcrypt.DefaultCost)
				if err != nil {
					return nil, err
				}
				val = string(hashed)
			}
		}

		columns = append(columns, k)
		values = append(values, val)
	}

	query, args, err := s.dao.GetBuilder().Insert(entity.TableName).Columns(columns...).Values(values...).ToSql()
	if err != nil {
		return nil, err
	}

	if strings.Contains(query, "$1") {
		var newId interface{}
		err = s.dao.GetDb().QueryRowContext(ctx, query+" RETURNING "+entity.PrimaryKey, args...).Scan(&newId)
		if err != nil {
			return nil, err
		}
		return s.Single(ctx, name, newId)
	} else {
		res, err := s.dao.GetDb().ExecContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		newId, _ := res.LastInsertId()
		return s.Single(ctx, name, newId)
	}
}

func (s *EntityService) Update(ctx context.Context, name string, data datamodels.Record) (datamodels.Record, error) {
	entity, err := s.schemaService.LoadEntity(ctx, name)
	if err != nil {
		return nil, err
	}

	roles, _ := ctx.Value("roles").([]string)
	fieldPerms, _ := s.permissionService.GetFieldPermissions(ctx, name, roles)

	id := data[entity.PrimaryKey]
	sb := s.dao.GetBuilder().Update(entity.TableName)
	for k, v := range data {
		if k == entity.PrimaryKey {
			continue
		}
		if p, ok := fieldPerms[k]; ok && !p["write"] {
			continue // Skip unauthorized fields
		}

		val := v
		// Handle password hashing if it's the User entity and password_hash field
		if name == "User" && k == "password_hash" {
			if str, ok := v.(string); ok && str != "" {
				hashed, err := bcrypt.GenerateFromPassword([]byte(str), bcrypt.DefaultCost)
				if err != nil {
					return nil, err
				}
				val = string(hashed)
			} else {
				continue // Don't update password if it's empty
			}
		}

		sb = sb.Set(k, val)
	}

	query, args, err := sb.Where(squirrel.Eq{entity.PrimaryKey: id}).ToSql()
	if err != nil {
		return nil, err
	}

	_, err = s.dao.GetDb().ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	return s.Single(ctx, name, id)
}

func (s *EntityService) Delete(ctx context.Context, name string, id interface{}) error {
	entity, err := s.schemaService.LoadEntity(ctx, name)
	if err != nil {
		return err
	}

	// For delete, we might also want to apply row filters to ensure user can only delete what they can see
	userId, _ := ctx.Value("userId").(int64)
	rowFilters, _ := s.permissionService.GetRowFilters(ctx, userId, name)

	sb := s.dao.GetBuilder().Delete(entity.TableName).Where(squirrel.Eq{entity.PrimaryKey: id})
	// Note: squirrel DeleteBuilder doesn't directly support the same complex where as SelectBuilder in some versions, 
	// but squirrel.Eq should work fine.
	
	// Complex row filters might need careful application to Delete
	if len(rowFilters) > 0 {
		// This is a simplified application
		for _, f := range rowFilters {
			for _, c := range f.Constraints {
				if c.Match == "equals" && len(c.Values) > 0 {
					sb = sb.Where(squirrel.Eq{f.FieldName: c.Values})
				}
			}
		}
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return err
	}

	_, err = s.dao.GetDb().ExecContext(ctx, query, args...)
	return err
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
		return nil, 0, fmt.Errorf("collection attribute %s not found in entity %s", attrName, name)
	}

	collection := collectionAttr.Collection
	targetEntity := collection.TargetEntity

	userId, _ := ctx.Value("userId").(int64)
	rowFilters, _ := s.permissionService.GetRowFilters(ctx, userId, targetEntity.Name)
	filters = append(filters, rowFilters...)

	sb := s.dao.GetBuilder().Select("*").From(targetEntity.TableName).
		Where(squirrel.Eq{collection.LinkAttribute.Field: id})

	sb = s.applyFilters(sb, filters)
	sb = s.applySorts(sb, sorts)
	sb = s.applyPagination(sb, pagination, uint64(targetEntity.DefaultPageSize))

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.dao.GetDb().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	roles, _ := ctx.Value("roles").([]string)
	fieldPerms, _ := s.permissionService.GetFieldPermissions(ctx, targetEntity.Name, roles)

	results, err := s.scanRows(rows, fieldPerms)
	return results, int64(len(results)), err
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
	le, err := s.schemaService.LoadLoadedEntity(ctx, name)
	if err != nil {
		return nil, 0, err
	}

	var junctionAttr *descriptors.LoadedAttribute
	for i := range le.LoadedAttributes {
		if le.LoadedAttributes[i].Field == attrName {
			junctionAttr = &le.LoadedAttributes[i]
			break
		}
	}
	if junctionAttr == nil || junctionAttr.Junction == nil {
		return nil, 0, fmt.Errorf("junction attribute %s not found", attrName)
	}

	junction := junctionAttr.Junction
	targetEntity := junction.TargetEntity

	userId, _ := ctx.Value("userId").(int64)
	rowFilters, _ := s.permissionService.GetRowFilters(ctx, userId, targetEntity.Name)
	filters = append(filters, rowFilters...)

	subQuery, subArgs, _ := s.dao.GetBuilder().Select(junction.TargetAttribute.Field).
		From(junction.JunctionEntity.TableName).
		Where(squirrel.Eq{junction.SourceAttribute.Field: id}).ToSql()

	sb := s.dao.GetBuilder().Select("*").From(targetEntity.TableName)
	if exclude {
		sb = sb.Where(fmt.Sprintf("%s NOT IN (%s)", targetEntity.PrimaryKey, subQuery), subArgs...)
	} else {
		sb = sb.Where(fmt.Sprintf("%s IN (%s)", targetEntity.PrimaryKey, subQuery), subArgs...)
	}

	sb = s.applyFilters(sb, filters)
	sb = s.applySorts(sb, sorts)
	sb = s.applyPagination(sb, pagination, uint64(targetEntity.DefaultPageSize))

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.dao.GetDb().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	roles, _ := ctx.Value("roles").([]string)
	fieldPerms, _ := s.permissionService.GetFieldPermissions(ctx, targetEntity.Name, roles)

	results, err := s.scanRows(rows, fieldPerms)
	return results, int64(len(results)), err
}

func (s *EntityService) JunctionSave(ctx context.Context, name, id, attrName string, targetIds []interface{}) error {
	le, err := s.schemaService.LoadLoadedEntity(ctx, name)
	if err != nil {
		return err
	}

	var junctionAttr *descriptors.LoadedAttribute
	for i := range le.LoadedAttributes {
		if le.LoadedAttributes[i].Field == attrName {
			junctionAttr = &le.LoadedAttributes[i]
			break
		}
	}
	if junctionAttr == nil || junctionAttr.Junction == nil {
		return fmt.Errorf("junction attribute %s not found", attrName)
	}

	junction := junctionAttr.Junction
	tx, err := s.dao.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	delQuery, delArgs, _ := s.dao.GetBuilder().Delete(junction.JunctionEntity.TableName).
		Where(squirrel.Eq{junction.SourceAttribute.Field: id}).ToSql()
	if _, err := tx.ExecContext(ctx, delQuery, delArgs...); err != nil {
		return err
	}

	for _, tid := range targetIds {
		insQuery, insArgs, _ := s.dao.GetBuilder().Insert(junction.JunctionEntity.TableName).
			Columns(junction.SourceAttribute.Field, junction.TargetAttribute.Field).
			Values(id, tid).ToSql()
		if _, err := tx.ExecContext(ctx, insQuery, insArgs...); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *EntityService) JunctionDelete(ctx context.Context, name, id, attrName string, targetIds []interface{}) error {
	le, err := s.schemaService.LoadLoadedEntity(ctx, name)
	if err != nil {
		return err
	}

	var junctionAttr *descriptors.LoadedAttribute
	for i := range le.LoadedAttributes {
		if le.LoadedAttributes[i].Field == attrName {
			junctionAttr = &le.LoadedAttributes[i]
			break
		}
	}
	if junctionAttr == nil || junctionAttr.Junction == nil {
		return fmt.Errorf("junction attribute %s not found", attrName)
	}

	junction := junctionAttr.Junction
	query, args, err := s.dao.GetBuilder().Delete(junction.JunctionEntity.TableName).
		Where(squirrel.Eq{junction.SourceAttribute.Field: id, junction.TargetAttribute.Field: targetIds}).ToSql()
	if err != nil {
		return err
	}

	_, err = s.dao.GetDb().ExecContext(ctx, query, args...)
	return err
}

func (s *EntityService) applyFilters(sb squirrel.SelectBuilder, filters []datamodels.Filter) squirrel.SelectBuilder {
	for _, f := range filters {
		for _, c := range f.Constraints {
			if c.Match == "equals" && len(c.Values) > 0 {
				if len(c.Values) == 1 {
					sb = sb.Where(squirrel.Eq{f.FieldName: *c.Values[0]})
				} else {
					sb = sb.Where(squirrel.Eq{f.FieldName: c.Values})
				}
			}
		}
	}
	return sb
}

func (s *EntityService) applySorts(sb squirrel.SelectBuilder, sorts []datamodels.Sort) squirrel.SelectBuilder {
	for _, sort := range sorts {
		order := "ASC"
		if sort.Order == datamodels.SortOrderDesc {
			order = "DESC"
		}
		sb = sb.OrderBy(fmt.Sprintf("%s %s", sort.Field, order))
	}
	return sb
}

func (s *EntityService) applyPagination(sb squirrel.SelectBuilder, pagination datamodels.Pagination, defaultLimit uint64) squirrel.SelectBuilder {
	limit := defaultLimit
	if pagination.Limit != nil {
		if l, err := strconv.ParseUint(*pagination.Limit, 10, 64); err == nil {
			limit = l
		}
	}
	sb = sb.Limit(limit)

	if pagination.Offset != nil {
		if o, err := strconv.ParseUint(*pagination.Offset, 10, 64); err == nil {
			sb = sb.Offset(o)
		}
	}
	return sb
}

func (s *EntityService) scanRows(rows *sql.Rows, fieldPerms map[string]map[string]bool) ([]datamodels.Record, error) {
	var results []datamodels.Record
	columns, _ := rows.Columns()
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		record := make(datamodels.Record)
		for i, col := range columns {
			// Field-level read check
			if p, ok := fieldPerms[col]; ok && !p["read"] {
				continue // Skip unauthorized fields
			}

			val := values[i]
			if b, ok := val.([]byte); ok {
				record[col] = string(b)
			} else {
				record[col] = val
			}
		}
		results = append(results, record)
	}
	return results, nil
}

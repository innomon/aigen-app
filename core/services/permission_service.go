package services

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/innomon/aigen-cms/infrastructure/relationdbdao"
	"github.com/innomon/aigen-cms/utils/datamodels"
)

type PermissionService struct {
	dao           relationdbdao.IPrimaryDao
	schemaService ISchemaService
}

func NewPermissionService(dao relationdbdao.IPrimaryDao, schemaService ISchemaService) *PermissionService {
	return &PermissionService{
		dao:           dao,
		schemaService: schemaService,
	}
}

func (s *PermissionService) HasAccess(ctx context.Context, userId int64, roles []string, entityName, action string) (bool, error) {
	// SA always has access
	for _, r := range roles {
		if r == "sa" {
			return true, nil
		}
	}

	// Fetch doc perms for the roles and entity
	// Note: Action names should match the fields in __doc_perms: read, write, create, delete, etc.
	query, args, err := s.dao.GetBuilder().Select("count(*)").From("__doc_perms").
		Join("__roles ON __doc_perms.role = __roles.id").
		Where(squirrel.Eq{"__roles.name": roles, "parent": entityName, action: true, "permlevel": 0}).
		ToSql()

	if err != nil {
		return false, err
	}

	var count int
	err = s.dao.GetDb().QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *PermissionService) GetRowFilters(ctx context.Context, userId int64, entityName string) ([]datamodels.Filter, error) {
	// Fetch user permissions for the user
	query, args, err := s.dao.GetBuilder().Select("allow", "for_value").From("__user_perms").
		Where(squirrel.Eq{"user_id": userId}).ToSql()

	if err != nil {
		return nil, err
	}

	rows, err := s.dao.GetDb().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	userPerms := make(map[string][]string)
	for rows.Next() {
		var allow, forValue string
		if err := rows.Scan(&allow, &forValue); err != nil {
			return nil, err
		}
		userPerms[allow] = append(userPerms[allow], forValue)
	}

	if len(userPerms) == 0 {
		return nil, nil
	}

	// Load the entity to find fields that link to these allowed entities
	entity, err := s.schemaService.LoadEntity(ctx, entityName)
	if err != nil {
		return nil, err
	}

	var filters []datamodels.Filter
	for _, attr := range entity.Attributes {
		if attr.DataType == "Lookup" && userPerms[attr.Options] != nil {
			values := userPerms[attr.Options]
			ptrValues := make([]*string, len(values))
			for i := range values {
				ptrValues[i] = &values[i]
			}
			filters = append(filters, datamodels.Filter{
				FieldName: attr.Field,
				Constraints: []datamodels.Constraint{
					{
						Match:  "equals", // Use equals but we can handle slice of values in applyFilters
						Values: ptrValues,
					},
				},
			})
		}
	}

	return filters, nil
}

func (s *PermissionService) GetFieldPermissions(ctx context.Context, entityName string, roles []string) (map[string]map[string]bool, error) {
	// Load entity attributes first to know what fields we have
	entity, err := s.schemaService.LoadEntity(ctx, entityName)
	if err != nil {
		return nil, err
	}

	fieldPerms := make(map[string]map[string]bool)

	// SA always has full access to all fields
	for _, r := range roles {
		if r == "sa" {
			for _, attr := range entity.Attributes {
				fieldPerms[attr.Field] = map[string]bool{"read": true, "write": true}
			}
			return fieldPerms, nil
		}
	}

	// Fetch all doc perms for these roles and entity
	query, args, err := s.dao.GetBuilder().Select("permlevel", "read", "write").From("__doc_perms").
		Join("__roles ON __doc_perms.role = __roles.id").
		Where(squirrel.Eq{"__roles.name": roles, "parent": entityName}).
		ToSql()

	if err != nil {
		return nil, err
	}

	rows, err := s.dao.GetDb().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	permLevels := make(map[int]map[string]bool)
	for rows.Next() {
		var lvl int
		var r, w bool
		if err := rows.Scan(&lvl, &r, &w); err != nil {
			return nil, err
		}
		if _, ok := permLevels[lvl]; !ok {
			permLevels[lvl] = map[string]bool{"read": false, "write": false}
		}
		if r {
			permLevels[lvl]["read"] = true
		}
		if w {
			permLevels[lvl]["write"] = true
		}
	}

	// Match attributes with permlevels
	for _, attr := range entity.Attributes {
		lvl := attr.PermLevel
		if p, ok := permLevels[lvl]; ok {
			fieldPerms[attr.Field] = p
		} else {
			// Default to no access if no permlevel defined for these roles
			fieldPerms[attr.Field] = map[string]bool{"read": false, "write": false}
		}
	}

	return fieldPerms, nil
}

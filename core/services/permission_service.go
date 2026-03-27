package services

import (
	"context"
	"fmt"

	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/innomon/aigen-app/utils/datamodels"
)

const (
	DocPermNamespace  = "aigen.core.descriptors.DocPerm"
	UserPermNamespace = "aigen.core.descriptors.UserPerm"
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
	// Since we can't JOIN, we fetch doc_perms and filter by roles in Go
	filters := []datamodels.Filter{
		{
			FieldName: "parent",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{entityName}},
			},
		},
		{
			FieldName: "permlevel",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{0}},
			},
		},
		{
			FieldName: action,
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{true}},
			},
		},
	}

	recs, _, err := s.dao.List(ctx, DocPermNamespace, filters, datamodels.Pagination{}, nil)
	if err != nil {
		return false, err
	}

	for _, r := range recs {
		// rec.role is either role name or role id. 
		// Assuming it matches the role name for simplicity, or we should resolve role IDs.
		// If DocPerm.role is role name:
		data := r.Rec.(map[string]interface{})
		roleInPerm := fmt.Sprintf("%v", data["role"])
		for _, userRole := range roles {
			if roleInPerm == userRole {
				return true, nil
			}
		}
	}

	return false, nil
}

func (s *PermissionService) GetRowFilters(ctx context.Context, userId int64, entityName string) ([]datamodels.Filter, error) {
	// Fetch user permissions for the user
	filters := []datamodels.Filter{
		{
			FieldName: "userId",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{userId}},
			},
		},
	}

	recs, _, err := s.dao.List(ctx, UserPermNamespace, filters, datamodels.Pagination{}, nil)
	if err != nil {
		return nil, err
	}

	userPerms := make(map[string][]string)
	for _, r := range recs {
		data := r.Rec.(map[string]interface{})
		allow := fmt.Sprintf("%v", data["allow"])
		forValue := fmt.Sprintf("%v", data["for_value"])
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

	var rowFilters []datamodels.Filter
	for _, attr := range entity.Attributes {
		if attr.DataType == "Lookup" && userPerms[attr.Options] != nil {
			values := userPerms[attr.Options]
			ifaceValues := make([]interface{}, len(values))
			for i := range values {
				ifaceValues[i] = values[i]
			}
			rowFilters = append(rowFilters, datamodels.Filter{
				FieldName: attr.Field,
				Constraints: []datamodels.Constraint{
					{
						Match:  "equals",
						Values: ifaceValues,
					},
				},
			})
		}
	}

	return rowFilters, nil
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
	filters := []datamodels.Filter{
		{
			FieldName: "parent",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{entityName}},
			},
		},
	}

	recs, _, err := s.dao.List(ctx, DocPermNamespace, filters, datamodels.Pagination{}, nil)
	if err != nil {
		return nil, err
	}

	permLevels := make(map[int]map[string]bool)
	for _, r := range recs {
		data := r.Rec.(map[string]interface{})
		roleInPerm := fmt.Sprintf("%v", data["role"])
		
		match := false
		for _, userRole := range roles {
			if roleInPerm == userRole {
				match = true
				break
			}
		}

		if match {
			lvl := int(data["permlevel"].(float64))
			read := data["read"].(bool)
			write := data["write"].(bool)
			
			if _, ok := permLevels[lvl]; !ok {
				permLevels[lvl] = map[string]bool{"read": false, "write": false}
			}
			if read {
				permLevels[lvl]["read"] = true
			}
			if write {
				permLevels[lvl]["write"] = true
			}
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

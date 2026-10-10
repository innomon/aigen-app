package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/innomon/aigen-app/utils/datamodels"
	"github.com/innomon/aigen-app/utils/ids"
	"golang.org/x/crypto/bcrypt"
)

type EntityService struct {
	schemaService     ISchemaService
	evolutionService  IEvolutionService
	dao               relationdbdao.IPrimaryDao
	permissionService IPermissionService
}

func NewEntityService(schemaService ISchemaService, evolutionService IEvolutionService, dao relationdbdao.IPrimaryDao, permissionService IPermissionService) *EntityService {
	return &EntityService{
		schemaService:     schemaService,
		evolutionService:  evolutionService,
		dao:               dao,
		permissionService: permissionService,
	}
}

func (s *EntityService) getNamespace(entityName string) string {
	return fmt.Sprintf("aigen.bizdef.entities.%s", entityName)
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

		// JIT Evolution
		if _, _, err := s.evolutionService.EvolveRecord(name, recData, &r.MetaData); err != nil {
			// Log error but continue with original data
		}

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

	// JIT Evolution
	recData := rec.Rec.(map[string]interface{})
	if _, modified, err := s.evolutionService.EvolveRecord(name, recData, &rec.MetaData); err == nil && modified {
		rec.Rec = recData
		// We don't save back here to avoid write penalty on read, 
		// but we use the evolved data for the current request.
	}

	roles, _ := ctx.Value("roles").([]string)
	fieldPerms, _ := s.permissionService.GetFieldPermissions(ctx, name, roles)

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
		id = ids.NewRandomID()
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
				if !strings.HasPrefix(str, "$2a$") && !strings.HasPrefix(str, "$2b$") && !strings.HasPrefix(str, "$2y$") {
					hashed, err := bcrypt.GenerateFromPassword([]byte(str), bcrypt.DefaultCost)
					if err != nil {
						return nil, fmt.Errorf("failed to hash password: %w", err)
					}
					val = string(hashed)
				}
			}
		}
		cleanData[k] = val
	}

	cleanData["created_at"] = time.Now()
	cleanData["updated_at"] = time.Now()

	if name == "User" {
		roleStr, _ := cleanData["role"].(string)
		var rolesList []string
		if rRaw, ok := cleanData["roles"].([]interface{}); ok && len(rRaw) > 0 {
			for _, item := range rRaw {
				if sItem, ok := item.(string); ok && sItem != "" {
					rolesList = append(rolesList, sItem)
				}
			}
		} else if rList, ok := cleanData["roles"].([]string); ok && len(rList) > 0 {
			rolesList = rList
		}

		if len(rolesList) == 0 {
			switch roleStr {
			case descriptors.RoleSa, "superadmin":
				rolesList = []string{descriptors.RoleSa, descriptors.RoleAdmin, descriptors.RoleUser}
			case descriptors.RoleAdmin:
				rolesList = []string{descriptors.RoleAdmin, descriptors.RoleUser}
			case descriptors.RoleUser:
				rolesList = []string{descriptors.RoleUser}
			case descriptors.RoleGuest:
				rolesList = []string{descriptors.RoleGuest}
			case "":
				rolesList = []string{descriptors.RoleAdmin, descriptors.RoleUser}
				cleanData["role"] = descriptors.RoleAdmin
			default:
				rolesList = []string{roleStr, descriptors.RoleUser}
			}
		}
		cleanData["roles"] = rolesList
	}

	metadata := datamodels.MetaData{
		Roles: roles,
		Owner: ctx.Value("userId"),
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

	if name == "User" {
		if email, ok := cleanData["email"].(string); ok && email != "" {
			userIdInt := ids.StringToInt64ID(fmt.Sprintf("%v", id))
			phoneStr, _ := cleanData["phone"].(string)
			pwHash, _ := cleanData["password_hash"].(string)
			avatarStr, _ := cleanData["avatar_path"].(string)
			rolesList, _ := cleanData["roles"].([]string)

			userDesc := descriptors.User{
				Id:           userIdInt,
				Email:        email,
				Phone:        phoneStr,
				AvatarPath:   avatarStr,
				PasswordHash: pwHash,
				Roles:        rolesList,
				CreatedAt:    cleanData["created_at"].(time.Time),
				UpdatedAt:    cleanData["updated_at"].(time.Time),
			}
			_ = s.dao.Save(ctx, datamodels.RecJSON{
				Namespace: "aigen.core.descriptors.User",
				Key:       email,
				Rec:       userDesc,
				Tmstamp:   time.Now(),
			})
		}
	}

	return s.Single(ctx, name, id)
}

func (s *EntityService) Update(ctx context.Context, name string, data datamodels.Record) (datamodels.Record, error) {
	entity, err := s.schemaService.LoadEntity(ctx, name)
	if err != nil {
		return nil, err
	}

	id := data[entity.PrimaryKey]
	key := fmt.Sprintf("%v", id)
	rec, err := s.dao.Get(ctx, s.getNamespace(name), key)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("record not found")
	}

	// JIT Evolution before update
	recData, ok := rec.Rec.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid record data format")
	}
	evolved, _, err := s.evolutionService.EvolveRecord(name, recData, &rec.MetaData)
	if err != nil {
		return nil, fmt.Errorf("failed to evolve record: %w", err)
	}
	existing := evolved

	roles, _ := ctx.Value("roles").([]string)
	fieldPerms, _ := s.permissionService.GetFieldPermissions(ctx, name, roles)

	for k, v := range data {
		if p, ok := fieldPerms[k]; ok && !p["write"] {
			continue
		}

		val := v
		if name == "User" && k == "password_hash" {
			if str, ok := v.(string); ok && str != "" {
				if !strings.HasPrefix(str, "$2a$") && !strings.HasPrefix(str, "$2b$") && !strings.HasPrefix(str, "$2y$") {
					hashed, err := bcrypt.GenerateFromPassword([]byte(str), bcrypt.DefaultCost)
					if err != nil {
						return nil, fmt.Errorf("failed to hash password: %w", err)
					}
					val = string(hashed)
				}
			} else {
				continue
			}
		}
		existing[k] = val
	}

	existing["updated_at"] = time.Now()

	if name == "User" {
		roleStr, _ := existing["role"].(string)
		var rolesList []string
		if rRaw, ok := existing["roles"].([]interface{}); ok && len(rRaw) > 0 {
			for _, item := range rRaw {
				if sItem, ok := item.(string); ok && sItem != "" {
					rolesList = append(rolesList, sItem)
				}
			}
		} else if rList, ok := existing["roles"].([]string); ok && len(rList) > 0 {
			rolesList = rList
		}

		if len(rolesList) == 0 {
			switch roleStr {
			case descriptors.RoleSa, "superadmin":
				rolesList = []string{descriptors.RoleSa, descriptors.RoleAdmin, descriptors.RoleUser}
			case descriptors.RoleAdmin:
				rolesList = []string{descriptors.RoleAdmin, descriptors.RoleUser}
			case descriptors.RoleUser:
				rolesList = []string{descriptors.RoleUser}
			case descriptors.RoleGuest:
				rolesList = []string{descriptors.RoleGuest}
			case "":
				rolesList = []string{descriptors.RoleAdmin, descriptors.RoleUser}
				existing["role"] = descriptors.RoleAdmin
			default:
				rolesList = []string{roleStr, descriptors.RoleUser}
			}
		}
		existing["roles"] = rolesList
	}

	rec.Rec = existing
	rec.Tmstamp = time.Now()
	// Metadata is preserved and will be incremented by Dao.Save

	if err := s.dao.Save(ctx, *rec); err != nil {
		return nil, err
	}

	if name == "User" {
		if email, ok := existing["email"].(string); ok && email != "" {
			userIdInt := ids.StringToInt64ID(fmt.Sprintf("%v", id))
			phoneStr, _ := existing["phone"].(string)
			pwHash, _ := existing["password_hash"].(string)
			avatarStr, _ := existing["avatar_path"].(string)
			rolesList, _ := existing["roles"].([]string)

			userDesc := descriptors.User{
				Id:           userIdInt,
				Email:        email,
				Phone:        phoneStr,
				AvatarPath:   avatarStr,
				PasswordHash: pwHash,
				Roles:        rolesList,
				UpdatedAt:    time.Now(),
			}
			_ = s.dao.Save(ctx, datamodels.RecJSON{
				Namespace: "aigen.core.descriptors.User",
				Key:       email,
				Rec:       userDesc,
				Tmstamp:   time.Now(),
			})
		}
	}

	return s.Single(ctx, name, id)
}

func (s *EntityService) Delete(ctx context.Context, name string, id interface{}) error {
	if name == "User" {
		if rec, _ := s.dao.Get(ctx, s.getNamespace(name), fmt.Sprintf("%v", id)); rec != nil {
			if recMap, ok := rec.Rec.(map[string]interface{}); ok {
				if email, ok := recMap["email"].(string); ok && email != "" {
					_ = s.dao.Delete(ctx, "aigen.core.descriptors.User", email)
				}
			}
		}
	}
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

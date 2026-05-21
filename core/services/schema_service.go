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
)

const SchemaNamespace = "aigen.core.descriptors.Schema"

type SchemaService struct {
	dao               relationdbdao.IPrimaryDao
	permissionService IPermissionService
}

func NewSchemaService(dao relationdbdao.IPrimaryDao) *SchemaService {
	return &SchemaService{dao: dao}
}

func (s *SchemaService) SetPermissionService(ps IPermissionService) {
	s.permissionService = ps
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
			FieldName: "publicationStatus",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{*status}},
			},
		})
	} else {
		filters = append(filters, datamodels.Filter{
			FieldName: "isLatest",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{true}},
			},
		})
	}

	recs, _, err := s.dao.List(ctx, SchemaNamespace, filters, datamodels.Pagination{}, nil)
	if err != nil {
		return nil, err
	}

	results := []*descriptors.Schema{}
	userId, _ := ctx.Value("userId").(int64)
	roles, _ := ctx.Value("roles").([]string)

	for _, r := range recs {
		schema, err := descriptors.RecordToSchema(r.Rec.(map[string]interface{}))
		if err != nil {
			return nil, err
		}

		// Filter based on permissions
		if s.permissionService != nil && userId != 0 {
			// Resource name for schemas is its name (e.g. "Lead", "Role", "top-menu-bar")
			allowed, _ := s.permissionService.HasAccess(ctx, userId, roles, schema.Name, "read")
			if !allowed {
				continue
			}
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
			FieldName: "publicationStatus",
			Constraints: []datamodels.Constraint{
				{Match: "equals", Values: []interface{}{*status}},
			},
		})
	} else {
		filters = append(filters, datamodels.Filter{
			FieldName: "isLatest",
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
		schema.SchemaId = ids.NewRandomID()
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
				FieldName: "schemaId",
				Constraints: []datamodels.Constraint{
					{Match: "equals", Values: []interface{}{schema.SchemaId}},
				},
			},
			{
				FieldName: "isLatest",
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
				FieldName: "schemaId",
				Constraints: []datamodels.Constraint{
					{Match: "equals", Values: []interface{}{schema.SchemaId}},
				},
			},
			{
				FieldName: "publicationStatus",
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

func (s *SchemaService) BootstrapDefaultHomePage(ctx context.Context) error {
	schema, err := s.ByNameOrDefault(ctx, "home", descriptors.PageSchema, nil)
	if err != nil {
		return err
	}
	if schema != nil {
		return nil
	}

	htmlContent := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>AIGenApp - Next Generation Dynamic Engine</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;500;600;700;800&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-color: #0b0f19;
            --card-bg: rgba(17, 24, 39, 0.7);
            --card-border: rgba(255, 255, 255, 0.08);
            --text-primary: #f3f4f6;
            --text-secondary: #9ca3af;
            --accent-purple: #8b5cf6;
            --accent-cyan: #06b6d4;
            --accent-pink: #ec4899;
            --accent-gradient: linear-gradient(135deg, #8b5cf6 0%, #06b6d4 100%);
            --mesh-gradient: radial-gradient(at 0% 0%, rgba(139, 92, 246, 0.15) 0px, transparent 50%),
                             radial-gradient(at 100% 0%, rgba(6, 182, 212, 0.15) 0px, transparent 50%),
                             radial-gradient(at 50% 100%, rgba(236, 72, 153, 0.1) 0px, transparent 50%);
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: 'Outfit', sans-serif;
            background-color: var(--bg-color);
            background-image: var(--mesh-gradient);
            background-attachment: fixed;
            color: var(--text-primary);
            min-height: 100vh;
            overflow-x: hidden;
            line-height: 1.6;
        }

        header {
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            z-index: 100;
            backdrop-filter: blur(12px);
            -webkit-backdrop-filter: blur(12px);
            background-color: rgba(11, 15, 25, 0.7);
            border-bottom: 1px solid var(--card-border);
        }

        .nav-container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 1.25rem 2rem;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .logo {
            font-size: 1.5rem;
            font-weight: 800;
            background: linear-gradient(to right, #a78bfa, #22d3ee);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            text-decoration: none;
            letter-spacing: -0.5px;
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .logo-dot {
            width: 8px;
            height: 8px;
            background-color: var(--accent-cyan);
            border-radius: 50%;
            display: inline-block;
            box-shadow: 0 0 12px var(--accent-cyan);
        }

        .nav-links {
            display: flex;
            align-items: center;
            gap: 2rem;
        }

        .nav-link {
            color: var(--text-secondary);
            text-decoration: none;
            font-weight: 500;
            transition: color 0.3s ease;
        }

        .nav-link:hover {
            color: var(--text-primary);
        }

        .btn-nav-login {
            background: rgba(255, 255, 255, 0.05);
            border: 1px solid var(--card-border);
            padding: 0.5rem 1.25rem;
            border-radius: 9999px;
            font-weight: 600;
            font-size: 0.875rem;
            color: var(--text-primary);
            cursor: pointer;
            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
            text-decoration: none;
            display: inline-block;
        }

        .btn-nav-login:hover {
            background: rgba(255, 255, 255, 0.1);
            border-color: rgba(255, 255, 255, 0.2);
            transform: translateY(-2px);
        }

        main {
            max-width: 1200px;
            margin: 0 auto;
            padding: 8rem 2rem 4rem;
            display: flex;
            flex-direction: column;
            align-items: center;
            text-align: center;
        }

        .hero-badge {
            background: rgba(139, 92, 246, 0.1);
            border: 1px solid rgba(139, 92, 246, 0.3);
            color: #c084fc;
            padding: 0.5rem 1.25rem;
            border-radius: 9999px;
            font-size: 0.875rem;
            font-weight: 600;
            margin-bottom: 2rem;
            display: inline-flex;
            align-items: center;
            gap: 0.5rem;
            letter-spacing: 0.5px;
            box-shadow: 0 4px 20px rgba(139, 92, 246, 0.1);
            animation: pulse 3s infinite alternate;
        }

        @keyframes pulse {
            0% { transform: scale(1); box-shadow: 0 4px 20px rgba(139, 92, 246, 0.1); }
            100% { transform: scale(1.03); box-shadow: 0 4px 25px rgba(139, 92, 246, 0.25); }
        }

        .hero-title {
            font-size: 4rem;
            font-weight: 800;
            line-height: 1.15;
            letter-spacing: -1.5px;
            max-width: 900px;
            margin-bottom: 1.5rem;
        }

        .hero-title span {
            background: linear-gradient(135deg, #a78bfa 0%, #22d3ee 50%, #f472b6 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }

        .hero-description {
            font-size: 1.25rem;
            color: var(--text-secondary);
            max-width: 650px;
            margin-bottom: 3rem;
            font-weight: 400;
        }

        .cta-container {
            display: flex;
            gap: 1rem;
            margin-bottom: 5rem;
        }

        .btn-primary {
            background: var(--accent-gradient);
            border: none;
            padding: 0.875rem 2.25rem;
            border-radius: 9999px;
            font-weight: 600;
            color: white;
            font-size: 1rem;
            cursor: pointer;
            text-decoration: none;
            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
            box-shadow: 0 4px 20px rgba(6, 182, 212, 0.3);
            display: inline-block;
        }

        .btn-primary:hover {
            transform: translateY(-3px);
            box-shadow: 0 8px 30px rgba(6, 182, 212, 0.5);
        }

        .btn-secondary {
            background: rgba(255, 255, 255, 0.03);
            border: 1px solid var(--card-border);
            padding: 0.875rem 2.25rem;
            border-radius: 9999px;
            font-weight: 600;
            color: var(--text-primary);
            font-size: 1rem;
            cursor: pointer;
            text-decoration: none;
            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
            display: inline-block;
        }

        .btn-secondary:hover {
            background: rgba(255, 255, 255, 0.08);
            border-color: rgba(255, 255, 255, 0.2);
            transform: translateY(-3px);
        }

        .preview-container {
            width: 100%;
            max-width: 1000px;
            background: var(--card-bg);
            border: 1px solid var(--card-border);
            border-radius: 20px;
            padding: 0.75rem;
            backdrop-filter: blur(16px);
            -webkit-backdrop-filter: blur(16px);
            box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
            margin-bottom: 6rem;
            position: relative;
        }

        .preview-container::before {
            content: '';
            position: absolute;
            top: -2px;
            left: -2px;
            right: -2px;
            bottom: -2px;
            background: linear-gradient(135deg, rgba(139, 92, 246, 0.2), rgba(6, 182, 212, 0.2));
            z-index: -1;
            border-radius: 22px;
            filter: blur(10px);
        }

        .preview-header {
            display: flex;
            align-items: center;
            padding: 0.5rem 1rem 1rem;
            border-bottom: 1px solid rgba(255, 255, 255, 0.05);
            margin-bottom: 1rem;
        }

        .window-dots {
            display: flex;
            gap: 6px;
        }

        .window-dot {
            width: 10px;
            height: 10px;
            border-radius: 50%;
        }

        .window-dot.red { background: #ef4444; }
        .window-dot.yellow { background: #f59e0b; }
        .window-dot.green { background: #10b981; }

        .window-title {
            margin: 0 auto;
            font-size: 0.85rem;
            color: var(--text-secondary);
            font-weight: 500;
        }

        .preview-content {
            background: #060913;
            border-radius: 12px;
            padding: 2.5rem;
            text-align: left;
            overflow: hidden;
            position: relative;
        }

        .preview-schema-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
            gap: 1.5rem;
        }

        .schema-card {
            background: rgba(255, 255, 255, 0.02);
            border: 1px solid rgba(255, 255, 255, 0.05);
            border-radius: 12px;
            padding: 1.5rem;
            transition: all 0.3s ease;
        }

        .schema-card:hover {
            background: rgba(255, 255, 255, 0.05);
            border-color: rgba(139, 92, 246, 0.3);
            transform: translateY(-5px);
        }

        .schema-title {
            font-size: 1.1rem;
            font-weight: 600;
            margin-bottom: 0.75rem;
            color: var(--text-primary);
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .schema-title i {
            color: var(--accent-cyan);
        }

        .schema-desc {
            font-size: 0.875rem;
            color: var(--text-secondary);
            margin-bottom: 1.25rem;
        }

        .schema-tag {
            font-size: 0.75rem;
            background: rgba(6, 182, 212, 0.1);
            color: var(--accent-cyan);
            padding: 0.25rem 0.625rem;
            border-radius: 9999px;
            font-weight: 600;
            display: inline-block;
        }

        .section-title {
            font-size: 2.25rem;
            font-weight: 800;
            margin-bottom: 1rem;
            letter-spacing: -0.5px;
        }

        .section-desc {
            font-size: 1.1rem;
            color: var(--text-secondary);
            max-width: 600px;
            margin: 0 auto 4rem;
        }

        .features-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
            gap: 2rem;
            width: 100%;
            margin-bottom: 6rem;
        }

        .feature-card {
            background: var(--card-bg);
            border: 1px solid var(--card-border);
            border-radius: 16px;
            padding: 2.5rem 2rem;
            text-align: left;
            backdrop-filter: blur(10px);
            -webkit-backdrop-filter: blur(10px);
            transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
        }

        .feature-card:hover {
            transform: translateY(-8px);
            border-color: rgba(139, 92, 246, 0.4);
            box-shadow: 0 20px 40px -15px rgba(139, 92, 246, 0.2);
        }

        .feature-icon {
            width: 48px;
            height: 48px;
            border-radius: 12px;
            background: linear-gradient(135deg, rgba(139, 92, 246, 0.2) 0%, rgba(6, 182, 212, 0.2) 100%);
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 1.5rem;
            color: var(--accent-cyan);
            margin-bottom: 1.5rem;
            border: 1px solid rgba(6, 182, 212, 0.3);
        }

        .feature-name {
            font-size: 1.25rem;
            font-weight: 700;
            margin-bottom: 0.75rem;
        }

        .feature-desc {
            color: var(--text-secondary);
            font-size: 0.95rem;
            line-height: 1.5;
        }

        footer {
            border-top: 1px solid var(--card-border);
            width: 100%;
            padding: 3rem 2rem;
            background: rgba(11, 15, 25, 0.5);
            text-align: center;
            color: var(--text-secondary);
            font-size: 0.875rem;
        }

        footer p {
            margin-bottom: 1rem;
        }

        .footer-links {
            display: flex;
            justify-content: center;
            gap: 1.5rem;
        }

        .footer-link {
            color: var(--text-secondary);
            text-decoration: none;
            transition: color 0.3s ease;
        }

        .footer-link:hover {
            color: var(--text-primary);
        }

        @media (max-width: 768px) {
            .hero-title {
                font-size: 2.5rem;
            }
            .nav-links {
                display: none;
            }
            .preview-content {
                padding: 1.5rem;
            }
        }
    </style>
</head>
<body>
    <header>
        <div class="nav-container">
            <a href="#" class="logo">
                <span class="logo-dot"></span>
                AIGenApp
            </a>
            <div class="nav-links">
                <a href="#features" class="nav-link">Features</a>
                <a href="/admin/list.html" class="btn-nav-login">Sign In</a>
            </div>
        </div>
    </header>

    <main>
        <div class="hero-badge">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polygon points="12 2 2 7 12 12 22 7 12 2"></polygon><polyline points="2 17 12 22 22 17"></polyline><polyline points="2 12 12 17 22 12"></polyline></svg>
            Schema-on-Read Engine Ready
        </div>

        <h1 class="hero-title">
            The headless platform built for <span>Limitless Speed</span> and flexibility.
        </h1>

        <p class="hero-description">
            A dynamic application framework powered by a single-table JSON architecture in PostgreSQL, featuring custom schemas, role-based controls, and a developer-first GraphQL layer.
        </p>

        <div class="cta-container">
            <a href="/admin/list.html" class="btn-primary">Launch Dashboard</a>
            <a href="#features" class="btn-secondary">Learn More</a>
        </div>

        <div class="preview-container">
            <div class="preview-header">
                <div class="window-dots">
                    <span class="window-dot red"></span>
                    <span class="window-dot yellow"></span>
                    <span class="window-dot green"></span>
                </div>
                <div class="window-title">aigen-app_dashboard_preview.io</div>
            </div>
            <div class="preview-content">
                <div style="margin-bottom: 2rem;">
                    <h3 style="font-weight: 700; font-size: 1.5rem; margin-bottom: 0.5rem; color: #fff;">Interactive Workspace</h3>
                    <p style="color: var(--text-secondary); font-size: 0.95rem;">Experience live schema introspection and zero-downtime structural modifications.</p>
                </div>
                <div class="preview-schema-grid">
                    <div class="schema-card">
                        <div class="schema-title">
                            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="color:var(--accent-purple)"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"></circle><path d="M23 21v-2a4 4 0 0 0-3-3.87"></path><path d="M16 3.13a4 4 0 0 1 0 7.75"></path></svg>
                            User Role
                        </div>
                        <p class="schema-desc">Comprehensive system role descriptor mapped dynamically into active application contexts.</p>
                        <span class="schema-tag">Active Schema</span>
                    </div>
                    <div class="schema-card">
                        <div class="schema-title">
                            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="color:var(--accent-cyan)"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect><line x1="9" y1="3" x2="9" y2="21"></line></svg>
                            Lead Entity
                        </div>
                        <p class="schema-desc">Track and nurture visitor entries dynamically inside the automated marketing pipeline.</p>
                        <span class="schema-tag">Entity</span>
                    </div>
                    <div class="schema-card">
                        <div class="schema-title">
                            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="color:var(--accent-pink)"><polygon points="12 2 2 22 22 22"></polygon></svg>
                            Page Configuration
                        </div>
                        <p class="schema-desc">Design premium client-facing interfaces with our integrated Handlebars & GrapesJS renderer.</p>
                        <span class="schema-tag">Layout</span>
                    </div>
                </div>
            </div>
        </div>

        <h2 id="features" class="section-title">Designed for Modern Operations</h2>
        <p class="section-desc">Get the ultimate developer experience with all components tightly coupled and configured out of the box.</p>

        <div class="features-grid">
            <div class="feature-card">
                <div class="feature-icon">
                    <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><ellipse cx="12" cy="5" rx="9" ry="3"></ellipse><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"></path><path d="M3 12c0 1.66 4 3 9 3s9-1.34 9-3"></path></svg>
                </div>
                <h3 class="feature-name">Single-Table Persistence</h3>
                <p class="feature-desc">Engineered around a high-performance JSON-B schema-on-read database model inside PostgreSQL. Zero migration stress.</p>
            </div>
            <div class="feature-card">
                <div class="feature-icon">
                    <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="16 18 22 12 16 6"></polyline><polyline points="8 6 2 12 8 18"></polyline></svg>
                </div>
                <h3 class="feature-name">Dynamic GraphQL Engine</h3>
                <p class="feature-desc">All user schemas are dynamically compiled into functional GraphQL queries, mutations, and resolver schemas at runtime.</p>
            </div>
            <div class="feature-card">
                <div class="feature-icon">
                    <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path></svg>
                </div>
                <h3 class="feature-name">Robust Role RBAC</h3>
                <p class="feature-desc">Deeply granular, rule-based permission layers protecting field, entity, and layout levels natively with secure token validation.</p>
            </div>
        </div>
    </main>

    <footer>
        <p>&copy; 2026 AIGenApp Engine. All rights reserved.</p>
        <div class="footer-links">
            <a href="#" class="footer-link">Documentation</a>
            <a href="#" class="footer-link">Privacy Policy</a>
            <a href="#" class="footer-link">GitHub</a>
        </div>
    </footer>
</body>
</html>`

	page := &descriptors.Page{
		Name:  "home",
		Title: "AIGenApp - Next Generation Dynamic Engine",
		Html:  htmlContent,
	}

	defaultHomeSchema := &descriptors.Schema{
		SchemaId:          "default-home-page",
		Name:              "home",
		Type:              descriptors.PageSchema,
		Description:       "Default premium home page for guest and unauthenticated users.",
		Settings: &descriptors.SchemaSettings{
			Page: page,
		},
		IsLatest:          true,
		PublicationStatus: descriptors.Published,
		CreatedAt:         time.Now(),
	}

	_, err = s.Save(ctx, defaultHomeSchema, true)
	return err
}

package services

import (
	"context"
	"testing"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/innomon/aigen-app/utils/datamodels"
	"github.com/stretchr/testify/assert"
)

func TestRBACIntegration(t *testing.T) {
	ctx := context.Background()
	dao, _ := relationdbdao.CreateDao("memory://")
	dao.EnsureTable(ctx)
	
	schemaSvc := NewSchemaService(dao)
	permSvc := NewPermissionService(dao, schemaSvc)
	entitySvc := NewEntityService(schemaSvc, dao, permSvc)

	// 1. Setup Schemas
	companySchema := &descriptors.Entity{
		Name:       "Company",
		TableName:  "companies",
		PrimaryKey: "id",
		Attributes: []descriptors.Attribute{
			{Field: "id", DataType: "String"},
			{Field: "name", DataType: "String"},
		},
	}
	schemaSvc.Save(ctx, &descriptors.Schema{
		Name:     "Company",
		Type:     descriptors.EntitySchema,
		Settings: &descriptors.SchemaSettings{Entity: companySchema},
		IsLatest: true,
		PublicationStatus: descriptors.Published,
	}, true)

	invoiceSchema := &descriptors.Entity{
		Name:       "Invoice",
		TableName:  "invoices",
		PrimaryKey: "id",
		Attributes: []descriptors.Attribute{
			{Field: "id", DataType: "String"},
			{Field: "amount", DataType: "Float", PermLevel: 0},
			{Field: "secret_notes", DataType: "String", PermLevel: 1},
			{Field: "company", DataType: "Lookup", Options: "Company", PermLevel: 0},
		},
	}
	schemaSvc.Save(ctx, &descriptors.Schema{
		Name:     "Invoice",
		Type:     descriptors.EntitySchema,
		Settings: &descriptors.SchemaSettings{Entity: invoiceSchema},
		IsLatest: true,
		PublicationStatus: descriptors.Published,
	}, true)

	// 2. Setup RBAC Data
	// Role: Manager
	// DocPerm: Invoice, role: Manager, read: true, write: true, permlevel: 0
	dao.Save(ctx, datamodels.RecJSON{
		Namespace: DocPermNamespace,
		Key:       "perm1",
		Rec: map[string]interface{}{
			"role":      "Manager",
			"parent":    "Invoice",
			"permlevel": 0.0,
			"read":      true,
			"write":     true,
		},
	})

	// UserPermission: User 100, allow: Company, for_value: C1
	dao.Save(ctx, datamodels.RecJSON{
		Namespace: UserPermNamespace,
		Key:       "uperm1",
		Rec: map[string]interface{}{
			"userId":    100.0,
			"allow":     "Company",
			"for_value": "C1",
		},
	})

	// 3. Setup Test Data
	// Bypass RBAC for initial setup by using SA context
	saCtx := context.WithValue(ctx, "userId", int64(1))
	saCtx = context.WithValue(saCtx, "roles", []string{"sa"})

	entitySvc.Insert(saCtx, "Invoice", datamodels.Record{"id": "INV1", "amount": 100.0, "company": "C1", "secret_notes": "Very secret"})
	entitySvc.Insert(saCtx, "Invoice", datamodels.Record{"id": "INV2", "amount": 200.0, "company": "C2", "secret_notes": "Another secret"})

	t.Run("Row-level filtering for User 100", func(t *testing.T) {
		// Context with User 100 and Role Manager
		userCtx := context.WithValue(ctx, "userId", int64(100))
		userCtx = context.WithValue(userCtx, "roles", []string{"Manager"})

		recs, total, err := entitySvc.List(userCtx, "Invoice", datamodels.Pagination{}, nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, recs, 1)
		assert.Equal(t, "INV1", recs[0]["id"])
	})

	t.Run("Field-level filtering (Manager has no access to PermLevel 1)", func(t *testing.T) {
		userCtx := context.WithValue(ctx, "userId", int64(100))
		userCtx = context.WithValue(userCtx, "roles", []string{"Manager"})

		rec, err := entitySvc.Single(userCtx, "Invoice", "INV1")
		assert.NoError(t, err)
		assert.NotNil(t, rec["amount"])
		_, ok := rec["secret_notes"]
		assert.False(t, ok, "secret_notes should be filtered out")
	})

	t.Run("Field-level filtering (SA has access to all fields)", func(t *testing.T) {
		saCtx := context.WithValue(ctx, "userId", int64(1))
		saCtx = context.WithValue(saCtx, "roles", []string{"sa"})

		rec, err := entitySvc.Single(saCtx, "Invoice", "INV1")
		assert.NoError(t, err)
		assert.NotNil(t, rec["amount"])
		assert.NotNil(t, rec["secret_notes"])
		assert.Equal(t, "Very secret", rec["secret_notes"])
	})
}

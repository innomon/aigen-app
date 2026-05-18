package apps

import (
	"context"
	"testing"

	"github.com/innomon/aigen-app/core/services"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/innomon/aigen-app/utils/datamodels"
	"github.com/stretchr/testify/assert"
)

func TestERPNextAccountingIntegration(t *testing.T) {
	ctx := context.WithValue(context.Background(), "roles", []string{"sa"})

	// Use memory DAO for tests
	dao, err := relationdbdao.CreateDao("memory://")
	assert.NoError(t, err)
	defer dao.Close()

	err = dao.EnsureTable(ctx)
	assert.NoError(t, err)

	schemaService := services.NewSchemaService(dao)
	permissionService := services.NewPermissionService(dao, schemaService)
	entityService := services.NewEntityService(schemaService, dao, permissionService)

	appsDir := "../../apps"
	appName := "erpnext_accounting"

	// 1. Setup Schemas
	err = SetupApp(ctx, appsDir, appName, schemaService, dao)
	assert.NoError(t, err)

	// 2. Setup Test Data
	err = SetupAppTestData(ctx, appsDir, appName, entityService)
	assert.NoError(t, err)

	// 3. Verify Data
	t.Run("Verify Currency", func(t *testing.T) {
		currencies, _, err := entityService.List(ctx, "Currency", datamodels.Pagination{}, nil, nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, currencies)
		assert.Equal(t, "INR", currencies[0]["currency_name"])
	})

	t.Run("Verify Company", func(t *testing.T) {
		companies, _, err := entityService.List(ctx, "Company", datamodels.Pagination{}, nil, nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, companies)
		assert.Equal(t, "Tata Motors", companies[0]["company_name"])
		assert.NotNil(t, companies[0]["default_currency"])
	})

	t.Run("Verify Accounts", func(t *testing.T) {
		accounts, _, err := entityService.List(ctx, "Account", datamodels.Pagination{}, nil, nil)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(accounts), 5)

		// Check hierarchy (HDFC Bank parent is Bank Accounts)
		var hdfc, bankGroup map[string]interface{}
		for _, acc := range accounts {
			if acc["account_name"] == "HDFC Bank" {
				hdfc = acc
			}
			if acc["account_name"] == "Bank Accounts" {
				bankGroup = acc
			}
		}
		assert.NotNil(t, hdfc)
		assert.NotNil(t, bankGroup)
		assert.Equal(t, bankGroup["id"], hdfc["parent_account"])
	})

	t.Run("Verify Journal Entry and Children", func(t *testing.T) {
		jes, _, err := entityService.List(ctx, "JournalEntry", datamodels.Pagination{}, nil, nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, jes)

		je := jes[0]
		assert.Equal(t, 50000.0, je["total_debit"])

		// Check child records
		children, _, err := entityService.CollectionList(ctx, "JournalEntry", je["id"].(string), "accounts", datamodels.Pagination{}, nil, nil)
		assert.NoError(t, err)
		assert.Len(t, children, 2)
	})

	t.Run("Verify GL Entries", func(t *testing.T) {
		gles, _, err := entityService.List(ctx, "GLEntry", datamodels.Pagination{}, nil, nil)
		assert.NoError(t, err)
		assert.Len(t, gles, 2)
	})

    t.Run("Perform CRUD - Create Account", func(t *testing.T) {
        newAccData := map[string]interface{}{
            "account_name": "ICICI Bank",
            "is_group": false,
            "company": "Company_TATA_ID_PLACEHOLDER", 
            "root_type": "Asset",
            "account_type": "Bank",
        }
        
        // Get Tata Motors ID
        companies, _, _ := entityService.List(ctx, "Company", datamodels.Pagination{}, nil, nil)
        newAccData["company"] = companies[0]["id"]
        
        rec, err := entityService.Insert(ctx, "Account", newAccData)
        assert.NoError(t, err)
        assert.NotNil(t, rec["id"])
        
        // Read back
        found, err := entityService.Single(ctx, "Account", rec["id"])
        assert.NoError(t, err)
        assert.Equal(t, "ICICI Bank", found["account_name"])
        
        // Update
        updateData := map[string]interface{}{
            "id":           rec["id"],
            "account_name": "ICICI Bank - Updated",
        }
        _, err = entityService.Update(ctx, "Account", updateData)
        assert.NoError(t, err)
        
        found, _ = entityService.Single(ctx, "Account", rec["id"])
        assert.Equal(t, "ICICI Bank - Updated", found["account_name"])
        
        // Delete
        err = entityService.Delete(ctx, "Account", rec["id"])
        assert.NoError(t, err)
        
        _, err = entityService.Single(ctx, "Account", rec["id"])
        assert.Error(t, err) // Should not be found
    })

}

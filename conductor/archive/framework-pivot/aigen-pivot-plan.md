# Plan: AIGenApp Pivot - Single-Table JSON Store & Project Rename

This plan outlines the migration of the AIGen CMS (to be renamed to AIGenApp) from a dynamic relational table model to a single-table JSON storage model.

## 1. Project Renaming
- Rename module in `go.mod` to `github.com/innomon/aigen-app`.
- Update all Go import paths from `github.com/innomon/aigen-app` to `github.com/innomon/aigen-app`.
- Update references in `README.md`, `GEMINI.md`, and other documentation.
- Update UI strings in `core/api/ui/`.

## 2. Core Data Model: `RecJSON`
- Add `RecJSON` struct to `utils/datamodels/rec_json.go`.
```go
package datamodels

import "time"

type RecJSON struct {
	Namespace string      `json:"namespace"`
	Key       string      `json:"key"`
	Rec       interface{} `json:"rec"`
	MetaData  interface{} `json:"metadata"`
	Tmstamp   time.Time   `json:"tmstamp"`
}
```

## 3. Database Layer Overhaul
### 3.1. Settings Update
- Remove `DatabaseProvider` and related constants from `core/descriptors/settings.go`.
- The system will now automatically detect if it's using Postgres or SQLite based on the connection string or driver.

### 3.2. DAO Interface (`IPrimaryDao`)
- Refactor `infrastructure/relationdbdao/interface.go` to focus on `RecJSON` operations.
- Methods to include:
    - `Save(ctx, rec RecJSON) error`
    - `Get(ctx, namespace, key) (*RecJSON, error)`
    - `Delete(ctx, namespace, key) error`
    - `List(ctx, namespace, filters, pagination, sorts) ([]RecJSON, int64, error)`
    - `EnsureTable(ctx) error` (to create `aigen_records`)

### 3.3. DAO Implementations
- Update `postgres.go` and `sqlite.go` to implement the new interface.
- Implement JSON-path filtering for `List` operations.
    - Postgres: Use `->>` and `@>` operators.
    - SQLite: Use `->>` operator and `json_extract`.

## 4. Service Layer Refactoring
### 4.1. `SchemaService`
- Migrate all schema operations to use the `aigen_records` table.
- Namespace: `aigen.core.descriptors.Schema`.
- Remove use of `__schemas` table.

### 4.2. `EntityService`
- Migrate dynamic entity operations to use the `aigen_records` table.
- Namespace: `aigen.app.<appName>.<entityName>`.
- Update `Insert`, `Update`, `Delete`, `List`, `Single`.
- Support `MetaData` for RBAC.

### 4.3. Other Services
- Refactor `AssetService`, `AuditService`, `CommentService`, `EngagementService`, `NotificationService`, `PageService` to use the JSON store with appropriate namespaces (e.g., `aigen.core.descriptors.Asset`).

## 5. App Setup and Initialization
- Update `framework/init.go` to call `dao.EnsureTable(ctx)`.
- Update `core/apps/setup.go` to remove table creation logic (`dao.CreateTable`).

## 6. Verification and Testing
- Update existing tests in `core/services/` to work with the new DAO.
- Add new tests specifically for JSON filtering and metadata-based RBAC.
- Verify that the `aigen_records` table is correctly created and populated.

## 7. Migration of Existing Data (Optional/Manual)
- Since this is a pivot, we assume existing data in dynamic tables can be migrated or the database can be reset. We will provide a basic migration script if requested.

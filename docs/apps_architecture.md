# AiGen CMS App Architecture & Lifecycle

This document provides a comprehensive overview of the architecture, data structures, and lifecycle of applications ("apps") within the AiGen CMS ecosystem. It also serves as a guide on how to create, deploy, and modify these apps.

## 1. Architecture Overview

AiGen CMS is a headless Content Management System written in Go (migrated from C#). It is built for extreme flexibility, using dynamic data modeling where database tables are created and modified on the fly based on JSON schemas.

### Core Technology Stack
- **Routing**: `net/http` + `chi` router
- **Database Access**: Standard `database/sql` supporting SQLite, PostgreSQL, MySQL, and SQL Server.
- **Dynamic Queries**: `Masterminds/squirrel` for query building, as table schemas are not known at compile time (preventing the use of standard ORMs).
- **GraphQL**: `graphql-go/graphql` for dynamic API endpoints based on schemas.
- **Templating**: `aymerick/raymond` (Handlebars template engine) for dynamic page rendering.

### What is an "App" in AiGen CMS?
An "App" in AiGen CMS is essentially a bundle of predefined entity schemas and optional test data. These apps provide out-of-the-box functionality for specific domains (e.g., CRM, RBAC, ERPNext Accounting). Instead of writing complex migration scripts, an app developer defines their data model in JSON, and the CMS engine automatically handles the underlying database structure and CRUD/GraphQL APIs.

## 2. Data Structure

Apps reside in the `apps/` directory at the project root.

### Directory Layout
```text
apps/
├── apps.json                    # Registry of currently enabled apps
├── crm/                         # Example App: CRM
│   ├── data/
│   │   └── test_data.json       # Seed data for the app
│   └── schemas/                 # Entity schemas defining the app's structure
│       ├── crm_lead.json
│       ├── crm_deal.json
│       └── ...
└── rbac/                        # Example App: Role-Based Access Control
    ├── data/
    │   └── test_data.json
    └── schemas/
        └── role.json
```

### Schemas (`schemas/*.json`)
A schema defines an entity (which maps to a database table). It describes attributes (columns), relationships (lookups, junctions, collections), and UI metadata.

### Test Data (`data/test_data.json`)
A JSON array specifying seed records to insert upon app deployment. It supports reference linking (using `$Ref:<key>`) to handle relationships between newly inserted records.

---

## 3. App Lifecycle

The lifecycle of an app is managed by the core CMS initialization process (specifically inside `main.go` and `core/apps/setup.go`).

1. **Discovery & Configuration**: 
   Upon startup, the server reads `apps/apps.json` to determine which apps are enabled.
2. **Schema Setup (`SetupApp`)**:
   For each enabled app, the CMS scans the `apps/<app_name>/schemas/` directory.
   - Parses each `.json` file into an `Entity` descriptor.
   - Translates local data types (e.g., `Text`, `Int`, `Boolean`) into physical SQL column types.
   - Automatically adds system columns: `id`, `created_at`, `updated_at`, and `deleted`.
   - Executes a `CREATE TABLE` query to construct the physical table if it doesn't already exist.
   - Saves the schema definition into the core `__schemas` table, marking it as `Published`.
3. **Data Seeding (`SetupAppTestData`)**:
   After all tables are created, the CMS reads `apps/<app_name>/data/test_data.json`.
   - It checks if data already exists to prevent duplicate seeding.
   - Inserts the records, dynamically resolving any `$Ref` cross-references between records.

---

## 4. How to Create an App

To create a new app (e.g., `inventory`), follow these steps:

1. **Create the Directory Structure**:
   Create a new folder in the `apps` directory:
   ```bash
   mkdir -p apps/inventory/schemas
   mkdir -p apps/inventory/data
   ```

2. **Define Entity Schemas**:
   Create JSON schema files for your entities inside `apps/inventory/schemas/`. 
   For example, `product.json`:
   ```json
   {
     "name": "inventory_product",
     "tableName": "inventory_product",
     "attributes": [
       { "field": "name", "dataType": "String" },
       { "field": "price", "dataType": "Float" },
       { "field": "in_stock", "dataType": "Boolean" }
     ]
   }
   ```

3. **(Optional) Provide Seed Data**:
   Create `apps/inventory/data/test_data.json`:
   ```json
   [
     {
       "Entity": "inventory_product",
       "Ref": "prod_1",
       "Data": {
         "name": "Super Widget",
         "price": 19.99,
         "in_stock": true
       }
     }
   ]
   ```

---

## 5. How to Deploy an App

Deploying an app simply involves enabling it so the CMS picks it up on the next startup.

1. Open `apps/apps.json`.
2. Add your app's directory name to the `enabled_apps` array:
   ```json
   {
     "enabled_apps": [
       "rbac",
       "crm",
       "inventory"
     ]
   }
   ```
3. Restart the AiGen CMS backend server. The system will automatically build the tables, register the schemas for GraphQL/REST APIs, and seed the initial data.

---

## 6. How to Modify an App

Modifying an app's structure or behavior depends on what phase the modification occurs in.

**Modifying via Codebase (Pre-deployment or Development):**
- You can add new `.json` schemas to the app's `schemas/` directory. On the next restart, the CMS will detect the new schemas and create the corresponding tables.
- Because `SetupApp` currently uses `CREATE TABLE IF NOT EXISTS`, altering existing columns directly via JSON files requires manual database migrations if the table already exists.

**Modifying via CMS (Post-deployment):**
- Once deployed, the app's schemas are saved in the internal `__schemas` table. 
- You can use the CMS admin interface to dynamically add new columns, modify pages, or adjust Handlebars templates. These modifications are persisted to the database and take effect immediately via the dynamic `squirrel` query builder and `raymond` templating engine.
- Note: UI modifications currently exist in the database and would need to be exported back to `.json` files if you want to bundle them into the persistent app source code. You can use the `cmd/export` utility for this.

---

## 7. Exporting App Modifications

Since UI modifications and dynamically created data live in the database (e.g., in the `__schemas` table and individual dynamic tables), you can use the built-in export utility to export these modifications back into JSON files. This allows you to bundle them into your source control as part of your application.

### Using the Export Utility

The export utility connects to the SQLite database, extracts all published schemas (Entity, Page, Menu, Query) as well as the row data for `EntitySchema` objects, and saves them to an output directory.

From your project directory, run:

```bash
go run cmd/export/main.go
```

By default, this will read from `formcms.db` and output to an `exports` directory:
- **Schemas**: Extracted to `exports/schemas/<SchemaType>/<SchemaName>.json`
- **Data**: Extracted to `exports/data/<EntityName>.json`

You can customize the database path and output directory using flags:
```bash
go run cmd/export/main.go --db=path/to/my-db.sqlite --out=./my-export-dir
```

---

## 8. Importing App Modifications

You can restore your exported app modifications (both schemas and data) into any database using the `cmd/import` utility. This is extremely useful for migrating modifications across environments, resetting test databases, or seeding production instances.

### Using the Import Utility

The import tool dynamically reads your exported `.json` schemas and data files, creates the necessary core tables (`__schemas`) and dynamic entity tables if they do not exist, and inserts all definitions and row data back into the target database.

From your project directory, run:

```bash
go run cmd/import/main.go
```

By default, it reads from the `./exports` directory and imports into `formcms.db`.

You can optionally specify the input folder and target database:
```bash
go run cmd/import/main.go --in=./my-export-dir --db=path/to/target-db.sqlite
```

**Idempotency & Preventing Duplicates:**
The import script is designed to be completely safe to run multiple times:
1. **Schemas:** Before inserting a schema into the `__schemas` table, the utility checks if a schema of the same type and name already exists. If it does, it skips it.
2. **Tables:** It uses `CREATE TABLE IF NOT EXISTS` when constructing physical SQL tables, safely ignoring tables that are already built.
3. **Data:** Exported records contain their original primary key (`id`). During import, the utility verifies if a record with that specific `id` already exists in the target table. Existing records are skipped, meaning running the import twice will **not** duplicate your exported data. (Note: If you manually author new data in the `.json` files without providing an `"id"`, the database will treat those as brand-new records and auto-assign them IDs, which could cause duplication if run multiple times).

**Note:** Export from a local SQLite development environment and import directly into a production PostgreSQL environment without modifying the JSON files, is supported.

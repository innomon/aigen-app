# AiGen CMS BizDef Architecture & Lifecycle

This document provides a comprehensive overview of the architecture, data structures, and lifecycle of BizDefs within the AiGen CMS ecosystem. It also serves as a guide on how to create, deploy, and modify these BizDefs.

## 1. Architecture Overview

AiGen CMS is a headless Content Management System written in Go (migrated from C#). It is built for extreme flexibility, using dynamic data modeling where database tables are created and modified on the fly based on JSON schemas.

### Core Technology Stack
- **Routing**: `net/http` + `chi` router
- **Database Access**: A unified `IPrimaryDao` abstraction supporting PostgreSQL and Google Cloud Firestore.
- **Dynamic Queries**: `Masterminds/squirrel` for query building, as table schemas are not known at compile time (preventing the use of standard ORMs).
- **GraphQL**: `graphql-go/graphql` for dynamic API endpoints based on schemas.
- **Templating**: `aymerick/raymond` (Handlebars template engine) for dynamic page rendering.

### What is a "BizDef"?
A "BizDef" in AiGen CMS is essentially a bundle of predefined entity schemas and optional test data. These BizDefs provide out-of-the-box functionality for specific domains (e.g., CRM, RBAC, ERPNext Accounting). Instead of writing complex migration scripts, a BizDef developer defines their data model in JSON, and the CMS engine automatically handles the underlying database structure and CRUD/GraphQL APIs.

## 2. Data Structure

BizDefs reside in the `bizdefs/` directory at the project root.

### Directory Layout
```text
bizdefs/
├── bizdefs.json                    # Registry of currently enabled BizDefs
├── crm/                         # Example BizDef: CRM
│   ├── data/
│   │   └── seed_data.json       # Seed data for the BizDef
│   └── schemas/                 # Entity schemas defining the BizDef's structure
│       ├── crm_lead.json
│       ├── crm_deal.json
│       └── ...
└── rbac/                        # Example BizDef: Role-Based Access Control
    ├── data/
    │   └── seed_data.json
    └── schemas/
        └── role.json
```

### Schemas (`schemas/*.json`)
A schema defines an entity (which maps to a database table). It describes attributes (columns), relationships (lookups, junctions, collections), and UI metadata.

### Seed Data (`data/seed_data.json`)
A JSON array specifying seed records to insert upon BizDef deployment. It supports reference linking (using `$Ref:<key>`) to handle relationships between newly inserted records.

---

## 3. BizDef Lifecycle

The lifecycle of a BizDef is managed by the core CMS initialization process (specifically inside `main.go` and `core/bizdefs/setup.go`).

1. **Discovery & Configuration**: 
   Upon startup, the server reads `bizdefs/bizdefs.json` to determine which BizDefs are enabled.
2. **Schema Setup (`SetupBizDef`)**:
   For each enabled BizDef, the CMS scans the `bizdefs/<bizdef_name>/schemas/` directory.
   - Parses each `.json` file into an `Entity` descriptor.
   - Registers the schema definition for dynamic API generation.
   - Saves the schema definition as a JSON record in the core `aigen_records` table, marking it as `Published`.
3. **Data Seeding (`SetupBizDefTestData`)**:
   After schemas are registered, the CMS reads `bizdefs/<bizdef_name>/data/seed_data.json`.
   - It checks if data already exists to prevent duplicate seeding.
   - Inserts the records, dynamically resolving any `$Ref` cross-references between records.

---

## 4. How to Create a BizDef

To create a new BizDef (e.g., `inventory`), follow these steps:

1. **Create the Directory Structure**:
   Create a new folder in the `bizdefs` directory:
   ```bash
   mkdir -p bizdefs/inventory/schemas
   mkdir -p bizdefs/inventory/data
   ```

2. **Define Entity Schemas**:
   Create JSON schema files for your entities inside `bizdefs/inventory/schemas/`. 
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
   Create `bizdefs/inventory/data/seed_data.json`:
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

## 5. How to Deploy a BizDef

Deploying a BizDef simply involves enabling it so the CMS picks it up on the next startup.

1. Open `bizdefs/bizdefs.json`.
2. Add your BizDef's directory name to the `enabled_bizdefs` array:
   ```json
   {
     "enabled_bizdefs": [
       "rbac",
       "crm",
       "inventory"
     ]
   }
   ```
3. Restart the AiGen CMS backend server. The system will automatically build the tables, register the schemas for GraphQL/REST APIs, and seed the initial data.

---

## 6. How to Modify a BizDef

Modifying a BizDef's structure or behavior depends on what phase the modification occurs in.

**Modifying via Codebase (Pre-deployment or Development):**
- You can add new `.json` schemas to the BizDef's `schemas/` directory. On the next restart, the CMS will detect the new schemas and register them.
- Because all data is stored in the single `aigen_records` JSON table, altering existing attributes directly via JSON files requires no physical database migrations; schema-on-read handles it natively.

**Modifying via CMS (Post-deployment):**
- Once deployed, the BizDef's schemas are saved in the internal `aigen_records` table. 
- You can use the CMS admin interface to dynamically add new columns, modify pages, or adjust Handlebars templates. These modifications are persisted to the database and take effect immediately via the dynamic `squirrel` query builder and `raymond` templating engine.
- Note: UI modifications currently exist in the database and would need to be exported back to `.json` files if you want to bundle them into the persistent BizDef source code. You can use the `cmd/export` utility for this.

**Modifying/Overriding via App Extensions (Downstream Customization):**
- Downstream projects can completely customize and override built-in BizDef schemas and default configurations without modifying the core codebase or maintaining a fork.
- The primary and recommended mechanism for downstream overrides is through **Applet Extensions** (JAR files).
- When an Applet Extension containing a `bizdef/` directory is placed in the extensions folder (configured via `app_extensions_dir` or the `FORMCMS_APP_EXTENSIONS_DIR` environment variable), the host system automatically mounts it on startup.
- The **Smart Schema Evolution Engine** dynamically compares the extension's schema definitions with existing definitions in the core database (`aigen_records`). If differences are detected, the system performs a safe, in-place, JIT schema upgrade of the database definitions. This JIT process is fully idempotent—if the schema structures match, no database write is performed.
- Through this mechanism, downstream projects can cleanly introduce, override, or evolve standard BizDefs using the unified extension system.

---

## 7. Exporting BizDef Modifications

Since UI modifications and dynamically created data live in the database (e.g., in the single `aigen_records` table), you can use the built-in export utility to export these modifications back into JSON files. This allows you to bundle them into your source control as part of your application.

### Using the Export Utility

The export utility connects to the database, extracts all published schemas (Entity, Page, Menu, Query) as well as the row data for entities, and saves them to an output directory.

From your project directory, run:

```bash
go run cmd/export/main.go
```

By default, this will output to an `exports` directory:
- **Schemas**: Extracted to `exports/schemas/<SchemaType>/<SchemaName>.json`
- **Data**: Extracted to `exports/data/<EntityName>.json`

You can customize the database connection and output directory using flags:
```bash
go run cmd/export/main.go --db="postgres://..." --out=./my-export-dir
```

---

## 8. Importing BizDef Modifications

You can restore your exported BizDef modifications (both schemas and data) into any database using the `cmd/import` utility. This is extremely useful for migrating modifications across environments, resetting test databases, or seeding production instances.

### Using the Import Utility

The import tool dynamically reads your exported `.json` schemas and data files, ensures the core `aigen_records` table exists, and inserts all definitions and row data back into the target database.

From your project directory, run:

```bash
go run cmd/import/main.go
```

By default, it reads from the `./exports` directory.

You can optionally specify the input folder and target database connection:
```bash
go run cmd/import/main.go --in=./my-export-dir --db="postgres://..."
```

**Idempotency & Preventing Duplicates:**
The import script is designed to be completely safe to run multiple times:
1. **Schemas:** Before inserting a schema into the `aigen_records` table, the utility checks if a schema of the same type and name already exists. If it does, it skips it.
2. **Data:** Exported records contain their original primary key (`id`). During import, the utility verifies if a record with that specific `id` already exists in the target table. Existing records are skipped, meaning running the import twice will **not** duplicate your exported data. (Note: If you manually author new data in the `.json` files without providing an `"id"`, the database will treat those as brand-new records and auto-assign them IDs, which could cause duplication if run multiple times).

**Note:** Exporting from a development environment and importing into a production PostgreSQL environment without modifying the JSON files is fully supported.

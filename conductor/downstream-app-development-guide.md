# Agent Guide: Downstream App Development

This document provides a comprehensive guide for LLM agents to develop "Downstream Apps" using the AIGenApp framework. By following these steps, you can create a fully functional, schema-driven application that integrates seamlessly with the core system's JSON-based persistence and GraphQL API.

## 1. Directory Structure

Every app must reside in the `apps/` directory and follow this structure:

```text
apps/{app_name}/
├── app_def.json         # Manifest and metadata
├── schemas/             # Entity JSON definitions
│   ├── {entity_1}.json
│   └── {entity_2}.json
├── docs/                # Extended business context (Markdown)
│   ├── {entity_1}.md
│   └── {entity_2}.md
├── data/                # Initial/Test data
│   └── test_data.json
└── migrations/          # (Optional) SQL DDL for relational mapping/documentation
    └── 001_initial.sql
```

## 2. Step-by-Step Development Workflow

### Step 1: Define the App Manifest (`app_def.json`)
The `app_def.json` file describes the app's purpose, roles, and entities.

```json
{
  "name": "my_app",
  "display_name": "My Custom App",
  "description": "Brief summary of the app.",
  "context": "Comprehensive description of the business domain and workflows.",
  "roles": ["Admin", "User"],
  "entities": {
    "my_entity": {
      "description": "Purpose of the entity.",
      "context_file": "docs/my_entity.md"
    }
  }
}
```

### Step 2: Define Entity Schemas (`schemas/*.json`)
Each entity is defined by a JSON schema. Use the following structure:

- **`Name`**: PascalCase name (e.g., `CRMLead`). Used in GraphQL and Lookups.
- **`DisplayName`**: Human-readable label.
- **`TableName`**: snake_case name used for storage namespaces.
- **`Attributes`**: Array of field definitions.

#### Attribute Properties:
- `Field`: The database field name.
- `Header`: UI column header.
- `DataType`: `String`, `Float`, `Int`, `Boolean`, `DateTime`, `Lookup`, `Collection`.
- `DisplayType`: `Text`, `Number`, `Checkbox`, `Date`, `Select`, `Image`, `RichText`.
- `Options`: 
    - For `Select`: Newline-separated list of options.
    - For `Lookup`: The `Name` of the target entity.
    - For `Collection`: `TargetEntityName|ForeignKeyName`.

### Step 3: Provide Business Context (`docs/*.md`)
Create Markdown files in the `docs/` folder for complex entities. Document lifecycles, relationships, and validation rules. These are automatically indexed for LLM discovery.

### Step 4: Add Test Data (`data/test_data.json`)
Initialize the app with sample data. Use the `$Ref:RefName` syntax to link related records.

```json
[
  {
    "Entity": "MyCategory",
    "Ref": "Cat_1",
    "Data": { "name": "Category A" }
  },
  {
    "Entity": "MyEntity",
    "Ref": "Ent_1",
    "Data": {
      "title": "Sample Record",
      "category": "$Ref:Cat_1"
    }
  }
]
```

### Step 5: (Optional) SQL Migrations & JSON Portability
The framework's primary mechanism for portability is JSON-based export/import.

- **JSON Portability**: All schemas and data are serialized to JSON. Use the core CLI tools (`cmd/export` and `cmd/import`) for system-wide migration and environment synchronization.
- **Relational Mapping (Migrations)**: While the framework uses a single-table JSON architecture (`aigen_records`), you can optionally provide standard `CREATE TABLE` SQL scripts in `migrations/`. These serve as documentation for external reporting (BI tools) or for mapping the JSON structure to a traditional relational schema in external systems.

### Step 6: Register the App
Add your app's folder name to the `enabled_apps` array in `apps/apps.json`.

```json
{
  "enabled_apps": [
    "rbac",
    "crm",
    "my_app"
  ]
}
```

## 3. Best Practices

1. **Naming Conventions**: Use `snake_case` for `app_name`, `TableName`, and `Field`. Use `PascalCase` for entity `Name`.
2. **Persistence**: Remember that all data is stored in `aigen_records`. Do NOT attempt to create tables via Go code. Use the Schema Service to register your JSON definitions.
3. **Relationships**: Favor `Lookup` for many-to-one and `Collection` for one-to-many relationships.
4. **Validation**: Document strict validation rules in the entity context files so that agents can enforce them at the application layer.
5. **Idempotency**: The `SetupAppTestData` function skips data insertion if the entity already contains records, ensuring safe restarts.

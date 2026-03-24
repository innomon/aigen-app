# Agent Guide: Downstream App Development

This document provides a comprehensive guide for LLM agents to develop, define, and enhance "Downstream Apps" using the AIGenApp framework. Following these guidelines ensures that your application is fully self-describing, allowing other agents (like `cms_agent` and `ui_agent`) to autonomously discover its purpose, roles, and entities.

## 1. Goal

Your objective when creating or enhancing an application is to ensure it is fully self-describing. When another agent uses the `cms_app_get` tool, it should receive enough context to autonomously interact with the app's entities, enforce its business rules, and construct appropriate UIs.

## 2. Directory Structure

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

## 3. Step-by-Step Development Workflow

### Step 1: Define the App Manifest (`app_def.json`)
Every application must have an `app_def.json` file located at its root. This manifest describes the app's purpose, roles, and entities.

#### Structure Requirements:
- **`name`**: The system name of the app (snake_case, matches the folder name).
- **`display_name`**: A human-readable title (e.g., "ERPNext Accounting").
- **`description`**: A concise, 1-2 sentence summary of what the app does.
- **`context`**: A comprehensive paragraph explaining the app's business domain, primary workflows, and how it integrates into the broader system.
- **`roles`**: A JSON array of string values representing the roles that interact with this app.
- **`entities`**: A JSON object mapping entity names (keys) to their definitions.
    - **`description`**: A brief summary of the entity's purpose.
    - **`context_file`**: (Optional but highly recommended) A relative path to a Markdown file containing deep context for the entity (e.g., `docs/{entity_name}.md`).

#### Example `app_def.json`:
```json
{
  "name": "crm",
  "display_name": "Customer Relationship Management",
  "description": "Manages leads, deals, and customer interactions.",
  "context": "The CRM app tracks the entire sales lifecycle from lead acquisition to deal closure. It models organizations, contacts, and tasks associated with deals.",
  "roles": ["Sales Manager", "Sales Representative"],
  "entities": {
    "crm_lead": {
      "description": "A potential customer who has shown interest but is not yet qualified.",
      "context_file": "docs/crm_lead.md"
    },
    "crm_deal": {
      "description": "A qualified opportunity currently in the sales pipeline.",
      "context_file": "docs/crm_deal.md"
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
- `Field`: The database field name (snake_case).
- `Header`: UI column header.
- `DataType`: `String`, `Float`, `Int`, `Boolean`, `DateTime`, `Lookup`, `Collection`.
- `DisplayType`: `Text`, `Number`, `Checkbox`, `Date`, `Select`, `Image`, `RichText`.
- `Options`: 
    - For `Select`: Newline-separated list of options.
    - For `Lookup`: The `Name` of the target entity.
    - For `Collection`: `TargetEntityName|ForeignKeyName`.

### Step 3: Provide Business Context (`docs/*.md`)
When an entity requires more explanation than a simple description, you must create a Markdown file in the `docs/` folder and link it via the `context_file` property in `app_def.json`.

#### What to Include in a Context File:
1. **Business Purpose**: Why does this entity exist in the domain?
2. **Relationships**: How does this entity relate to other entities? (e.g., "A `crm_deal` must be linked to a `crm_organization`").
3. **State Transitions / Lifecycle**: If the entity has statuses, explain the flow (e.g., "A Lead moves from 'Open' to 'Contacted' to 'Qualified' or 'Lost'").
4. **Key Fields & Validation Rules**: Note any fields that have specific formatting or business constraints (e.g., "The `amount` field must be strictly positive").
5. **Role Permissions**: Specify if certain roles have restricted access to this entity (e.g., "Only Sales Managers can delete deals").

#### Example `docs/crm_lead.md`:
```markdown
# CRM Lead Context

A `crm_lead` represents an unqualified prospect in the CRM pipeline. 

## Lifecycle
Leads are typically generated from marketing campaigns or manual entry. Their status is tracked via the `status` field, which references the `crm_lead_status` entity. 
Once a lead is 'Qualified', it should be converted into a `crm_organization` and a corresponding `crm_deal`.

## Relationships
- **Lead Status**: Must map to a valid `crm_lead_status`.
- **Assigned To**: Often linked to a user in the system (Sales Rep).

## Constraints
- An email address or phone number is strictly required for contact purposes.
```

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

## 4. Best Practices

1. **Analyze First**: When tasked with creating a new app or documenting an existing one, use `codebase_investigator` to review schemas and logic.
2. **Naming Conventions**: Use `snake_case` for `app_name`, `TableName`, and `Field`. Use `PascalCase` for entity `Name`.
3. **Persistence**: Remember that all data is stored in `aigen_records`. Do NOT attempt to create tables via Go code. Use the Schema Service to register your JSON definitions.
4. **Relationships**: Favor `Lookup` for many-to-one and `Collection` for one-to-many relationships.
5. **Validation**: Document strict validation rules in the entity context files so that agents can enforce them at the application layer.
6. **Idempotency**: The `SetupAppTestData` function skips data insertion if the entity already contains records, ensuring safe restarts.
7. **Verification**: The resulting structure should enable the `cms_app_get` tool to automatically aggregate the `app_def.json` and all linked Markdown files into a single context payload.

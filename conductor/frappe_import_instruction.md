# Agent Instruction: Importing Frappe DocTypes to AiGen CMS

This document serves as the Standard Operating Procedure (SOP) and execution plan for any AI coding agent tasked with importing a DocType from a Frappe/ERPNext application into the `AiGen CMS` framework.

## 🎯 Core Objectives & Rules

1. **No Hardcoded Go Descriptors:** Schema definitions MUST strictly be created as `.json` files inside the target module's `schemas/` directory.
2. **Indian Locale Mandate:** All mock/test data generated MUST use the Indian locale (e.g., Currency: INR `₹`, Example Companies: Tata Motors, Reliance, Indian cities/addresses, standard Indian financial years like starting April 1st).
3. **JSON Test Data:** Test data MUST be created as `.json` files in the module's `data/` directory, utilizing the `$Ref:Key` referencing mechanism to link relational entities.
4. **SQL Migrations:** Alongside JSON schemas, migration SQL scripts (DDL/DML) MUST be generated for users who need to deploy to bare-metal databases without relying solely on application-level DDL generation.
5. **Temporary Workspace:** The target Frappe/ERPNext application MUST be cloned into a temporary directory for exploration to avoid cluttering the main workspace.

---

## 🗺️ Execution Plan

### Phase 1: Repository Exploration & Preparation
1. **Clone the Target App:** 
   Clone the desired Frappe application from GitHub into the system's temporary directory.
   ```bash
   mkdir -p .gemini/tmp/temp_repos
   cd .gemini/tmp/temp_repos
   git clone --depth 1 https://github.com/frappe/<app_name>.git
   ```
2. **Locate DocTypes:**
   Navigate the cloned repository to find the target DocType definitions, typically located at `<app_name>/<app_name>/<module_name>/doctype/<doctype_name>/<doctype_name>.json`.
3. **Analyze the DocType:**
   Read the Frappe JSON to understand the fields, data types, mandatory constraints, and relational links (`options` targeting other DocTypes).

### Phase 2: Schema Translation (JSON Creation)
Create a new JSON file in `apps/<module_name>/schemas/<doctype_snake_case>.json`.

**Translation Rules:**
* **DocType -> Entity:** Map the DocType name directly to the `Entity` name (e.g., "Journal Entry" -> "JournalEntry").
* **Fields -> Attributes:** Map Frappe fields to FormCMS `Attributes` array.
* **Type Mapping:**
  * Frappe `Data`, `Select`, `Small Text` ➔ FormCMS DataType: `String`
  * Frappe `Text`, `Text Editor` ➔ FormCMS DataType: `String` or `Text`, DisplayType: `Textarea` or `Editor`
  * Frappe `Int` ➔ FormCMS DataType: `Int`
  * Frappe `Float`, `Currency`, `Percent` ➔ FormCMS DataType: `Float`, DisplayType: `Number`
  * Frappe `Check` ➔ FormCMS DataType: `Boolean`
  * Frappe `Date`, `Datetime` ➔ FormCMS DataType: `Datetime`, DisplayType: `Date` or `Datetime`
  * Frappe `Link` ➔ FormCMS DataType: `Lookup`, Options: `<TargetEntityName>`
  * Frappe `Table` ➔ FormCMS DataType: `Collection`, Options: `<ChildEntityName>|<ParentLinkField>`

**Example Schema (`apps/<module_name>/schemas/currency.json`):**
```json
{
  "Name": "Currency",
  "DisplayName": "Currency",
  "TableName": "currency",
  "LabelAttributeName": "currency_name",
  "PrimaryKey": "id",
  "Attributes": [
    { "Field": "currency_name", "Header": "Currency", "DataType": "String", "DisplayType": "Text", "InList": true, "InDetail": true }
  ]
}
```

### Phase 3: SQL Migration Generation
Generate standard SQL migration files (e.g., `apps/<module_name>/migrations/001_initial_schema.sql`). 
1. Write the `CREATE TABLE` statements equivalent to the JSON schemas for PostgreSQL and SQLite.
2. Include system columns: `id`, `created_at`, `updated_at`, `deleted`.
3. Generate `INSERT INTO` statements containing baseline production data (if applicable, e.g., default Indian states, tax categories).

### Phase 4: Test Data Generation (Indian Locale)
Create a JSON file in `apps/<module_name>/data/test_data.json`.
1. **Locale Requirements:** Use INR (`₹`), Indian addresses (Mumbai, Delhi), Indian tax logic (GST, SGST, CGST), and standard Indian corporate names.
2. **Relational Data Mapping:** Use the application's built-in reference resolver to manage foreign keys in test data. Prefix referenced keys with `$Ref:`.

**Example Test Data:**
```json
[
  {
    "Entity": "Currency",
    "Ref": "Currency_INR",
    "Data": {
      "currency_name": "INR",
      "symbol": "₹"
    }
  },
  {
    "Entity": "Company",
    "Ref": "Company_TATA",
    "Data": {
      "company_name": "Tata Motors India",
      "default_currency": "$Ref:Currency_INR"
    }
  }
]
```

### Phase 5: Implementation & Wiring (Configuration)
The `AiGen CMS` framework utilizes a fully configuration-driven initialization system for apps.
1. Place the generated application module inside `apps/<module_name>`.
2. Add the `<module_name>` to the `enabled_apps` array inside `apps/apps.json` so the core system knows to load the schemas, run migrations (if applicable), and process the test data during startup. Do NOT modify the Go source code.

**Example Configuration (`apps/apps.json`):**
```json
{
  "enabled_apps": [
    "erpnext_accounting",
    "crm",
    "<module_name>"
  ]
}
```

---

## 🚦 Verification Steps for Agents
1. Have I created **only** `.json` files for the DocType representations?
2. Are all monetary values, companies, and date structures conforming to the **Indian locale**?
3. Did I generate a corresponding `.sql` migration script?
4. Is the Frappe source code appropriately isolated in `.gemini/tmp/temp_repos/`?
5. Did I update `apps/apps.json` to enable the new module?
6. Does the test server boot up and seed the configuration data without errors?
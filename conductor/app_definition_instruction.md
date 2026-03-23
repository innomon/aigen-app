# Agent Instruction: App Capability Discovery & Definition

This document provides instructions for LLM agents on how to create, format, and enhance App Definitions (`app_def.json`) and Entity Context files within the AiGen CMS. Following these guidelines ensures that other agents (like `cms_agent` and `ui_agent`) can dynamically discover and understand the purpose, roles, and entities of installed applications.

## 1. Goal

Your objective when enhancing or creating an application is to ensure it is fully self-describing. When another agent uses the `cms_app_get` tool, it should receive enough context to autonomously interact with the app's entities, enforce its business rules, and construct appropriate UIs.

## 2. The `app_def.json` Manifest

Every application must have an `app_def.json` file located at its root (e.g., `apps/{app_name}/app_def.json`).

### Structure Requirements:
- **`name`**: The system name of the app (snake_case, matches the folder name).
- **`display_name`**: A human-readable title (e.g., "ERPNext Accounting").
- **`description`**: A concise, 1-2 sentence summary of what the app does.
- **`context`**: A comprehensive paragraph explaining the app's business domain, primary workflows, and how it integrates into the broader system.
- **`roles`**: A JSON array of string values representing the roles that interact with this app.
- **`entities`**: A JSON object mapping entity names (keys) to their definitions.

### Entity Definition Requirements:
For each entity in the `entities` object, provide:
- **`description`**: A brief summary of the entity's purpose.
- **`context_file`**: (Optional but highly recommended) A relative path to a Markdown file containing deep context for the entity (e.g., `docs/{entity_name}.md`).

### Example `app_def.json`:
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

## 3. Entity Context Files (Markdown)

When an entity requires more explanation than a simple description, you must create a Markdown file and link it via the `context_file` property in `app_def.json`. These files should be placed in the app's `docs/` directory.

### What to Include in a Context File:
1. **Business Purpose**: Why does this entity exist in the domain?
2. **Relationships**: How does this entity relate to other entities? (e.g., "A `crm_deal` must be linked to a `crm_organization`").
3. **State Transitions / Lifecycle**: If the entity has statuses, explain the flow (e.g., "A Lead moves from 'Open' to 'Contacted' to 'Qualified' or 'Lost'").
4. **Key Fields & Validation Rules**: Note any fields that have specific formatting or business constraints (e.g., "The `amount` field must be strictly positive").
5. **Role Permissions**: Specify if certain roles have restricted access to this entity (e.g., "Only Sales Managers can delete deals").

### Example `docs/crm_lead.md`:
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

## 4. Workflow for Creating/Enhancing an App

When tasked with creating a new app or documenting an existing one, follow these steps:
1. **Analyze**: Use `codebase_investigator` to review the schemas and logic of the application.
2. **Create `app_def.json`**: Generate the manifest in the app's root folder.
3. **Write Context Files**: Identify complex entities and write corresponding Markdown files in the `docs/` directory.
4. **Register**: Ensure the app's folder name is added to the `"enabled_apps"` array in `apps/apps.json`.
5. **Verify**: The resulting structure should enable the `cms_app_get` tool to automatically aggregate the `app_def.json` and all linked Markdown files into a single context payload.
# Specification: BizDef Evolution Service

## Goal
Implement a robust, declarative system for evolving BizDef entity schemas in AIGenApp's single-table JSON architecture. This system must support Just-In-Time (JIT) upgrades during reads/writes and asynchronous batch upgrades via background workers, while maintaining data integrity through Optimistic Concurrency Control (OCC).

## Core Concepts

### 1. Evolution Manifests
- **`evolution.json`**: A machine-readable timeline of schema changes.
    - Structure: `Entity -> Version -> {Date, Description, Actions[]}`.
    - Actions: `rename`, `add`, `drop`, `transform` (placeholder for complex logic).
- **`evolution.md`**: A human and AI-readable history of the business reasoning behind schema changes.

### 2. EvolutionService
- A new core service in `core/services/`.
- **Responsibilities**:
    - Parsing and caching `evolution.json`.
    - Executing transformation chains on JSON records.
    - Coordinating JIT upgrades in the `EntityService`.
    - Managing background "scrubber" routines for batch migrations.

### 3. Data Integrity & Concurrency
- **Metadata Enhancement**: Add `schema_version` and `schema_version_date` to the record metadata.
- **Optimistic Concurrency Control (OCC)**: Use a `revision` field in metadata to prevent lost updates during background migrations.

## Technical Requirements

### Schema Definition (evolution.json)
```json
{
  "entity_name": {
    "v2": {
      "date": "2024-05-19T14:30:00Z",
      "description": "Short description",
      "actions": [
        { "action": "rename", "from": "old", "to": "new" },
        { "action": "add", "field": "f1", "default": "val" },
        { "action": "drop", "field": "f2" }
      ]
    }
  }
}
```

### JIT Upgrade Logic
1. `EntityService.Get` or `EntityService.Update` is called.
2. `EvolutionService.EvolveRecord` is invoked with the record's current JSON and Metadata.
3. If `MetaData.schema_version` is older than the latest in `evolution.json`, the transformation chain is applied.
4. The record is returned in its modernized state.

### Background Scrubber (via MCP)
- A tool `cms_bizdef_evolve` triggers a background goroutine.
- The routine fetches records in batches using the DAO.
- It applies `EvolutionService.EvolveRecord`.
- It saves back to the DAO using a conditional update: `WHERE key = ? AND revision = ?`.

# Implementation Plan: BizDef Evolution Service

## Phase 1: Foundation & Data Models
- [x] Update `MetaData` struct in `core/descriptors/` to include `SchemaVersion`, `SchemaVersionDate`, and `Revision`.
- [x] Update `IPrimaryDao` implementations (Postgres, Memory) to support conditional updates based on `Revision`.
- [x] Define the `evolution.json` structure in Go types within a new `core/descriptors/evolution.go`.

## Phase 2: The EvolutionEngine & Service
- [x] Create `core/services/evolution_service.go`.
- [x] Implement the `EvolutionEngine` logic: a transformation chain that mutates `map[string]interface{}` based on actions.
- [x] Implement `LoadEvolutionManifests` in `SchemaService` to scan `bizdefs/*/evolution.json`.
- [x] Unit test the transformation logic with various edge cases (missing fields, nested renames, etc.).

## Phase 3: EntityService Integration (JIT)
- [x] Modify `EntityService.Get` to check and apply transformations in-memory.
- [x] Modify `EntityService.Update` to ensure data is "upgraded" before being saved to the database.
- [x] Add integration tests for JIT upgrades.

## Phase 4: Asynchronous Batch Migration
- [x] Implement the "Scrubber" logic in `EvolutionService`.
- [x] Implement Optimistic Concurrency Control handling (retry/skip on revision mismatch).
- [x] Expose the scrubber via a new MCP tool in `core/services/cms_tools.go`.
- [x] Implement progress tracking for long-running migrations.

## Phase 5: Documentation & BizDef Alignment
- [x] Create `evolution.json` and `evolution.md` for existing BizDefs (e.g., `crm`, `rbac`) as samples.
- [x] Update `ARCHITECTURE.md` to include the Schema Evolution section.
- [x] Update the `Downstream BizDef Development Guide` to include instructions on managing evolution.

## Phase 6: Verification
- [x] Full integration test: Create v1 records, update schema to v2, verify JIT upgrade on read.
- [x] Full integration test: Trigger batch migration via MCP and verify all records are consistent.
- [x] Verify AI agents can correctly read and explain `evolution.md`.

# Implementation Plan: BizDef Evolution Service

## Phase 1: Foundation & Data Models
- [ ] Update `MetaData` struct in `core/descriptors/` to include `SchemaVersion`, `SchemaVersionDate`, and `Revision`.
- [ ] Update `IPrimaryDao` implementations (Postgres, Memory) to support conditional updates based on `Revision`.
- [ ] Define the `evolution.json` structure in Go types within a new `core/descriptors/evolution.go`.

## Phase 2: The EvolutionEngine & Service
- [ ] Create `core/services/evolution_service.go`.
- [ ] Implement the `EvolutionEngine` logic: a transformation chain that mutates `map[string]interface{}` based on actions.
- [ ] Implement `LoadEvolutionManifests` in `SchemaService` to scan `bizdefs/*/evolution.json`.
- [ ] Unit test the transformation logic with various edge cases (missing fields, nested renames, etc.).

## Phase 3: EntityService Integration (JIT)
- [ ] Modify `EntityService.Get` to check and apply transformations in-memory.
- [ ] Modify `EntityService.Update` to ensure data is "upgraded" before being saved to the database.
- [ ] Add integration tests for JIT upgrades.

## Phase 4: Asynchronous Batch Migration
- [ ] Implement the "Scrubber" logic in `EvolutionService`.
- [ ] Implement Optimistic Concurrency Control handling (retry/skip on revision mismatch).
- [ ] Expose the scrubber via a new MCP tool in `core/services/cms_tools.go`.
- [ ] Implement progress tracking for long-running migrations.

## Phase 5: Documentation & BizDef Alignment
- [ ] Create `evolution.json` and `evolution.md` for existing BizDefs (e.g., `crm`, `rbac`) as samples.
- [ ] Update `ARCHITECTURE.md` to include the Schema Evolution section.
- [ ] Update the `Downstream BizDef Development Guide` to include instructions on managing evolution.

## Phase 6: Verification
- [ ] Full integration test: Create v1 records, update schema to v2, verify JIT upgrade on read.
- [ ] Full integration test: Trigger batch migration via MCP and verify all records are consistent.
- [ ] Verify AI agents can correctly read and explain `evolution.md`.

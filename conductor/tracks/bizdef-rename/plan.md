# Implementation Plan: Rename App to BizDef

This plan outlines the steps to transition the "App" concept to "BizDef".

## Phase 1: Documentation & Discovery
- [x] Update `ARCHITECTURE.md` with new terminology.
- [x] Update `README.md`.
- [x] Update `conductor/docs/downstream-app-development-guide.md` (and rename to `downstream-bizdef-development-guide.md`).
- [x] Search the codebase for all occurrences of `App`, `app`, `apps` to identify candidates for renaming.

## Phase 2: File System Changes
- [x] Rename `apps/` directory to `bizdefs/`.
- [x] Update all `app_def.json` files to `bizdef.json`.
- [x] Update references to these paths in the code (e.g., path constants, directory scanning logic).

## Phase 3: Code Refactoring (Backend)
- [x] **Data Structures**: Rename `AppManifest`, `AppDescriptor`, etc. (Refactored `core/bizdefs/setup.go` types).
- [x] **Services**: Rename `AppService` and related interfaces/implementations.
- [x] **APIs**: Rename `AppApi` and internal routing logic (Updated `core/services/cms_tools.go` and `core/agentic/agents/router_agent.go`).
- [x] **Variables**: Systematic rename of local variables and struct fields.
- [x] **Tests**: Update test names and mock data paths.

## Phase 4: Persistence & Data
- [x] Ensure the `Namespace` logic in `IPrimaryDao` correctly maps to the folders in `bizdefs/`.
- [x] Update any hardcoded "app" namespace strings in the core logic to "bizdef" if applicable. (Updated `aigen.app` -> `aigen.bizdef`).

## Phase 5: Verification
- [x] Run all tests (`go test ./...`).
- [x] Manually verify that BizDefs are still discovered and loaded correctly on startup.
- [x] Verify that GraphQL and Entity CRUD operations still function for registered BizDefs.

# Specification: Rename App to BizDef

## Background
The term "App" in AIGenApp currently refers to a logical container for business logic, schemas, and data. However, in the context of a headless CMS and dynamic application framework, "App" can be ambiguous (often implying a standalone executable or a frontend application). 

"BizDef" (Business Definition) has been selected as an alternative that better reflects the system's "Schema-on-Read" architecture, where these containers are declarative specifications of business domains.

## Objective
Systematically rename all occurrences of "App" (when referring to the logical business container) to "BizDef" across the codebase, documentation, and file structure.

## Scope
- **Documentation**: Update `ARCHITECTURE.md`, `README.md`, and development guides.
- **File System**:
    - Rename `apps/` directory to `bizdefs/`.
    - Rename `app_def.json` manifest files to `bizdef.json`.
- **Backend Code (Go)**:
    - Rename structs (e.g., `AppManifest` -> `BizDefManifest`).
    - Rename services (e.g., `AppService` -> `BizDefService`).
    - Rename variables, function names, and comments.
- **Database/Persistence**: Update namespacing logic if "App" is used as a hardcoded string key (keeping in mind backward compatibility for existing data).
- **API**: (Optional/Staged) Update API endpoints from `/api/app/...` to `/api/bizdef/...`.

## Constraints
- **Idiomatic Go**: Maintain consistency with Go naming conventions during refactoring.
- **Minimal Disruption**: The refactoring should be surgical and not introduce regression in core logic.

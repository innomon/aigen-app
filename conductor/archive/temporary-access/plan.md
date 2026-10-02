# Implementation Plan: Temporary File Access

This plan outlines the steps to implement the temporary file access system with TTL-based expiry and MCP management.

## Phase 1: Configuration and Infrastructure

### 1.1 Update Configuration Schema
- Add a `TemporaryAccessConfig` struct to `framework/config.go`.
- Update the global `Config` struct to include a list of these configurations.
- Update `config.yaml.sample` with example settings.

### 1.2 Extend Filestore Capability (if needed)
- Ensure `infrastructure/filestore/interface.go` and its implementations (local, S3, GCS) provide access to file creation/modification times.
- Verify that `local.go` and `postgres.go` (if used for file storage) support retrieving this metadata.

## Phase 2: Core Expiry Logic

### 2.1 Implement Expiry Service
- Create `core/services/temp_access_service.go`.
- Function: `IsExpired(path string, filename string) (bool, error)`.
    - Retrieve config for `path`.
    - Get file metadata from `IFileStore`.
    - Compare `CreatedTime + TTL` with `Now`.

## Phase 3: Public Retrieval API

### 3.1 Implement Public GET Handler
- Create a new API handler in `core/api/static_api.go` or a new `temp_access_api.go`.
- Register routes for each configured temporary path in `main.go` or the relevant router setup.
- The handler should:
    1. Call `temp_access_service.IsExpired`.
    2. If expired, return `410 Gone`.
    3. If not expired, stream the file from `IFileStore`.

## Phase 4: Management API (MCP)

### 4.1 Implement MCP Management Endpoints
- Add endpoints to `core/api/mcp_api.go`:
    - `POST /mcp/temp-access/:path/upload`: Uploads a file, requires JWT and path-specific role.
    - `DELETE /mcp/temp-access/:path/:filename`: Deletes a file.
- Implement RBAC check using `core/services/permission_service.go` or a custom check against the `TemporaryAccessConfig`.

## Phase 5: Testing and Validation

### 5.1 Unit Tests
- Test `IsExpired` logic with mocked file metadata.
- Test configuration loading.

### 5.2 Integration Tests
- Upload a file via MCP API.
- Verify public access works immediately.
- Wait for TTL (or simulate/mock time) and verify `410 Gone`.
- Verify RBAC prevents unauthorized management.

## Phase 6: Documentation
- Update `README.md` or create a new doc in `docs/` explaining how to configure and use temporary access.

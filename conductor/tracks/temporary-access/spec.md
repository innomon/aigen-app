# Specification: Temporary File Access

## Overview
Implement a system for generating and serving short-lived, unauthenticated URLs for file access. This is intended for scenarios like one-time document downloads, temporary media sharing, or ephemeral assets that should not require full session authentication for the recipient but must expire quickly.

## Requirements

### 1. Configuration
The system must support multiple temporary access configurations (buckets/directories).
Each configuration includes:
- `Path`: The URL path prefix (e.g., `tmp`, `reports`).
- `TTL_SEC`: Time-to-live in seconds (default: 300).
- `Role`: The RBAC role required to manage (upload/delete) files in this path.

Example configuration structure:
```yaml
temporary_access:
  - path: "tmp"
    ttl_sec: 300
    role: "editor"
  - path: "secure-reports"
    ttl_sec: 3600
    role: "admin"
```

### 2. Storage
- All files are stored using the existing `infrastructure/filestore/provider.go` abstraction.
- The creation time of the file in the filestore (or metadata) is used to calculate expiry.

### 3. Public Access (Unauthenticated)
- Endpoint: `GET /<TMP_DIR>/<filename>`
- Logic:
    1. Identify the configuration based on `<TMP_DIR>`.
    2. Check if the file exists.
    3. Retrieve the file's creation time.
    4. If `now > creation_time + TTL_SEC`, return `410 Gone` or `404 Not Found` with an expiry message.
    5. If valid, serve the file content.

### 4. Management API (Authenticated)
- The lifecycle of these files is managed via the **MCP Server API**.
- Authentication: JWT required.
- Authorization: User must have the `Role` defined in the configuration for the target path.
- Operations:
    - Upload file to a temporary path.
    - Delete file.
    - (Optional) List files in a path.

## Security Considerations
- **URL Guessability**: Filenames should be sufficiently random (e.g., UUIDs) to prevent enumeration, as the URLs are unauthenticated.
- **Expiry Enforcement**: Strict enforcement of TTL to ensure data is not accessible after the intended period.
- **RBAC**: Management operations must be strictly guarded by JWT and role-based checks.

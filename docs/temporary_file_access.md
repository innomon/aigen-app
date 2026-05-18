# Temporary File Access Guide

This guide explains how to configure and use the Temporary File Access system in AIGenApp.

## Overview
The Temporary File Access system allows you to create short-lived, unauthenticated URLs for file access. This is ideal for one-time document downloads, temporary media sharing, or ephemeral assets.

## Configuration

Temporary access is configured in `config.yaml` under the `temporary_access` key. You can define multiple paths, each with its own Time-To-Live (TTL) and RBAC role required for management.

```yaml
temporary_access:
  - path: "tmp"
    ttl_sec: 300
    role: "admin"
  - path: "reports"
    ttl_sec: 3600
    role: "editor"
```

- **path**: The URL path prefix (e.g., `/tmp/my-file.txt`).
- **ttl_sec**: How long the file remains accessible after creation (in seconds).
- **role**: The RBAC role required to upload, delete, or trigger cleanup for files in this path.

## Public Access (Unauthenticated)

Files stored in temporary paths can be accessed via a simple GET request:

`GET /{path}/{filename}`

Example: `https://api.example.com/tmp/report-123.pdf`

If the file has exceeded its `ttl_sec` since creation, the server will return a `410 Gone` error.

## Management API (Authenticated)

Management operations are handled via the MCP API and require JWT authentication. The user must have the role specified in the configuration for the target path.

### 1. Upload a Temporary File
`POST /api/mcp/temp-access/{path}/upload`

- **Auth**: Bearer Token (JWT)
- **Body**: `multipart/form-data` with a `file` field.
- **Success**: `201 Created`

### 2. Delete a Temporary File
`DELETE /api/mcp/temp-access/{path}/{filename}`

- **Auth**: Bearer Token (JWT)
- **Success**: `200 OK`

### 3. Cleanup Expired Files
`POST /api/mcp/temp-access/{path}/cleanup`

- **Auth**: Bearer Token (JWT)
- **Description**: Triggers an optimized batch deletion of all files in the path that have exceeded their TTL.
- **Success**: `200 OK` (returns count of deleted files)

## Optimized Garbage Collection

The system uses optimized cleanup strategies for different storage providers:
- **Postgres**: Executes a single `DELETE` query with date arithmetic.
- **S3**: Uses batch `DeleteObjects` based on metadata from the list operation.
- **GCS**: Iterates through object attributes to avoid redundant metadata calls.
- **Local**: Performs a single filesystem walk.

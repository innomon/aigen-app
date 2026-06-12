# Specification: SurrealDB Infrastructure Support

## Overview
This specification details the design and requirements for adding SurrealDB database support to both `relationdbdao` (for persistent entity storage) and `filestore` (for asset storage).

## Requirements

### 1. Connection URL Format
The database connections will be configured via a standard URI string.
* Scheme: `surreal://` or `surrealdb://` (e.g. `surreal://root:root@localhost:8000/my_namespace/my_database`)
* Host/Port: Hostname and port of the SurrealDB server.
* Username/Password: Handled via standard URI user info.
* Path: `/namespace/database` (defaults to `/aigen/aigen` if not provided).

### 2. RelationDBDAO Integration
`SurrealDBDao` will implement the `IPrimaryDao` interface.
* **Storage Pattern**: Uses a single SurrealDB table `aigen_records`.
* **Record ID Mapping**: Document ID in SurrealDB will be structured as `aigen_records:⟨namespace⟩___⟨key⟩`.
* **Operations**:
  * `Save`: Upserts the record. Increments the revision metadata field.
  * `SaveConditional`: Performs an atomic update conditioned on the document's current revision (`metadata.revision = expectedRevision`).
  * `Get`: Selects a single document by matching namespace and key.
  * `Delete`: Deletes a document by matching namespace and key.
  * `List`: Fetches records filtered by namespace and custom query filters, supporting sorting and offset/limit pagination using SurrealQL (`LIMIT ... START ...`).
  * `Close`: Closes the connection to SurrealDB.

### 3. FileStore Integration
`SurrealFileStore` will implement the `IFileStore` interface.
* **Storage Pattern**: Uses a SurrealDB table/collection `filesys`.
* **Document Schema**:
  ```go
  type SurrealFileDoc struct {
      Path     string    `json:"path"`
      Metadata string    `json:"metadata"`
      Content  []byte    `json:"content"`
      Tmstamp  time.Time `json:"tmstamp"`
  }
  ```
* **Operations**:
  * `Upload`: Writes/replaces a file document.
  * `GetMetadata`: Extracts file size, modified times, and other properties.
  * `GetUrl`: Generates public-facing URL path `/api/files/surreal/⟨path⟩`.
  * `Download`: Streams the binary file content.
  * `Delete`: Deletes the file by path.
  * `DeleteByPrefix`: Deletes all files matching a path prefix using wildcard matches (`path LIKE prefix%`).
  * `List`: Lists all files matching a prefix.
  * `PurgeExpired`: Deletes expired files matching a prefix older than the TTL.

## Verification
- Compilation must succeed.
- Implementation must pass tests validating basic CRUD for both DAO and FileStore.

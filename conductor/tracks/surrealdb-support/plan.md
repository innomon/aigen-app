# Implementation Plan: SurrealDB Infrastructure Support

This plan outlines the steps to implement and test SurrealDB support for both `relationdbdao` and `filestore`.

## Phase 1: RelationDBDAO Implementation
- Create `/infrastructure/relationdbdao/surreal.go` implementing `IPrimaryDao`.
- Implement connection parsing (`surreal://` / `surrealdb://`).
- Implement basic operations (`Save`, `SaveConditional`, `Get`, `Delete`).
- Implement the query engine inside `List` supporting dynamic filters, sorting, and pagination (using `LIMIT` and `START`).
- Update `/infrastructure/relationdbdao/provider.go` to recognize `surreal://` / `surrealdb://` URLs and instantiate `SurrealDBDao`.

## Phase 2: FileStore Implementation
- Create `/infrastructure/filestore/surreal.go` implementing `IFileStore`.
- Update `/infrastructure/filestore/provider.go` to include the `surrealdb` driver in `Config` and handle it in `CreateFileStore`.
- Implement `Upload`, `Download`, `GetMetadata`, `Delete`, `DeleteByPrefix`, `List`, and `PurgeExpired` in `SurrealFileStore`.

## Phase 3: Validation and Testing
- Create unit/integration tests for `SurrealDBDao` and `SurrealFileStore`.
- Verify compilation succeeds.

# Implementation Plan: SQLite P2P Support for DAO & Filestore

This plan outlines the steps to integrate `sqlite-p2p` into `AIGenApp` for both the DAO layer (`IPrimaryDao`) and the Filestore layer (`IFileStore`), setting it as a default option.

## Phase 1: Module Setup & Dependency Wiring
- [x] Add local module dependency or replace directive in `go.mod` referencing `/home/innomon/B204-zone/owly-sewa/sqlite-p2p`.
- [x] Verify module dependencies build and vendor properly.

## Phase 2: DAO Layer Integration (`IPrimaryDao`)
- [x] Create `/infrastructure/relationdbdao/sqlite_p2p.go` implementing `IPrimaryDao`.
- [x] Handle `sqlite-p2p://` and `p2p://` URI schemes in `/infrastructure/relationdbdao/provider.go`.
- [x] Set `sqlite-p2p` as default database provider option when URI is unspecified or set to default.

## Phase 3: FileStore Layer Integration (`IFileStore`)
- [x] Create `/infrastructure/filestore/sqlite_p2p.go` implementing `IFileStore`.
- [x] Update `/infrastructure/filestore/provider.go` to support `sqlite-p2p` driver as default.

## Phase 4: Validation & Unit Tests
- [x] Add unit tests for `SqliteP2PDao` and `SqliteP2PFileStore`.
- [x] Run full `go test ./...` test suite to verify stability.


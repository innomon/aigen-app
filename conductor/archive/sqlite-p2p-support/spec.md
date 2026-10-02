# Specification: SQLite P2P Support for DAO & Filestore

## Overview
This specification details the design and requirements for integrating `sqlite-p2p` (locally available at `/home/innomon/B204-zone/owly-sewa/sqlite-p2p`) as a default option for both the `IPrimaryDao` database layer and `IFileStore` asset storage layer in `AIGenApp`.

## Requirements

### 1. Local Module Resolution
* Wire the `sqlite-p2p` Go module via a local `replace` directive in `go.mod` (or direct path import reference as appropriate) pointing to `/home/innomon/B204-zone/owly-sewa/sqlite-p2p`.

### 2. URL Scheme & Default Configuration
* Support standard URI schemes `sqlite-p2p://` and `p2p://` for instantiating the storage engines.
* Make `sqlite-p2p` the default driver option when database URL or filestore driver configuration is unset or specified as `sqlite-p2p`.

### 3. RelationDBDAO Integration (`SqliteP2PDao`)
* Implement `IPrimaryDao` using `sqlite-p2p` connection / peer instance.
* Store data using single-table JSON persistence (`aigen_records`) compatible with SQLite JSON functions.
* Implement `Save`, `SaveConditional`, `Get`, `Delete`, `List` (with pagination, filtering, sorting), and `Close`.

### 4. FileStore Integration (`SqliteP2PFileStore`)
* Implement `IFileStore` backed by `sqlite-p2p` database table for peer-synchronized binary asset storage.
* Implement `Upload`, `Download`, `GetMetadata`, `GetUrl`, `Delete`, `DeleteByPrefix`, `List`, and `PurgeExpired`.

### 5. Verification
* Ensure clean compilation and test execution.

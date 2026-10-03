# Implementation Plan: Full Codebase Review Remediation

This plan outlines the phased remediation of all Critical, High, Medium, and Low issues identified in the repository code review report.

---

## Phase 1: Security & Critical Vulnerability Fixes
- [x] **1.1 Configurable JWT Secret (`SEC-1`)**: Update `framework/config.go` & `framework/init.go` to require `AIGEN_JWT_SECRET` / `cfg.Auth.JWTSecret` with safe defaults/checks.
- [x] **1.2 RBAC Middleware Fail-Closed (`SEC-2`)**: Update `core/api/auth_api.go:390` to return `403 Forbidden` on empty resource lookup.
- [x] **1.3 Asset API Auth & Identity (`SEC-3`)**: Add JWT middleware to `/api/assets` in `core/api/asset_api.go` and safely extract `userId` from context.
- [x] **1.4 SQL/SurrealQL Injection Prevention (`C-4`, `H-6`, `H-7`)**: Add `ValidateFieldName` across `postgres.go`, `sqlite_p2p.go`, `surreal.go` DAOs and filestores.
- [x] **1.5 Secure JWT Cookie Flags (`H-17`)**: Set `Secure: true`, `HttpOnly: true`, and `SameSite: Strict` across all auth cookies in `core/api/auth_api.go`.

---

## Phase 2: Core Services & API Correctness Fixes
- [x] **2.1 Password Hashing Error Handling (`C-8`)**: Check error on `bcrypt.GenerateFromPassword` in `core/services/entity_service.go`.
- [x] **2.2 Comma-ok Type Assertions (`C-12`, `C-13`, `M-7`, `M-10`)**: Replace bare assertions with comma-ok checks across `permission_service.go`, `auth_api.go`, `memory.go`, and `auth_service.go`.
- [x] **2.3 Schema Version Promotion Errors & ID Generation (`C-9`, `C-10`, `H-13`)**: Handle DAO errors in `core/services/schema_service.go` and use `ids.NewRandomID()` for schemas and users.
- [x] **2.4 Admin Bootstrap Scan (`C-11`)**: Fix 100-user scan cap in `core/services/auth_service.go:301`.
- [x] **2.5 HTTP Error Sanitization (`H-16`)**: Sanitize error strings returned in `core/api/entity_api.go`.

---

## Phase 3: Infrastructure (DAO & FileStore) Fixes
- [x] **3.1 S3 Driver Pointer Safety & Download Corruption (`C-5`, `C-6`)**: Nil-safe pointer dereferencing and sequential downloads in `infrastructure/filestore/s3.go`.
- [x] **3.2 SQLite P2P Chunk Uploads (`C-7`, `H-1`, `H-2`)**: Fix chunk index sorting, ensure atomic cleanup on upload success, and support persistent chunk storage.
- [x] **3.3 SQL List Row Errors & JSON Unmarshal (`H-4`, `H-9`)**: Check `rows.Err()` and JSON unmarshal errors across `postgres.go` and `sqlite_p2p.go`.
- [x] **3.4 Filestore Cross-Platform URL & Contract Normalization (`M-2`, `M-4`)**: Use `path.Join` in `local.go` and unify `GetMetadata` not-found error behavior.

---

## Phase 4: Framework Defaults, Context Keys & CLI Polish
- [x] **4.1 Default Config (`C-14`)**: Set default `DatabaseDSN` and storage `Driver` to `sqlite-p2p` in `framework/config.go`.
- [x] **4.2 Typed Context Keys (`M-13`)**: Implement `ctxKey` type to avoid plain string context values.
- [x] **4.3 Test Env Decoupling (`M-12`)**: Decouple `isTestEnv` in `framework/init.go` from database DSN.
- [x] **4.4 CLI Messages & Help Strings (`L-6`, `L-7`)**: Update admin CLI help text and error logging.

---

## Phase 5: Test Suite Execution & Verification
- [x] **5.1 Package-level Unit Tests**: Run and expand unit tests for DAO, filestore, services, and API packages.
- [x] **5.2 Full Test Suite**: Execute `go test ./...` and ensure all tests pass with zero regressions.

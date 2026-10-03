# Specification: Full Codebase Review Remediation

## Overview
This specification defines the comprehensive remediation requirements for all issues identified in the repository-wide code review report (`/brain/069aed54-b1ea-4c2d-88de-2fa3799bc8e0/code-review-report.md`). It addresses 14 Critical, 19 High, 27 Medium, and 24 Low severity issues across all layers of `AIGenApp` (Security, Core Services, API, Infrastructure DAO/Filestore, Framework, and Configuration).

---

## 1. Security & Authentication Requirements

### 1.1 Configurable JWT Secret (`SEC-1`)
* Remove hardcoded `"your-secret-key"` placeholder from `framework/init.go`.
* Source JWT signing secret from `AIGEN_JWT_SECRET` environment variable or `Config.Auth.JWTSecret`.
* Fail fast (`log.Fatal` / return error) on startup in production if secret is empty or insecurely short (< 32 characters).

### 1.2 RBAC Middleware Bypass Elimination (`SEC-2`)
* Fix `core/api/auth_api.go:390` `RBACMiddleware` where empty `resourceName` (`""`) bypasses role evaluation and calls `next.ServeHTTP`.
* When a protected route does not specify a resource or fails resource extraction, fail closed with `403 Forbidden`.

### 1.3 Asset API Authentication & Identity (`SEC-3`)
* Apply JWT authentication middleware to `/api/assets` endpoints.
* Replace hardcoded `userId = "admin"` in `core/api/asset_api.go:34` with authenticated user identity extracted safely from request context.
* Refactor `AssetApi` to depend on interface abstractions (`IAssetService` / `IFileStore`) rather than concrete struct pointers.

### 1.4 SQL / SurrealQL Injection Prevention (`C-4`, `H-6`, `H-7`)
* Implement field name validation (`ValidateFieldName`) in all SQL DAOs (`postgres.go`, `sqlite_p2p.go`, `surreal.go`) matching `^[a-zA-Z_][a-zA-Z0-9_.]*$` before constructing JSON extraction or sorting SQL fragments (`rec->>'...'`).
* Sanitize record path identifiers and eliminate raw string interpolation in SurrealDB queries.

### 1.5 Secure Cookie Policy (`H-17`)
* Enforce `Secure: true`, `HttpOnly: true`, and `SameSite: http.SameSiteStrictMode` on all JWT authentication cookies in `core/api/auth_api.go`.

---

## 2. Core Services & API Correctness

### 2.1 Password Hashing & Error Propagation (`C-8`)
* Handle `bcrypt.GenerateFromPassword` errors explicitly in `core/services/entity_service.go` (lines 126, 189). Never store empty string password hashes.

### 2.2 Panic Elimination & Type Assertions (`C-12`, `C-13`, `M-7`, `M-10`)
* Replace bare type assertions (e.g. `userId := r.Context().Value("userId").(int64)`, `data["read"].(bool)`) with comma-ok assertions across `permission_service.go`, `auth_api.go`, `memory.go`, and `auth_service.go`.
* Return appropriate HTTP 401/403 or error responses when context values are absent or invalid.

### 2.3 Schema Version Promotion Consistency (`C-9`, `C-10`)
* Propagate errors from DAO operations during schema version deprecation (`isLatest=false`) in `core/services/schema_service.go`.
* Replace `time.Now().UnixNano()` temporary ID generation with cryptographically secure / collision-free IDs (`ids.NewRandomID()`).

### 2.4 Admin Bootstrapping Robustness (`C-11`)
* Replace 100-user limit scan in `auth_service.go:301` with targeted role/admin existence check.

### 2.5 Error Sanitization in HTTP Responses (`H-16`)
* Sanitize error responses in `core/api/entity_api.go` and other handlers to avoid leaking internal database identifiers and file paths to API clients.

---

## 3. Infrastructure Layer (DAO & Filestore)

### 3.1 Filestore S3 Driver Fixes (`C-5`, `C-6`)
* Safe dereferencing of pointer fields (`ContentLength`, `ContentType`, `LastModified`) in `infrastructure/filestore/s3.go`.
* Fix byte stream corruption in `fakeWriterAt` by using sequential `s3.GetObject` + `io.Copy` or forcing concurrency=1.

### 3.2 Filestore SQLite P2P Driver Fixes (`C-7`, `H-1`, `H-2`)
* Fix chunk numbering iteration in `SqliteP2PFileStore.CommitChunks` to support 0-based and arbitrary contiguous chunk keys.
* Retain chunk buffers until commit operation succeeds to allow client upload retries.
* Implement database table persistence for multi-part file chunks.

### 3.3 Driver Error Checking & Consistency (`H-4`, `H-9`, `M-2`, `M-4`)
* Add `rows.Err()` checks across all `List()` implementations in SQL DAOs and FileStores.
* Propagate unmarshaling errors in `postgres.go` and `sqlite_p2p.go`.
* Standardize `GetMetadata` not-found contract across all storage drivers.
* Fix path separator bug on Windows in `local.go` using `path.Join`.

---

## 4. Framework, Defaults & Cleanups

### 4.1 Default Configuration Alignment (`C-14`)
* Update `framework/config.go` `DefaultConfig` to use `sqlite-p2p://.aigen.db` as default DSN and `sqlite-p2p` as default storage driver.
* Decouple `isTestEnv` detection in `framework/init.go` from database DSN value.

### 4.2 Typed Context Keys (`M-13`)
* Define package-private / shared typed context key types (`type ctxKey string`) to prevent string context collisions and satisfy `go vet`.

---

## 5. Verification & Testing
* Unit and integration test coverage across all affected components.
* Zero regressions on full test suite: `go test ./...`.

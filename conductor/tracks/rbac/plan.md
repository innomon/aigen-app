# Implementation Plan: Fine-Grained RBAC

## 1. Phase 1: Data Model & Setup (DONE)
- [x] Create `apps/rbac/schemas/` directory.
- [x] Define `Role` schema (`role.json`).
- [x] Define `UserRole` junction schema (`user_role.json`).
- [x] Define `DocPerm` schema (`doc_perm.json`).
- [x] Define `UserPermission` schema (`user_perm.json`).
- [x] Update `main.go` to ensure `rbac` app is loaded.
- [x] Implement a migration/bootstrap to create default roles.

## 2. Phase 2: Core Services (DONE)
- [x] Create `core/services/permission_service.go`.
- [x] Implement `HasAccess(userId, entityName, action)` in `PermissionService`.
- [x] Implement `GetRowFilters(userId, entityName)` in `PermissionService`.
- [x] Implement `GetFieldPermissions(userId, entityName, roleIds)` in `PermissionService`.
- [x] Update `AuthService` to return multiple roles for a user.

## 3. Phase 3: Middleware & Integration (DONE)
- [x] Implement `RBACMiddleware` in `core/api/auth_api.go`.
- [x] Update `EntityApi` to use `RBACMiddleware`.
- [x] Integrate `PermissionService` into `EntityService`:
    - [x] Apply `GetRowFilters` in `List` and `Single` methods.
    - [x] Apply field-level filtering in `scanRows`.
    - [x] Apply field-level filtering in `Insert` and `Update` methods.
- [ ] Integrate `PermissionService` into `SchemaService` to restrict schema visibility.

## 4. Phase 4: API & Testing (DONE)
- [x] Implement `RBACApi` to manage roles and permissions via REST.
- [x] Add unit tests for `PermissionService`.
- [x] Add integration tests for protected `EntityApi` endpoints.
- [x] Verify row-level filtering with multiple `UserPermission` rules.


## Implementation Checklist

- [x] Define RBAC system schemas
- [x] Bootstrap RBAC tables and initial data
- [x] Implement core PermissionService logic
- [x] Update JWT and Context to handle multiple roles
- [x] Implement Authorization Middleware
- [ ] Integrate RBAC into EntityService (Row-level)
- [ ] Integrate RBAC into EntityService (Field-level)
- [x] Implement RBAC Management APIs
- [ ] Final end-to-end verification

# Implementation Plan: Fine-Grained RBAC

## 1. Phase 1: Data Model & Setup
- [ ] Create `apps/rbac/schemas/` directory.
- [ ] Define `Role` schema (`role.json`).
- [ ] Define `UserRole` junction schema (`user_role.json`).
- [ ] Define `DocPerm` schema (`doc_perm.json`).
- [ ] Define `UserPermission` schema (`user_perm.json`).
- [ ] Update `main.go` to ensure `rbac` app is loaded (or add to `apps.json`).
- [ ] Implement a migration/bootstrap to create default roles (SA, Admin, User, Guest) and link existing users to them.

## 2. Phase 2: Core Services
- [ ] Create `core/services/permission_service.go`.
- [ ] Implement `HasAccess(userId, entityName, action)` in `PermissionService`.
- [ ] Implement `GetRowFilters(userId, entityName)` in `PermissionService`.
- [ ] Implement `GetFieldPermissions(userId, entityName, roleIds)` in `PermissionService`.
- [ ] Update `AuthService` to return multiple roles for a user.

## 3. Phase 3: Middleware & Integration
- [ ] Implement `RBACMiddleware` in `core/api/auth_api.go` or a new file.
- [ ] Update `EntityApi` to use `RBACMiddleware`.
- [ ] Integrate `PermissionService` into `EntityService`:
    - [ ] Apply `GetRowFilters` in `List` and `Single` methods.
    - [ ] Apply field-level filtering in `scanRows`.
    - [ ] Apply field-level filtering in `Insert` and `Update` methods.
- [ ] Integrate `PermissionService` into `SchemaService` to restrict schema visibility.

## 4. Phase 4: API & Testing
- [ ] Implement `RBACApi` to manage roles and permissions via REST.
- [ ] Add unit tests for `PermissionService`.
- [ ] Add integration tests for protected `EntityApi` endpoints.
- [ ] Verify row-level filtering with multiple `UserPermission` rules.

## Implementation Checklist

- [ ] Define RBAC system schemas
- [ ] Bootstrap RBAC tables and initial data
- [ ] Implement core PermissionService logic
- [ ] Update JWT and Context to handle multiple roles
- [ ] Implement Authorization Middleware
- [ ] Integrate RBAC into EntityService (Row-level)
- [ ] Integrate RBAC into EntityService (Field-level)
- [ ] Implement RBAC Management APIs
- [ ] Final end-to-end verification

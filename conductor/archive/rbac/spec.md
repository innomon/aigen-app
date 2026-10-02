# Fine-Grained RBAC Specification (Frappe-inspired)

## 1. Overview
The goal is to implement a robust, flexible, and fine-grained Role-Based Access Control (RBAC) system in FormCMS Go, similar to the one used in Frappe/ERPNext. This will allow administrators to define complex permission rules for different user roles, including row-level and field-level access control.

## 2. Core Concepts

### 2.1. User & Roles
- **User**: Can be assigned multiple **Roles**.
- **Role**: A named entity (e.g., "Blogger", "Sales Manager", "System Manager") that groups a set of permissions.

### 2.2. Document Permissions (DocPerm)
Permissions are defined per **Role** per **Entity** (DocType).
- **Actions**:
    - `Read`: Can view records.
    - `Write`: Can update existing records.
    - `Create`: Can create new records.
    - `Delete`: Can delete records.
    - `Submit`: For workflow-enabled entities, can transition to a "Submitted" state.
    - `Cancel`: For workflow-enabled entities, can transition to a "Cancelled" state.
    - `Amend`: Can create a new version of a cancelled record.
    - `Report`: Can view reports for this entity.
    - `Export`: Can export data.
    - `Import`: Can import data.
    - `Print`: Can generate print views.
    - `Email`: Can send records via email.
    - `Share`: Can share records with other users.
- **Permission Level (PermLevel)**:
    - An integer (0-9).
    - Attributes (fields) in an Entity are assigned a `PermLevel`.
    - A Role must have permission for a specific `PermLevel` to access those fields.
    - `PermLevel 0` is the default for most fields and general access.

### 2.3. User Permissions (Row-Level)
Restricts access to specific records based on field values.
- **Rule**: "User X is allowed to see 'Invoice' where 'Company' is 'MyCompany'".
- **Logic**: When a user queries an entity, the system automatically appends filters based on these rules.

## 3. Data Models

### 3.1. Role (`__roles`)
| Field | Type | Description |
|-------|------|-------------|
| id | ID | Primary Key |
| name | String | Role name (Unique) |
| disabled | Boolean | If the role is inactive |

### 3.2. User Role (`__user_roles`)
Junction table between Users and Roles.
| Field | Type | Description |
|-------|------|-------------|
| user_id | ID | Link to `__users` |
| role_id | ID | Link to `__roles` |

### 3.3. Document Permission (`__doc_perms`)
| Field | Type | Description |
|-------|------|-------------|
| role | Link | Link to `__roles` |
| parent | String | Entity name (e.g., "Invoice") |
| permlevel | Int | Permission level (0-9) |
| read | Boolean | |
| write | Boolean | |
| create | Boolean | |
| delete | Boolean | |
| submit | Boolean | |
| cancel | Boolean | |
| amend | Boolean | |
| report | Boolean | |
| export | Boolean | |
| import | Boolean | |
| print | Boolean | |
| email | Boolean | |
| share | Boolean | |

### 3.4. User Permission (`__user_perms`)
| Field | Type | Description |
|-------|------|-------------|
| user | Link | Link to `__users` |
| allow | String | Entity name (e.g., "Company") |
| for_value | String | Allowed record ID |

## 4. Implementation Logic

### 4.1. Permission Evaluation
For a given `(userId, entityName, action)`:
1. Fetch all `Roles` assigned to the `User`.
2. Fetch all `DocPerms` for these `Roles` where `parent == entityName` and `permlevel == 0`.
3. If any `DocPerm` allows the `action`, access is granted (subject to Row-Level checks).

### 4.2. Row-Level Enforcement (User Permissions)
For a query on `Entity A`:
1. Identify all fields in `Entity A` that link to other entities (e.g., `company` links to `Company`).
2. Fetch `UserPermissions` for the `User` where `allow` matches the linked entity names.
3. If rules exist, append filters: `field IN (allowed_values)`.

### 4.3. Field-Level Enforcement
- **Read**: When scanning rows, nullify or omit fields where the user's roles don't have `read` permission for the field's `permlevel`.
- **Write/Create**: When processing input data, ignore fields where the user doesn't have `write` permission for the field's `permlevel`.

## 5. API & Middleware
- **AuthMiddleware**: Load user's roles into context.
- **RBACMiddleware**: Check `permlevel 0` permissions for the requested route/action.
- **PermissionService**: Centralize logic for `HasPermission`, `ApplyRowFilters`, and `FilterFields`.

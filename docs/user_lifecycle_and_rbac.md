# User Lifecycle and Role-Based Access Control (RBAC)

## 1. Overview

This document details the user lifecycle and the Role-Based Access Control (RBAC) architecture for the FormCMS Go system. The RBAC system is heavily inspired by Frappe/ERPNext, providing fine-grained permissions at the entity (document), row, and field levels.

---

## 2. User Lifecycle

The user lifecycle in FormCMS outlines how a user entity is created, authenticated, authorized, and managed throughout its existence in the system.

### 2.1. Registration & Provisioning
* **Creation:** Users are created via an API or admin interface. The primary entity representing a user is stored in the `__users` table (based on the `User` schema).
* **Attributes:** Core attributes include `email`, `password_hash`, `role` (legacy support), and `avatar_path`.
* **Default Role Assignment:** Upon creation, a user typically receives a default system role (e.g., "Guest" or "User") to ensure immediate basic access, before higher-level privileges are granted by an Administrator.

### 2.2. Authentication
* **Login:** The user provides credentials (email and password), which are verified against `password_hash`.
* **Token Generation:** A successful login results in a JWT (JSON Web Token) or session token. The `AuthService` embeds the user's ID and associated Roles within the token context for subsequent API requests.

### 2.3. Authorization & Active Usage
* **Middleware Integration:** During an active session, every incoming request passes through the `AuthMiddleware` (which extracts the user context) and the `RBACMiddleware` (which validates permissions).
* **Role Expansion:** A single User can be assigned multiple Roles via the `__user_roles` junction table.

### 2.4. De-provisioning & Updates
* **Profile Management:** Users can update certain attributes (like `avatar_path`).
* **Role Revocation:** Administrators can modify `__user_roles` to escalate or revoke access.
* **Deletion/Disabling:** Users can be deleted or disabled. Disabling is preferred to maintain referential integrity in audits and logs.

---

## 3. Core RBAC Concepts

The RBAC system allows administrators to define complex permission rules.

### 3.1. Roles
* **Definition:** A Role is a named grouping of permissions (e.g., "System Manager", "Sales Manager", "Blogger").
* **Multiple Roles:** Users can have multiple roles. Their effective permissions are typically a union of all permissions granted by their assigned roles.

### 3.2. Document Permissions (DocPerm)
Permissions are defined per **Role** and per **Entity (DocType)**.
* **Supported Actions:**
    * `Read`: View records.
    * `Write`: Update existing records.
    * `Create`: Create new records.
    * `Delete`: Remove records.
    * `Submit` / `Cancel` / `Amend`: Workflow state transitions.
    * `Report` / `Export` / `Import` / `Print` / `Email` / `Share`: Extended system actions.
* **Permission Level (`PermLevel`):**
    * Integer value (0-9).
    * Entity attributes (fields) can be assigned a `PermLevel`.
    * A Role must have explicit permission for a specific `PermLevel` to read/write those fields. `PermLevel 0` is the baseline for general entity access.

### 3.3. Row-Level Security (User Permissions)
Row-level access restricts the records a user can see based on specific field values.
* **Rule Syntax:** e.g., "User X is allowed to see 'Invoice' where 'Company' is 'MyCompany'".
* **Enforcement:** The `PermissionService` dynamically appends filters (`field IN (allowed_values)`) to queries when fetching data.

### 3.4. Field-Level Security
* **Read Enforcement:** When scanning rows from the database, the system will nullify or omit fields where the user's roles lack `read` permission for that field's `PermLevel`.
* **Write/Create Enforcement:** During data ingestion, the system silently drops or rejects payload fields where the user lacks `write` permission for that field's `PermLevel`.

### 3.5. Dynamic Role-Based Dashboard & Navigation
To provide a tailored experience, the system supports dynamic dashboards and menus based on the user's active role.
* **Role Configuration:** Each Role can be linked to a `dashboard_page_id` (a dynamic Page created via the system's GrapesJS page builder) and a `menu_id` (a Menu entity).
* **Default Assignment:** Users have a `default_role_id`. Upon login, the frontend routes the user to the dashboard corresponding to this default role.
* **Role Switching:** If a user holds multiple roles, a role switcher is presented in the navigation bar. Selecting a different role updates the frontend state (`activeRoleId`), immediately reloading the UI to display the new role's dashboard and navigation menu.
* **Fallback:** If a role does not have a configured dashboard, or the fetch fails, the system automatically falls back to a generic default dashboard.

---

## 4. Data Models

The RBAC system introduces several dynamic schemas in the database.

### 4.1. User (`__users`)
| Field | Type | Description |
|-------|------|-------------|
| id | ID | Primary Key |
| email | String | User email (Login ID) |
| password_hash| String | Encrypted password |
| avatar_path | String | URL/Path to avatar |
| default_role_id | Integer | The default role id for the dashboard |
| role | String | Legacy role field |

### 4.2. Role (`__roles`)
| Field | Type | Description |
|-------|------|-------------|
| id | ID | Primary Key |
| name | String | Role name (Unique) |
| disabled | Boolean | Inactive flag |
| dashboard_page_id | String | The Page entity ID assigned to this role's dashboard |
| menu_id | String | The Menu entity ID assigned to this role's navigation |

### 4.3. User Role (`__user_roles`)
*Junction table for Many-to-Many User-to-Role mapping.*
| Field | Type | Description |
|-------|------|-------------|
| user_id | ID | Link to `__users` |
| role_id | ID | Link to `__roles` |

### 4.4. Document Permission (`__doc_perms`)
| Field | Type | Description |
|-------|------|-------------|
| role | Link | Link to `__roles` |
| parent | String | Entity name (e.g., "Invoice") |
| permlevel | Int | Permission level (0-9) |
| read, write, create, delete, etc. | Boolean | Action flags |

### 4.5. User Permission (`__user_perms`)
| Field | Type | Description |
|-------|------|-------------|
| user | Link | Link to `__users` |
| allow | String | Entity name (e.g., "Company") |
| for_value | String | Allowed record ID / Value |

---

## 5. Implementation Logic & Services

### 5.1. PermissionService
The core engine for access checks:
* `HasAccess(userId, entityName, action)`: Evaluates if any of the user's roles grant the requested action on the entity at `PermLevel 0`.
* `GetRowFilters(userId, entityName)`: Retrieves applicable User Permissions to append as SQL WHERE clauses.
* `GetFieldPermissions(userId, entityName, roleIds)`: Dictates which fields are readable/writable based on `PermLevel` matching.

### 5.2. Middlewares
* **AuthMiddleware:** Authenticates the incoming request, parses the JWT, and loads the user ID and Roles into the HTTP Request Context. If no valid authentication is provided, it assigns the user ID `0` and the `guest` role to allow anonymous access to permitted resources.
* **RBACMiddleware:** Intercepts route requests. Uses `PermissionService.HasAccess()` to verify if the Context user has the necessary `PermLevel 0` permissions for the target endpoint. It supports entity-based access control as well as explicit resource names (e.g., `graphql`, `chat`, `a2ui`) to secure custom APIs.

### 5.3. Anonymous / Guest Access
* **The Guest Role:** Anonymous users interacting with the system are assigned the `guest` role by the `AuthMiddleware`.
* **Public APIs:** Open APIs like GraphQL, Stored Queries, Comments, Engagements, and A2UI are secured using `RBACMiddleware` with explicit resource names. Guests only have access to these APIs if the `guest` role is explicitly granted permissions for those resources in the RBAC configuration.
* **Guest Dashboard:** The `guest` role can be configured with a `dashboard_page_id`. If a guest visits the root URL (`/`), the system will serve this custom dashboard page instead of redirecting to the admin login.

### 5.4. EntityService Integration
The `EntityService` integrates deeply with `PermissionService`:
* **List/Single Queries:** Calls `GetRowFilters` before executing the underlying `squirrel` SQL builder to ensure Row-Level security.
* **Data Scanning:** Uses `GetFieldPermissions` to strip unauthorized fields from the JSON response.
* **Data Mutations (Insert/Update):** Strips unauthorized fields from the incoming JSON payload before constructing the SQL INSERT/UPDATE statements.

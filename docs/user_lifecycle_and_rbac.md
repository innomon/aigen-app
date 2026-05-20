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

---

## 6. Administrator Account Bootstrapping & CLI Management

To ensure an out-of-the-box experience while maintaining maximum production security, AIGenApp features a secure, database-agnostic administrator account bootstrapping system and a dedicated handcrafted CLI administration utility.

### 6.1. Bootstrapping Lifecycle

On server startup, after database initialization, the framework automatically scans for any existing users with administrative privileges (`admin` or `sa` roles) in the user records. 

* **Skip Condition:** If at least one administrative user is found, the bootstrapping sequence is completely skipped. This guarantees that existing administrative setups and credentials are never overwritten or altered.
* **First-Time Boot:** If no administrative users are found in the system, a new super-admin account is generated based on the environment type.

### 6.2. Environment-Specific Behavior

The bootstrapping behavior adapts dynamically to the environment configuration:

#### A. In-Memory & Test Runs (`memory://` or `FORMCMS_ENV=test`)
When running locally with an in-memory database or in test mode, the system assumes a developer/test environment and auto-seeds standard credentials for immediate access:
* **Default Email:** `admin@aigen.local`
* **Default Password:** `adminpassword`
* **Assigned Roles:** `sa` (Super Admin), `admin`, `user`

#### B. Production & Persistent Database Runs (`postgres://...`)
For live databases (such as PostgreSQL), security-first policies are enforced:

1. **Configured Overrides:** The system checks the environment variables:
   * `AIGEN_ADMIN_EMAIL`
   * `AIGEN_ADMIN_PASSWORD`
   
   If these are set, they take absolute precedence. You can also specify them inside the YAML configuration file under:
   ```yaml
   admin:
     email: "admin@company.com"
     password: "super-secure-password"
   ```

2. **Secure Auto-Generation Fallback:** If no environment overrides or YAML configurations are provided, and no admins exist:
   * A high-entropy 16-character password is generated using Go's cryptographically secure random generator `crypto/rand`.
   * A super-admin account with the email `admin@aigen.local` is registered.
   * A highly visible stdout warning banner is printed to the console containing the generated password:
     ```
     ========================================================================
     [WARNING] FIRST TIME STARTUP DETECTED: NO ADMIN ACCOUNT FOUND.
     We have safely bootstrapped a super-admin account for you:

       Username/Email: admin@aigen.local
       Password:       [16-character-random-password]

     Please log in and CHANGE this password immediately inside the admin panel!
     ========================================================================
     ```

---

### 6.3. Handcrafted CLI Management Tool (`aigen-admin`)

For direct administrative intervention on host environments, the project provides a dedicated Cobra-free/Pflag-free CLI tool compiled from `cmd/admin/main.go`. This utility operates directly on the underlying database schema utilizing standard Go flag parsing.

#### Building the Utility
```bash
go build -o aigen-admin cmd/admin/main.go
```

#### Subcommands

##### 1. `create`
Manually registers a new super-admin user directly into the database.
```bash
./aigen-admin create -db="postgres://user:pass@host:5432/db" -email="admin@my-company.com" -password="secure-password"
```

##### 2. `reset-pass`
Resets the password of any existing user inside the database.
```bash
./aigen-admin reset-pass -db="postgres://user:pass@host:5432/db" -email="admin@my-company.com" -password="new-secure-password"
```


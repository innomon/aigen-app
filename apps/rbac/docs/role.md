# Role Context

The `role` entity defines a set of permissions that govern what actions a user can perform within the system. It is the cornerstone of the Role-Based Access Control (RBAC) implementation.

## Lifecycle
Roles are typically predefined for the application (e.g., 'System Manager', 'Sales Representative') but can be dynamically created or disabled by administrators.

## Key Fields & Behaviors
- **Name**: The unique identifier for the role (e.g., 'Sales Manager').
- **Disabled**: A flag to quickly suspend all permissions associated with this role without deleting it.
- **Dashboard Page / Menu**: Specifies the default UI experience (dashboard layout and navigation menu) for users logging in with this role.

## Relationships
- **Document Permissions**: A role is granted specific permissions (read, write, delete, etc.) on various entities through the `doc_perm` entity.
- **User Assignment**: Roles are assigned to users via the `user_role` junction entity.
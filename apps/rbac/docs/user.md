# User Context

The `user` entity represents individuals who have access to the AiGen CMS ecosystem. It serves as the primary identity for authentication and authorization.

## Lifecycle
Users are created by administrators or through self-registration (if enabled). They can be assigned a legacy role or be linked to multiple roles via the `user_role` entity.

## Key Fields & Validation Rules
- **Email**: The primary identifier and username for login. Must be unique.
- **Password**: Stored as a secure hash (`password_hash`), never in plain text.
- **Default Role**: Determines the user's primary dashboard and menu navigation if they have multiple roles.
- **Legacy Role**: An older, simpler string-based role field, primarily kept for backward compatibility. Modern role assignment uses `user_role`.

## Relationships
- **User Roles**: A user can have many roles, defined in the `user_role` mapping entity.
- **User Permissions**: Specific data-level restrictions (row-level security) are applied via the `user_perm` entity.
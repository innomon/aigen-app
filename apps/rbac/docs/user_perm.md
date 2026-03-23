# User Permission Context

The `user_perm` entity provides row-level security (RLS) or data filtering for specific users. While `doc_perm` controls *if* a role can access an entity, `user_perm` controls *which specific records* of that entity the user can access.

## Business Purpose
Often used in multi-tenant or highly restricted environments, `user_perm` ensures a user only sees data relevant to them. For example, ensuring a Sales Rep only sees `crm_lead` records where the assigned territory matches their own territory.

## Key Fields
- **User ID**: The user this restriction applies to.
- **Allow Entity**: The target entity being restricted (e.g., `crm_lead`).
- **For Value**: The specific value the user is allowed to access. The exact matching mechanism depends on the core system's query parser, but typically forces a mandatory `WHERE field = for_value` on queries against the allowed entity.

## Relationships
- **User**: Must link to a valid `user` entity.
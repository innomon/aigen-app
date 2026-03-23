# Document Permission Context

The `doc_perm` entity acts as the central authorization matrix. It defines exactly what actions a specific `role` can perform on a specific entity type (document/table).

## Business Purpose
Instead of hardcoding permissions, the system uses `doc_perm` records to dynamically resolve access rights when a user queries or mutates an entity. 

## Key Fields & Permissions
- **Role**: The role being granted these permissions.
- **Entity Name (Parent)**: The system name of the entity these rules apply to (e.g., `crm_lead`).
- **Perm Level**: Allows hierarchical permission structures (e.g., base level 0, higher levels for specific fields/actions).
- **CRUD Operations**: Standard `read`, `write`, `create`, `delete` boolean flags.
- **Workflow Operations**: Extended flags like `submit`, `cancel`, `amend` for documents with complex lifecycles (like accounting journals).
- **Utility Operations**: Flags for `report`, `export`, `import`, `print`, `email`, and `share`.

## Relationships
- **Role**: Must link to a valid `role` entity. 
- **Entity**: Maps to any dynamically defined schema within the CMS.
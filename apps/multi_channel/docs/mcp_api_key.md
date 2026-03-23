# MCP API Key Context

The `mcp_api_key` represents a secure credential used by external agents (e.g., Claude, ChatGPT, custom LLM co-workers) to connect to the CMS via the **Model Context Protocol (MCP)**.

## Business Purpose
To provide a gated, traceable, and role-based access mechanism for external AI agents to interact with CMS entities and tools. This allows "human-in-the-loop" co-working where an external agent can perform tasks (like summarizing leads or updating orders) on behalf of a specific system user.

## Relationships
- **User ID**: Every API key must be linked to a valid system `user`. 
- **Roles**: The linked user **must** have the `MCP` role. The permissions of the `MCP` role determine which tools and resources are available through the MCP server.

## Lifecycle
1.  **Creation**: A `channel_admin` generates a new key and assigns it to a user.
2.  **Activation**: The key is used in the `X-API-Key` header or `apiKey` query parameter for MCP SSE connections.
3.  **Rotation/Disabling**: Keys can be toggled via the `enabled` field to immediately revoke access without deleting the user.

## Constraints
- **Uniqueness**: Each API key string must be globally unique.
- **Role Requirement**: A key associated with a user lacking the `MCP` role will be rejected during authentication.

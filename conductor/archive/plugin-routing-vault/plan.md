# Track: Plugin-Aware Routing & Secure Vault

## Goal
Enhance the RouterAgent to dynamically route requests to Applet plugins by analyzing their documentation and metadata. Implement a secure permission system and a vault for plugin-specific environment variables.

## Objectives
1.  [x] **Plugin-Aware Routing**: RouterAgent discovers active plugins and routes chats to them if they match the user's intent.
2.  [x] **Agent Transfer**: Automatically load a plugin's `agentic.yaml` and delegate execution to its `root_agent`.
3.  [x] **Permission Management**: A manifest-based permission system where admins authorize sensitive operations (e.g., HTTP access).
4.  [x] **Secure Vault**: A mechanism to securely store and retrieve environment variables required by plugins.
5.  [x] **Audit Logging**: Logged all admin authorizations and security-sensitive transitions.

## Proposed Changes

### Core Models (`core/plugins/models.go`)
- Update `PluginManifest` to include:
    - `EnvVars []string`: List of required environment variable keys.
    - `Permissions []PermissionRequirement`: Structured permission requests (e.g., `{"type": "http", "value": "*.openai.com"}`).
- Create `PermissionGrant` and `VaultEntry` models.

### Plugin Service (`core/plugins/service.go`)
- Add `GetRoutingDocs()`: Returns a list of plugin IDs and descriptions for the RouterAgent.
- Add `LoadAgenticConfig(id string)`: Extracts and parses `agentic/agentic.yaml` from the JAR.
- Implement `AuthorizePermission(pluginID string, permission PermissionRequirement, adminID string)`: Persists authorization and logs it.
- Implement `Vault`: `SetSecret(pluginID string, key string, value string)` and `GetSecret(...)`.

### Router Agent (`core/agentic/agents/router_agent.go`)
- Refactor to take `IPluginProvider` interface.
- Update `Run` loop:
    - Include plugin descriptions in the classification prompt.
    - If a plugin is selected:
        1. Fetch its `agentic.yaml`.
        2. Instantiate its `root_agent` in memory.
        3. Transfer control.

### API & UI
- Admin endpoints for:
    - Listing pending permissions.
    - Authorizing permissions.
    - Managing vault secrets.

## Success Criteria
- RouterAgent correctly identifies when a query should be handled by a plugin (e.g., "What's the weather?" routed to `weather-plugin`).
- Sandbox scripts fail if they attempt unauthorized HTTP calls.
- Sensitive environment variables are not stored in plain text or committed to code.
- Admin authorizations are visible in audit logs.

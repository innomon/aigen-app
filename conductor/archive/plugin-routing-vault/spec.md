# Specification: Plugin-Aware Routing & Secure Vault

## 1. Overview
This track specifies the integration of the `RouterAgent` with the Applet Plugin system and the implementation of a secure sandbox environment with manifest-based permissions and encrypted vault access.

## 2. Requirements

### 2.1 Plugin-Aware RouterAgent
- **Discovery**: Must query `PluginService` for active plugins and their descriptions.
- **Classification**: Include plugin descriptions in the classification prompt to allow the LLM to route intent to plugins.
- **Transfer**: When a plugin is selected, the system must:
    1. Extract `agentic/agentic.yaml` from the plugin JAR.
    2. Instantiate the plugin's root agent.
    3. Transfer the `InvocationContext` to the plugin agent for execution.

### 2.2 Security Sandbox & Permissions
- **Manifest Declaration**: Plugins must list required permissions in `metadata.json` (e.g., `http`, `bizdef:crm`).
- **Authorization Enforcement**: 
    - The `SandboxDispatcher` must check for an authorized `PermissionGrant` before allowing sensitive host API calls (like HTTP fetch).
    - Authorization must be granted by an admin and recorded in the audit logs.
- **Host API Bridge**: Scripts interact via `AIGenHostAPI`, which acts as a secure, audited gateway.

### 2.3 Secure Vault
- **Isolation**: Plugins can only access secret keys that are declared in their `EnvVars` manifest section.
- **Persistence**: Secrets must be stored securely (encrypted in production, simplified vault for POC).
- **Access**: Provided via `GetSecret(key)` within the sandbox environment.

## 3. Data Models

### 3.1 PermissionRequirement
```go
type PermissionRequirement struct {
    Type  string // e.g., "http"
    Value string // e.g., "*.google.com"
}
```

### 3.2 PermissionGrant
```go
type PermissionGrant struct {
    PluginID  string
    Type      string
    Value     string
    GrantedBy string
    GrantedAt time.Time
}
```

### 3.3 VaultEntry
```go
type VaultEntry struct {
    PluginID  string
    Key       string
    Value     string
    UpdatedAt time.Time
}
```

## 4. Audit & Compliance
- Every administrative permission grant must generate an `Audit` record.
- Unauthorized attempts to access secrets or external URLs must be logged as security alerts.

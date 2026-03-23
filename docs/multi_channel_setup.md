# Multi-Channel Communication Setup Guide (A2A & MCP)

This guide provides instructions for setting up and configuring the multi-channel communication system in your `aigen-cms` powered application using the A2A and MCP protocols.

## Prerequisites

1.  **Aigen CMS Instance**: A running instance of `aigen-cms`.
2.  **External A2A Agents**: External agents (WhatsApp, Email gateways) must support the [A2A Protocol](https://github.com/a2aproject/A2A).
3.  **MCP Clients**: External agents (e.g., Claude, custom co-workers) that wish to access CMS tools must support the [Model Context Protocol](https://modelcontextprotocol.io).

## 1. A2A (Agent2Agent) Configuration

### Update `config.yaml`

Enable A2A and add public keys for trusted external agents:

```yaml
channels:
  a2a_enabled: true
  trusted_keys:
    - id: "whatsapp-gateway"
      public_key: "YOUR_ED25519_PUBLIC_KEY_BASE64URL"
```

### A2A Authentication
External agents must include an Ed25519-signed JWT in the `Authorization: Bearer <JWT>` header. The JWT must include:
- `iss`: Matching the `id` in your `trusted_keys` config.
- `sub`: The user identifier (or `guest`).

### A2A Endpoints
- **JSON-RPC Invoke**: `POST /api/a2a`
- **Agent Card**: `GET /.well-known/a2a-agent-card`

---

## 2. MCP (Model Context Protocol) Configuration

### Update `config.yaml`

Enable the MCP server and configure API keys for external agents:

```yaml
mcp:
  enabled: true
  api_keys:
    - key: "cms_mcp_test_key_123"
      user_id: 1 # System user with "MCP" role
```

### Role Setup
Ensure the `user_id` linked to the API key has the **"MCP" role**. This role gates access to the tools exposed by the MCP server.

### MCP Endpoints
- **SSE Connection**: `GET /api/mcp/sse?apiKey=...`
- **Authentication**: Use `X-API-Key` header or `apiKey` query parameter.

### Exposed Tools
- `list_entities`: Lists all available CMS entities.
- `get_entity_records`: Fetches records for a specific entity.

---

## E-trail Logging
All A2A and MCP interactions are logged in the `__auth_logs` table for non-repudiation, including IP addresses, User Agents, and success status.

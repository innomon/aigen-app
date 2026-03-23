# Specification: Multi-Channel Communication (A2A & MCP Redesign)

## 1. Overview
This specification defines a redesigned multi-channel communication system for `aigen-cms` using the **Agent2Agent (A2A)** protocol and providing external access via the **Model Context Protocol (MCP)**. All channels (WhatsApp, Email, Signal, Telegram) now act as A2A agents, communicating via structured JSON-RPC messages and authenticated using Ed25519 JWTs.

## 2. A2A Communication Layer

### 2.1. Channel as A2A Agent
Each communication channel (WhatsApp, Email, etc.) is treated as an independent A2A agent.
- **Protocol**: A2A (JSON-RPC 2.0 over HTTP/SSE).
- **Authentication**: Ed25519 JWT.
- **Claims**:
    - `iss`: Channel ID (e.g., `whatsapp-gateway`).
    - `sub`: User ID or `guest`.
    - `channel_id`: Unique identifier for the specific channel instance.
    - `iat`, `exp`: Standard time claims.

### 2.2. Trusted Channel Auth
- `aigen-cms` maintains a registry of trusted channel public keys (Ed25519).
- JWTs signed by these keys are accepted for direct user identification or guest session initialization.
- Channels can also request a JWT from the CMS to authenticate outbound requests.

### 2.3. Messaging Flow
- **Inbound**: Channel Agent sends an A2A `sendMessage` request to the CMS.
- **Outbound**: CMS sends an A2A `sendMessage` request to the Channel Agent's gateway URL.
- **Task Management**: Uses A2A `Task` objects to track long-running interactions (e.g., order tracking).

## 3. MCP Server Integration

### 3.1. Purpose
Expose CMS capabilities (entity management, querying, agent tools) to external agents like Claude, ChatGPT, or custom co-workers.

### 3.2. Implementation
- Built using the official **MCP Go SDK** (`github.com/modelcontextprotocol/go-sdk`).
- **Endpoint**: `/api/mcp` (supporting SSE transport).

### 3.3. Authentication & Authorization
- **Mechanism**: API Key (conceptually similar to Gemini API Key).
- **Gating**: Each API Key is linked to a system user with the **"MCP" role**.
- **Role Permissions**: The "MCP" role determines which tools and resources are visible and executable via the MCP server.

## 4. E-trail & Non-Repudiation
- Every A2A message and MCP request is logged in the `__auth_logs` table.
- Logs include: Message Hash, Signature, IP, User Agent, and Timestamp.

## 5. Configuration (`config.yaml`)
```yaml
channels:
  a2a_enabled: true
  trusted_keys:
    - id: "whatsapp-gateway"
      public_key: "..."
mcp:
  enabled: true
  api_keys:
    - key: "cms_mcp_..."
      user_id: 100 # User with "MCP" role
```

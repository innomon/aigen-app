# Auth Log (E-trail) Context

The `auth_log` entity provides a cryptographically grounded audit trail (e-trail) for all interactions occurring via multi-channel protocols.

## Business Purpose
To ensure **non-repudiation**. In an agentic ecosystem where actions can be triggered by external agents (A2A) or authenticated via messaging platforms, it is critical to have an immutable record of who performed what action, when, and from where.

## Scope
Logs are generated for:
- **Channel Logins**: When a user logs in via WhatsApp, Email, etc.
- **A2A Invocations**: Incoming requests from external A2A agents.
- **MCP Tool Execution**: Any tool call made by an external agent using an MCP API Key.

## Data Points Captured
- **User Context**: The system user ID and the external channel identity.
- **Location & Device**: IP address and User Agent.
- **Action Details**: The specific protocol method or tool invoked.
- **Security Metadata**: Nonces, message hashes, or signature verification status.

## Governance
Auth logs are read-only for most users and should be periodically reviewed by a `channel_admin` to identify anomalous agent behavior or unauthorized access attempts.

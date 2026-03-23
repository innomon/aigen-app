# Implementation Plan: A2A & MCP Redesign

## Phase 1: A2A Core Integration
- [ ] Integrate `github.com/a2aproject/a2a-go` SDK.
- [ ] Implement `A2AHandler` to receive A2A messages (JSON-RPC over HTTP).
- [ ] Update `descriptors/channel.go` to include `AgentID` and A2A-specific metadata.
- [ ] Update `ChannelService` to use A2A `sendMessage` and `streamMessage`.
- [ ] Implement Ed25519 JWT verification for trusted A2A channels.

## Phase 2: MCP Server Implementation
- [ ] Setup `api/mcp_api.go` using `github.com/modelcontextprotocol/go-sdk`.
- [ ] Implement `MCPServer` with tool registration (exposed entity CRUD & query tools).
- [ ] Add SSE transport support for MCP.
- [ ] Gate MCP tool execution based on the "MCP" role.

## Phase 3: Authentication & Security
- [ ] Implement API Key management for MCP users.
- [ ] Ensure all A2A and MCP interactions are logged in `__auth_logs` for non-repudiation.
- [ ] Implement guest-to-user promotion flow within A2A tasks.

## Phase 4: Configuration & Initialization
- [ ] Update `framework/config.go` with `A2AConfig` and `MCPConfig`.
- [ ] Initialize A2A and MCP services in `framework/init.go`.
- [ ] Register `/api/a2a` and `/api/mcp` routes.

## Phase 5: Testing & Validation
- [ ] Unit tests for A2A message parsing and JWT verification.
- [ ] Integration tests for MCP server tool execution with API keys.
- [ ] Verify A2A multi-channel message delivery (mocking external A2A agents).

## Checklist
- [ ] `a2aproject/a2a-go` integration
- [ ] `modelcontextprotocol/go-sdk` integration
- [ ] Ed25519 JWT verification for A2A
- [ ] API Key concept for MCP
- [ ] "MCP" role gating
- [ ] Unified `__auth_logs` for A2A & MCP
- [ ] Updated `config.yaml` support

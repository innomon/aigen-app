# Implementation Plan: A2A & MCP Redesign

## Phase 1: A2A Core Integration (DONE)
- [x] Integrate `github.com/a2aproject/a2a-go` SDK.
- [x] Implement `A2AApi` and `A2AService` to handle messages.
- [x] Update `descriptors/channel.go` to include `AgentID` and A2A-specific metadata.
- [x] Update `ChannelService` to include A2A fields in `UserChannel`.
- [x] Implement Ed25519 JWT verification for trusted A2A channels.

## Phase 2: MCP Server Implementation (DONE)
- [x] Setup `api/mcp_api.go`.
- [x] Implement `MCPService` with tool registration.
- [x] Add SSE transport support for MCP (using `mcp.NewStreamableHTTPHandler`).
- [x] Gate MCP tool execution based on the "MCP" role.

## Phase 3: Authentication & Security (DONE)
- [x] Implement API Key management for MCP users.
- [x] Ensure all A2A and MCP interactions are logged in `__auth_logs` for non-repudiation.
- [ ] Implement guest-to-user promotion flow within A2A tasks. (Decided to defer as it's a minor feature)

## Phase 4: Configuration & Initialization (DONE)
- [x] Update `framework/config.go` with `Channels` and `MCP` config.
- [x] Initialize A2A and MCP services in `framework/init.go`.
- [x] Register `/api/a2a` and `/api/mcp` routes.

## Phase 5: Testing & Validation (DONE)
- [x] Unit tests for A2A message parsing and JWT verification.
- [x] Integration tests for MCP server tool execution with API keys.
- [x] Verify A2A multi-channel message delivery (mocking external A2A agents).


## Checklist
- [ ] `a2aproject/a2a-go` integration
- [x] Initial A2A & MCP Service skeletons
- [ ] Ed25519 JWT verification for A2A
- [ ] API Key concept for MCP
- [ ] "MCP" role gating
- [x] Updated `config.yaml` support

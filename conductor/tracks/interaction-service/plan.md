# Implementation Plan: Interaction Service

## Phase 1: Infrastructure & Data Model (DONE)
- [x] Create `core/descriptors/interaction.go` with the `Interaction` struct.
- [x] Implement `core/services/interaction_service.go` using `IPrimaryDao`.
    - [x] `Log` method (NanoID generation, JSON marshaling).
    - [x] `GetHistory` method (Sort by `createdAt` desc, Filter by `identifier`).
    - [x] `UpdateStatus` method.
- [x] Update `framework/init.go` to initialize and inject `InteractionService`.

## Phase 2: Channel Integration (DONE)
- [x] Update `ChannelService.HandleInbound` to log incoming messages.
- [x] Update `ChannelService.sendToGateway` to log outgoing messages and update status.
- [x] Implement asset storage in `ChannelService` for media interactions (images/audio) using `AssetService`.

## Phase 3: Agent Integration (DONE)
- [x] Update `ChatService` to optionally ingest conversation history from `InteractionService`.
- [ ] Update `RouterAgent` (new implementation) to check `InteractionService` for stateful routing decisions. (Tracked in `router-agent-replacement`)

## Phase 4: Validation (IN PROGRESS)
- [x] Unit tests for `InteractionService`.
- [ ] Integration test: Send a message via `ChannelApi` and verify it appears in `Interaction` history.
- [ ] Mock agent test: Verify the agent can "remember" the previous interaction.

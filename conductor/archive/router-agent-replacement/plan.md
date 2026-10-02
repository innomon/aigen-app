# Implementation Plan: Router Agent Replacement

## Phase 1: Research and Adaptation (DONE)
- [x] Analyze `IPrimaryDao` to find the best way to store transient routing state. (Leveraged `InteractionService`).
- [x] Map `whatsadk`'s `RouterConfig` and `ClassifierConfig` to `aigen-app`.
- [x] Design the prompt templates for intent classification.

## Phase 2: Implementation (DONE)
- [x] **Step 1: Refactor `RouterAgentConfig`**: Updated in `core/agentic/agents/router_agent.go`.
- [x] **Step 2: Implement Persistence Layer**: Integrated with `InteractionService`.
- [x] **Step 3: Port Routing Logic**: Implemented stateful routing and LLM classification.
- [x] **Step 4: Update Registry**: Updated `RegisterRouterAgent` to accept `InteractionService`.

## Phase 3: Testing and Validation (DONE)
- [x] Create integration tests for the new router agent in `core/services/chat_service_test.go`.
- [x] Verify keyword-based routing.
- [x] Verify stateful routing (prompting for app selection).
- [x] Verify LLM-based classification with a test model.
- [x] Perform build verification.


## Phase 4: Cleanup (DONE)
- [x] Remove old keyword matching logic (Integrated into new logic).
- [x] Update documentation (Managed via track updates).


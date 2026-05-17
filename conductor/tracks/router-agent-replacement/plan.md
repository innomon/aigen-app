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

## Phase 3: Testing and Validation (IN PROGRESS)
- [ ] Create unit tests for the new router agent using a mock DAO.
- [ ] Verify keyword-based routing.
- [ ] Verify stateful routing (prompting for app selection).
- [ ] Verify LLM-based classification with a test model.
- [x] Perform build verification.

## Phase 4: Cleanup (TODO)
- [ ] Remove old keyword matching logic (Integrated into new logic).
- [ ] Update documentation.

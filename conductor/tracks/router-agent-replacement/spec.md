# Specification: Router Agent Replacement

## Goal
Replace the current simplistic keyword-based router agent in `core/agentic/agents/router_agent.go` with a more advanced routing logic inspired by the `whatsadk` router example.

## Current Implementation
- Located at `core/agentic/agents/router_agent.go`.
- Uses simple keyword matching ("data", "ui") to route to sub-agents.
- Fallback is the first sub-agent.
- Registered as "router" in `registry`.

## Proposed Features (from whatsadk)
- **User-Specific Routing**: Route based on user configuration stored in the database.
- **Stateful Routing**: Support multi-turn routing (e.g., asking user to select an app).
- **LLM Classification**: Use an LLM to classify user intent when simple matching fails.
- **Integration with A2A**: Ability to route to remote agents via URL if needed (optional for first phase).
- **Persistent State**: Use `IPrimaryDao` to store user-specific app lists and routing state.

## Architecture Changes
- Update `RouterAgentConfig` to include classifier settings and prompts.
- Implement `routerRun` logic that uses `IPrimaryDao` for persistence.
- Adapt `whatsadk` logic to use `aigen-app`'s `agent.InvocationContext` and `session.Event`.
- Ensure compatibility with existing `agentic.yaml` configuration.

## Dependencies
- `IPrimaryDao` for state and configuration persistence.
- `registry` for sub-agent lookup and registration.
- LLM provider (Gemini or OpenAI) for classification.

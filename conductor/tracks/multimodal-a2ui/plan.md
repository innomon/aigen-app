# Multimodal Chat & Agentic Integration Plan

## Objective
Implement a Multimodal Chat interface powered by the `agentic` framework with A2UI component rendering, exposing core CMS capabilities via function calling, managed by a deterministic router agent.

## Checklist

### 1. Framework Integration
- [ ] **Import Agentic Dependency**: Update `go.mod` in `AiGen CMS` to include `github.com/innomon/agentic`. We MUST NOT make any modifications to the `agentic` project's source code.
- [ ] **Agent Config Initialization**: Set up `agentic` config loading (YAMLs) and registry initialization in `core/apps/setup.go` or a dedicated `chat_service.go`.

### 2. Function Tool Integration
- [ ] **Define CMS Tool Handlers**: (`core/agentic/tools/cms_tools.go`)
    - [ ] Create `ToolHandler` wrappers for `EntityService` (List, Get, Create, Update, Delete).
    - [ ] Create `ToolHandler` wrappers for `SchemaService` (GetSchema, ListSchemas).
    - [ ] Create `ToolHandler` wrappers for `A2UIService` (RenderComponent).
- [ ] **Register Tools**:
    - [ ] Import `github.com/innomon/agentic/pkg/registry` and invoke `RegisterToolHandler` for each CMS capability during application startup.

### 3. Deterministic Router Agent
- [ ] **Implement Router Agent**: (`core/agentic/agents/router_agent.go`)
    - [ ] Create a custom Go struct implementing the `agentic` Agent-like interface/behavior.
    - [ ] Implement the routing logic (regex/intent-based matching to forward prompts to registered LLM sub-agents).
- [ ] **Register Router Agent**: Register this Go struct into the `agentic` agent registry so that it can be retrieved generically.

### 4. API & Backend Wiring
- [ ] **Create Chat Controller**: (`core/api/chat_api.go`)
    - [ ] Add `/api/chat/message` (POST, Multimodal support).
    - [ ] Add `/api/chat/stream` (SSE) endpoint to push agent responses.
- [ ] **Create Chat Service**: (`core/services/chat_service.go`)
    - [ ] Handle session context, chat history management.
    - [ ] Invoke the Deterministic Router Agent with the incoming message and stream the output back.

### 5. Frontend UI Development
- [ ] **Implement Chat Interface**: (`core/api/ui/chat.html`, `core/api/ui/js/chat/app.js`)
    - [ ] Build a Multimodal input field (supporting file drops/uploads).
    - [ ] Implement SSE Listener for the chat stream.
- [ ] **Integrate A2UI Renderer**: (`core/api/ui/js/chat/renderer.js`)
    - [ ] Intercept messages containing A2UI JSON payload.
    - [ ] Dynamically render the components using the `A2UI Component Catalog` within the chat bubble.

## Deliverables
1. **Registered Go Tools**: Core services exposed to the `agentic` framework.
2. **Deterministic Router Agent**: Go-based root agent for intelligent delegation.
3. **Multimodal A2UI Chat Interface**: A fully functioning frontend capable of interacting with agents and rendering dynamic UI components.

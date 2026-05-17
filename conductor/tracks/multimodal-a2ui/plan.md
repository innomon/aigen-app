# Multimodal Chat & Agentic Integration Plan

## Objective
Implement a Multimodal Chat interface powered by the `agentic` framework with A2UI component rendering, exposing core CMS capabilities via function calling, managed by a deterministic router agent.

## Checklist

### 1. Framework Integration (DONE)
- [x] **Import Agentic Dependency**: Updated `go.mod`.
- [x] **Agent Config Initialization**: Set up `agentic` config loading and registry initialization in `framework/init.go`.

### 2. Function Tool Integration (DONE)
- [x] **Define CMS Tool Handlers**: (`core/services/cms_tools.go`)
    - [x] Create `ToolHandler` wrappers for `EntityService`.
    - [x] Create `ToolHandler` wrappers for `SchemaService`.
    - [x] Create `ToolHandler` wrappers for `A2UIService`.
- [x] **Register Tools**: Tools are registered during application startup.

### 3. Deterministic Router Agent (IN PROGRESS)
- [x] **Implement Router Agent**: Initial implementation in `core/agentic/agents/router_agent.go`.
- [ ] **Advanced Routing**: Plan for replacement with stateful/LLM-based router (see `router-agent-replacement` track).
- [x] **Register Router Agent**: Registered in the `agentic` agent registry.

### 4. API & Backend Wiring (DONE)
- [x] **Create Chat Controller**: (`core/api/chat_api.go`)
    - [x] Add `/api/chat/message`.
- [x] **Create Chat Service**: (`core/services/chat_service.go`)
    - [x] Handle session context, chat history management.
    - [x] Invoke the Deterministic Router Agent.

### 5. Frontend UI Development (IN PROGRESS)
- [ ] **Implement Chat Interface**: (`core/api/ui/chat.html`, `core/api/ui/js/chat/app.js`)
- [x] **Implement A2UI Service**: (`core/services/a2ui_service.go`)
- [x] **Implement A2UI API**: (`core/api/a2ui_api.go`)
- [ ] **Integrate A2UI Renderer**: Dynamically render components using the `A2UI Component Catalog`.

## Deliverables
1. **Registered Go Tools**: Core services exposed to the `agentic` framework.
2. **Deterministic Router Agent**: Go-based root agent for intelligent delegation.
3. **Multimodal A2UI Chat Interface**: A fully functioning frontend capable of interacting with agents and rendering dynamic UI components.

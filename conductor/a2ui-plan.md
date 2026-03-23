# A2UI Implementation Plan

## Objective
Implement a fully functional A2UI-compatible streaming UI layer that allows the backend to generate and update administrative interfaces dynamically.

## Checklist

### 1. Foundation (Backend)
- [ ] **Create `A2UIService`**: (`core/services/a2ui_service.go`)
    - [ ] Define `A2UIComponent` and `A2UIMessage` structs in Go.
    - [ ] Implement `BuildAdjacencyList` helper to generate the flat JSON array.
    - [ ] Manage a basic state store for UI components (start with a simple map).
- [ ] **Implement SSE Handler**: (`core/api/a2ui_api.go`)
    - [ ] Add `/api/a2ui/stream` endpoint with `Content-Type: text/event-stream`.
    - [ ] Implement channel-based streaming to push `A2UIMessage` as JSON.
- [ ] **Implement Action Handler**:
    - [ ] Add `/api/a2ui/action` POST endpoint.
    - [ ] Parse `userAction` payload and trigger updates via the service.

### 2. Renderer (Frontend)
- [ ] **Develop `a2ui-renderer.js`**: (`core/api/ui/js/a2ui/renderer.js`)
    - [ ] Implement `SSEListener` to handle incoming JSON messages.
    - [ ] Build `ComponentStore` to manage the local adjacency list.
    - [ ] Create `renderRecursive(id)` to traverse and build the DOM tree.
- [ ] **Build Component Catalog**: (`core/api/ui/js/a2ui/catalog.js`)
    - [ ] Map `Text`, `Heading`, `Button`, `TextField`, `Card`, `Column`, `Row` to Bootstrap 5 templates.
    - [ ] Implement "Signal Dispatcher" to send `userAction` back to the backend.

### 3. Integration & Testing
- [ ] **Create Prototype Page**: (`core/api/ui/a2ui_test.html`)
    - [ ] Add a `surface-root` div.
    - [ ] Initialize the A2UI Renderer.
- [ ] **Verify Interaction**:
    - [ ] Confirm backend can push a "Counter" component.
    - [ ] Verify clicking a button sends an action to the backend.
    - [ ] Verify the backend updates the counter and streams a delta back.
- [ ] **Refine CMS Use Case**:
    - [ ] Implement an "Agent Dashboard" using A2UI that shows recent system activity (Audits/Logins).

## Deliverables
1. **A2UI Protocol Layer**: Core Go and JS libraries for Adjacency List handling.
2. **Streaming Backend**: SSE-based real-time UI updates.
3. **Interactive Demo**: A functional "Counter" or "Form" example rendered entirely via the protocol.

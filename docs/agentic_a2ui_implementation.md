# AiGen CMS: Agentic SDK, Chat, and A2UI Implementation Guide

## 1. Overview
The AiGen CMS framework incorporates an AI-driven architecture to enhance how users interact with the CMS data and dashboard. By leveraging the **Agentic Development Kit (ADK)** (`google.golang.org/adk/agent`), the application implements a multi-agent system that seamlessly blends natural language chat interfaces with rich, dynamic User Interfaces generated on the fly (A2UI).

## 2. Agentic SDK (ADK) Implementation

The agentic capabilities are built using a declarative configuration approach backed by a Go-based registry and execution engine.

### 2.1. Agent Configuration (`agentic.yaml`)
The core of the multi-agent system is defined in `agentic.yaml`. This file dictates the models, available tools, and the agent hierarchy.

- **Models**: Defines the underlying LLM (e.g., `gemini-2.5-pro` via the `gemini` provider).
- **Agents**:
  - **Router Agent (`router_agent`)**: The root agent responsible for triaging incoming user requests and delegating them to the appropriate specialized sub-agent.
  - **CMS Agent (`cms_agent`)**: A specialized LLM agent instructed to act as the CMS data assistant. It has access to tools like `cms_entity_list`, `cms_entity_get`, `cms_entity_create`, and `cms_schema_list`.
  - **UI Agent (`ui_agent`)**: A specialized LLM agent instructed to manage the A2UI dashboard. It uses the `cms_a2ui_update` tool to dynamically render or modify UI components.
- **Tools**: Built-in functions registered in Go that the LLM can invoke to interact with the underlying `EntityService` and `SchemaService`. Key tools include:
  - `cms_entity_list`, `cms_entity_get`, `cms_entity_create`: For data manipulation.
  - `cms_schema_list`: For listing available schemas.
  - `cms_app_list`, `cms_app_get`: For exploring installed apps, their definitions, roles, and entity contexts.
  - `cms_a2ui_update`: For modifying the frontend A2UI state.

### 2.2. The Router Agent (`router_agent.go`)
The `RouterAgent` is a custom Go implementation registered with the ADK `registry`. It implements a lightweight, keyword-based intent classification system.

- **Execution Flow**: When a user sends a message, the `RouterAgent` analyzes the text (`ic.UserContent()`).
- **Intent Classification**:
  - If the prompt contains keywords like "data", "schema", "entity", or "record", it routes the request to the `cms_agent`.
  - If the prompt contains keywords like "ui", "component", "dashboard", or "view", it routes the request to the `ui_agent`.
- **Delegation**: Once the target sub-agent is selected, the router delegates the execution context (`targetAgent.Run(ic)`) and yields the events back to the session.

### 2.3. Initialization (`chat_service.go`)
The `ChatService` serves as the bridge between the standard Go web backend and the ADK. 
Upon initialization (`NewChatService`), it:
1. Registers the custom `RouterAgent` type.
2. Registers the specific CMS tools bound to the `EntityService`, `SchemaService`, and `A2UIService`.
3. Loads the `agentic.yaml` configuration into the `registry`.
4. Initializes an in-memory session service to maintain chat history.

---

## 3. Chat Features

The chat system is the primary entry point for AI interactions. Instead of relying on a single monolithic LLM prompt, the chat is driven by the **Router -> Sub-Agent** architecture.

### Conversational Workflows
1. **Data Operations**: A user asks, "Show me the latest CRM leads." The `RouterAgent` forwards this to the `cms_agent`. The `cms_agent` invokes the `cms_entity_list` tool, retrieves the data, and formulates a natural language response.
2. **UI Generation**: A user asks, "Create a dashboard view for the leads." The router forwards this to the `ui_agent`. The `ui_agent` invokes the `cms_a2ui_update` tool, which pushes a new UI component structure to the frontend.

By separating the "Data" persona from the "UI" persona, the system maintains strict boundaries and prevents the LLM from hallucinating UI components when the user only wants raw data.

---

## 4. Agent-to-User Interface (A2UI)

A2UI is a protocol that allows the backend Agent to stream rich, interactive UI structures as JSON data to the frontend. This bypasses traditional static templating, allowing the AI to generate bespoke interfaces based on user intent.

### 4.1. The Adjacency List Model
A2UI utilizes a flat **Adjacency List** data structure rather than deeply nested JSON. This makes it highly efficient for streaming deltas and updating specific nodes.

```go
type A2UIComponent struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`       // e.g., "Card", "TextField", "Button"
	Attributes map[string]interface{} `json:"attributes,omitempty"`
	Children   []string               `json:"children,omitempty"` // IDs of child components
}
```

### 4.2. State Management (`a2ui_service.go`)
The `A2UIService` acts as the definitive "Source of Truth" for the active dashboard UI.

- **Pub/Sub via SSE**: The service uses Go channels (`s.subscribers`) and `sync.RWMutex` to manage state safely across concurrent connections.
- **Streaming Updates**: When the `ui_agent` modifies a component via the `cms_a2ui_update` tool, the `UpdateComponent` method is called. This updates the internal component map and immediately broadcasts the updated state array to all active Server-Sent Events (SSE) subscribers.

### 4.3. The Interaction Loop
The integration between the Frontend (Renderer), Backend (Agent), and Database (Entity Service) follows a strict Producer-Consumer model:

1. **Rendering**: The frontend JavaScript connects to `/api/a2ui/stream`. It maintains a local Component Map and recursively renders the UI using a trusted Bootstrap 5 component catalog.
2. **User Action**: The user interacts with the generated UI (e.g., clicks "Save Lead").
3. **Action Signal**: The frontend sends a JSON payload `{ componentId, actionType, data }` to the `/api/a2ui/action` endpoint.
4. **Agent Processing**: The backend processes this action (potentially invoking the `EntityService` to save the data).
5. **UI Delta Update**: The `A2UIService` pushes any resulting UI updates (e.g., showing a success message or updating a chart) back through the SSE stream, seamlessly updating the user's view.

### 4.4. Security and RBAC
The A2UI layer is "security-aware". When the Agent fetches data to populate an A2UI component (like a `DataTable`), it executes through the `EntityService` using the user's authenticated context. This ensures that all Role-Based Access Control (RBAC), row-level filters, and field-level permissions are strictly enforced before the data ever reaches the UI generation phase. Client-side execution of arbitrary code is prevented by restricting rendering to a pre-defined, trusted component catalog.

---

## 5. App Capability Discovery

To allow the Agent to dynamically discover and understand the purpose, roles, and entities of installed applications, AiGen CMS utilizes an expanded App Definition framework.

### 5.1. The `app_def.json` Specification
Each application can define an `app_def.json` file in its root directory (e.g., `apps/erpnext_accounting/app_def.json`). This file acts as a manifest that provides the LLM with deep context:
- `name`: The system name of the app.
- `display_name`: Human-readable name.
- `description`: A short summary of the app's purpose.
- `context`: A longer, detailed explanation of the app's business domain and usage.
- `roles`: An array of applicable roles for the app (e.g., `["System Manager", "Auditor"]`).
- `entities`: A mapping of entity schemas to their descriptions and, optionally, a `context_file`.

### 5.2. Context Files
Entities defined in `app_def.json` can point to external Markdown context files via the `context_file` property (e.g., `docs/account.md`). These files contain detailed business rules, relationships, or specific instructions on how to handle the entity, which is directly injected into the LLM's context window.

### 5.3. Discovery Tools
The Agent discovers this information through two primary tools:
1. **`cms_app_list`**: Reads the global `apps.json` to find enabled apps, then parses each app's `app_def.json` to return a summarized list of available applications.
2. **`cms_app_get`**: Retrieves the full definition of a specific app. Crucially, this tool automatically traverses the `entities` mapping, reads any referenced `context_file`s from the disk, and embeds their raw text directly into the response payload. This allows the LLM to acquire comprehensive entity knowledge in a single tool call without manual file system navigation.

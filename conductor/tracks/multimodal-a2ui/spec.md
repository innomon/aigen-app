# Multimodal Chat & Agentic Integration Specification

## Overview
This specification outlines the integration of a Multimodal Chat interface with A2UI capability into `AiGen CMS`. At its backend, it integrates the Application Development Kit (ADK) using the `agentic` framework from `/home/innomon/orez/agentic`. 
The system relies on configurable agent creation, internal capabilities mapped via function calling, and a deterministic Go-based router agent to delegate traffic to specialized sub-agents.

## Core Architectural Components

### 1. The Agentic ADK Backend
- **Dependency**: The `agentic` module will be imported from `github.com/innomon/agentic`.
- **No Modifications**: We MUST NOT modify the `agentic` project's source code; all integration must be done via its public API and registry.
- **Configurable Agent Creation**: The backend will utilize the `agentic` configuration YAMLs and registry to instantiate agents at runtime.
- **Agent Registry**: Agents and tools are registered natively in the `agentic` registry.

### 2. Internal Capabilities as Function Calls
To bridge the LLM capabilities with the core CMS services (`EntityService`, `SchemaService`, `A2UIService`), we will register custom function tools in the `agentic` registry.
- **Go Custom Tool Definition**: We will define `ToolHandler` implementations (matching `func(ctx context.Context, args map[string]any) (any, error)`) that wrap our internal service calls.
- **Registration**: Use `registry.RegisterToolHandler(name, handler)` from `github.com/innomon/agentic/pkg/registry` to make these core capabilities accessible to the LLMs through function calling.

### 3. Deterministic Router Agent
- **Purpose**: A specialized, highly predictable Go-based agent that acts as the entry point for chat requests.
- **Implementation**: Written in Go (not LLM-based), evaluating inputs based on intent, metadata, and routing logic to pick the correct LLM sub-agent (e.g., "Data Query Agent", "Schema Design Agent").
- **Registration**: This custom agent will be registered into the `agentic` registry to act seamlessly as the root node of the agent hierarchy.

### 4. Multimodal Chat & A2UI Frontend
- **Interface**: A chat UI supporting text and attachments (multimodal).
- **A2UI Streaming**: The backend responds with a mix of text messages and A2UI Adjacency List payload (JSON array of UI components). 
- **Renderer**: The frontend will use the existing A2UI Renderer (`a2ui-renderer.js`) to display rich interactive responses (e.g., DataTables, Charts, Forms) natively inside the chat thread.

## Interaction Flow
1. User sends a chat message (with optional file attachments).
2. The Chat API endpoint `/api/chat/stream` receives the message.
3. The request is passed to the **Deterministic Router Agent**.
4. The router identifies the correct **Sub-Agent** and forwards the prompt.
5. The Sub-Agent uses **Function Calling** (accessing our registered Go tools) to query/modify CMS data.
6. The Sub-Agent formulates a response, optionally structuring it as an **A2UI Payload**.
7. The response is streamed back to the frontend via Server-Sent Events (SSE).
8. The frontend displays the chat bubble and renders the embedded A2UI components dynamically.

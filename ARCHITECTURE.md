# AIGenApp Architecture

This document describes the high-level architecture of AIGenApp, its core concepts, and how different components interact.

## Core Philosophy: Schema-on-Read

Unlike traditional CMS or ERP systems that create physical database tables for each entity, AIGenApp uses a **Single-Table JSON Store** approach. 

- **Persistence Layer**: All data is stored in a single table (e.g., `aigen_records` in Postgres or a single collection in Firestore).
- **Structure**: Each record consists of a Namespace (BizDef), a Key (Unique ID), a JSON blob (`Rec`), and Metadata.
- **Flexibility**: Changes to schemas do not require SQL migrations. The application logic interprets the JSON data based on dynamic definitions.

---

## Key Concepts

### 1. BizDefs (Business Definitions)
A **BizDef** in AIGenApp is a logical container for business logic, schemas, and data.
- **Discovery**: Defined in `bizdefs/` directories via a `bizdef.json` manifest.
- **Components**: A BizDef includes its own schemas, default data, UI configurations, and documentation.
- **Isolation**: Roles and permissions are typically scoped to a BizDef namespace.

### 2. Entities & Schemas
Data within a BizDef is organized into **Entities**.
- **Schema**: Defined in JSON files (e.g., `crm_lead.json`). These schemas describe attributes, data types, validation rules, and UI hints (e.g., "this field is an email").
- **Dynamic Modeling**: Entities are "realized" at runtime. The `SchemaService` parses these definitions to provide CRUD operations and GraphQL types.

### 3. Channels
**Channels** represent the communication interfaces between the system and the outside world.
- **Multi-Channel**: Supports WhatsApp, Email, Signal, Telegram, etc.
- **Identity**: Channels map external platform IDs (e.g., a phone number) to internal User profiles.
- **Guest Support**: Each channel can be configured to allow or deny unauthenticated interactions.

### 4. Services (The Business Logic Layer)
Services are the heart of the system. They are responsible for business logic, data validation, and orchestration.
- **Stateless**: Services generally do not hold state themselves; they rely on DAOs for persistence.
- **Isolation**: Services should not know about HTTP or specific API details. They take a `context.Context` and specialized descriptors as input.
- **Examples**: `EntityService`, `AuthService`, `ChannelService`, `TempAccessService`.

### 5. APIs (The Transport Layer)
APIs handle the "outer shell" of the application.
- **Routing**: Built on `net/http` and `chi`.
- **Handling**: APIs parse HTTP requests (JSON, Multipart, Query params), enforce basic authentication (JWT Middleware), and delegate work to Services.
- **MCP API**: A specialized management API for administrative tasks (like clearing caches or managing temporary files).

---

## Architectural Layers

```mermaid
graph TD
    User((User/Client)) --> API[API Layer: chi Router]
    API --> Service[Service Layer: Business Logic]
    Service --> DAO[DAO Layer: Database Abstraction]
    Service --> Filestore[Filestore Layer: S3/GCS/Local]
    DAO --> Postgres[(PostgreSQL)]
    DAO --> Firestore[(Firestore)]
    Filestore --> Cloud[(Cloud Storage)]
    
    subgraph Core Concepts
        BizDef[BizDef Manifests]
        Schema[JSON Schemas]
        Channel[Communication Channels]
    end
    
    Service -.-> BizDef
    Service -.-> Schema
    Service -.-> Channel
```

### Infrastructure Abstraction
AIGenApp is designed to be cloud-agnostic:
- **`IPrimaryDao`**: Abstracts the database (Postgres, Firestore, or Memory).
- **`IFileStore`**: Abstracts the storage (Local, S3, GCS, or Postgres-BLOB).

## Relationship & Data Flow

1. **Request**: A user sends a request to an API endpoint (e.g., `GET /api/entity/crm/lead/123`).
2. **Auth**: The `AuthMiddleware` verifies the JWT and populates the context with the user's role and identity.
3. **Logic**: The `EntityApi` calls the `EntityService`.
4. **Schema Check**: The `EntityService` consults the `SchemaService` to ensure the requested entity (`crm/lead`) is valid and the user has permission to read it.
5. **Fetch**: The `EntityService` calls the `IPrimaryDao` to retrieve the JSON record from the database.
6. **Transformation**: The data is mapped to the requested format (JSON or GraphQL) and returned up the chain.

## Extension Points

- **Downstream BizDefs**: Add new business domains by creating a folder in `bizdefs/`.
- **Custom Tools**: Extend the agentic capabilities by adding tools to `core/agentic/tools/`.
- **New Channels**: Implement the `ChannelService` patterns for new messaging platforms.

## BizDef Integration & Invocation

BizDefs (like `bizdefs/crm`) are not standalone executables, but rather collections of business logic, schemas, and metadata managed by the system's core services.

### Integration Flow
1. **Manifest Registration**: Each BizDef defines its capabilities, entities, and context in a `bizdef.json` manifest.
2. **Schema Discovery**: Upon initialization, the `SchemaService` automatically scans the `bizdefs/` directory to discover these manifests and their associated JSON schemas (e.g., `bizdefs/crm/schemas/crm_lead.json`).
3. **Data Modeling**: The system registers these entities into memory, allowing the `EntityService` to perform CRUD operations on the BizDef's entities using the standard JSON store pattern.
4. **Agent-Awareness**: The `cms_agent` is "BizDef-aware." When a user asks a question, the agent uses tools like `cms_bizdef_list` and `cms_bizdef_get` to introspect the registered BizDef manifests, allowing it to understand the business domain and translate natural language into specific entity queries.
5. **Polymorphic Execution**: Because the backend uses a single-table architecture, BizDef "invocation" is effectively the dynamic routing of requests through the `EntityService` to the appropriate entity table as defined in the BizDef's schema.

## Agentic Interaction

Users interact with the AI-driven agentic framework through the `Chat API` endpoint. This allows for natural language interaction with defined downstream BizDefs and tools.

### Interaction Flow
1. **Request**: A user sends a POST request to `/api/chat/message` with their prompt.
2. **Entry**: The `ChatApi` layer authenticates the request and delegates the message to the `ChatService`.
3. **Session**: The `ChatService` loads the user's session and historical context from the `InteractionService`.
4. **Agent Orchestration**:
    - The `ChatService` invokes the `router_agent` (the root agent defined in `agentic.yaml`).
    - The `router_agent` analyzes the input using heuristics or a configured LLM (`gemma4_26b`) to identify the appropriate downstream `cms_agent` or `ui_agent`.
5. **Execution**: The chosen sub-agent executes, potentially invoking registered tools (implemented in `core/services/cms_tools.go`) to fetch data from the CMS or update the UI dashboard.
6. **Response**: The agent's final output is streamed back, packaged as a JSON response, and returned to the user via the `ChatApi`.

```mermaid
sequenceDiagram
    participant User
    participant API as ChatApi
    participant Service as ChatService
    participant Router as RouterAgent
    participant Sub as Sub-Agent (CMS/UI)
    participant Tools as Tool Registry

    User->>API: POST /api/chat/message
    API->>Service: ProcessMessage
    Service->>Router: Execute (Run)
    Router->>Router: Analyze Intent
    Router->>Sub: Route to Agent
    Sub->>Tools: Invoke Tool
    Tools->>Sub: Tool Result
    Sub->>Router: Final Response
    Router->>Service: Return Response
    Service->>API: Return Response
    API->>User: JSON Response
```

## UI Architecture

AIGenApp provides a hybrid interface that blends traditional structured administration with modern, conversational AI interactions.

- **Dynamic Dashboards**: The primary administrative interface (`dashboard.html`) is role-aware and highly customizable. It dynamically renders pages based on a user's assigned role, allowing for tailored workflows and specialized dashboards defined by system administrators.
- **Entity Administration**: Provides standard, form-based interfaces for direct entity and user management (`entity_list.html`, `entity_edit.html`), ensuring complete control over system data.
- **Conversational Agent (Chat UI)**: The chat interface (`chat.html`) serves as an agent-led, conversational alternative to the structured UI. It allows users to query data, trigger actions, and manage entities using natural language via the AI-driven agentic framework.

Users are free to navigate between these structured dashboards and the conversational chat assistant depending on their current workflow requirements.

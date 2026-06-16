# Specification: ADK to App Gateway (adk2app)

## Background
To integrate `aigen-app` with the WhatsApp gateway (`whatsadk`), the application needs to act as an ADK-compatible server. The gateway will forward WhatsApp chat events as standard ADK REST requests (session initialization, agent runs, etc.). To secure these interactions, all requests must be authenticated using the gateway's RS256 JWT signature, mapped to the correct user in `aigen-app`, and authorized against the RBAC framework before invoking the requested agent.

## Objectives
1. **ADK-Compliant REST Interface**: Expose endpoints matching the ADK REST API specification under `/api/adk2app/*`.
2. **Gateway Authentication**: Verify incoming `RS256` JWT tokens in the `Authorization` header using the gateway's configured public key.
3. **User mapping & Registration**: Map the gateway's `user_id` claim (phone number) to the internal `aigen-app` user ID and roles, creating a user and channel linkage dynamically if permitted.
4. **RBAC Validation**: Restrict execution based on the requested `appName` (the sub-agent or extension name) and user roles.
5. **Request Execution**: Load the correct agent (sub-agent or app extension) and execute it using the ADK runner, supporting both standard JSON and SSE streaming responses.

## Scope & Design

### 1. API Endpoints
The gateway expects the following standard ADK routes, which we will expose under the `/api/adk2app` prefix:
- `POST /api/adk2app/apps/{appName}/users/{userID}/sessions/{sessionID}`: Verify or create a session.
- `POST /api/adk2app/run`: Execute an agent turn (synchronous).
- `POST /api/adk2app/run_sse`: Execute an agent turn with SSE streaming response.

### 2. Authentication Flow
- Extract the Bearer token from the `Authorization` header.
- Use the gateway public key (`gatewayPub` loaded from `ChannelConfig`) to verify the token signature.
- Verify claims:
  - `exp`: Expiry time (enforce short TTL).
  - `channel`: Must be `"whatsapp"`.
- Verify that the `user_id` claim matches the requested `userID` (or `user_id` path parameter in session endpoints) to prevent impersonation.

### 3. User Resolution & RBAC check
- Resolve the internal `aigen-app` user using the phone number (`claims.UserID`) via `authService.LoginByChannel`. This automatically handles user registration, channel linking, and audits.
- Parse the resulting internal token using `authService.ValidateToken` to obtain the internal `userId` and `roles`.
- Perform the RBAC check:
  ```go
  hasAccess, err := permissionService.HasAccess(ctx, userId, roles, appName, "read")
  ```
- Reject the request with `403 Forbidden` if the user is unauthorized.

### 4. Integration with ADK REST Server (`adkrest`)
Instead of duplicating execution logic, we will leverage the official ADK REST server implementation in `google.golang.org/adk/server/adkrest`.
- Implement a custom `agent.Loader` that resolves agents dynamically:
  - If the agent exists in the main `ChatService.Registry`, load it.
  - If the agent is an app extension, load its config, build the extension registry, and return its root agent.
- Initialize the `adkrest.Server` using this custom loader and the shared `SessionService` from `ChatService`.
- Mount the server as a handler under `/api/adk2app` using `http.StripPrefix`.

```
[WhatsADK Gateway]
       │
       │ (RS256 JWT)
       ▼
[Chi Router: /api/adk2app/*]
       │
       ├─► [Auth & RBAC Middleware]
       │     ├─ Verify Gateway JWT (RS256)
       │     ├─ Resolve user via LoginByChannel
       │     └─ Verify RBAC access to appName
       │
       ▼ (if valid)
[http.StripPrefix]
       │
       ▼
[ADK REST Server (adkrest)] ◄──► [Custom Agent Loader]
       │                                │
       ▼                                ├─► Check main registry
[Agent Runner (runner.Runner)] ◄────────┴─► Check App Extension
```

## Configuration
The public key is configured under `channels.whatsapp.public_key` in the `config.yaml` file, which is already loaded into `WhatsAppService`.

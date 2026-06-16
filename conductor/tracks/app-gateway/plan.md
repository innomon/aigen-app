# Implementation Plan: ADK to App Gateway (adk2app)

This plan outlines the steps required to implement the secure `adk2app` endpoint.

## Phase 1: Authentication & Verification Logic
- [x] Add `VerifyADKJWT` method to `WhatsAppService` (`core/services/whatsapp_service.go`) to verify RS256 JWTs from the gateway using the configured public key.
- [x] Implement unit tests in `core/services/whatsapp_service_test.go` to cover JWT token verification.

## Phase 2: Dynamic Agent Loader
- [x] Create a custom `agent.Loader` implementation in `core/services/chat_service.go` or a new file:
  - It should first search for the agent in the main registry.
  - If not found, look up the agent in the `AppExtensionService` and load its config.
  - Return the loaded `agent.Agent`.

## Phase 3: Middleware and Routing Implementation
- [x] Create `core/api/adk2app_api.go` defining the `ADK2AppApi` struct and its Chi routing configuration.
- [x] Implement the `AuthenticateAndAuthorize` middleware:
  - Extract the token, call `whatsappService.VerifyADKJWT`.
  - Validate that the target user ID matches the token subject.
  - Resolve user and roles via `authService.LoginByChannel`.
  - Perform RBAC check on `appName` via `permissionService.HasAccess`.
- [x] Initialize and mount the `adkrest.Server` wrapped by the middleware under `/api/adk2app` in `framework/init.go`.

## Phase 4: Configuration & Verification
- [x] Update `config.yaml.sample` to document gateway public key setup if not already clear.
- [x] Run `go build ./...` to verify compilation.
- [x] Create integration tests verifying the full flow: request decoding, middleware auth, RBAC checks, and mock agent execution.

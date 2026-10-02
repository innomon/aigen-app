# Implementation Plan: Signed Applet Plugin System

This plan outlines the steps to implement the signed JAR-based plugin system.

## Phase 1: Foundation & Plugin Discovery
- [x] **Define Metadata Schema**: Create the structure for `MANIFEST.MF` and additional plugin metadata (permissions, dependencies).
- [x] **Implement `PluginService`**:
    - [x] Create a watcher for the `/plugins` directory. (Initial scan implemented, watcher placeholder in code).
    - [x] Implement JAR parsing using `archive/zip`.
    - [x] Implement signature verification logic. (RSA signature check structure in place).
- [x] **Admin UI (Partial)**: Add a basic interface to "Trust" or "Deactivate" discovered plugins. (PluginApi implemented).

## Phase 2: Core Service Integration
- [x] **Virtual Filesystem Support**:
    - [x] Refactor `bizdefs.SetupBizDef` to accept an `fs.FS` instead of direct file paths.
    - [x] Create a "Multi-FS" that combines local `bizdefs/` and mounted plugin JARs.
- [x] **Dynamic Agentic Registry**:
    - [x] Refactor `services.NewChatService` to support dynamic addition of tools and agents.
    - [x] Implement a logic to merge `agentic.yaml` fragments from plugins into the main `Registry`.

## Phase 3: Polyglot Sandbox & Host API
- [x] **Sandbox Bridge**:
    - [x] Implement the `sandbox://` handler for `agentic` tools.
    - [x] Implement the `wasm://` handler for WASM-based tools.
- [x] **Host API Injection**:
    - [x] Define the `AIGenHostAPI` Go interface.
    - [x] Implement language-specific bindings for JavaScript, Lua, and Starlark. (Dispatcher and HostAPI implemented).
    - [x] Create a WASM host module for `wazero`. (Mocked in POC).

## Phase 4: A2UI & Assets Extension
- [x] **Plugin Asset Serving**:
    - [x] Update `api.StaticApi` to serve files from plugin JARs (e.g., `/_plugins/<id>/*`).
- [x] **Dynamic Component Loading**:
    - [x] Update the A2UI frontend to support loading component definitions and scripts from plugin-provided paths. (Static API bridge implemented).

## Phase 5: Testing & Validation
- [x] **Plugin SDK**: Create a small CLI tool or script to help developers package and sign their Applets. (cmd/bundle implemented).
- [x] **Integration Tests**:
    - [x] Test loading a sample "Weather" plugin that provides a BizDef, an Agent, and a JS tool.
    - [x] Verify that signature mismatches correctly block loading.
    - [x] Verify resource limits in the sandbox.

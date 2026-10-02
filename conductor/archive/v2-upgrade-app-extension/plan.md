# Implementation Plan: V2 Alignment and App-Extension Rename

This plan details the steps required to transition AIGenApp to ADK v2 / agentic v2 and rename plugins to app-extensions.

## Phase 1: Dependencies & V2 Alignment
- [x] Add `replace google.golang.org/adk => /home/innomon/sandbox/adk-go` in `go.mod`.
- [x] Run `go mod tidy`.
- [x] Investigate and fix any compilation issues arising from ADK v2 APIs (e.g. `CallbackContext` or `InvocationContext` signature changes).

## Phase 2: Refactoring package `core/plugins` to `core/app_extensions`
- [x] Rename directory `core/plugins` to `core/app_extensions`.
- [x] Rename Go files and package declarations to `package app_extensions`:
  - `service.go` -> `service.go`
  - `models.go` -> `models.go`
  - `sandbox.go` -> `sandbox.go`
  - `host_api.go` -> `host_api.go`
  - `plugin_integration_test.go` -> `app_extension_integration_test.go`
- [x] Perform systematic rename inside the new package:
  - `PluginService` -> `AppExtensionService`
  - `PluginInfo` -> `AppExtensionInfo`
  - `PluginManifest` -> `AppExtensionManifest`
  - `PluginStatus` -> `AppExtensionStatus` (e.g. `StatusActive`, `StatusUntrusted`)
  - Update `PluginID` properties to `ExtensionID` in `PermissionGrant`, `VaultEntry`, etc.
- [x] Rename files and API handler structs:
  - `core/api/plugin_api.go` -> `core/api/app_extension_api.go` (Struct: `PluginApi` -> `AppExtensionApi`)
- [x] Update interface `IPluginProvider` -> `IAppExtensionProvider` in `core/agentic/agents/router_agent.go`.

## Phase 3: Configuration & Filesystem Renames
- [x] Update `framework/config.go` (`PluginsDir` -> `AppExtensionsDir`, env var `FORMCMS_PLUGINS_DIR` -> `FORMCMS_APP_EXTENSIONS_DIR`).
- [x] Rename folder `plugins/` to `app-extensions/`.
- [x] Rename folder `sample-plugin/` to `sample-app-extension/` (and update manifest identifiers/paths inside it).
- [x] Update `config.yaml` and `config.yaml.sample` (`plugins_dir: plugins` -> `app_extensions_dir: app-extensions`).

## Phase 4: API Routing & Bundler Alignment
- [x] Update routes in `core/api/app_extension_api.go`: `/api/plugins` -> `/api/app-extensions`.
- [x] Update static serving logic in `core/api/static_api.go`: prefix `/_plugins/{id}/*` -> `/_extensions/{id}/*`.
- [x] Update bundler tool `cmd/bundle/main.go`:
  - `PLUGIN.SF` -> `EXTENSION.SF`
  - `PLUGIN.RSA` -> `EXTENSION.RSA`
- [x] Update `inspectJar` signature checks in `core/app_extensions/service.go` to accept both new signatures and legacy ones.

## Phase 5: Verification & Testing
- [x] Update integration tests in `core/app_extensions/app_extension_integration_test.go`.
- [x] Run `go test ./...` and verify clean build.
- [x] Package `sample-app-extension` using the new bundler and run the application to verify discovery, verification, mounting, and execution in the sandbox.

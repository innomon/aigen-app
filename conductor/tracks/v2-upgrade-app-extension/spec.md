# Specification: V2 Alignment and App-Extension Rename

## Background
AIGenApp is built on the Agentic Development Kit (ADK) specification. 
We are upgrading to ADK 2.0 and agentic v2.0.0. The Go SDK v2 is locally available at `/home/innomon/sandbox/adk-go`.

In ADK 2.0 and agentic v2, a "plugin" refers to agent execution pipeline callbacks/hooks (defined in `google.golang.org/adk/plugin`). However, AIGenApp currently uses the word "plugin" to denote dynamic, signed jar/zip archives containing BizDef schemas, UI assets, and sandboxed scripts. 

To avoid namespace collision and architectural confusion between ADK pipelines and AIGenApp's modular components, we will rename the modular components to **App Extensions** (or "extensions").

## Objectives
1. **Dependency Upgrade**: Align go.mod dependencies with ADK v2 and agentic v2, mapping `google.golang.org/adk` to the local v2 SDK path `/home/innomon/sandbox/adk-go`.
2. **Terminology Separation**: Systematically rename all internal references to the dynamic modular components from `Plugin` to `App Extension` (represented as `app-extension` in paths/directories, and `AppExtension` in Go code).

## Scope

### 1. File & Directory Renames
- Rename directory `/plugins/` (where jar packages reside) to `/app-extensions/`.
- Rename directory `/sample-plugin/` to `/sample-app-extension/`.
- Rename Go package directory `core/plugins/` to `core/app_extensions/` (Go package name will be `app_extensions`).
- Rename files within `core/plugins/` (e.g., `plugin_integration_test.go` -> `app_extension_integration_test.go`).
- Update bundling tool `/cmd/bundle/main.go` to package signed archives as extensions.

### 2. Go Code Refactoring
- Rename the package `plugins` to `app_extensions`.
- Rename structs:
  - `PluginService` -> `AppExtensionService`
  - `PluginInfo` -> `AppExtensionInfo`
  - `PluginManifest` -> `AppExtensionManifest`
  - `PluginStatus` -> `AppExtensionStatus`
  - `PluginApi` -> `AppExtensionApi`
- Interfaces & Fields:
  - `IPluginProvider` (in `router_agent.go`) -> `IAppExtensionProvider`
  - Update `PluginID` fields to `ExtensionID` in structs like `PermissionGrant`, `VaultEntry`, `AppExtensionInfo`.
  - Update variables and fields: `plugins` map, `pluginService` instances, `pluginsDir`, etc.

### 3. API Routes
- REST management endpoints: `/api/plugins` -> `/api/app-extensions`.
- Static/asset endpoints: `/_plugins/{id}/*` -> `/_extensions/{id}/*` (or `/_app-extensions/{id}/*`).

### 4. Configuration & Environment Variables
- Configuration yaml: `plugins_dir` -> `app_extensions_dir`.
- Default configuration fallback: `"plugins"` -> `"app-extensions"`.
- Environment variable: `FORMCMS_PLUGINS_DIR` -> `FORMCMS_APP_EXTENSIONS_DIR`.

### 5. Packaging & Signature Formats
- Packaged archives will produce signature files:
  - `META-INF/PLUGIN.SF` -> `META-INF/EXTENSION.SF`
  - `META-INF/PLUGIN.RSA` -> `META-INF/EXTENSION.RSA`
- The inspector logic in the service will search for both new signature files (`EXTENSION.SF`/`EXTENSION.RSA`) and fall back to the old signature files (`PLUGIN.SF`/`PLUGIN.RSA`) to maintain compatibility with legacy plugins if necessary.

## Constraints & Compatibility
- **ADK 2.0 Integration**: Go package imports of `google.golang.org/adk` must match ADK 2.0 semantics (e.g., handling unified `CallbackContext` which incorporates action confirmation, memory search, etc.).
- **Gofmt & Goimports**: All refactored files must be styled cleanly according to Go standards.
- **Go Mod Tidy**: Ensure `go mod tidy` successfully cleans up old dependencies.

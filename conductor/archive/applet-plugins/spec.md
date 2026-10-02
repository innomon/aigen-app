# Specification: Signed Applet Plugin System

## 1. Overview
The **Applet** plugin system enables dynamic extension of `AIGenApp` via signed JAR files. It allows third-party developers to package business definitions (`BizDef`), agentic workflows, sandboxed logic, and custom UI components into a single, verifiable unit.

## 2. Core Requirements
- **Verification**: Plugins must be digitally signed. Only signers on a "Public Key Whitelist" are accepted by default.
- **Dynamic Loading**: Plugins can be added/removed at runtime without restarting the main server (where possible).
- **Isolation**: Custom logic (JS, Lua, Starlark, WASM) must run in a secure sandbox with restricted resource access.
- **Interoperability**: Plugins must seamlessly integrate with existing `SchemaService` (BizDefs) and `ChatService` (Agentic ADK).

## 3. Architecture

### 3.1. The Applet JAR Structure
```text
my-plugin.jar
├── META-INF/                 # Verification & Metadata
├── bizdef/                   # Entities, schemas, and test data
├── agentic/                  # agentic.yaml extensions
├── a2ui/                     # Custom UI component definitions
├── wwwroot/                  # Static assets (JS/CSS for A2UI)
├── scripts/                  # Sandboxed scripts (Polyglot)
└── wasm/                     # WASM-based tools/agents
```

### 3.2. Plugin Lifecycle
1. **Discovery**: `PluginService` scans the `/plugins` folder for `.jar` files.
2. **Verification**: 
   - Check JAR integrity via `MANIFEST.MF`.
   - Verify digital signature against a whitelist of trusted certificates.
3. **Mounting**:
   - Register BizDefs with `SchemaService`.
   - Merge `agentic.yaml` into the `Registry`.
   - Add asset paths to `StaticApi`.
4. **Execution**:
   - Tools defined in `agentic.yaml` are executed via the `agentic` library's sandbox VMs.
   - Host API is injected into VMs to provide managed access to CMS services.

### 3.3. Polyglot Sandbox
Support all VM engines provided by the `agentic` library:
- **JavaScript**: via `goja`/`otto`.
- **Lua**: via `gopher-lua`.
- **Starlark**: via `starlark-go`.
- **WASM**: via `wazero`.

## 4. Host API (`AIGenHostAPI`)
A standardized bridge exposed to sandboxed scripts:
- `cms.Get(entity, id)`: Retrieve a record.
- `cms.List(entity, params)`: Search/List records.
- `cms.Save(entity, data)`: Insert/Update a record (subject to permissions).
- `a2ui.Update(id, data)`: Push updates to the dynamic UI.
- `log.Info/Error(msg)`: Audited logging.

## 5. Security Model
- **Signature Enforcement**: Unsigned or untrusted plugins remain inactive.
- **Resource Capping**: Memory and CPU limits per VM instance.
- **Capability-Based Security**: Plugins must declare requested permissions (e.g., `access:bizdef:crm`) in metadata.

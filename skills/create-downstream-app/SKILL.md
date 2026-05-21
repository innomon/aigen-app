---
name: create-downstream-app
description: Guides and automates the creation of a fully customized downstream application using AIGenApp as an upstream dependency. Enables full UI and BizDef overrides via configurations and plugins, without forking or modifying the core codebase.
---
# Creating a Downstream Application with AIGenApp

This skill guides and automates the creation of a customized downstream application without forking, copying, or modifying the core `AIGenApp` (upstream) repository.

## 1. Overview & Architecture

To customize AIGenApp for a downstream project, you utilize two main configuration-driven extensibility layers:

```mermaid
graph TD
    Downstream[Downstream Project Folder] --> CustomUI[Custom UI Folder: OverlayFS]
    Downstream --> Plugins[Plugins Folder: BizDef/Agent JARs]
    Downstream --> EnvConfig[.env Configuration File]
    
    EnvConfig --> Upstream[AIGenApp Engine - Upstream]
    CustomUI --> Upstream
    Plugins --> Upstream
    
    Upstream --> DB[(Single-Table Postgres Store)]
```

### 1. Presentation Layer Overrides (OverlayFS)
- **Virtual Overlay Filesystem**: The web server merges the built-in embedded UI assets (React Admin panels, Page builder, etc.) with a local directory path of your choice.
- **Priority serving**: Any asset placed in your custom UI folder takes absolute priority. If a file is requested but not overridden, it seamlessly falls back to the embedded system assets.
- **Configuration**: Set `FORMCMS_CUSTOM_UI_PATH` in your `.env` or configuration.

### 2. Business Logic & Schema Overrides (Applet Plugins)
- **Unified Extension Point**: Instead of physical files in `bizdefs/` or standard migrations, you package custom entities, GraphQL schemas, migration timelines (`evolution.json`), scripts (JS/Lua), and tools into signed `.jar` plugin applets.
- **Smart Evolution Engine**: When a plugin is loaded, the engine dynamically performs deep JSON structural comparison on schemas, JIT-migrating target definitions in the single-table `aigen_records` database only when changes actually occur. No database locks or SQL modifications are performed.
- **Configuration**: Set `FORMCMS_PLUGINS_DIR` to your compiled plugins folder.

---

## 2. Automated Downstream Scaffolding

To quickly scaffold a downstream project structure, run the automated script provided by this skill:

```bash
./skills/create-downstream-app/scripts/scaffold_downstream.sh <app-path>
```

### Scaffolded Directory Structure
The script generates a standalone directory layout for your downstream application:
```text
<app-path>/
├── .env                   # Configuration parameters pointing to your paths
├── custom_ui/             # Place custom UI overrides here (OverlayFS)
│   └── admin/
│       └── index.html     # Overridden admin index dashboard
├── plugins/               # Place compiled plugin JARs here
├── wwwroot/               # Runtime uploads and static uploads directory
├── README.md              # Documentation on how to develop and run
└── run.sh                 # Convenient script to start the upstream engine
```

---

## 3. Step-by-Step Customization Guide

### Step 3.1: Configure Your Downstream Project
Modify the generated `.env` file to customize ports, database connections, and paths:
```env
PORT=5000
FORMCMS_DB_DSN=postgres://admin:secret@localhost:5432/my_downstream_db
FORMCMS_CUSTOM_UI_PATH=./custom_ui
FORMCMS_PLUGINS_DIR=./plugins
FORMCMS_WWW_ROOT=./wwwroot
```

### Step 3.2: Customize the User Interface
To override any page in the admin interface (served at `/admin/*`), place the file with the matching relative path inside `custom_ui/`.
For example, to customize the dashboard header or style, you can edit `custom_ui/admin/index.html`. When you load `http://localhost:5000/admin/`, this file will be loaded instead of the embedded core admin panel.

### Step 3.3: Extend the Business Logic & Schemas
To define custom tables/schemas, create a signed JAR plugin:
1. Create a `bizdef/schemas/` folder in your plugin workspace.
2. Define your entity schemas (e.g. `bizdef/schemas/my_entity.json`).
3. (Optional) Provide a `bizdef/evolution.json` for JIT database migrations.
4. Package the directory as a ZIP and rename the extension to `.jar` (ensure it is signed if running in production verified modes).
5. Copy the `.jar` into the `plugins/` directory of your downstream project.
6. The schema is automatically mounted, compared, and evolved in the database during startup.

---

## 4. Running the Application

To run the application, ensure the `.env` file is loaded and launch the upstream server:
```bash
./run.sh
```
This executes the upstream server binary, applying all configurations, serving overrides from `custom_ui/`, and mounting BizDef plugins from `plugins/`.

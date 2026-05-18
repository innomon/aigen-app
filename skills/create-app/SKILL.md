---
name: create-app
description: Automates the creation of new downstream apps (e.g., apps/new-app) based on the AIGenApp conventions. Use this skill when tasked with generating the structure, manifests, and boilerplate for a new business app.
---
# Create App

## Overview
This skill automates the generation of a new downstream app following the project's architecture.

## Workflow
1. Use this skill when asked to "create a new app".
2. The skill will create the necessary directory structure under `apps/`:
   - `apps/<app-name>/`
   - `apps/<app-name>/data/`
   - `apps/<app-name>/docs/`
   - `apps/<app-name>/schemas/`
   - `apps/<app-name>/migrations/`
3. It will generate a default `app_def.json` template.
4. It will register the new app in `apps/apps.json`.

## Usage
Run the following script to scaffold the app:
```bash
./skills/create-app/scripts/scaffold_app.sh <app-name>
```

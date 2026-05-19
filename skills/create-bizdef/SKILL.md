---
name: create-bizdef
description: Automates the creation of new downstream BizDefs (e.g., bizdefs/new-bizdef) based on the AIGenApp conventions. Use this skill when tasked with generating the structure, manifests, and boilerplate for a new business definition.
---
# Create BizDef

## Overview
This skill automates the generation of a new downstream BizDef following the project's architecture.

## Workflow
1. Use this skill when asked to "create a new BizDef".
2. The skill will create the necessary directory structure under `bizdefs/`:
   - `bizdefs/<bizdef-name>/`
   - `bizdefs/<bizdef-name>/data/`
   - `bizdefs/<bizdef-name>/docs/`
   - `bizdefs/<bizdef-name>/schemas/`
   - `bizdefs/<bizdef-name>/migrations/`
   - `bizdefs/<bizdef-name>/evolution.json` (New: evolution manifest)
   - `bizdefs/<bizdef-name>/evolution.md` (New: evolution history)
3. It will generate a default `bizdef.json` template.
4. It will register the new BizDef in `bizdefs/bizdefs.json`.

## Usage
Run the following script to scaffold the BizDef:
```bash
./skills/create-bizdef/scripts/scaffold_bizdef.sh <bizdef-name>
```

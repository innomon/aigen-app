---
name: mermaid-to-image
description: Converts Mermaid diagrams into PNG or SVG images. Use this skill when you need to render technical diagrams defined in Mermaid syntax into visual assets for documentation.
---
# Mermaid to Image

## Overview
This skill provides a workflow for converting Mermaid.js diagrams into image files (PNG/SVG).

## How to use
1. Save your Mermaid code to a `.mmd` file.
2. Use the `mmdc` command to convert:
   ```bash
   mmdc -i <input-file>.mmd -o <output-file>.<ext>
   ```

## Requirements
- `mmdc` (Mermaid CLI) must be installed.
- Ensure Chromium is installed if using the NPM version of `mermaid-cli`.

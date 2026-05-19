#!/bin/bash
set -e

BIZDEF_NAME=$1
if [ -z "$BIZDEF_NAME" ]; then
    echo "Usage: $0 <bizdef-name>"
    exit 1
fi

BIZDEF_DIR="bizdefs/$BIZDEF_NAME"
mkdir -p $BIZDEF_DIR/{data,docs,migrations,schemas}

# Manifest
cat <<JSON > $BIZDEF_DIR/bizdef.json
{
  "name": "$BIZDEF_NAME",
  "display_name": "$BIZDEF_NAME",
  "description": "New BizDef for $BIZDEF_NAME",
  "context": "Context for $BIZDEF_NAME bizdef.",
  "roles": ["Admin"],
  "entities": {}
}
JSON

# Evolution Manifest
echo "{}" > $BIZDEF_DIR/evolution.json

# Evolution History
cat <<MD > $BIZDEF_DIR/evolution.md
# $BIZDEF_NAME Evolution History

## Initial Version (v1)
- Initial scaffold created.
MD

# Register in bizdefs.json if not present
BIZDEFS_FILE="bizdefs/bizdefs.json"
if [ -f "$BIZDEFS_FILE" ]; then
    if ! grep -q "\"$BIZDEF_NAME\"" "$BIZDEFS_FILE"; then
        # Insert the new bizdef at the beginning of the enabled_bizdefs array
        sed -i "s/\"enabled_bizdefs\": \[/\"enabled_bizdefs\": [\n    \"$BIZDEF_NAME\",/" "$BIZDEFS_FILE"
    fi
else
    echo "Warning: $BIZDEFS_FILE not found. BizDef not registered."
fi

echo "Successfully created BizDef: $BIZDEF_NAME (including evolution tracking)"

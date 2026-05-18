#!/bin/bash
set -e

APP_NAME=$1
if [ -z "$APP_NAME" ]; then
    echo "Usage: $0 <app-name>"
    exit 1
fi

APP_DIR="apps/$APP_NAME"
mkdir -p $APP_DIR/{data,docs,migrations,schemas}

cat <<JSON > $APP_DIR/app_def.json
{
  "name": "$APP_NAME",
  "display_name": "$APP_NAME",
  "description": "New app for $APP_NAME",
  "entities": {}
}
JSON

# Register in apps.json if not present
if ! grep -q "$APP_NAME" apps/apps.json; then
    sed -i "s/\[/\[\n    \"$APP_NAME\",/" apps/apps.json
fi

echo "Successfully created app: $APP_NAME"

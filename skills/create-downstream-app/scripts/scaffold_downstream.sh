#!/bin/bash
set -e

APP_PATH=$1
if [ -z "$APP_PATH" ]; then
    echo "Usage: $0 <downstream-app-path>"
    exit 1
fi

UPSTREAM_DIR=$(pwd)

echo "Scaffolding downstream application at: $APP_PATH"
mkdir -p "$APP_PATH"/{custom_ui/admin,plugins,wwwroot}

# 1. Create .env file
cat <<ENV > "$APP_PATH/.env"
# Downstream Application Configuration
PORT=5000
FORMCMS_DB_DSN=postgres://admin:admin@localhost:5433/aigen?sslmode=disable
FORMCMS_CUSTOM_UI_PATH=./custom_ui
FORMCMS_PLUGINS_DIR=./plugins
FORMCMS_WWW_ROOT=./wwwroot
ENV

# 2. Create sample Custom UI Override
cat <<HTML > "$APP_PATH/custom_ui/admin/index.html"
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Custom Downstream Dashboard</title>
    <style>
        body {
            font-family: 'Outfit', -apple-system, sans-serif;
            background: linear-gradient(135deg, #0f172a, #1e1b4b);
            color: #f8fafc;
            display: flex;
            align-items: center;
            justify-content: center;
            height: 100vh;
            margin: 0;
        }
        .container {
            text-align: center;
            padding: 2.5rem;
            background: rgba(255, 255, 255, 0.05);
            backdrop-filter: blur(10px);
            border-radius: 16px;
            border: 1px solid rgba(255, 255, 255, 0.1);
            box-shadow: 0 10px 30px rgba(0, 0, 0, 0.5);
            max-width: 480px;
        }
        h1 {
            background: linear-gradient(to right, #6366f1, #a855f7);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            font-size: 2.5rem;
            margin-bottom: 0.5rem;
        }
        p {
            color: #94a3b8;
            line-height: 1.6;
        }
        .badge {
            background: rgba(99, 102, 241, 0.2);
            color: #818cf8;
            padding: 0.35rem 0.75rem;
            border-radius: 9999px;
            font-size: 0.85rem;
            font-weight: bold;
            display: inline-block;
            margin-top: 1rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Welcome Downstream!</h1>
        <p>This page is served dynamically via AIGenApp's <strong>OverlayFS</strong> configuration-driven architecture.</p>
        <p>You can customize the entire administrator panel by adding files inside the <code>custom_ui/admin/</code> folder.</p>
        <span class="badge">OverlayFS Active</span>
    </div>
</body>
</html>
HTML

# 3. Create run.sh
cat <<'RUN' > "$APP_PATH/run.sh"
#!/bin/bash
# Convenient runner script for the downstream application

# Load environment variables from .env
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi

echo "Starting Upstream AIGenApp Engine..."
echo "Custom UI Overrides: $FORMCMS_CUSTOM_UI_PATH"
echo "BizDef Plugins: $FORMCMS_PLUGINS_DIR"
echo "Listening on port: $PORT"

# Execute the upstream engine pointing to this directory's configurations
cd "UPSTREAM_DIR_PLACEHOLDER"
if [ -f ./aigen-app ]; then
    # Run the compiled binary if available
    ./aigen-app
else
    # Fallback to go run
    go run main.go
fi
RUN

sed -i "s|UPSTREAM_DIR_PLACEHOLDER|$UPSTREAM_DIR|g" "$APP_PATH/run.sh"
chmod +x "$APP_PATH/run.sh"

# 4. Create README.md
cat <<'MD' > "$APP_PATH/README.md"
# Downstream Application

A zero-fork downstream application built on top of the **AIGenApp** headless CMS and dynamic application framework.

## Getting Started

1. **Configure Environment**: Review and adjust database credentials and configurations in `.env`.
2. **Launch the Server**: Run the convenient startup script:
   ```bash
   ./run.sh
   ```
3. **Verify Custom UI**: Navigate to `http://localhost:5000/admin/` in your browser. You will see the customized dashboard served from `custom_ui/admin/index.html` via the **OverlayFS** engine.
4. **Deploy Custom BizDefs**: Package your schemas and agents in a signed JAR file and copy it into the `plugins/` directory to seamlessly migrate and register your custom entities.

## Folder Layout

- `custom_ui/`: Customize static assets (HTML/CSS/JS) and the admin panels here. Files take absolute priority over embedded upstream assets.
- `plugins/`: Place your Applet JARs here to inject dynamic BizDefs, sandboxed script triggers, and agent profiles.
- `wwwroot/`: Runtime storage directory for uploads and dynamic assets.
MD

echo "---------------------------------------------------------"
echo "Scaffold successfully created at: $APP_PATH"
echo "To run your downstream application, execute:"
echo "  cd $APP_PATH && ./run.sh"
echo "---------------------------------------------------------"

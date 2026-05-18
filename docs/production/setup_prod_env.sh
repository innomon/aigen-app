#!/bin/bash

# Setup script for AIGenApp production environment variables
# Usage: sudo ./setup_prod_env.sh [user_name]

USER_NAME=${1:-aigen-user}
CONFIG_DIR="/etc/aigen"
ENV_FILE="$CONFIG_DIR/aigen.env"

echo "Creating configuration directory: $CONFIG_DIR"
sudo mkdir -p $CONFIG_DIR

if [ ! -f "$ENV_FILE" ]; then
    echo "Creating template environment file: $ENV_FILE"
    sudo bash -c "cat > $ENV_FILE" <<EOF
# --- Production Environment Variables ---
FORMCMS_DB_DSN="postgres://user:password@localhost:5432/aigen_db?sslmode=disable"
GEMINI_API_KEY=""
GOOGLE_API_KEY=""
OPENAI_API_KEY=""
PORT="5000"
DOMAIN=""
FORMCMS_APPS_DIR="apps"
FORMCMS_WWW_ROOT="wwwroot"
AGENT_ENCRYPTION_KEY="$(openssl rand -hex 16)"
EOF
else
    echo "Environment file already exists at $ENV_FILE, skipping creation."
fi

echo "Setting strict permissions on $ENV_FILE"
sudo chown root:root $ENV_FILE
sudo chmod 600 $ENV_FILE

echo "--------------------------------------------------------"
echo "Setup complete!"
echo "1. Edit $ENV_FILE with your production secrets."
echo "2. Copy aigen.service to /etc/systemd/system/"
echo "3. Run 'sudo systemctl daemon-reload'"
echo "4. Run 'sudo systemctl enable --now aigen'"
echo "--------------------------------------------------------"

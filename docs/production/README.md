# Production Deployment Scripts

This directory contains templates and scripts to help you deploy AIGenApp securely in a production environment, especially when running behind a firewall or Cloudflare proxy.

## Files

- **`aigen.service`**: A systemd unit file template.
- **`setup_prod_env.sh`**: A helper script to create the `/etc/aigen/` directory and set up a secure, root-owned environment file with restricted permissions (`600`).

## Deployment Steps

1.  **Build the App**:
    ```bash
    go build -o aigen-app .
    ```

2.  **Run the Setup Script**:
    ```bash
    chmod +x docs/production/setup_prod_env.sh
    sudo ./docs/production/setup_prod_env.sh
    ```

3.  **Configure Secrets**:
    Edit `/etc/aigen/aigen.env` and fill in your production database credentials and API keys.

4.  **Install the Service**:
    ```bash
    sudo cp docs/production/aigen.service /etc/systemd/system/
    # Update the WorkingDirectory and ExecStart paths in the file if necessary
    sudo nano /etc/systemd/system/aigen.service
    ```

5.  **Start and Enable**:
    ```bash
    sudo systemctl daemon-reload
    sudo systemctl enable --now aigen
    ```

6.  **Verify**:
    ```bash
    sudo systemctl status aigen
    journalctl -u aigen -f
    ```

## Cloudflare Integration

If using Cloudflare, we recommend using **Cloudflare Tunnels** (`cloudflared`). This allows you to serve your application without opening any inbound ports on your firewall.

- [Cloudflare Tunnel Documentation](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/)

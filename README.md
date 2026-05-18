# AIGenApp

A headless CMS and dynamic application framework in Go, evolved from the FormCMS original C# implementation. It uses a single-table JSON store for maximum schema flexibility.

## Features

- **Agentic Workflows**: Integrated multi-agent system powered by Gemini models for orchestrating tasks.
  - **Router Agent**: Intelligently routes user requests between specialized sub-agents.
  - **App Agent**: Manages and queries app data and schemas autonomously.
  - **UI Agent**: Dynamically updates the A2UI dashboard components based on user interactions and data changes.
- **App Capability Discovery**: Built-in `app_def.json` and context file framework allowing LLM agents to dynamically discover app purpose, roles, and entity relationships.
- **A2UI Protocol**: Real-time Agent-to-User Interface for streaming backend-driven UI updates (SSE) using a high-performance adjacency list model.
- **Multi-Channel Communication**: Seamlessly interact with users via WhatsApp, Email, Signal, Telegram, X.com, and Bluesky.
  - **Authenticated Channels**: Link verified platform identities to user profiles.
  - **E-trail Logging**: Secure audit logs with IP and User Agent for non-repudiation.
  - **Guest Support**: Configurable guest access across different channels.
- **Frappe/ERPNext Integration**: Built-in support for importing and mapping Frappe Doctypes to native app schemas.
- **Advanced RBAC**: Granular Role-Based Access Control with field-level and row-level security filters managed via JSON metadata.
- **Schema-on-Read Data Modeling**: Define entities and attributes dynamically. All data is stored in a highly flexible single-table JSON schema (`aigen_records`), making migrations a thing of the past.
- **REST & GraphQL APIs**: Auto-generated CRUD and GraphQL endpoints.
- **File Storage**: Local and S3 support with image processing.
- **Social Engagement**: Built-in likes, bookmarks, and comments.
- **Embedded UI**: React Admin panel, GrapesJS page builder, and dynamic A2UI renderer included.

## Getting Started

AIGenApp is a reusable Go framework. To use it, create a new Go project and import the framework.

### Prerequisites

- Go 1.25+

### Creating a Project

1. **Initialize a new Go module:**
```bash
mkdir my-app
cd my-app
go mod init my-app
```

2. **Create a `main.go` file:**
```go
package main

import (
	"log"
	"os"

	"github.com/innomon/aigen-app/framework"
)

func main() {
	configPath := ""
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	config, err := framework.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	if err := framework.Start(config); err != nil {
		log.Fatalf("Framework failed to start: %v", err)
	}
}
```

3. **Create a `config.yaml` file:**
```yaml
apps_dir: "apps"
www_root: "wwwroot"
database_dsn: "postgres://user:pass@localhost:5432/aigen_db"
domain: ""
port: "5000"
agentic_config_path: "agentic.yaml"
```

4. **Run the server:**
```bash
go run main.go config.yaml
```

The server will start on `http://localhost:5000`.

## Deployment

### Environment Variables

| Variable | Description | Default |
| :--- | :--- | :--- |
| `DOMAIN` | Your external domain name (e.g., `example.com`). If set, enables automatic HTTPS via `autocert`. | `""` |
| `PORT` | The port to listen on for HTTP. Ignored if `DOMAIN` is set. | `5000` |
| `FORMCMS_WWW_ROOT` | The directory for serving static files and storing uploaded assets. | `wwwroot` |
| `FORMCMS_APPS_DIR` | The directory where app definitions and data are located. | `apps` |
| `FORMCMS_DB_DSN` | Database connection string (e.g., `postgres://user:pass@host:port/db`). | `""` |
| `FORMCMS_CONFIG_PATH` | Path to the YAML/JSON configuration file. | `""` |
| `FORMCMS_AGENTIC_CONFIG_PATH` | Path to the `agentic.yaml` configuration for LLM workflows. | `agentic.yaml` |
| `GEMINI_API_KEY` | API key for Google Gemini models. | `""` |
| `GOOGLE_API_KEY` | Alternative API key variable for Google Gemini/Vertex AI. | `""` |
| `OPENAI_API_KEY` | API key for OpenAI models. | `""` |
| `OLLAMA_BASE_URL` | Base URL for local Ollama models (e.g., `http://localhost:11434/v1`). | `""` |
| `AGENT_ENCRYPTION_KEY` | 32-byte key for state encryption in specific agent types (e.g., GnoVM). | `""` |
| `BYPASS_AUTH` | Set to `true` to bypass JWT authentication for local development. | `false` |
| `AWS_ACCESS_KEY_ID` | AWS access key for S3 storage. | `""` |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key for S3 storage. | `""` |
| `AWS_REGION` | AWS region for S3 storage. | `us-east-1` |
| `GOOGLE_APPLICATION_CREDENTIALS` | Path to Google Cloud service account JSON for GCS/Firestore. | `""` |

### Production Best Practices

#### 1. Local Production (Behind Firewall / Cloudflare)
For deployments on a Linux VPS or On-Prem server behind a Cloudflare proxy/tunnel, use **systemd** with a restricted environment file.

1.  **Create a Secure Env File**: Store your secrets in a system-protected directory (e.g., `/etc/aigen/aigen.env`) and set permissions to `600`.
    ```bash
    FORMCMS_DB_DSN="postgres://user:pass@localhost:5432/aigen_prod"
    GEMINI_API_KEY="your-prod-key"
    ```
2.  **Configure systemd**: Create a service unit (e.g., `/etc/systemd/system/aigen.service`) that references this file:
    ```ini
    [Service]
    ExecStart=/path/to/aigen-app
    EnvironmentFile=/etc/aigen/aigen.env
    Restart=always
    User=aigen-user
    ```

#### 2. Cloud Native (Containerized)
When deploying to AWS (ECS/EKS), GCP (Cloud Run/GKE), or Azure:

1.  **Use Secret Managers**: Do not bake `.env` files into your Docker image. Instead, use AWS Secrets Manager or GCP Secret Manager.
2.  **Injection**: Configure your orchestrator to inject these secrets as environment variables at runtime.
    *   **Kubernetes**: Use `secretKeyRef` or the [External Secrets Operator](https://external-secrets.io/).
    *   **Cloud Run**: Directly map GCP Secrets to environment variables in the service configuration.

## Static File Serving

AIGenApp serves static files from two main sources:

1.  **Embedded UI Assets**: The core admin panel and static system assets are embedded in the binary and served under `/admin` and `/static`.
2.  **Dynamic Static Files**: Files stored in the directory specified by `www_root` (default `wwwroot`) are served via HTTP:
    *   **Uploaded Assets**: By default, uploaded files are stored in `wwwroot/files` and are served under the `/files/*` path.
    *   **Custom Assets**: Any directory or file placed within `www_root` can be accessed if a corresponding route is registered. By default, the `/files/*` route is mapped to the `www_root` directory, meaning `wwwroot/files/logo.png` is available at `/files/logo.png`.

## Root Route Handling

The root route (`/`) is dynamically handled by the `PageApi` and follows a tiered resolution logic:

1.  **Dynamic "Home" Page**: It first looks for a page entity in the database specifically named `home`. If found, it renders this page using the application's Handlebars-based template engine.
2.  **Role-Based Dashboard**: If no `home` page exists, the system checks the current user's role (or the `guest` role if not authenticated). If that role has a `DashboardPageId` configured, it renders that specific page.
3.  **Admin Redirect**: If neither of the above is found, the system redirects the user to the admin interface (`/admin/list.html`).

## Framework Structure

- `framework`: The main entry point `Start()` function to initialize the application.
- `apps`: Pre-packaged data models, test data, and UI logic that load dynamically.
- `core/api`: HTTP handlers and routing.
- `core/descriptors`: Data models and schema definitions.
- `core/services`: Business logic and orchestration.
- `infrastructure/filestore`: File storage implementations (Local, S3).
- `infrastructure/relationdbdao`: Database abstraction layer (PostgreSQL and Firestore using single JSON store).
- `utils`: Shared utilities and data models.

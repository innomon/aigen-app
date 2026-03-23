# AiGen CMS

A headless CMS framework in Go, rewritten from the [FormCMS](https://github.com/formcms/formcms) original C# implementation (**NOTE:** no upstream dependency).

## Features

- **Agentic Workflows**: Integrated multi-agent system powered by Gemini models for orchestrating tasks.
  - **Router Agent**: Intelligently routes user requests between specialized sub-agents.
  - **CMS Agent**: Manages and queries CMS data and schemas autonomously.
  - **UI Agent**: Dynamically updates the A2UI dashboard components based on user interactions and data changes.
- **App Capability Discovery**: Built-in `app_def.json` and context file framework allowing LLM agents to dynamically discover app purpose, roles, and entity relationships.
- **A2UI Protocol**: Real-time Agent-to-User Interface for streaming backend-driven UI updates (SSE) using a high-performance adjacency list model.
- **Multi-Channel Communication**: Seamlessly interact with users via WhatsApp, Email, Signal, Telegram, X.com, and Bluesky.
  - **Authenticated Channels**: Link verified platform identities to user profiles.
  - **E-trail Logging**: Secure audit logs with IP and User Agent for non-repudiation.
  - **Guest Support**: Configurable guest access across different channels.
- **Frappe/ERPNext Integration**: Built-in support for importing and mapping Frappe Doctypes to native CMS schemas.
- **Advanced RBAC**: Granular Role-Based Access Control with field-level and row-level security filters.
- **Dynamic Data Modeling**: Define entities and attributes via UI or API.
- **REST & GraphQL APIs**: Auto-generated CRUD and GraphQL endpoints.
- **File Storage**: Local and S3 support with image processing.
- **Social Engagement**: Built-in likes, bookmarks, and comments.
- **Embedded UI**: React Admin panel, GrapesJS page builder, and dynamic A2UI renderer included.

## Getting Started

AiGen CMS is now a reusable Go framework. To use it, create a new Go project and import the framework.

### Prerequisites

- Go 1.25+

### Creating a Project

1. **Initialize a new Go module:**
```bash
mkdir my-cms
cd my-cms
go mod init my-cms
```

2. **Create a `main.go` file:**
```go
package main

import (
	"log"
	"os"

	"github.com/innomon/aigen-cms/framework"
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
database_type: "SQLite"
database_dsn: "formcms.db"
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

### Automatic HTTPS (autocert)

FormCMS Go supports automatic TLS certificate provisioning via Let's Encrypt using `autocert`. To enable this:

1. Point your domain's DNS A/AAAA records to your server's IP.
2. Ensure ports `80` and `443` are open and not in use by other processes.
3. Run the application with the `DOMAIN` environment variable:

```bash
DOMAIN=yourdomain.com go run main.go config.yaml
```

The server will automatically handle HTTP-to-HTTPS redirection and store certificates in a local `certs` directory.

## Framework Structure

- `framework`: The main entry point `Start()` function to initialize the CMS.
- `apps`: Pre-packaged data models, test data, and UI logic that load dynamically.
- `core/api`: HTTP handlers and routing.
- `core/descriptors`: Data models and schema definitions.
- `core/services`: Business logic and orchestration.
- `infrastructure/filestore`: File storage implementations (Local, S3).
- `infrastructure/relationdbdao`: Database abstraction layer (PostgreSQL, SQLite).
- `utils`: Shared utilities and data models.

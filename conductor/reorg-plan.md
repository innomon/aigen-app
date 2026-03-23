# Reorganization Plan: Reusable Framework

The primary goal of this reorganization is to allow other developers to use this framework by creating a `main.go`, adding `apps`, and configuring agentic settings.

## Phase 1: Configuration Management
The first step is to define the new configuration structure that downstream projects will use to configure the CMS framework.

- [ ] **Create `framework` package:** Create a new directory `framework` (e.g., `framework/config.go`).
- [ ] **Define `Config` struct:** Include the necessary fields for downstream customization:
  ```go
  type Config struct {
      AppsDir           string `json:"apps_dir" yaml:"apps_dir"`
      WWWRoot           string `json:"www_root" yaml:"www_root"`
      DatabaseType      string `json:"database_type" yaml:"database_type"`
      DatabaseDSN       string `json:"database_dsn" yaml:"database_dsn"` // Database name or connection string
      Domain            string `json:"domain" yaml:"domain"`
      Port              string `json:"port" yaml:"port"`
      AgenticConfigPath string `json:"agentic_config_path" yaml:"agentic_config_path"`
  }
  ```
- [ ] **Implement Config Loader:** Write a function `LoadConfig(path string) (*Config, error)` to parse the YAML/JSON config file, with fallbacks to environment variables.

## Phase 2: Refactoring `main.go` to `framework/init.go`
Move the monolithic `main.go` script into a reusable framework lifecycle function.

- [ ] **Create `framework/init.go`:** Move the core logic of `main.go` into a new exported function: `func Start(cfg *Config) error`.
- [ ] **Dynamic Database Initialization:** Update the `relationdbdao.CreateDao(descriptors.SQLite, "formcms.db")` call to use `cfg.DatabaseType` and `cfg.DatabaseDSN`.
- [ ] **Dynamic Apps Loading:** Update `apps.LoadAppsConfig()` and the `apps.SetupApp()` loops to use `cfg.AppsDir` instead of hardcoded relative paths.
- [ ] **Dynamic Static Files:** Update `api.NewStaticApi()` and other components that rely on static files to serve from `cfg.WWWRoot`.
- [ ] **Dynamic Agentic Config:** Update `services.NewChatService("agentic.yaml", ...)` to use `cfg.AgenticConfigPath`.
- [ ] **Dynamic Server Port/Domain:** Replace `os.Getenv("PORT")` and `os.Getenv("DOMAIN")` with `cfg.Port` and `cfg.Domain`.

## Phase 3: Removing Hardcoded Paths in Core Packages
Ensure that underlying core packages don't rely on the current monolithic repository structure.

- [ ] **Update `apps` package:** Modify `LoadAppsConfig()` and `SetupApp()` to accept an absolute or relative directory path (`appsDir`) instead of assuming an `apps/` directory in the working directory.
- [ ] **Update `filestore` package:** Ensure local file storage respects directory configurations provided by the new config if needed.

## Phase 4: Create the new downstream `main.go`
Create a clean, minimalistic entry point that represents how a downstream project will use the framework.

- [ ] **Create new `main.go`:**
  - Parse the configuration file location.
  - Priority: 1. Command-line argument (`os.Args[1]`), 2. Local directory (e.g., `./config.yaml`), 3. Environment variable (e.g., `FORMCMS_CONFIG_PATH`).
  - Load the configuration.
  - Call `framework.Start(config)`.

## Phase 5: Testing & Cleanup
Verify that the standalone application still behaves as it originally did.

- [ ] **Create a sample `config.yaml`:** Create a default configuration in the repository root for testing.
- [ ] **Test the server startup:** Run `go run main.go config.yaml` to verify all APIs, databases, and apps load correctly.
- [ ] **Test external apps:** Temporarily move the `apps` and `wwwroot` directories outside of the project root and start the application using an updated `config.yaml` to ensure it works externally.
